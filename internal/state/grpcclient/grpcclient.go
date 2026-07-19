package grpcclient

import (
	"context"
	"server/internal/contract/state"
	"server/internal/contract/statepb"
	"server/internal/state/stateproto"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client adapts the generated gRPC client to the state contract interfaces.
type Client struct {
	grpc statepb.StateServiceClient
}

// GetAccount loads an account by username through state-server.
func (c *Client) GetAccount(ctx context.Context, username string) (*state.Account, error) {
	res, err := c.grpc.GetAccount(ctx, &statepb.GetAccountRequest{Username: username})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoAccount(res.GetAccount()), nil
}

// RegisterAccount creates account, player, and session state in one gRPC call.
func (c *Client) RegisterAccount(ctx context.Context, input state.RegisterAccountInput) (*state.RegisterAccountResult, error) {
	res, err := c.grpc.RegisterAccount(ctx, &statepb.RegisterAccountRequest{
		Username:         input.Username,
		PasswordHash:     input.PasswordHash,
		Nickname:         input.Nickname,
		Avatar:           input.Avatar,
		Email:            input.Email,
		Phone:            input.Phone,
		SessionToken:     input.SessionToken,
		SessionExpiresAt: stateproto.ToProtoTime(input.SessionExpiresAt),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &state.RegisterAccountResult{
		Account: stateproto.FromProtoAccount(res.GetAccount()),
		Player:  stateproto.FromProtoPlayer(res.GetPlayer()),
		Session: stateproto.FromProtoSession(res.GetSession()),
	}, nil
}

// CreateSession persists a login session through state-server.
func (c *Client) CreateSession(ctx context.Context, session *state.Session) error {
	_, err := c.grpc.CreateSession(ctx, &statepb.CreateSessionRequest{Session: stateproto.ToProtoSession(session)})
	return mapGRPCError(err)
}

// GetSession loads a session by token through state-server.
func (c *Client) GetSession(ctx context.Context, token string) (*state.Session, error) {
	res, err := c.grpc.GetSession(ctx, &statepb.GetSessionRequest{Token: token})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoSession(res.GetSession()), nil
}

// DeleteSession removes a login session through state-server.
func (c *Client) DeleteSession(ctx context.Context, token string) error {
	_, err := c.grpc.DeleteSession(ctx, &statepb.DeleteSessionRequest{Token: token})
	return mapGRPCError(err)
}

// CreatePlayer persists player profile state through state-server.
func (c *Client) CreatePlayer(ctx context.Context, player *state.Player) error {
	_, err := c.grpc.CreatePlayer(ctx, &statepb.CreatePlayerRequest{Player: stateproto.ToProtoPlayer(player)})
	return mapGRPCError(err)
}

// GetPlayer loads a player profile by ID through state-server.
func (c *Client) GetPlayer(ctx context.Context, id int64) (*state.Player, error) {
	res, err := c.grpc.GetPlayer(ctx, &statepb.GetPlayerRequest{Id: id})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoPlayer(res.GetPlayer()), nil
}

// NextPlayerID allocates the next player ID through state-server.
func (c *Client) NextPlayerID(ctx context.Context) (int64, error) {
	res, err := c.grpc.NextPlayerID(ctx, &statepb.NextPlayerIDRequest{})
	if err != nil {
		return 0, mapGRPCError(err)
	}
	return res.GetId(), nil
}

// CreateAccount persists account credentials through state-server.
func (c *Client) CreateAccount(ctx context.Context, account *state.Account) error {
	_, err := c.grpc.CreateAccount(ctx, &statepb.CreateAccountRequest{Account: stateproto.ToProtoAccount(account)})
	return mapGRPCError(err)
}

// SetPresence records a player's current logic-server connection.
func (c *Client) SetPresence(ctx context.Context, presence *state.Presence, ttl time.Duration) error {
	_, err := c.grpc.SetPresence(ctx, &statepb.SetPresenceRequest{
		Presence: stateproto.ToProtoPresence(presence),
		Ttl:      stateproto.ToProtoDuration(ttl),
	})
	return mapGRPCError(err)
}

// GetPresence loads a player's current online-state record.
func (c *Client) GetPresence(ctx context.Context, playerID int64) (*state.Presence, error) {
	res, err := c.grpc.GetPresence(ctx, &statepb.GetPresenceRequest{PlayerId: playerID})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoPresence(res.GetPresence()), nil
}

// ClearPresence removes a presence record when it still belongs to serverName.
func (c *Client) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	_, err := c.grpc.ClearPresence(ctx, &statepb.ClearPresenceRequest{PlayerId: playerID, ServerName: serverName})
	return mapGRPCError(err)
}

// NewClient creates a state contract client from generated gRPC bindings.
func NewClient(grpcClient statepb.StateServiceClient) *Client {
	return &Client{grpc: grpcClient}
}

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}
	st := status.Convert(err)
	switch st.Code() {
	case codes.NotFound:
		switch st.Message() {
		case state.ErrAccountNotFound.Error():
			return state.ErrAccountNotFound
		case state.ErrPlayerNotFound.Error():
			return state.ErrPlayerNotFound
		case state.ErrSessionNotFound.Error():
			return state.ErrSessionNotFound
		case state.ErrPresenceNotFound.Error():
			return state.ErrPresenceNotFound
		}
	case codes.AlreadyExists:
		switch st.Message() {
		case state.ErrAccountExists.Error():
			return state.ErrAccountExists
		}
	case codes.InvalidArgument:
		switch st.Message() {
		case state.ErrInvalidPresence.Error():
			return state.ErrInvalidPresence
		}
	}

	return err
}
