package grpcserver

import (
	"context"
	statecontract "server/internal/contract/state"
	"server/internal/contract/statepb"
	"server/internal/state/stateproto"

	"google.golang.org/grpc"
)

// Server 以 protobuf/gRPC 方法暴露状态契约操作。
type Server struct {
	statepb.UnimplementedStateServiceServer
	stateClient    statecontract.Client
	presenceClient statecontract.PresenceClient
	friendClient   statecontract.FriendClient
	realtimeClient statecontract.RealtimeClient
	growthClient   statecontract.GrowthClient
	coinClient     statecontract.CoinClient
}

func (s *Server) AddPlayerCoins(ctx context.Context, request *statepb.AddPlayerCoinsRequest) (*statepb.AddPlayerCoinsResponse, error) {
	res, err := s.coinClient.AddPlayerCoins(ctx, statecontract.AddPlayerCoinsInput{
		PlayerID: request.GetPlayerId(),
		Amount:   request.GetAmount(),
	})
	if err != nil {
		return nil, mapStateError(err)
	}

	return &statepb.AddPlayerCoinsResponse{
		PlayerId: res.PlayerID,
		Coins:    res.Coins,
	}, nil
}

func (s *Server) GetGrowth(ctx context.Context, request *statepb.GetGrowthRequest) (*statepb.GetGrowthResponse, error) {
	growth, err := s.growthClient.GetGrowth(ctx, request.PlayerId)
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetGrowthResponse{
		Growth: stateproto.ToProtoGrowth(growth),
	}, nil
}

func (s *Server) UpgradeGrowth(ctx context.Context, request *statepb.UpgradeGrowthRequest) (*statepb.UpgradeGrowthResponse, error) {
	res, err := s.growthClient.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{
		PlayerID:     request.GetPlayerId(),
		UpgradeField: request.GetUpgradeField(),
		Cost:         request.GetCost(),
		MaxLevel:     request.GetMaxLevel(),
	})
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.UpgradeGrowthResponse{
		Growth:         stateproto.ToProtoGrowth(res.Growth),
		RemainingCoins: res.RemainingCoins,
	}, nil
}

// CreateAccount 处理创建账号凭据的 gRPC 请求。
func (s *Server) CreateAccount(ctx context.Context, request *statepb.CreateAccountRequest) (*statepb.CreateAccountResponse, error) {
	err := s.stateClient.CreateAccount(ctx, stateproto.FromProtoAccount(request.Account))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.CreateAccountResponse{}, nil
}

// GetAccount 处理读取账号凭据的 gRPC 请求。
func (s *Server) GetAccount(ctx context.Context, request *statepb.GetAccountRequest) (*statepb.GetAccountResponse, error) {
	account, err := s.stateClient.GetAccount(ctx, request.GetUsername())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetAccountResponse{Account: stateproto.ToProtoAccount(account)}, nil
}

// RegisterAccount 在一次调用中处理账号、玩家和会话创建。
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

// CreateSession 处理持久化登录会话的 gRPC 请求。
func (s *Server) CreateSession(ctx context.Context, request *statepb.CreateSessionRequest) (*statepb.CreateSessionResponse, error) {
	err := s.stateClient.CreateSession(ctx, stateproto.FromProtoSession(request.Session))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.CreateSessionResponse{}, nil
}

// GetSession 处理读取登录会话的 gRPC 请求。
func (s *Server) GetSession(ctx context.Context, request *statepb.GetSessionRequest) (*statepb.GetSessionResponse, error) {
	session, err := s.stateClient.GetSession(ctx, request.GetToken())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetSessionResponse{Session: stateproto.ToProtoSession(session)}, nil
}

// DeleteSession 处理删除登录会话的 gRPC 请求。
func (s *Server) DeleteSession(ctx context.Context, request *statepb.DeleteSessionRequest) (*statepb.DeleteSessionResponse, error) {
	err := s.stateClient.DeleteSession(ctx, request.GetToken())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.DeleteSessionResponse{}, nil
}

// CreatePlayer 处理持久化玩家档案状态的 gRPC 请求。
func (s *Server) CreatePlayer(ctx context.Context, request *statepb.CreatePlayerRequest) (*statepb.CreatePlayerResponse, error) {
	err := s.stateClient.CreatePlayer(ctx, stateproto.FromProtoPlayer(request.Player))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.CreatePlayerResponse{}, nil
}

