package player

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const nextIDKey = "game:next_player_id"

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func Key(id int64) string {
	return "game:player:" + strconv.FormatInt(id, 10)
}

func (r *RedisRepository) NextID(ctx context.Context) (int64, error) {
	return r.client.Incr(ctx, nextIDKey).Result()
}

func (r *RedisRepository) Create(ctx context.Context, p *Player) error {
	return r.client.HSet(ctx, Key(p.ID), map[string]any{
		"id":   p.ID,
		"name": p.Name,
	}).Err()
}

func (r *RedisRepository) Get(ctx context.Context, id int64) (*Player, error) {
	value, err := r.client.HGetAll(ctx, Key(id)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrNotFound
	}
	return &Player{ID: id, Name: value["name"]}, nil
}

func (r *RedisRepository) Exists(ctx context.Context, id int64) (bool, error) {
	exists, err := r.client.Exists(ctx, Key(id)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
