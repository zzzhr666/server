package redisstore

import (
	"context"
	statecontract "server/internal/contract/state"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const nextPlayerIDKey = "game:next_player_id"

type Store struct {
	client *redis.Client
}

func (s *Store) CreatePlayer(ctx context.Context, player *statecontract.Player) error {
	return s.client.HSet(ctx, playerKey(player.ID), map[string]any{
		"id":       player.ID,
		"nickname": player.Nickname,
		"avatar":   player.Avatar,
		"email":    player.Email,
		"phone":    player.Phone,
	}).Err()

}

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

func (s *Store) NextPlayerID(ctx context.Context) (int64, error) {
	return s.client.Incr(ctx, nextPlayerIDKey).Result()
}

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

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	return s.client.Del(ctx, sessionKey(token)).Err()
}

func (s *Store) CreateAccount(ctx context.Context, account *statecontract.Account) error {
	key := accountKey(account.Username)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists > 0 {
		return statecontract.ErrAccountExists
	}
	return s.client.HSet(ctx, key, map[string]any{
		"username":      account.Username,
		"password_hash": account.PasswordHash,
		"player_id":     account.PlayerID,
	}).Err()
}

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

func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

func accountKey(username string) string {
	return "game:account:" + username
}

func sessionKey(token string) string {
	return "game:session:" + token
}
func playerKey(id int64) string {
	return "game:player:" + strconv.FormatInt(id, 10)
}
