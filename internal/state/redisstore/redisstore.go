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
