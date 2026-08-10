package redisstore

import (
	"context"
	"encoding/json"
	"errors"
	statecontract "server/internal/contract/state"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const nextPlayerIDKey = "game:next_player_id"
const optimisticLockRetries = 3

// Store 在 Redis 中持久化状态契约模型。
type Store struct {
	client *redis.Client
}

func (s *Store) AddPlayerCoins(ctx context.Context, input statecontract.AddPlayerCoinsInput) (*statecontract.AddPlayerCoinsResult, error) {
	if input.PlayerID <= 0 || input.Amount <= 0 {
		return nil, statecontract.ErrInvalidPlayer
	}
	key := playerKey(input.PlayerID)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, statecontract.ErrPlayerNotFound
	}
	coins, err := s.client.HIncrBy(ctx, key, "coins", input.Amount).Result()
	if err != nil {
		return nil, err
	}
	return &statecontract.AddPlayerCoinsResult{
		PlayerID: input.PlayerID,
		Coins:    coins,
	}, nil
}

// CreatePlayer 按玩家 ID 存储玩家档案。
func (s *Store) CreatePlayer(ctx context.Context, player *statecontract.Player) error {
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
func (s *Store) GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error) {
	key := playerKey(id)
	value, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, statecontract.ErrPlayerNotFound
	}
	coins, err := strconv.ParseInt(value["coins"], 10, 64)
	if err != nil {
		coins = 0
	}
	return &statecontract.Player{
		ID:       id,
		Nickname: value["nickname"],
		Avatar:   value["avatar"],
		Email:    value["email"],
		Phone:    value["phone"],
		Coins:    coins,
	}, nil

}

// NextPlayerID 递增并返回 Redis 支持的玩家 ID 序列。
func (s *Store) NextPlayerID(ctx context.Context) (int64, error) {
	return s.client.Incr(ctx, nextPlayerIDKey).Result()
}

// CreateSession 使用由 ExpiresAt 推导的 Redis TTL 存储会话。
func (s *Store) CreateSession(ctx context.Context, session *statecontract.Session) error {
	key := sessionKey(session.Token)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return statecontract.ErrSessionNotFound
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
func (s *Store) GetSession(ctx context.Context, token string) (*statecontract.Session, error) {
	value, err := s.client.HGetAll(ctx, sessionKey(token)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, statecontract.ErrSessionNotFound
	}
	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	expiresAtUnix, err := strconv.ParseInt(value["expires_at"], 10, 64)
	if err != nil {
		return nil, err
	}
	session := &statecontract.Session{
		Token:     value["token"],
		PlayerID:  playerID,
		ExpiresAt: time.Unix(expiresAtUnix, 0),
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, statecontract.ErrSessionNotFound
	}

	return session, nil
}

// DeleteSession 按令牌删除会话。
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	return s.client.Del(ctx, sessionKey(token)).Err()
}

