package redisstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"server/internal/contract/state"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const nextPlayerIDKey = "game:next_player_id"
const optimisticLockRetries = 3
const settlementRetention = 7 * 24 * time.Hour

// Store 在 Redis 中持久化状态契约模型。
type Store struct {
	client *redis.Client
}

// SettleMatchRewards 使用乐观事务原子结算多人奖励与排行榜数据，并按结算 ID 保证幂等。
func (s *Store) SettleMatchRewards(ctx context.Context, input state.SettleMatchRewardsInput) (*state.SettleMatchRewardsResult, error) {
	if input.SettlementID == "" || len(input.Rewards) == 0 {
		return nil, state.ErrInvalidSettlement
	}

	playerIDs := make(map[int64]struct{}, len(input.Rewards))
	markerKey := settlementKey(input.SettlementID)
	watchKeys := []string{markerKey}
	for _, playerReward := range input.Rewards {
		if playerReward.PlayerID <= 0 || playerReward.Amount <= 0 {
			return nil, state.ErrInvalidPlayer
		}
		if _, ok := playerIDs[playerReward.PlayerID]; ok {
			return nil, state.ErrInvalidSettlement
		}
		watchKeys = append(watchKeys, playerKey(playerReward.PlayerID))
		playerIDs[playerReward.PlayerID] = struct{}{}
	}
	leaderboard := input.Leaderboard
	clearTimeMember, err := validateLeaderboardRecord(leaderboard, playerIDs)
	if err != nil {
		return nil, err
	}
	applied := false
	err = retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			applied = false
			exists, err := tx.Exists(ctx, markerKey).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return nil
			}
			newCoins := make([]int64, len(input.Rewards))
			for i, reward := range input.Rewards {
				coins, err := tx.HGet(ctx, playerKey(reward.PlayerID), "coins").Int64()
				if errors.Is(err, redis.Nil) {
					return state.ErrPlayerNotFound
				} else if err != nil {
					return state.ErrInvalidPlayer
				} else if coins < 0 || coins > (1<<63-1)-reward.Amount {
					return state.ErrInvalidPlayer
				}
				newCoins[i] = coins + reward.Amount
			}
			_, err = tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				for i, reward := range input.Rewards {
					pipeliner.HSet(ctx, playerKey(reward.PlayerID), "coins", newCoins[i])
				}

				if leaderboard != nil {
					for _, player := range leaderboard.Players {
						if player.TotalKills == 0 {
							continue
						}
						pipeliner.ZIncrBy(ctx, totalKillsLeaderboardKey, float64(player.TotalKills), strconv.FormatInt(player.PlayerID, 10))
					}
					if leaderboard.Cleared {
						pipeliner.ZAddArgs(ctx, clearTimeLeaderboardKey(leaderboard.Mode, leaderboard.MapVersion), redis.ZAddArgs{
							LT: true,
							Members: []redis.Z{
								{
									Score:  float64(leaderboard.CombatDurationMS),
									Member: clearTimeMember,
								},
							},
						})
					}
				}
				pipeliner.Set(ctx, markerKey, "applied", settlementRetention)
				return nil
			})
			if err != nil {
				return err
			}
			applied = true
			return nil
		}, watchKeys...)
	})
	if err != nil {
		return nil, err
	}
	return &state.SettleMatchRewardsResult{
		Applied: applied,
	}, nil
}

// CreatePlayer 按玩家 ID 存储玩家档案。
func (s *Store) CreatePlayer(ctx context.Context, player *state.Player) error {
	return s.client.HSet(ctx, playerKey(player.ID), map[string]any{
		"id":       player.ID,
		"nickname": player.Nickname,
		"avatar":   player.Avatar,
		"email":    player.Email,
		"phone":    player.Phone,
		"coins":    player.Coins,
	}).Err()

}

// GetPlayer 按玩家 ID 读取玩家档案。
func (s *Store) GetPlayer(ctx context.Context, id int64) (*state.Player, error) {
	key := playerKey(id)
	value, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, state.ErrPlayerNotFound
	}
	return playerFromHash(id, value), nil
}

func playerFromHash(id int64, value map[string]string) *state.Player {
	coins, err := strconv.ParseInt(value["coins"], 10, 64)
	if err != nil {
		coins = 0
	}
	return &state.Player{
		ID:       id,
		Nickname: value["nickname"],
		Avatar:   value["avatar"],
		Email:    value["email"],
		Phone:    value["phone"],
		Coins:    coins,
	}
}

