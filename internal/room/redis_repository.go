package room

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	nextIDKey = "game:next_room_id"
	roomsKey  = "game:rooms"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func key(id int64) string {
	return "game:room:" + strconv.FormatInt(id, 10)
}

func playersKey(id int64) string {
	return "game:room:" + strconv.FormatInt(id, 10) + ":players"
}

func (r *RedisRepository) NextID(ctx context.Context) (int64, error) {
	return r.client.Incr(ctx, nextIDKey).Result()
}

func (r *RedisRepository) Create(ctx context.Context, room *Room) error {
	_, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key(room.ID), map[string]any{
			"id":       room.ID,
			"owner_id": room.OwnerID,
		})
		for playerID := range room.Players {
			pipe.SAdd(ctx, playersKey(room.ID), playerID)
		}
		pipe.SAdd(ctx, roomsKey, room.ID)
		return nil
	})
	return err
}

func (r *RedisRepository) Get(ctx context.Context, roomID int64) (*Room, error) {
	value, err := r.client.HGetAll(ctx, key(roomID)).Result()
	if err != nil {
		return nil, err
	}
	if len(value) == 0 {
		return nil, ErrNotFound
	}
	ownerID, err := strconv.ParseInt(value["owner_id"], 10, 64)
	if err != nil {
		return nil, err
	}
	players, err := r.client.SMembers(ctx, playersKey(roomID)).Result()
	if err != nil {
		return nil, err
	}
	playerIDs := make(map[int64]struct{}, len(players))
	for _, playerIDString := range players {
		playerID, err := strconv.ParseInt(playerIDString, 10, 64)
		if err != nil {
			return nil, err
		}
		playerIDs[playerID] = struct{}{}
	}
	return &Room{ID: roomID, OwnerID: ownerID, Players: playerIDs}, nil
}

func (r *RedisRepository) ListIDs(ctx context.Context) ([]int64, error) {
	values, err := r.client.SMembers(ctx, roomsKey).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *RedisRepository) Exists(ctx context.Context, roomID int64) (bool, error) {
	exists, err := r.client.Exists(ctx, key(roomID)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *RedisRepository) AddPlayer(ctx context.Context, roomID, playerID int64) error {
	added, err := r.client.SAdd(ctx, playersKey(roomID), playerID).Result()
	if err != nil {
		return err
	}
	if added == 0 {
		return ErrPlayerAlreadyIn
	}
	return nil
}

func (r *RedisRepository) RemovePlayer(ctx context.Context, roomID, playerID int64) error {
	removed, err := r.client.SRem(ctx, playersKey(roomID), playerID).Result()
	if err != nil {
		return err
	}
	if removed == 0 {
		return ErrPlayerNotIn
	}
	return nil
}
