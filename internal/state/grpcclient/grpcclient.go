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

// Client 将生成的 gRPC 客户端适配到状态契约接口。
type Client struct {
	grpc statepb.StateServiceClient
}

func (c *Client) AddPlayerCoins(ctx context.Context, input state.AddPlayerCoinsInput) (*state.AddPlayerCoinsResult, error) {
	res, err := c.grpc.AddPlayerCoins(ctx, &statepb.AddPlayerCoinsRequest{
		PlayerId: input.PlayerID,
		Amount:   input.Amount,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &state.AddPlayerCoinsResult{
		PlayerID: res.GetPlayerId(),
		Coins:    res.GetCoins(),
	}, nil
}

func (c *Client) GetGrowth(ctx context.Context, playerID int64) (*state.Growth, error) {
	res, err := c.grpc.GetGrowth(ctx, &statepb.GetGrowthRequest{
		PlayerId: playerID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoGrowth(res.GetGrowth()), nil
}

func (c *Client) UpgradeGrowth(ctx context.Context, input state.UpgradeGrowthInput) (*state.UpgradeGrowthResult, error) {
	res, err := c.grpc.UpgradeGrowth(ctx, &statepb.UpgradeGrowthRequest{
		PlayerId:     input.PlayerID,
		UpgradeField: input.UpgradeField,
		Cost:         input.Cost,
		MaxLevel:     input.MaxLevel,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return &state.UpgradeGrowthResult{
		Growth:         stateproto.FromProtoGrowth(res.GetGrowth()),
		RemainingCoins: res.GetRemainingCoins(),
	}, nil
}

// GetAccount 通过 state-server 按用户名读取账号。
func (c *Client) GetAccount(ctx context.Context, username string) (*state.Account, error) {
	res, err := c.grpc.GetAccount(ctx, &statepb.GetAccountRequest{Username: username})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoAccount(res.GetAccount()), nil
}

// RegisterAccount 通过一次 gRPC 调用创建账号、玩家和会话状态。
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

// CreateSession 通过 state-server 持久化登录会话。
func (c *Client) CreateSession(ctx context.Context, session *state.Session) error {
	_, err := c.grpc.CreateSession(ctx, &statepb.CreateSessionRequest{Session: stateproto.ToProtoSession(session)})
	return mapGRPCError(err)
}

// GetSession 通过 state-server 按令牌读取会话。
func (c *Client) GetSession(ctx context.Context, token string) (*state.Session, error) {
	res, err := c.grpc.GetSession(ctx, &statepb.GetSessionRequest{Token: token})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoSession(res.GetSession()), nil
}

// DeleteSession 通过 state-server 删除登录会话。
func (c *Client) DeleteSession(ctx context.Context, token string) error {
	_, err := c.grpc.DeleteSession(ctx, &statepb.DeleteSessionRequest{Token: token})
	return mapGRPCError(err)
}

// CreatePlayer 通过 state-server 持久化玩家档案状态。
func (c *Client) CreatePlayer(ctx context.Context, player *state.Player) error {
	_, err := c.grpc.CreatePlayer(ctx, &statepb.CreatePlayerRequest{Player: stateproto.ToProtoPlayer(player)})
	return mapGRPCError(err)
}

// GetPlayer 通过 state-server 按 ID 读取玩家档案。
func (c *Client) GetPlayer(ctx context.Context, id int64) (*state.Player, error) {
	res, err := c.grpc.GetPlayer(ctx, &statepb.GetPlayerRequest{Id: id})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoPlayer(res.GetPlayer()), nil
}

// NextPlayerID 通过 state-server 分配下一个玩家 ID。
func (c *Client) NextPlayerID(ctx context.Context) (int64, error) {
	res, err := c.grpc.NextPlayerID(ctx, &statepb.NextPlayerIDRequest{})
	if err != nil {
		return 0, mapGRPCError(err)
	}
	return res.GetId(), nil
}

// CreateAccount 通过 state-server 持久化账号凭据。
func (c *Client) CreateAccount(ctx context.Context, account *state.Account) error {
	_, err := c.grpc.CreateAccount(ctx, &statepb.CreateAccountRequest{Account: stateproto.ToProtoAccount(account)})
	return mapGRPCError(err)
}

// SetPresence 记录玩家当前连接的 logic-server。
func (c *Client) SetPresence(ctx context.Context, presence *state.Presence, ttl time.Duration) error {
	_, err := c.grpc.SetPresence(ctx, &statepb.SetPresenceRequest{
		Presence: stateproto.ToProtoPresence(presence),
		Ttl:      stateproto.ToProtoDuration(ttl),
	})
	return mapGRPCError(err)
}

// GetPresence 读取玩家当前在线状态记录。
func (c *Client) GetPresence(ctx context.Context, playerID int64) (*state.Presence, error) {
	res, err := c.grpc.GetPresence(ctx, &statepb.GetPresenceRequest{PlayerId: playerID})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return stateproto.FromProtoPresence(res.GetPresence()), nil
}

// ClearPresence 在在线记录仍属于 serverName 时删除它。
func (c *Client) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	_, err := c.grpc.ClearPresence(ctx, &statepb.ClearPresenceRequest{PlayerId: playerID, ServerName: serverName})
	return mapGRPCError(err)
}

// RefreshPresence 在在线记录仍属于 serverName 时延长其 TTL。
func (c *Client) RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	_, err := c.grpc.RefreshPresence(ctx, &statepb.RefreshPresenceRequest{
		PlayerId:   playerID,
		ServerName: serverName,
		UpdatedAt:  stateproto.ToProtoTime(updatedAt),
		Ttl:        stateproto.ToProtoDuration(ttl),
	})
	return mapGRPCError(err)
}

func (c *Client) SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	_, err := c.grpc.SendFriendRequest(ctx, &statepb.SendFriendRequestRequest{
		FromPlayerId: fromPlayerID,
		ToPlayerId:   toPlayerID,
	})
	return mapGRPCError(err)
}

func (c *Client) ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error) {
	res, err := c.grpc.ListIncomingRequest(ctx, &statepb.ListFriendRequestRequest{PlayerId: playerID})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	var requests []*state.FriendRequest
	for _, request := range res.GetRequests() {
		requests = append(requests, stateproto.FromProtoFriendRequest(request))
	}
	return requests, nil
}

func (c *Client) ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error) {
	res, err := c.grpc.ListOutgoingRequest(ctx, &statepb.ListFriendRequestRequest{PlayerId: playerID})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	var requests []*state.FriendRequest
	for _, request := range res.GetRequests() {
		requests = append(requests, stateproto.FromProtoFriendRequest(request))
	}
	return requests, nil
}

func (c *Client) AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	_, err := c.grpc.AcceptFriendRequest(ctx, &statepb.HandleFriendRequestRequest{
		FromPlayerId: fromPlayerID,
		ToPlayerId:   toPlayerID,
	})
	return mapGRPCError(err)
}