// UpdatePlayerAvatar 原子更新已有玩家的头像并返回更新后的完整档案。
func (s *Store) UpdatePlayerAvatar(ctx context.Context, playerID int64, avatar string) (*state.Player, error) {
	if playerID <= 0 || avatar == "" {
		return nil, state.ErrInvalidPlayer
	}
	key := playerKey(playerID)
	var values map[string]string
	err := retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists == 0 {
				return state.ErrPlayerNotFound
			}

			var playerResult *redis.MapStringStringCmd
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, key, "avatar", avatar)
				playerResult = pipe.HGetAll(ctx, key)
				return nil
			})
			if err != nil {
				return err
			}
			values, err = playerResult.Result()
			return err
		}, key)
	})
	if err != nil {
		return nil, err
	}
	return playerFromHash(playerID, values), nil
}

// NextPlayerID 递增并返回 Redis 支持的玩家 ID 序列。
func (s *Store) NextPlayerID(ctx context.Context) (int64, error) {
	return s.client.Incr(ctx, nextPlayerIDKey).Result()
}

// CreateSession 使用由 ExpiresAt 推导的 Redis TTL 存储会话。
func (s *Store) CreateSession(ctx context.Context, session *state.Session) error {
	key := sessionKey(session.Token)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return state.ErrSessionNotFound
	}
	_, err := s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, key, map[string]any{
			"token":      session.Token,
			"player_id":  session.PlayerID,
			"expires_at": session.ExpiresAt.Unix(),
		})
		p.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

// GetSession 按令牌读取会话，并将过期会话视为不存在。
func (s *Store) GetSession(ctx context.Context, token string) (*state.Session, error) {
	value, err := s.client.HGetAll(ctx, sessionKey(token)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, state.ErrSessionNotFound
	}
	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	expiresAtUnix, err := strconv.ParseInt(value["expires_at"], 10, 64)
	if err != nil {
		return nil, err
	}
	session := &state.Session{
		Token:     value["token"],
		PlayerID:  playerID,
		ExpiresAt: time.Unix(expiresAtUnix, 0),
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, state.ErrSessionNotFound
	}

	return session, nil
}

// DeleteSession 按令牌删除会话。
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	return s.client.Del(ctx, sessionKey(token)).Err()
}

// CreateAccount 在用户名未被使用时存储账号凭据。
func (s *Store) CreateAccount(ctx context.Context, account *state.Account) error {
	key := accountKey(account.Username)
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return state.ErrAccountExists
			}
			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.HSet(ctx, key, map[string]any{
					"username":      account.Username,
					"password_hash": account.PasswordHash,
					"player_id":     account.PlayerID,
				})
				return nil
			})
			return err
		}, key)
	})
}

// GetAccount 按用户名读取账号凭据。
func (s *Store) GetAccount(ctx context.Context, username string) (*state.Account, error) {
	value, err := s.client.HGetAll(ctx, accountKey(username)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, state.ErrAccountNotFound
	}
	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &state.Account{
		Username:     username,
		PasswordHash: value["password_hash"],
		PlayerID:     playerID,
	}, nil
}

// SetPresence 使用 TTL 记录玩家当前连接的 logic-server。
func (s *Store) SetPresence(ctx context.Context, presence *state.Presence, ttl time.Duration) error {
	if presence == nil || ttl <= 0 {
		return state.ErrInvalidPresence
	}
	if presence.PlayerID <= 0 || presence.ServerName == "" || presence.Status == "" {
		return state.ErrInvalidPresence
	}

	key := presenceKey(presence.PlayerID)
	_, err := s.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.HSet(ctx, key, map[string]any{
			"player_id":   presence.PlayerID,
			"server_name": presence.ServerName,
			"status":      presence.Status,
			"updated_at":  presence.UpdatedAt.Unix(),
		})
		p.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

// GetPresence 读取玩家当前的在线状态记录。
func (s *Store) GetPresence(ctx context.Context, playerID int64) (*state.Presence, error) {
	if playerID <= 0 {
		return nil, state.ErrInvalidPresence
	}
	value, err := s.client.HGetAll(ctx, presenceKey(playerID)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, state.ErrPresenceNotFound
	}
	id, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	updatedAtUnix, err := strconv.ParseInt(value["updated_at"], 10, 64)
	if err != nil {
		return nil, err
	}
	updatedAt := time.Unix(updatedAtUnix, 0)
	return &state.Presence{
		PlayerID:   id,
		ServerName: value["server_name"],
		Status:     value["status"],
		UpdatedAt:  updatedAt,
	}, nil
}

// ClearPresence 仅在 serverName 仍持有记录时删除在线状态。
func (s *Store) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return state.ErrInvalidPresence
	}
	key := presenceKey(playerID)
	// TCP 的旧连接可能在新连接写入 presence 后才关闭。WATCH 将“仍由本
	// serverName 持有”的读取和删除绑定为一次乐观事务，避免旧实例删掉新实例的记录。
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			storedServerName, err := tx.HGet(ctx, key, "server_name").Result()
			if errors.Is(err, redis.Nil) {
				return nil
			}
			if err != nil {
				return err
			}

			if storedServerName != serverName {
				return nil
			}

			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.Del(ctx, key)
				return nil
			})
			return err
		}, key)
	})
}

