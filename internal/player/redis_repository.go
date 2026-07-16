package player

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const nextIDKey = "game:next_player_id"

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

// UpdateProfile stores the latest player profile fields in Redis.
func (r *RedisRepository) UpdateProfile(ctx context.Context, p *Player) error {
	return r.client.HSet(ctx, Key(p.ID), map[string]any{
		"name":   p.Name,
		"avatar": p.Avatar,
		"email":  p.Email,
		"phone":  p.Phone,
	}).Err()
}
