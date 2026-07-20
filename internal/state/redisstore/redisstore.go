package redisstore

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const nextPlayerIDKey = "game:next_player_id"
const optimisticLockRetries = 3

// Store persists state contract models in Redis.
type Store struct {
	client *redis.Client
}

// CreatePlayer stores a player profile by player ID.
func (s *Store) CreatePlayer(ctx context.Context, player *statecontract.Player) error {
	return s.client.HSet(ctx, playerKey(player.ID), map[string]any{
		"id":       player.ID,
		"nickname": player.Nickname,
		"avatar":   player.Avatar,
		"email":    player.Email,
		"phone":    player.Phone,
	}).Err()

}

// GetPlayer loads a player profile by player ID.
func (s *Store) GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error) {
	key := playerKey(id)
	value, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, statecontract.ErrPlayerNotFound
	}
	return &statecontract.Player{
		ID:       id,
		Nickname: value["nickname"],
		Avatar:   value["avatar"],
		Email:    value["email"],
		Phone:    value["phone"],
	}, nil

}

// NextPlayerID increments and returns the Redis-backed player ID sequence.
func (s *Store) NextPlayerID(ctx context.Context) (int64, error) {
	return s.client.Incr(ctx, nextPlayerIDKey).Result()
}

// CreateSession stores a session with a Redis TTL derived from ExpiresAt.
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

// GetSession loads a session by token and treats expired sessions as missing.
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

// DeleteSession removes a session by token.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	return s.client.Del(ctx, sessionKey(token)).Err()
}

// CreateAccount stores account credentials if the username is still unused.
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

// GetAccount loads account credentials by username.
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

// SetPresence records a player's current logic-server connection with a TTL.
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

// GetPresence loads a player's current online-state record.
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

// ClearPresence deletes a presence record only when serverName still owns it.
func (s *Store) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	if playerID <= 0 || serverName == "" {
		return statecontract.ErrInvalidPresence
	}
	key := presenceKey(playerID)
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

// RefreshPresence extends a presence TTL only while serverName still owns it.
func (s *Store) RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	if playerID <= 0 || serverName == "" || ttl <= 0 {
		return statecontract.ErrInvalidPresence
	}
	key := presenceKey(playerID)
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

// RegisterAccount creates account, player, and session records together.
func (s *Store) RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	var result *statecontract.RegisterAccountResult
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

// NewStore creates a Redis-backed state store.
func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

// key helper
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