// RefreshPresence 仅在 serverName 仍持有记录时延长在线状态 TTL。
func (s *Store) RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	if playerID <= 0 || serverName == "" || ttl <= 0 {
		return state.ErrInvalidPresence
	}
	key := presenceKey(playerID)
	// 心跳只能续期当前 logic-server 持有的记录。所有权已经被重连实例替换时返回
	// ErrPresenceNotFound，让旧 TCP 连接主动退出，而不是把新记录的 TTL 延长。
	return retryOptimisticLock(ctx, func() error {
		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			storedServerName, err := tx.HGet(ctx, key, "server_name").Result()
			if errors.Is(err, redis.Nil) {
				return state.ErrPresenceNotFound
			}
			if err != nil {
				return err
			}
			if storedServerName != serverName {
				return state.ErrPresenceNotFound
			}
			_, err = tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				pipeliner.HSet(ctx, key, "updated_at", updatedAt.Unix())
				pipeliner.Expire(ctx, key, ttl)
				return nil
			})
			return err
		}, key)

		return err
	})
}

// RegisterAccount 一并创建账号、玩家和会话记录。
func (s *Store) RegisterAccount(ctx context.Context, input state.RegisterAccountInput) (*state.RegisterAccountResult, error) {
	var result *state.RegisterAccountResult
	// 用户名唯一性检查、玩家 ID 分配以及账号/玩家/会话/初始成长写入必须处于同一
	// WATCH + 事务提交范围。发生竞争时 retryOptimisticLock 重新读取，不能留下
	// 只有账号或只有玩家的半注册状态。
	err := retryOptimisticLock(ctx, func() error {
		accountKey := accountKey(input.Username)
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, accountKey).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return state.ErrAccountExists
			}

			playerID, err := tx.Incr(ctx, nextPlayerIDKey).Result()
			if err != nil {
				return err
			}
			player := &state.Player{
				ID:       playerID,
				Nickname: input.Nickname,
				Avatar:   input.Avatar,
				Email:    input.Email,
				Phone:    input.Phone,
				Coins:    0,
			}
			account := &state.Account{
				Username:     input.Username,
				PasswordHash: input.PasswordHash,
				PlayerID:     playerID,
			}
			session := &state.Session{
				Token:     input.SessionToken,
				PlayerID:  playerID,
				ExpiresAt: input.SessionExpiresAt,
			}
			sessionTTL := time.Until(session.ExpiresAt)
			if sessionTTL <= 0 {
				return state.ErrSessionNotFound
			}

			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.HSet(ctx, playerKey(player.ID), map[string]any{
					"id":       player.ID,
					"nickname": player.Nickname,
					"avatar":   player.Avatar,
					"email":    player.Email,
					"phone":    player.Phone,
					"coins":    player.Coins,
				})
				p.HSet(ctx, accountKey, map[string]any{
					"username":      account.Username,
					"password_hash": account.PasswordHash,
					"player_id":     account.PlayerID,
				})
				p.HSet(ctx, sessionKey(session.Token), map[string]any{
					"token":      session.Token,
					"player_id":  session.PlayerID,
					"expires_at": session.ExpiresAt.Unix(),
				})
				p.HSet(ctx, growthKey(playerID), map[string]any{
					"player_id":          playerID,
					"attack_level":       1,
					"attack_speed_level": 1,
					"health_level":       1,
					"move_speed_level":   1,
				})
				p.Expire(ctx, sessionKey(session.Token), sessionTTL)
				return nil
			})
			if err != nil {
				return err
			}

			result = &state.RegisterAccountResult{
				Account: account,
				Player:  player,
				Session: session,
			}
			return nil
		}, accountKey)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := validateFriendPair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	fromPlayerKey := playerKey(fromPlayerID)
	toPlayerKey := playerKey(toPlayerID)
	requestKey := friendRequestKey(fromPlayerID, toPlayerID)
	reverseRequestKey := friendRequestKey(toPlayerID, fromPlayerID)
	fromFriendKey := friendsKey(fromPlayerID)
	toFriendKey := friendsKey(toPlayerID)
	createdAt := time.Now().UTC()
	score := float64(createdAt.UnixMilli())
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			playerExists, err := tx.Exists(ctx, fromPlayerKey, toPlayerKey).Result()
			if err != nil {
				return err
			}
			if playerExists < 2 {
				return state.ErrPlayerNotFound
			}
			isFriend, err := tx.SIsMember(ctx, fromFriendKey, toPlayerID).Result()
			if err != nil {
				return err
			}
			reverseFriend, err := tx.SIsMember(ctx, toFriendKey, fromPlayerID).Result()
			if err != nil {
				return err
			}
			if isFriend || reverseFriend {
				return state.ErrFriendAlreadyExists
			}
			exists, err := tx.Exists(ctx, requestKey, reverseRequestKey).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return state.ErrFriendRequestExists
			}
			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.HSet(ctx, requestKey, map[string]any{
					"from_player_id": fromPlayerID,
					"to_player_id":   toPlayerID,
					"created_at":     createdAt.UnixMilli(),
				})
				p.ZAdd(ctx, friendIncomingKey(toPlayerID), redis.Z{
					Score:  score,
					Member: strconv.FormatInt(fromPlayerID, 10),
				})
				p.ZAdd(ctx, friendOutgoingKey(fromPlayerID), redis.Z{
					Score:  score,
					Member: strconv.FormatInt(toPlayerID, 10),
				})
				return nil
			})
			if err != nil {
				return err
			}
			return nil
		}, fromPlayerKey, toPlayerKey, requestKey, reverseRequestKey, fromFriendKey, toFriendKey)
	})
}

