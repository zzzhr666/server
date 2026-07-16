package auth

import (
	"context"
	"learning/internal/redisdb"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRepository stores auth accounts and sessions in Redis.
type RedisRepository struct {
	client *redis.Client
}

const maxTxRetries = 8

// CreateAccount stores account credentials only when the username key is free.
func (r *RedisRepository) CreateAccount(ctx context.Context, account *Account) error {
	key := accountKey(account.Username)
	return redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			exists, err := tx.Exists(ctx, key).Result()
			if err != nil {
				return err
			}
			if exists > 0 {
				return ErrAccountExists
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, key, map[string]any{
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

// GetAccount loads an account by username from Redis.
func (r *RedisRepository) GetAccount(ctx context.Context, username string) (*Account, error) {
	value, err := r.client.HGetAll(ctx, accountKey(username)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrAccountNotFound
	}
	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &Account{
		Username:     value["username"],
		PasswordHash: value["password_hash"],
		PlayerID:     playerID,
	}, nil
}

// CreateSession stores a session hash and applies Redis key expiration.
func (r *RedisRepository) CreateSession(ctx context.Context, session *Session) error {
	key := sessionKey(session.Token)
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return ErrSessionNotFound
	}
	_, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, map[string]any{
			"token":      session.Token,
			"player_id":  session.PlayerID,
			"expires_at": session.ExpiresAt.Unix(),
		})
		pipe.Expire(ctx, key, ttl)
		return nil
	})
	return err
}

// GetSession loads a session and treats expired sessions as missing.
func (r *RedisRepository) GetSession(ctx context.Context, token string) (*Session, error) {
	key := sessionKey(token)
	value, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrSessionNotFound
	}

	playerID, err := strconv.ParseInt(value["player_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	expiresAtUnix, err := strconv.ParseInt(value["expires_at"], 10, 64)
	if err != nil {
		return nil, err
	}
	session := &Session{
		Token:     value["token"],
		PlayerID:  playerID,
		ExpiresAt: time.Unix(expiresAtUnix, 0),
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// DeleteSession removes a session token from Redis.
func (r *RedisRepository) DeleteSession(ctx context.Context, token string) error {
	_, err := r.client.Del(ctx, sessionKey(token)).Result()
	return err
}

// NewRedisRepository creates a Redis-backed auth repository.
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func accountKey(username string) string {
	return "game:account:" + username
}

func sessionKey(token string) string {
	return "game:session:" + token
}