// CreateAccount 在用户名未被使用时存储账号凭据。
func (s *Store) CreateAccount(ctx context.Context, account *statecontract.Account) error {
	key := accountKey(account.Username)
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return statecontract.ErrAccountExists
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
func (s *Store) GetAccount(ctx context.Context, username string) (*statecontract.Account, error) {
	value, err := s.client.HGetAll(ctx, accountKey(username)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, statecontract.ErrAccountNotFound
	}
	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &statecontract.Account{
		Username:     username,
		PasswordHash: value["password_hash"],
		PlayerID:     playerID,
	}, nil
}

// SetPresence 使用 TTL 记录玩家当前连接的 logic-server。
func (s *Store) SetPresence(ctx context.Context, presence *statecontract.Presence, ttl time.Duration) error {
	if presence == nil || ttl <= 0 {
		return statecontract.ErrInvalidPresence
	}
	if presence.PlayerID <= 0 || presence.ServerName == "" || presence.Status == "" {
		return statecontract.ErrInvalidPresence
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
func (s *Store) GetPresence(ctx context.Context, playerID int64) (*statecontract.Presence, error) {
	if playerID <= 0 {
		return nil, statecontract.ErrInvalidPresence
	}
	value, err := s.client.HGetAll(ctx, presenceKey(playerID)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, statecontract.ErrPresenceNotFound
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
	return &statecontract.Presence{
		PlayerID:   id,
		ServerName: value["server_name"],
		Status:     value["status"],
		UpdatedAt:  updatedAt,
	}, nil
}

// ClearPresence 仅在 serverName 仍持有记录时删除在线状态。
func (s *Store) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return statecontract.ErrInvalidPresence
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
		return statecontract.ErrInvalidPresence
	}
	key := presenceKey(playerID)
	// 心跳只能续期当前 logic-server 持有的记录。所有权已经被重连实例替换时返回
	// ErrPresenceNotFound，让旧 TCP 连接主动退出，而不是把新记录的 TTL 延长。
	return retryOptimisticLock(ctx, func() error {
		err := s.client.Watch(ctx, func(tx *redis.Tx) error {
			storedServerName, err := tx.HGet(ctx, key, "server_name").Result()
			if errors.Is(err, redis.Nil) {
				return statecontract.ErrPresenceNotFound
			}
			if err != nil {
				return err
			}
			if storedServerName != serverName {
				return statecontract.ErrPresenceNotFound
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
func (s *Store) RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	var result *statecontract.RegisterAccountResult
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
				return statecontract.ErrAccountExists
			}

			playerID, err := tx.Incr(ctx, nextPlayerIDKey).Result()
			if err != nil {
				return err
			}
			player := &statecontract.Player{
				ID:       playerID,
				Nickname: input.Nickname,
				Avatar:   input.Avatar,
				Email:    input.Email,
				Phone:    input.Phone,
				Coins:    0,
			}
			account := &statecontract.Account{
				Username:     input.Username,
				PasswordHash: input.PasswordHash,
				PlayerID:     playerID,
			}
			session := &statecontract.Session{
				Token:     input.SessionToken,
				PlayerID:  playerID,
				ExpiresAt: input.SessionExpiresAt,
			}
			sessionTTL := time.Until(session.ExpiresAt)
			if sessionTTL <= 0 {
				return statecontract.ErrSessionNotFound
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

			result = &statecontract.RegisterAccountResult{
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
	requestKey := friendRequestKey(fromPlayerID, toPlayerID)
	reverseRequestKey := friendRequestKey(toPlayerID, fromPlayerID)
	fromFriendKey := friendsKey(fromPlayerID)
	toFriendKey := friendsKey(toPlayerID)
	createdAt := time.Now().UTC()
	score := float64(createdAt.UnixMilli())
	return retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			isFriend, err := tx.SIsMember(ctx, fromFriendKey, toPlayerID).Result()
			if err != nil {
				return err
			}
			reverseFriend, err := tx.SIsMember(ctx, toFriendKey, fromPlayerID).Result()
			if err != nil {
				return err
			}
			if isFriend || reverseFriend {
				return statecontract.ErrFriendAlreadyExists
			}
			exists, err := tx.Exists(ctx, requestKey, reverseRequestKey).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return statecontract.ErrFriendRequestExists
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
		}, requestKey, reverseRequestKey, fromFriendKey, toFriendKey)
	})
}

func (s *Store) ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	if err := validateFriendPlayerID(playerID); err != nil {
		return nil, err
	}
	fromIDs, err := s.client.ZRange(ctx, friendIncomingKey(playerID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	requests := make([]*statecontract.FriendRequest, 0, len(fromIDs))
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

func (s *Store) ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	if err := validateFriendPlayerID(playerID); err != nil {
		return nil, err
	}
	toIDs, err := s.client.ZRange(ctx, friendOutgoingKey(playerID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	requests := make([]*statecontract.FriendRequest, 0, len(toIDs))
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
				return statecontract.ErrFriendRequestNotFound
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
				return statecontract.ErrFriendRequestNotFound
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
				return statecontract.ErrFriendNotFound
			}
			friendExists, err := tx.SIsMember(ctx, friendFriendKey, playerID).Result()
			if err != nil {
				return err
			}
			if !friendExists {
				return statecontract.ErrFriendNotFound
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

// PublishRealtimeToServer 向一个 logic-server 实时频道发布事件。
func (s *Store) PublishRealtimeToServer(ctx context.Context, serverName string, event *statecontract.RealtimeEvent) error {
	if serverName == "" || event == nil || event.Type == "" || event.TargetPlayerID <= 0 {
		return statecontract.ErrInvalidPresence
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, realtimeChannelKey(serverName), payload).Err()
}

// SubscribeRealtime 订阅发送给一个 logic-server 的实时事件。
func (s *Store) SubscribeRealtime(ctx context.Context, serverName string) (<-chan *statecontract.RealtimeEvent, error) {
	if serverName == "" {
		return nil, statecontract.ErrInvalidPresence
	}
	pubsub := s.client.Subscribe(ctx, realtimeChannelKey(serverName))
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}
	events := make(chan *statecontract.RealtimeEvent, 16)
	go func() {
		// pubsub 的生命周期绑定调用方 context。关闭 events 前先关闭 Redis 订阅，
		// 使 logic-server 停止时不会泄漏 goroutine 或保留无消费者的频道连接。
		defer close(events)
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
				event := &statecontract.RealtimeEvent{}
				if err := json.Unmarshal([]byte(msg.Payload), event); err != nil {
					continue
				}
				select {
				case events <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return events, nil
}

func (s *Store) GetGrowth(ctx context.Context, playerID int64) (*statecontract.Growth, error) {
	if playerID <= 0 {
		return nil, statecontract.ErrInvalidGrowth
	}
	value, err := s.client.HGetAll(ctx, growthKey(playerID)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, statecontract.ErrGrowthNotFound
	}
	return parseGrowth(value)
}

func (s *Store) UpgradeGrowth(ctx context.Context, input statecontract.UpgradeGrowthInput) (*statecontract.UpgradeGrowthResult, error) {

	if input.PlayerID <= 0 || input.Cost < 0 || input.MaxLevel < 1 {
		return nil, statecontract.ErrInvalidGrowth
	}
	field, err := growthFieldName(input.UpgradeField)
	if err != nil {
		return nil, err
	}
	playerKey := playerKey(input.PlayerID)
	growthKey := growthKey(input.PlayerID)
	var result *statecontract.UpgradeGrowthResult
	err = retryOptimisticLock(ctx, func() error {
		return s.client.Watch(ctx, func(tx *redis.Tx) error {
			coins, err := tx.HGet(ctx, playerKey, "coins").Int64()
			if errors.Is(err, redis.Nil) {
				return statecontract.ErrPlayerNotFound
			}
			if err != nil {
				return err
			}
			if coins < input.Cost {
				return statecontract.ErrInsufficientCoins
			}
			value, err := tx.HGetAll(ctx, growthKey).Result()
			if err != nil {
				return err
			}
			if len(value) == 0 {
				return statecontract.ErrGrowthNotFound
			}
			growth, err := parseGrowth(value)
			if err != nil {
				return err
			}
			currentLevel64, err := strconv.ParseInt(value[field], 10, 32)
			if err != nil {
				return statecontract.ErrInvalidGrowth
			}
			if currentLevel64 < 1 {
				return statecontract.ErrInvalidGrowth
			}
			if currentLevel64 >= int64(input.MaxLevel) {
				return statecontract.ErrMaxGrowthLevel
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

			result = &statecontract.UpgradeGrowthResult{
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

func parseGrowth(value map[string]string) (*statecontract.Growth, error) {
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

	return &statecontract.Growth{
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

func validateFriendPair(fromPlayerID, toPlayerID int64) error {
	if fromPlayerID <= 0 || toPlayerID <= 0 || fromPlayerID == toPlayerID {
		return statecontract.ErrInvalidFriendRequest
	}
	return nil
}

func validateFriendPlayerID(playerID int64) error {
	if playerID <= 0 {
		return statecontract.ErrInvalidFriendRequest
	}
	return nil
}

func parseFriendRequest(value map[string]string) (*statecontract.FriendRequest, error) {
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
	return &statecontract.FriendRequest{
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
		return "", statecontract.ErrInvalidGrowthField
	}
}

func setGrowthLevel(growth *statecontract.Growth, field string, level int32) {
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