func (s *Store) ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error) {
	if err := validateFriendPlayerID(playerID); err != nil {
		return nil, err
	}
	fromIDs, err := s.client.ZRange(ctx, friendIncomingKey(playerID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	requests := make([]*state.FriendRequest, 0, len(fromIDs))
	for _, id := range fromIDs {
		fromPlayerID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		value, err := s.client.HGetAll(ctx, friendRequestKey(fromPlayerID, playerID)).Result()
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			continue
		}
		request, err := parseFriendRequest(value)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (s *Store) ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error) {
	if err := validateFriendPlayerID(playerID); err != nil {
		return nil, err
	}
	toIDs, err := s.client.ZRange(ctx, friendOutgoingKey(playerID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	requests := make([]*state.FriendRequest, 0, len(toIDs))
	for _, id := range toIDs {
		toPlayerID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
		value, err := s.client.HGetAll(ctx, friendRequestKey(playerID, toPlayerID)).Result()
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			continue
		}
		request, err := parseFriendRequest(value)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func (s *Store) AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := validateFriendPair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	requestKey := friendRequestKey(fromPlayerID, toPlayerID)
	fromFriendKey := friendsKey(fromPlayerID)
	toFriendKey := friendsKey(toPlayerID)
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, requestKey).Result()
			if err != nil {
				return err
			}
			if exists == 0 {
				return state.ErrFriendRequestNotFound
			}
			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.SAdd(ctx, fromFriendKey, toPlayerID)
				p.SAdd(ctx, toFriendKey, fromPlayerID)
				p.Del(ctx, requestKey)
				p.ZRem(ctx, friendIncomingKey(toPlayerID), strconv.FormatInt(fromPlayerID, 10))
				p.ZRem(ctx, friendOutgoingKey(fromPlayerID), strconv.FormatInt(toPlayerID, 10))
				return nil
			})

			return err
		}, requestKey)
	})
}

