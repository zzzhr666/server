package room

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	nextIDKey = "game:next_room_id"
	roomsKey  = "game:rooms"
)

// RedisRepository stores rooms and room membership in Redis.
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a Redis-backed room repository.
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// roomKey returns the Redis hash key for a room.
func roomKey(id int64) string {
	return "game:room:" + strconv.FormatInt(id, 10)
}

// playersKey returns the Redis set key for room members.
func playersKey(id int64) string {
	return "game:room:" + strconv.FormatInt(id, 10) + ":players"
}

// readyPlayersKey returns the Redis set key for ready room members.
func readyPlayersKey(id int64) string {
	return "game:room:" + strconv.FormatInt(id, 10) + ":ready_players"
}

// playerRoomKey returns the Redis string key that maps a player to its room.
func playerRoomKey(id int64) string {
	return "game:player:" + strconv.FormatInt(id, 10) + ":room"
}

// NextID increments and returns the next room ID.
func (r *RedisRepository) NextID(ctx context.Context) (int64, error) {
	return r.client.Incr(ctx, nextIDKey).Result()
}

// Create stores a room hash, room members, ready members, and the room index.
func (r *RedisRepository) Create(ctx context.Context, room *Room) error {
	_, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, roomKey(room.ID), map[string]any{
			"id":          room.ID,
			"owner_id":    room.OwnerID,
			"status":      string(room.Status),
			"max_players": room.MaxPlayers,
		})
		for playerID := range room.Players {
			pipe.SAdd(ctx, playersKey(room.ID), playerID)
		}
		for playerID := range room.ReadyPlayers {
			pipe.SAdd(ctx, readyPlayersKey(room.ID), playerID)
		}
		pipe.SAdd(ctx, roomsKey, room.ID)
		return nil
	})
	return err
}

// Get loads a room and its member sets from Redis.
func (r *RedisRepository) Get(ctx context.Context, roomID int64) (*Room, error) {
	value, err := r.client.HGetAll(ctx, roomKey(roomID)).Result()
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
	status := Status(value["status"])
	maxPlayers, err := strconv.Atoi(value["max_players"])
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
	readyPlayers, err := r.client.SMembers(ctx, readyPlayersKey(roomID)).Result()
	if err != nil {
		return nil, err
	}
	readyPlayersIDs := make(map[int64]struct{}, len(readyPlayers))
	for _, readyPlayerIDString := range readyPlayers {
		readyPlayerID, err := strconv.ParseInt(readyPlayerIDString, 10, 64)
		if err != nil {
			return nil, err
		}
		readyPlayersIDs[readyPlayerID] = struct{}{}
	}
	return &Room{
		ID:           roomID,
		OwnerID:      ownerID,
		Status:       status,
		MaxPlayers:   maxPlayers,
		Players:      playerIDs,
		ReadyPlayers: readyPlayersIDs,
	}, nil
}

// ListIDs returns all room IDs from the room index set.
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

// Exists reports whether the room hash exists in Redis.
func (r *RedisRepository) Exists(ctx context.Context, roomID int64) (bool, error) {
	exists, err := r.client.Exists(ctx, roomKey(roomID)).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// AddPlayer adds a player to the room member set.
func (r *RedisRepository) AddPlayer(ctx context.Context, roomID, playerID int64) error {
	added, err := r.client.SAdd(ctx, playersKey(roomID), playerID).Result()
	if err != nil {
		return err
	}
	if added == 0 {
		return ErrPlayerAlreadyInThisRoom
	}
	return nil
}

// RemovePlayer removes a player from the room member set.
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

// RemoveReadyPlayer removes a player from the room ready set.
func (r *RedisRepository) RemoveReadyPlayer(ctx context.Context, roomID, playerID int64) error {
	_, err := r.client.SRem(ctx, readyPlayersKey(roomID), playerID).Result()
	if err != nil {
		return err
	}
	return nil
}

// AddReadyPlayer adds a player to the room ready set.
func (r *RedisRepository) AddReadyPlayer(ctx context.Context, roomID, playerID int64) error {
	_, err := r.client.SAdd(ctx, readyPlayersKey(roomID), playerID).Result()
	if err != nil {
		return err
	}
	return nil
}

// UpdateOwner stores a new room owner ID.
func (r *RedisRepository) UpdateOwner(ctx context.Context, roomID, ownerID int64) error {
	_, err := r.client.HSet(ctx, roomKey(roomID), "owner_id", ownerID).Result()
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a room hash, membership sets, and the room index entry.
func (r *RedisRepository) Delete(ctx context.Context, roomID int64) error {
	_, err := r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, roomKey(roomID))
		pipe.Del(ctx, playersKey(roomID))
		pipe.Del(ctx, readyPlayersKey(roomID))

		pipe.SRem(ctx, roomsKey, roomID)
		return nil
	})
	return err
}

// UpdateStatus stores a new room status.
func (r *RedisRepository) UpdateStatus(ctx context.Context, roomID int64, status Status) error {
	_, err := r.client.HSet(ctx, roomKey(roomID), "status", string(status)).Result()
	return err
}

// FindRoomByPlayer returns the room ID currently stored for a player.
func (r *RedisRepository) FindRoomByPlayer(ctx context.Context, playerID int64) (int64, bool, error) {
	roomIDStr, err := r.client.Get(ctx, playerRoomKey(playerID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return roomID, true, nil

}

// SetPlayerRoom stores the current room ID for a player.
func (r *RedisRepository) SetPlayerRoom(ctx context.Context, playerID, roomID int64) error {
	_, err := r.client.Set(ctx, playerRoomKey(playerID), roomID, 0).Result()
	return err
}

// ClearPlayerRoom removes the current room ID stored for a player.
func (r *RedisRepository) ClearPlayerRoom(ctx context.Context, playerID int64) error {
	_, err := r.client.Del(ctx, playerRoomKey(playerID)).Result()
	return err
}