// GetPlayer 处理读取玩家档案状态的 gRPC 请求。
func (s *Server) GetPlayer(ctx context.Context, request *statepb.GetPlayerRequest) (*statepb.GetPlayerResponse, error) {
	player, err := s.stateClient.GetPlayer(ctx, request.GetId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetPlayerResponse{Player: stateproto.ToProtoPlayer(player)}, nil
}

// NextPlayerID 处理分配玩家 ID 的 gRPC 请求。
func (s *Server) NextPlayerID(ctx context.Context, request *statepb.NextPlayerIDRequest) (*statepb.NextPlayerIDResponse, error) {
	id, err := s.stateClient.NextPlayerID(ctx)
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.NextPlayerIDResponse{Id: id}, nil
}

// SetPresence 处理记录在线状态的 gRPC 请求。
func (s *Server) SetPresence(ctx context.Context, request *statepb.SetPresenceRequest) (*statepb.SetPresenceResponse, error) {
	err := s.presenceClient.SetPresence(ctx, stateproto.FromProtoPresence(request.GetPresence()), stateproto.FromProtoDuration(request.GetTtl()))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.SetPresenceResponse{}, nil
}

// GetPresence 处理读取在线状态的 gRPC 请求。
func (s *Server) GetPresence(ctx context.Context, request *statepb.GetPresenceRequest) (*statepb.GetPresenceResponse, error) {
	presence, err := s.presenceClient.GetPresence(ctx, request.GetPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.GetPresenceResponse{Presence: stateproto.ToProtoPresence(presence)}, nil
}

// ClearPresence 处理删除所属在线状态的 gRPC 请求。
func (s *Server) ClearPresence(ctx context.Context, request *statepb.ClearPresenceRequest) (*statepb.ClearPresenceResponse, error) {
	err := s.presenceClient.ClearPresence(ctx, request.GetPlayerId(), request.GetServerName())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.ClearPresenceResponse{}, nil
}

// RefreshPresence 处理延长所属在线状态的 gRPC 请求。
func (s *Server) RefreshPresence(ctx context.Context, request *statepb.RefreshPresenceRequest) (*statepb.RefreshPresenceResponse, error) {
	err := s.presenceClient.RefreshPresence(ctx, request.GetPlayerId(), request.GetServerName(), stateproto.FromProtoTime(request.GetUpdatedAt()), stateproto.FromProtoDuration(request.GetTtl()))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.RefreshPresenceResponse{}, nil
}

func (s *Server) SendFriendRequest(ctx context.Context, request *statepb.SendFriendRequestRequest) (*statepb.SendFriendRequestResponse, error) {
	err := s.friendClient.SendFriendRequest(ctx, request.GetFromPlayerId(), request.GetToPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.SendFriendRequestResponse{}, nil
}

func (s *Server) ListIncomingRequest(ctx context.Context, request *statepb.ListFriendRequestRequest) (*statepb.ListFriendRequestResponse, error) {
	stateReq, err := s.friendClient.ListIncomingFriendRequests(ctx, request.GetPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	requests := make([]*statepb.FriendRequest, 0, len(stateReq))
	for _, req := range stateReq {
		requests = append(requests, stateproto.ToProtoFriendRequest(req))
	}
	return &statepb.ListFriendRequestResponse{
		Requests: requests,
	}, nil
}

func (s *Server) ListOutgoingRequest(ctx context.Context, request *statepb.ListFriendRequestRequest) (*statepb.ListFriendRequestResponse, error) {
	stateReq, err := s.friendClient.ListOutgoingFriendRequests(ctx, request.GetPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	requests := make([]*statepb.FriendRequest, 0, len(stateReq))
	for _, req := range stateReq {
		requests = append(requests, stateproto.ToProtoFriendRequest(req))
	}
	return &statepb.ListFriendRequestResponse{
		Requests: requests,
	}, nil
}

func (s *Server) AcceptFriendRequest(ctx context.Context, request *statepb.HandleFriendRequestRequest) (*statepb.HandleFriendRequestResponse, error) {
	err := s.friendClient.AcceptFriendRequest(ctx, request.GetFromPlayerId(), request.GetToPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.HandleFriendRequestResponse{}, nil
}

func (s *Server) RejectFriendRequest(ctx context.Context, request *statepb.HandleFriendRequestRequest) (*statepb.HandleFriendRequestResponse, error) {
	err := s.friendClient.RejectFriendRequest(ctx, request.GetFromPlayerId(), request.GetToPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.HandleFriendRequestResponse{}, nil
}

func (s *Server) ListFriendIDs(ctx context.Context, request *statepb.ListFriendIDsRequest) (*statepb.ListFriendIDsResponse, error) {
	IDs, err := s.friendClient.ListFriendIDs(ctx, request.GetPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.ListFriendIDsResponse{FriendPlayerIds: IDs}, nil
}

func (s *Server) DeleteFriend(ctx context.Context, request *statepb.DeleteFriendRequest) (*statepb.DeleteFriendResponse, error) {
	err := s.friendClient.DeleteFriend(ctx, request.GetPlayerId(), request.GetFriendPlayerId())
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.DeleteFriendResponse{}, nil
}

func (s *Server) PublishRealtime(ctx context.Context, request *statepb.PublishRealtimeRequest) (*statepb.PublishRealtimeResponse, error) {
	err := s.realtimeClient.PublishRealtimeToServer(ctx, request.GetServerName(), stateproto.FromProtoRealtimeEvent(request.GetEvent()))
	if err != nil {
		return nil, mapStateError(err)
	}
	return &statepb.PublishRealtimeResponse{}, nil
}

func (s *Server) SubscribeRealtime(request *statepb.SubscribeRealtimeRequest, g grpc.ServerStreamingServer[statepb.RealtimeEvent]) error {
	events, err := s.realtimeClient.SubscribeRealtime(g.Context(), request.GetServerName())
	if err != nil {
		return mapStateError(err)
	}
	for {
		select {
		case <-g.Context().Done():
			return g.Context().Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if event == nil {
				continue
			}
			if err := g.Send(stateproto.ToProtoRealtimeEvent(event)); err != nil {
				return err
			}
		}
	}
}

// ServerConfig 提供 gRPC 适配器使用的状态客户端。
type ServerConfig struct {
	StateClient    statecontract.Client
	PresenceClient statecontract.PresenceClient
	FriendClient   statecontract.FriendClient
	RealtimeClient statecontract.RealtimeClient
	GrowthClient   statecontract.GrowthClient
	CoinClient     statecontract.CoinClient
}

// NewServer 创建 gRPC 状态服务适配器。
func NewServer(config ServerConfig) *Server {
	return &Server{
		stateClient:    config.StateClient,
		presenceClient: config.PresenceClient,
		friendClient:   config.FriendClient,
		realtimeClient: config.RealtimeClient,
		growthClient:   config.GrowthClient,
		coinClient:     config.CoinClient,
	}
}