func (s *Store) RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := validateFriendPair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	requestKey := friendRequestKey(fromPlayerID, toPlayerID)

	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, requestKey).Result()
			if err != nil {
				return err
			}
			if exists == 0 {
				return state.ErrFriendRequestNotFound
			}
			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.Del(ctx, requestKey)
				p.ZRem(ctx, friendIncomingKey(toPlayerID), strconv.FormatInt(fromPlayerID, 10))
				p.ZRem(ctx, friendOutgoingKey(fromPlayerID), strconv.FormatInt(toPlayerID, 10))
				return nil
			})
			return err
		}, requestKey)
	})
}

func (s *Store) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	if err := validateFriendPlayerID(playerID); err != nil {
		return nil, err
	}
	values, err := s.client.SMembers(ctx, friendsKey(playerID)).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(values))
	for _, v := range values {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *Store) DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error {
	if err := validateFriendPair(playerID, friendPlayerID); err != nil {
		return err
	}
	playerFriendKey := friendsKey(playerID)
	friendFriendKey := friendsKey(friendPlayerID)
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			playerExists, err := tx.SIsMember(ctx, playerFriendKey, friendPlayerID).Result()
			if err != nil {
				return err
			}
			if !playerExists {
				return state.ErrFriendNotFound
			}
			friendExists, err := tx.SIsMember(ctx, friendFriendKey, playerID).Result()
			if err != nil {
				return err
			}
			if !friendExists {
				return state.ErrFriendNotFound
			}
			_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.SRem(ctx, playerFriendKey, friendPlayerID)
				p.SRem(ctx, friendFriendKey, playerID)
				return nil
			})
			return err
		}, playerFriendKey, friendFriendKey)
	})
}

// PublishRealtime 按投递路由发布实时事件。
func (s *Store) PublishRealtime(ctx context.Context, delivery *state.RealtimeDelivery) error {
	if !validRealtimeDelivery(delivery) {
		return state.ErrInvalidRealtimeRoute
	}
	payload, err := json.Marshal(delivery)
	if err != nil {
		return err
	}
	channel, ok := realtimeDeliveryChannel(delivery.Route)
	if !ok {
		return state.ErrInvalidRealtimeRoute
	}

	return s.client.Publish(ctx, channel, payload).Err()
}

