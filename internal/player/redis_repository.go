package player

import (
	"context"
	"learning/internal/redisdb"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const nextIDKey = "game:next_player_id"

const maxTxRetries = 8

// RedisRepository stores players in Redis.
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a Redis-backed player repository.
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// Key returns the Redis hash key for a player.
func Key(id int64) string {
	return "game:player:" + strconv.FormatInt(id, 10)
}

// NextID increments and returns the next player ID.
func (r *RedisRepository) NextID(ctx context.Context) (int64, error) {
	return r.client.Incr(ctx, nextIDKey).Result()
}

// Create stores the player fields in Redis.
func (r *RedisRepository) Create(ctx context.Context, p *Player) error {
	return r.client.HSet(ctx, Key(p.ID), map[string]any{
		"id":     p.ID,
		"name":   p.Name,
		"avatar": p.Avatar,
		"email":  p.Email,
		"phone":  p.Phone,
	}).Err()
}

// Get loads a player from Redis by ID.
func (r *RedisRepository) Get(ctx context.Context, id int64) (*Player, error) {
	value, err := r.client.HGetAll(ctx, Key(id)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrNotFound
	}
	return &Player{
		ID:     id,
		Name:   value["name"],
		Avatar: value["avatar"],
		Email:  value["email"],
		Phone:  value["phone"],
	}, nil
}

// Exists reports whether the player hash exists in Redis.
func (r *RedisRepository) Exists(ctx context.Context, id int64) (bool, error) {
	exists, err := r.client.Exists(ctx, Key(id)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// UpdateProfile applies profile field changes using a Redis WATCH transaction.
func (r *RedisRepository) UpdateProfile(ctx context.Context, id int64, input UpdateProfileInput) (*Player, error) {
	key := Key(id)
	var updated *Player
	err := redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			value, err := tx.HGetAll(ctx, key).Result()
			if err != nil {
				return err
			}
			if len(value) == 0 {
				return ErrNotFound
			}
			p := &Player{
				ID:     id,
				Name:   value["name"],
				Avatar: value["avatar"],
				Email:  value["email"],
				Phone:  value["phone"],
			}
			changes := make(map[string]any)
			if input.Name != nil {
				p.Name = *input.Name
				changes["name"] = *input.Name
			}
			if input.Avatar != nil {
				p.Avatar = *input.Avatar
				changes["avatar"] = *input.Avatar
			}
			if input.Email != nil {
				p.Email = *input.Email
				changes["email"] = *input.Email
			}
			if input.Phone != nil {
				p.Phone = *input.Phone
				changes["phone"] = *input.Phone
			}
			if len(changes) == 0 {
				updated = p
				return nil
			}
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, key, changes)
				return nil
			})
			if err != nil {
				return err
			}
			updated = p
			return nil
		}, key)
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