func (c *Client) RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	_, err := c.grpc.RejectFriendRequest(ctx, &statepb.HandleFriendRequestRequest{
		FromPlayerId: fromPlayerID,
		ToPlayerId:   toPlayerID,
	})
	return mapGRPCError(err)
}

func (c *Client) ListFriendIDs(ctx context.Context, fromPlayerID int64) ([]int64, error) {
	res, err := c.grpc.ListFriendIDs(ctx, &statepb.ListFriendIDsRequest{
		PlayerId: fromPlayerID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	return res.GetFriendPlayerIds(), nil
}

func (c *Client) DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error {
	_, err := c.grpc.DeleteFriend(ctx, &statepb.DeleteFriendRequest{
		PlayerId:       playerID,
		FriendPlayerId: friendPlayerID,
	})
	return mapGRPCError(err)
}

func (c *Client) PublishRealtimeToServer(ctx context.Context, serverName string, event *state.RealtimeEvent) error {
	_, err := c.grpc.PublishRealtime(ctx, &statepb.PublishRealtimeRequest{
		ServerName: serverName,
		Event:      stateproto.ToProtoRealtimeEvent(event),
	})
	return mapGRPCError(err)
}

func (c *Client) SubscribeRealtime(ctx context.Context, serverName string) (<-chan *state.RealtimeEvent, error) {
	stream, err := c.grpc.SubscribeRealtime(ctx, &statepb.SubscribeRealtimeRequest{
		ServerName: serverName,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}
	events := make(chan *state.RealtimeEvent, 16)
	go func() {
		defer close(events)
		for {
			event, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case events <- stateproto.FromProtoRealtimeEvent(event):
			case <-ctx.Done():
				return
			}
		}
	}()
	return events, nil
}

// NewClient 使用生成的 gRPC 绑定创建状态契约客户端。
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
		case state.ErrFriendNotFound.Error():
			return state.ErrFriendNotFound
		case state.ErrFriendRequestNotFound.Error():
			return state.ErrFriendRequestNotFound
		case state.ErrGrowthNotFound.Error():
			return state.ErrGrowthNotFound
		}
	case codes.AlreadyExists:
		switch st.Message() {
		case state.ErrAccountExists.Error():
			return state.ErrAccountExists
		case state.ErrFriendAlreadyExists.Error():
			return state.ErrFriendAlreadyExists
		case state.ErrFriendRequestExists.Error():
			return state.ErrFriendRequestExists
		}
	case codes.InvalidArgument:
		switch st.Message() {
		case state.ErrInvalidPresence.Error():
			return state.ErrInvalidPresence
		case state.ErrInvalidFriendRequest.Error():
			return state.ErrInvalidFriendRequest
		case state.ErrInvalidGrowth.Error():
			return state.ErrInvalidGrowth
		case state.ErrInvalidGrowthField.Error():
			return state.ErrInvalidGrowthField
		case state.ErrInvalidPlayer.Error():
			return state.ErrInvalidPlayer

		}
	case codes.FailedPrecondition:
		switch st.Message() {
		case state.ErrInsufficientCoins.Error():
			return state.ErrInsufficientCoins
		case state.ErrMaxGrowthLevel.Error():
			return state.ErrMaxGrowthLevel
		}
	}
	return err
}