// SubscribeRealtime 订阅指定路由上的实时投递。
func (s *Store) SubscribeRealtime(ctx context.Context, route state.RealtimeRoute) (<-chan *state.RealtimeDelivery, error) {
	if !validRealtimeRoute(route) {
		return nil, state.ErrInvalidRealtimeRoute
	}
	channel, ok := realtimeDeliveryChannel(route)
	if !ok {
		return nil, state.ErrInvalidRealtimeRoute
	}
	pubsub := s.client.Subscribe(ctx, channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}
	deliveries := make(chan *state.RealtimeDelivery, 16)
	go func() {
		// pubsub 的生命周期绑定调用方 context。关闭 events 前先关闭 Redis 订阅，
		// 使 logic-server 停止时不会泄漏 goroutine 或保留无消费者的频道连接。
		defer close(deliveries)
		defer func() {
			_ = pubsub.Close()
		}()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				delivery := &state.RealtimeDelivery{}
				if err := json.Unmarshal([]byte(msg.Payload), delivery); err != nil || !validRealtimeDelivery(delivery) {
					continue
				}
				select {
				case deliveries <- delivery:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return deliveries, nil
}

func validRealtimeDelivery(delivery *state.RealtimeDelivery) bool {
	if delivery == nil || delivery.Event == nil || delivery.Event.Type == "" || !validRealtimeRoute(delivery.Route) {
		return false
	}
	return (delivery.Route.Type == state.RealtimeRouteServer && delivery.Event.TargetPlayerID > 0) ||
		delivery.Route.Type == state.RealtimeRouteBroadcast

}

func validRealtimeRoute(route state.RealtimeRoute) bool {
	if route.Type == state.RealtimeRouteServer {
		return route.ServerName != ""
	}
	return route.Type == state.RealtimeRouteBroadcast && route.ServerName == ""
}

func realtimeDeliveryChannel(route state.RealtimeRoute) (string, bool) {
	switch route.Type {
	case state.RealtimeRouteBroadcast:
		return realtimeChannelKey(state.RealtimeBroadcastChannelName), true
	case state.RealtimeRouteServer:
		return realtimeChannelKey(route.ServerName), true
	}
	return "", false
}

func (s *Store) GetGrowth(ctx context.Context, playerID int64) (*state.Growth, error) {
	if playerID <= 0 {
		return nil, state.ErrInvalidGrowth
	}
	value, err := s.client.HGetAll(ctx, growthKey(playerID)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, state.ErrGrowthNotFound
	}
	return parseGrowth(value)
}

func (s *Store) UpgradeGrowth(ctx context.Context, input state.UpgradeGrowthInput) (*state.UpgradeGrowthResult, error) {

	if input.PlayerID <= 0 || input.Cost < 0 || input.MaxLevel < 1 {
		return nil, state.ErrInvalidGrowth
	}
	field, err := growthFieldName(input.UpgradeField)
	if err != nil {
		return nil, err
	}
	playerKey := playerKey(input.PlayerID)
	growthKey := growthKey(input.PlayerID)
	var result *state.UpgradeGrowthResult
	err = retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			coins, err := tx.HGet(ctx, playerKey, "coins").Int64()
			if errors.Is(err, redis.Nil) {
				return state.ErrPlayerNotFound
			}
			if err != nil {
				return err
			}
			if coins < input.Cost {
				return state.ErrInsufficientCoins
			}
			value, err := tx.HGetAll(ctx, growthKey).Result()
			if err != nil {
				return err
			}
			if len(value) == 0 {
				return state.ErrGrowthNotFound
			}
			growth, err := parseGrowth(value)
			if err != nil {
				return err
			}
			currentLevel64, err := strconv.ParseInt(value[field], 10, 32)
			if err != nil {
				return state.ErrInvalidGrowth
			}
			if currentLevel64 < 1 {
				return state.ErrInvalidGrowth
			}
			if currentLevel64 >= int64(input.MaxLevel) {
				return state.ErrMaxGrowthLevel
			}
			remainingCoins := coins - input.Cost
			nextLevel := currentLevel64 + 1
			setGrowthLevel(growth, field, int32(nextLevel))
			_, err = tx.TxPipelined(ctx, func(pipeliner redis.Pipeliner) error {
				pipeliner.HSet(ctx, playerKey, "coins", remainingCoins)
				pipeliner.HSet(ctx, growthKey, field, nextLevel)

				return nil
			})
			if err != nil {
				return err
			}

			result = &state.UpgradeGrowthResult{
				Growth:         growth,
				RemainingCoins: remainingCoins,
			}
			return nil
		}, playerKey, growthKey)
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListLeaderboard 按排行榜类型读取指定数量的排名记录。
func (s *Store) ListLeaderboard(ctx context.Context, input state.ListLeaderboardInput) (*state.ListLeaderboardResult, error) {
	key, descending, err := resolveLeaderboardQuery(input)
	if err != nil {
		return nil, err
	}
	stop := input.Limit - 1

	var records []redis.Z
	if descending {
		records, err = s.client.ZRevRangeWithScores(ctx, key, 0, stop).Result()
	} else {
		records, err = s.client.ZRangeWithScores(ctx, key, 0, stop).Result()
	}
	if err != nil {
		return nil, err
	}
	entries := make([]state.LeaderboardEntry, 0, len(records))
	playerInfos := make(map[int64]*redis.MapStringStringCmd)
	for index, record := range records {
		member, ok := record.Member.(string)
		if !ok {
			return nil, fmt.Errorf("invalid member type")
		}
		playerIDs, err := parseLeaderboardMember(input.Type, member)
		if err != nil {
			return nil, err
		}
		entry := state.LeaderboardEntry{
			Rank:    int64(index + 1),
			Players: make([]state.LeaderboardPlayer, len(playerIDs)),
			Score:   int64(record.Score),
		}
		for i, playerID := range playerIDs {
			entry.Players[i].PlayerID = playerID
			playerInfos[playerID] = nil
		}
		entries = append(entries, entry)
	}

	if len(playerInfos) > 0 {
		_, err = s.client.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
			for playerID := range playerInfos {
				playerInfos[playerID] = pipeliner.HGetAll(ctx, playerKey(playerID))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for i := range entries {
		for j := range entries[i].Players {
			player := &entries[i].Players[j]
			values, err := playerInfos[player.PlayerID].Result()
			if err != nil {
				return nil, err
			}
			if len(values) == 0 {
				continue
			}
			player.Nickname = values["nickname"]
			player.Avatar = values["avatar"]
		}
	}
	resultMapVersion := input.MapVersion
	if input.Type == state.LeaderboardTypeTotalKills {
		resultMapVersion = ""
	}

	return &state.ListLeaderboardResult{
		Type:       input.Type,
		MapVersion: resultMapVersion,
		Entries:    entries,
	}, nil
}

func parseGrowth(value map[string]string) (*state.Growth, error) {
	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	attackLevel, err := strconv.ParseInt(value["attack_level"], 10, 32)
	if err != nil {
		return nil, err
	}
	attackSpeedLevel, err := strconv.ParseInt(value["attack_speed_level"], 10, 32)
	if err != nil {
		return nil, err
	}
	healthLevel, err := strconv.ParseInt(value["health_level"], 10, 32)
	if err != nil {
		return nil, err
	}
	moveSpeedLevel, err := strconv.ParseInt(value["move_speed_level"], 10, 32)
	if err != nil {
		return nil, err
	}

	return &state.Growth{
		PlayerID:         playerID,
		AttackLevel:      int32(attackLevel),
		AttackSpeedLevel: int32(attackSpeedLevel),
		HealthLevel:      int32(healthLevel),
		MoveSpeedLevel:   int32(moveSpeedLevel),
	}, nil
}

func retryOptimisticLock(ctx context.Context, operation func() error) error {
	var err error
	for range optimisticLockRetries {
		if err = operation(); !errors.Is(err, redis.TxFailedErr) {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// NewStore 创建由 Redis 支持的状态存储。
func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

// key 构造业务 Redis 键。
func accountKey(username string) string {
	return "game:account:" + username
}
func sessionKey(token string) string {
	return "game:session:" + token
}
func playerKey(id int64) string {
	return "game:player:" + strconv.FormatInt(id, 10)
}
func presenceKey(playerID int64) string {
	return "game:presence:" + strconv.FormatInt(playerID, 10)
}
func growthKey(playerID int64) string {
	return "game:growth:" + strconv.FormatInt(playerID, 10)
}

func friendRequestKey(fromPlayerID, toPlayerID int64) string {
	return "game:friend_request:" + strconv.FormatInt(fromPlayerID, 10) + ":" + strconv.FormatInt(toPlayerID, 10)
}

func friendIncomingKey(playerID int64) string {
	return "game:friend_request:incoming:" + strconv.FormatInt(playerID, 10)
}

func friendOutgoingKey(playerID int64) string {
	return "game:friend_request:outgoing:" + strconv.FormatInt(playerID, 10)
}

func friendsKey(playerID int64) string {
	return "game:friends:" + strconv.FormatInt(playerID, 10)
}

func realtimeChannelKey(serverName string) string {
	return "game:realtime:" + serverName
}

func settlementKey(settlementID string) string {
	return "game:settlement:" + settlementID
}

const (
	totalKillsLeaderboardKey = "game:leaderboard:total_kills"
	leaderboardModeSolo      = "solo"
	leaderboardModeDuo       = "duo"

	maxLeaderboardLimit int64 = 100
)

func resolveLeaderboardQuery(input state.ListLeaderboardInput) (key string, descending bool, err error) {
	if input.Limit <= 0 || input.Limit > maxLeaderboardLimit {
		return "", false, state.ErrInvalidLeaderboardQuery
	}
	if input.Type == state.LeaderboardTypeSoloClearTime {
		if input.MapVersion == "" {
			return "", false, state.ErrInvalidLeaderboardQuery
		}
		return clearTimeLeaderboardKey(leaderboardModeSolo, input.MapVersion), false, nil
	}
	if input.Type == state.LeaderboardTypeDuoClearTime {
		if input.MapVersion == "" {
			return "", false, state.ErrInvalidLeaderboardQuery
		}
		return clearTimeLeaderboardKey(leaderboardModeDuo, input.MapVersion), false, nil
	}
	if input.Type == state.LeaderboardTypeTotalKills {
		return totalKillsLeaderboardKey, true, nil
	}
	return "", false, state.ErrInvalidLeaderboardQuery
}

func parseLeaderboardMember(leaderboardType state.LeaderboardType, member string) ([]int64, error) {
	if leaderboardType == state.LeaderboardTypeDuoClearTime {
		playerIDsString := strings.Split(member, ":")
		if len(playerIDsString) != 2 {
			return nil, fmt.Errorf("invalid leaderboard member %q", member)
		}
		playerIDs := make([]int64, 0, len(playerIDsString))
		for _, playerID := range playerIDsString {
			playerIDInt, err := strconv.ParseInt(playerID, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid leaderboard member %q: %w", member, err)
			}
			if playerIDInt <= 0 {
				return nil, fmt.Errorf("invalid leaderboard member %q", member)
			}
			playerIDs = append(playerIDs, playerIDInt)
		}
		if playerIDs[0] >= playerIDs[1] {
			return nil, fmt.Errorf("invalid leaderboard member %q", member)
		}
		return playerIDs, nil
	}

	if leaderboardType == state.LeaderboardTypeSoloClearTime || leaderboardType == state.LeaderboardTypeTotalKills {
		playerID, err := strconv.ParseInt(member, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid leaderboard member %q: %w", member, err)
		}
		if playerID <= 0 {
			return nil, fmt.Errorf("invalid leaderboard member %q", member)
		}
		return []int64{playerID}, nil
	}

	return nil, state.ErrInvalidLeaderboardQuery
}

func clearTimeLeaderboardKey(mode, mapVersion string) string {
	return "game:leaderboard:clear_time:" + mode + ":" + mapVersion
}

func duoLeaderboardMember(first, second int64) string {
	if first > second {
		first, second = second, first
	}
	return strconv.FormatInt(first, 10) + ":" + strconv.FormatInt(second, 10)
}

func validateLeaderboardRecord(record *state.MatchLeaderboardRecord, rewardPlayerIDs map[int64]struct{}) (string, error) {
	if record == nil {
		return "", nil
	}
	if record.MapVersion == "" || record.CombatDurationMS < 0 || record.Cleared && record.CombatDurationMS == 0 {
		return "", state.ErrInvalidSettlement
	}
	expectedPlayers := 0
	switch record.Mode {
	case leaderboardModeSolo:
		expectedPlayers = 1
	case leaderboardModeDuo:
		expectedPlayers = 2
	default:
		return "", state.ErrInvalidSettlement
	}
	if len(record.Players) != expectedPlayers || len(record.Players) != len(rewardPlayerIDs) {
		return "", state.ErrInvalidSettlement
	}

	seenPlayerIDs := make(map[int64]struct{}, len(record.Players))
	for _, player := range record.Players {
		if player.PlayerID <= 0 {
			return "", state.ErrInvalidPlayer
		}
		if player.TotalKills < 0 {
			return "", state.ErrInvalidSettlement
		}
		if _, rewarded := rewardPlayerIDs[player.PlayerID]; !rewarded {
			return "", state.ErrInvalidSettlement
		}
		if _, duplicate := seenPlayerIDs[player.PlayerID]; duplicate {
			return "", state.ErrInvalidSettlement
		}
		seenPlayerIDs[player.PlayerID] = struct{}{}
	}

	if record.Mode == leaderboardModeSolo {
		return strconv.FormatInt(record.Players[0].PlayerID, 10), nil
	}
	return duoLeaderboardMember(record.Players[0].PlayerID, record.Players[1].PlayerID), nil
}

func validateFriendPair(fromPlayerID, toPlayerID int64) error {
	if fromPlayerID <= 0 || toPlayerID <= 0 || fromPlayerID == toPlayerID {
		return state.ErrInvalidFriendRequest
	}
	return nil
}

func validateFriendPlayerID(playerID int64) error {
	if playerID <= 0 {
		return state.ErrInvalidFriendRequest
	}
	return nil
}

func parseFriendRequest(value map[string]string) (*state.FriendRequest, error) {
	fromPlayerID, err := strconv.ParseInt(value["from_player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	toPlayerID, err := strconv.ParseInt(value["to_player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	createdAtMilli, err := strconv.ParseInt(value["created_at"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &state.FriendRequest{
		FromPlayerID: fromPlayerID,
		ToPlayerID:   toPlayerID,
		CreatedAt:    time.UnixMilli(createdAtMilli),
	}, nil
}

func growthFieldName(field string) (string, error) {
	switch field {
	case "Attack":
		return "attack_level", nil
	case "AttackSpeed":
		return "attack_speed_level", nil
	case "Health":
		return "health_level", nil
	case "MoveSpeed":
		return "move_speed_level", nil
	default:
		return "", state.ErrInvalidGrowthField
	}
}

func setGrowthLevel(growth *state.Growth, field string, level int32) {
	switch field {
	case "attack_level":
		growth.AttackLevel = level
	case "attack_speed_level":
		growth.AttackSpeedLevel = level
	case "health_level":
		growth.HealthLevel = level
	case "move_speed_level":
		growth.MoveSpeedLevel = level
	}
}
