package grpcserver

import (
	"context"
	statecontract "server/internal/contract/state"
	"server/internal/contract/statepb"
	"server/internal/state/stateproto"
)

// Server exposes state contract operations as protobuf/gRPC methods.
type Server struct {
	statepb.UnimplementedStateServiceServer
	stateClient    statecontract.Client
	presenceClient statecontract.PresenceClient
}

// CreateAccount handles a gRPC request to create account credentials.
func (s *Server) CreateAccount(ctx context.Context, request *statepb.CreateAccountRequest) (*statepb.CreateAccountResponse, error) {
	err := s.stateClient.CreateAccount(ctx, stateproto.FromProtoAccount(request.Account))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.CreateAccountResponse{}, nil
}

// GetAccount handles a gRPC request to load account credentials.
func (s *Server) GetAccount(ctx context.Context, request *statepb.GetAccountRequest) (*statepb.GetAccountResponse, error) {
	account, err := s.stateClient.GetAccount(ctx, request.GetUsername())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetAccountResponse{Account: stateproto.ToProtoAccount(account)}, nil
}

// RegisterAccount handles account, player, and session creation in one call.
func (s *Server) RegisterAccount(ctx context.Context, request *statepb.RegisterAccountRequest) (*statepb.RegisterAccountResponse, error) {
	res, err := s.stateClient.RegisterAccount(ctx, statecontract.RegisterAccountInput{
		Username:         request.GetUsername(),
		PasswordHash:     request.GetPasswordHash(),
		Nickname:         request.GetNickname(),
		Avatar:           request.GetAvatar(),
		Email:            request.GetEmail(),
		Phone:            request.GetPhone(),
		SessionToken:     request.GetSessionToken(),
		SessionExpiresAt: stateproto.FromProtoTime(request.GetSessionExpiresAt()),
	})
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.RegisterAccountResponse{
		Account: stateproto.ToProtoAccount(res.Account),
		Player:  stateproto.ToProtoPlayer(res.Player),
		Session: stateproto.ToProtoSession(res.Session),
	}, nil
}

// CreateSession handles a gRPC request to persist a login session.
func (s *Server) CreateSession(ctx context.Context, request *statepb.CreateSessionRequest) (*statepb.CreateSessionResponse, error) {
	err := s.stateClient.CreateSession(ctx, stateproto.FromProtoSession(request.Session))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.CreateSessionResponse{}, nil
}

// GetSession handles a gRPC request to load a login session.
func (s *Server) GetSession(ctx context.Context, request *statepb.GetSessionRequest) (*statepb.GetSessionResponse, error) {
	session, err := s.stateClient.GetSession(ctx, request.GetToken())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetSessionResponse{Session: stateproto.ToProtoSession(session)}, nil
}

// DeleteSession handles a gRPC request to remove a login session.
func (s *Server) DeleteSession(ctx context.Context, request *statepb.DeleteSessionRequest) (*statepb.DeleteSessionResponse, error) {
	err := s.stateClient.DeleteSession(ctx, request.GetToken())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.DeleteSessionResponse{}, nil
}

// CreatePlayer handles a gRPC request to persist player profile state.
func (s *Server) CreatePlayer(ctx context.Context, request *statepb.CreatePlayerRequest) (*statepb.CreatePlayerResponse, error) {
	err := s.stateClient.CreatePlayer(ctx, stateproto.FromProtoPlayer(request.Player))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.CreatePlayerResponse{}, nil
}

// GetPlayer handles a gRPC request to load player profile state.
func (s *Server) GetPlayer(ctx context.Context, request *statepb.GetPlayerRequest) (*statepb.GetPlayerResponse, error) {
	player, err := s.stateClient.GetPlayer(ctx, request.GetId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetPlayerResponse{Player: stateproto.ToProtoPlayer(player)}, nil
}

// NextPlayerID handles a gRPC request to allocate a player ID.
func (s *Server) NextPlayerID(ctx context.Context, request *statepb.NextPlayerIDRequest) (*statepb.NextPlayerIDResponse, error) {
	id, err := s.stateClient.NextPlayerID(ctx)
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.NextPlayerIDResponse{Id: id}, nil
}

// SetPresence handles a gRPC request to record online state.
func (s *Server) SetPresence(ctx context.Context, request *statepb.SetPresenceRequest) (*statepb.SetPresenceResponse, error) {
	err := s.presenceClient.SetPresence(ctx, stateproto.FromProtoPresence(request.GetPresence()), stateproto.FromProtoDuration(request.GetTtl()))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.SetPresenceResponse{}, nil
}

// GetPresence handles a gRPC request to load online state.
func (s *Server) GetPresence(ctx context.Context, request *statepb.GetPresenceRequest) (*statepb.GetPresenceResponse, error) {
	presence, err := s.presenceClient.GetPresence(ctx, request.GetPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetPresenceResponse{Presence: stateproto.ToProtoPresence(presence)}, nil
}

// ClearPresence handles a gRPC request to remove owned online state.
func (s *Server) ClearPresence(ctx context.Context, request *statepb.ClearPresenceRequest) (*statepb.ClearPresenceResponse, error) {
	err := s.presenceClient.ClearPresence(ctx, request.GetPlayerId(), request.GetServerName())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.ClearPresenceResponse{}, nil
}

// ServerConfig provides the state clients used by the gRPC adapter.
type ServerConfig struct {
	StateClient    statecontract.Client
	PresenceClient statecontract.PresenceClient
}

// NewServer creates a gRPC state server adapter.
func NewServer(config ServerConfig) *Server {
	return &Server{
		stateClient:    config.StateClient,
		presenceClient: config.PresenceClient,
	}
}
