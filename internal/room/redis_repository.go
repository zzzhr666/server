package room

import (
	"context"
	"errors"
	"learning/internal/redisdb"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	nextIDKey    = "game:next_room_id"
	roomsKey     = "game:rooms"
	maxTxRetries = 8
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

// CreateWithOwner stores a room and the owner room index atomically.
func (r *RedisRepository) CreateWithOwner(ctx context.Context, room *Room) error {
	ownerRoomKey := playerRoomKey(room.OwnerID)
	return redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			if _, ok, err := getPlayerRoomInTx(ctx, tx, room.OwnerID); err != nil {
				return err
			} else if ok {
				return ErrPlayerAlreadyInAnotherRoom
			}

			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
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
				pipe.Set(ctx, ownerRoomKey, room.ID, 0)
				return nil
			})
			return err
		}, ownerRoomKey, roomKey(room.ID), playersKey(room.ID), readyPlayersKey(room.ID), roomsKey)
	})
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

// JoinRoom adds a player to a waiting room and updates the player room index atomically.
func (r *RedisRepository) JoinRoom(ctx context.Context, playerID, roomID int64) error {
	playerRoom := playerRoomKey(playerID)
	return redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			if currentRoomID, ok, err := getPlayerRoomInTx(ctx, tx, playerID); err != nil {
				return err
			} else if ok {
				if currentRoomID == roomID {
					return ErrPlayerAlreadyInThisRoom
				}
				return ErrPlayerAlreadyInAnotherRoom
			}

			room, err := getRoomInTx(ctx, tx, roomID)
			if err != nil {
				return err
			}
			if room.Status != StatusWaiting {
				return ErrRoomNotWaiting
			}
			if _, ok := room.Players[playerID]; ok {
				return ErrPlayerAlreadyInThisRoom
			}
			if len(room.Players) >= room.MaxPlayers {
				return ErrRoomFull
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.SAdd(ctx, playersKey(roomID), playerID)
				pipe.Set(ctx, playerRoom, roomID, 0)
				return nil
			})
			return err
		}, roomKey(roomID), playersKey(roomID), playerRoom)
	})
}

// LeaveRoom removes a player and applies owner transfer or room deletion atomically.
func (r *RedisRepository) LeaveRoom(ctx context.Context, playerID, roomID int64) error {
	playerRoom := playerRoomKey(playerID)
	return redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			room, err := getRoomInTx(ctx, tx, roomID)
			if err != nil {
				return err
			}
			if _, ok := room.Players[playerID]; !ok {
				return ErrPlayerNotIn
			}
			delete(room.Players, playerID)

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.SRem(ctx, playersKey(roomID), playerID)
				pipe.SRem(ctx, readyPlayersKey(roomID), playerID)
				pipe.Del(ctx, playerRoom)

				if len(room.Players) == 0 {
					pipe.Del(ctx, roomKey(roomID))
					pipe.Del(ctx, playersKey(roomID))
					pipe.Del(ctx, readyPlayersKey(roomID))
					pipe.SRem(ctx, roomsKey, roomID)
					return nil
				}

				if playerID == room.OwnerID {
					newOwnerID := minPlayerID(room.Players)
					pipe.HSet(ctx, roomKey(roomID), "owner_id", newOwnerID)
					pipe.SRem(ctx, readyPlayersKey(roomID), newOwnerID)
				}
				return nil
			})
			return err
		}, roomKey(roomID), playersKey(roomID), readyPlayersKey(roomID), playerRoom, roomsKey)
	})
}

// SetReady updates a non-owner room player's ready state atomically.
func (r *RedisRepository) SetReady(ctx context.Context, playerID, roomID int64, ready bool) error {
	return redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			room, err := getRoomInTx(ctx, tx, roomID)
			if err != nil {
				return err
			}
			if room.Status != StatusWaiting {
				return ErrRoomNotWaiting
			}
			if _, ok := room.Players[playerID]; !ok {
				return ErrPlayerNotIn
			}
			if playerID == room.OwnerID {
				return ErrOwnerCannotReadyOrUnready
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				if ready {
					pipe.SAdd(ctx, readyPlayersKey(roomID), playerID)
				} else {
					pipe.SRem(ctx, readyPlayersKey(roomID), playerID)
				}
				return nil
			})
			return err
		}, roomKey(roomID), playersKey(roomID), readyPlayersKey(roomID))
	})
}

// StartRoom marks a waiting room as playing after owner and readiness checks.
func (r *RedisRepository) StartRoom(ctx context.Context, playerID, roomID int64) error {
	return redisdb.WithTxRetry(ctx, maxTxRetries, func() error {
		return r.client.Watch(ctx, func(tx *redis.Tx) error {
			room, err := getRoomInTx(ctx, tx, roomID)
			if err != nil {
				return err
			}
			if room.Status != StatusWaiting {
				return ErrRoomNotWaiting
			}
			if playerID != room.OwnerID {
				return ErrOnlyOwnerCanStart
			}
			for playerInRoomID := range room.Players {
				if playerInRoomID == playerID {
					continue
				}
				if _, ready := room.ReadyPlayers[playerInRoomID]; !ready {
					return ErrPlayersNotReady
				}
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HSet(ctx, roomKey(roomID), "status", string(StatusPlaying))
				return nil
			})
			return err
		}, roomKey(roomID), playersKey(roomID), readyPlayersKey(roomID))
	})
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

func getRoomInTx(ctx context.Context, tx *redis.Tx, roomID int64) (*Room, error) {
	value, err := tx.HGetAll(ctx, roomKey(roomID)).Result()
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
	maxPlayers, err := strconv.Atoi(value["max_players"])
	if err != nil {
		return nil, err
	}
	players, err := tx.SMembers(ctx, playersKey(roomID)).Result()
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
	readyPlayers, err := tx.SMembers(ctx, readyPlayersKey(roomID)).Result()
	if err != nil {
		return nil, err
	}
	readyPlayerIDs := make(map[int64]struct{}, len(readyPlayers))
	for _, readyPlayerIDString := range readyPlayers {
		readyPlayerID, err := strconv.ParseInt(readyPlayerIDString, 10, 64)
		if err != nil {
			return nil, err
		}
		readyPlayerIDs[readyPlayerID] = struct{}{}
	}
	return &Room{
		ID:           roomID,
		OwnerID:      ownerID,
		Status:       Status(value["status"]),
		MaxPlayers:   maxPlayers,
		Players:      playerIDs,
		ReadyPlayers: readyPlayerIDs,
	}, nil
}

func getPlayerRoomInTx(ctx context.Context, tx *redis.Tx, playerID int64) (int64, bool, error) {
	roomIDStr, err := tx.Get(ctx, playerRoomKey(playerID)).Result()
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

func minPlayerID(players map[int64]struct{}) int64 {
	var minID int64
	first := true
	for playerID := range players {
		if first || playerID < minID {
			minID = playerID
			first = false
		}
	}
	return minID
}
