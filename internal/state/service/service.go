package service

import (
	"context"
	"server/internal/contract/state"
	"server/internal/platform/logging"
	"time"
)

type accountStore interface {
	CreateAccount(ctx context.Context, account *state.Account) error
	GetAccount(ctx context.Context, username string) (*state.Account, error)
}

type sessionStore interface {
	CreateSession(ctx context.Context, account *state.Session) error
	GetSession(ctx context.Context, token string) (*state.Session, error)
	DeleteSession(ctx context.Context, token string) error
}

type playerStore interface {
	CreatePlayer(ctx context.Context, player *state.Player) error
	GetPlayer(ctx context.Context, id int64) (*state.Player, error)
	NextPlayerID(ctx context.Context) (int64, error)
	UpdatePlayerAvatar(ctx context.Context, playerID int64, avatar string) (*state.Player, error)
}

type presenceStore interface {
	SetPresence(ctx context.Context, presence *state.Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*state.Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error
}

type registrationStore interface {
	RegisterAccount(ctx context.Context, input state.RegisterAccountInput) (*state.RegisterAccountResult, error)
}

type friendStore interface {
	SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error)
	ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error)
	AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
	DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error
}

type realtimeStore interface {
	PublishRealtime(ctx context.Context, delivery *state.RealtimeDelivery) error
	SubscribeRealtime(ctx context.Context, route state.RealtimeRoute) (<-chan *state.RealtimeDelivery, error)
}

type growthStore interface {
	GetGrowth(ctx context.Context, playerID int64) (*state.Growth, error)
	UpgradeGrowth(ctx context.Context, input state.UpgradeGrowthInput) (*state.UpgradeGrowthResult, error)
}

type coinsStore interface {
	SettleMatchRewards(ctx context.Context, input state.SettleMatchRewardsInput) (*state.SettleMatchRewardsResult, error)
}

type chatStore interface {
	SaveChatMessage(ctx context.Context, input state.SaveChatMessageInput) (*state.ChatMessage, error)
	ListChatMessages(ctx context.Context, input state.ListChatMessagesInput) ([]*state.ChatMessage, error)
}

type leaderboardStore interface {
	ListLeaderboard(ctx context.Context, input state.ListLeaderboardInput) (*state.ListLeaderboardResult, error)
}

// Service 协调已配置存储上的状态操作。
type Service struct {
	registrations registrationStore
	accounts      accountStore
	sessions      sessionStore
	players       playerStore
	presences     presenceStore
	friends       friendStore
	realtime      realtimeStore
	growth        growthStore
	coins         coinsStore
	chats         chatStore
	leaderboards  leaderboardStore
	metrics       *Metrics
}

// ListLeaderboard 读取指定类型的排行榜。
func (s *Service) ListLeaderboard(ctx context.Context, input state.ListLeaderboardInput) (*state.ListLeaderboardResult, error) {
	return s.leaderboards.ListLeaderboard(ctx, input)
}

// UpdatePlayerAvatar 更新玩家头像并返回最新的完整玩家档案。
func (s *Service) UpdatePlayerAvatar(ctx context.Context, playerID int64, avatar string) (*state.Player, error) {
	return s.players.UpdatePlayerAvatar(ctx, playerID, avatar)
}

func (s *Service) SaveChatMessage(ctx context.Context, input state.SaveChatMessageInput) (*state.ChatMessage, error) {
	message, err := s.chats.SaveChatMessage(ctx, input)
	if err != nil {
		logging.Error("state save chat failed sender_id=%d receiver_id=%d: %v", input.SenderID, input.ReceiverID, err)
		return nil, err
	}
	logging.Debug("state chat saved sender_id=%d message_key=%s", input.SenderID, message.MessageKey)
	return message, nil
}

func (s *Service) ListChatMessages(ctx context.Context, input state.ListChatMessagesInput) ([]*state.ChatMessage, error) {
	messages, err := s.chats.ListChatMessages(ctx, input)
	if err != nil {
		logging.Error("state list chat failed channel=%s key=%s: %v", input.ChannelType, input.ChannelKey, err)
		return nil, err
	}
	logging.Trace("state chat page loaded channel=%s count=%d before=%s", input.ChannelType, len(messages), input.BeforeMessageKey)
	return messages, nil
}

// SettleMatchRewards 原子结算一场对局的多人奖励与排行榜数据，并由存储层保证幂等。
func (s *Service) SettleMatchRewards(ctx context.Context, input state.SettleMatchRewardsInput) (*state.SettleMatchRewardsResult, error) {
	return s.coins.SettleMatchRewards(ctx, input)
}

// PublishRealtime 发布一条包含明确路由的实时投递。
func (s *Service) PublishRealtime(ctx context.Context, delivery *state.RealtimeDelivery) (err error) {
	defer func() { s.observeRealtime("publish", err) }()
	if publishErr := s.realtime.PublishRealtime(ctx, delivery); publishErr != nil {
		logging.Error("state publish realtime failed: %v", publishErr)
		return publishErr
	}
	logging.Debug("state realtime published route_type=%s server=%s", delivery.Route.Type, delivery.Route.ServerName)
	return nil
}

// SubscribeRealtime 订阅指定路由上的实时投递。
func (s *Service) SubscribeRealtime(ctx context.Context, route state.RealtimeRoute) (deliveries <-chan *state.RealtimeDelivery, err error) {
	defer func() { s.observeRealtime("subscribe", err) }()
	deliveries, err = s.realtime.SubscribeRealtime(ctx, route)
	if err != nil {
		logging.Error("state subscribe realtime failed route_type=%s server=%s: %v", route.Type, route.ServerName, err)
		return nil, err
	}
	logging.Info("state realtime subscribed route_type=%s server=%s", route.Type, route.ServerName)
	return deliveries, nil
}

func (s *Service) observeRealtime(operation string, err error) {
	if s.metrics == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "error"
	}
	s.metrics.RealtimePubsub.WithLabelValues(operation, result).Inc()
}

// SetPresence 记录玩家在线状态。
func (s *Service) SetPresence(ctx context.Context, presence *state.Presence, ttl time.Duration) error {
	if err := s.presences.SetPresence(ctx, presence, ttl); err != nil {
		logging.Error("state set presence failed player_id=%d: %v", presence.PlayerID, err)
		return err
	}
	logging.Debug("state presence set player_id=%d server=%s", presence.PlayerID, presence.ServerName)
	return nil
}

// GetPresence 读取玩家在线状态。
func (s *Service) GetPresence(ctx context.Context, playerID int64) (*state.Presence, error) {
	return s.presences.GetPresence(ctx, playerID)
}

// ClearPresence 删除仍由 serverName 持有的在线状态。
func (s *Service) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	if err := s.presences.ClearPresence(ctx, playerID, serverName); err != nil {
		logging.Error("state clear presence failed player_id=%d server=%s: %v", playerID, serverName, err)
		return err
	}
	logging.Debug("state presence cleared player_id=%d server=%s", playerID, serverName)
	return nil
}

// RefreshPresence 延长仍由 serverName 持有的在线状态。
func (s *Service) RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	return s.presences.RefreshPresence(ctx, playerID, serverName, updatedAt, ttl)
}

// CreatePlayer 存储玩家档案。
func (s *Service) CreatePlayer(ctx context.Context, player *state.Player) error {
	return s.players.CreatePlayer(ctx, player)
}

// GetPlayer 按 ID 读取玩家档案。
func (s *Service) GetPlayer(ctx context.Context, id int64) (*state.Player, error) {
	return s.players.GetPlayer(ctx, id)
}

// NextPlayerID 分配下一个玩家 ID。
func (s *Service) NextPlayerID(ctx context.Context) (int64, error) {
	return s.players.NextPlayerID(ctx)
}

// CreateSession 存储登录会话。
func (s *Service) CreateSession(ctx context.Context, session *state.Session) error {
	return s.sessions.CreateSession(ctx, session)
}

// GetSession 按令牌读取登录会话。
func (s *Service) GetSession(ctx context.Context, token string) (*state.Session, error) {
	return s.sessions.GetSession(ctx, token)
}

// DeleteSession 删除登录会话。
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.sessions.DeleteSession(ctx, token)
}

// CreateAccount 存储账号凭据。
func (s *Service) CreateAccount(ctx context.Context, account *state.Account) error {
	return s.accounts.CreateAccount(ctx, account)
}

// GetAccount 按用户名读取账号凭据。
func (s *Service) GetAccount(ctx context.Context, username string) (*state.Account, error) {
	return s.accounts.GetAccount(ctx, username)
}

// RegisterAccount 一并创建账号、玩家和会话状态。
func (s *Service) RegisterAccount(ctx context.Context, input state.RegisterAccountInput) (*state.RegisterAccountResult, error) {
	return s.registrations.RegisterAccount(ctx, input)
}

func (s *Service) SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return s.friends.SendFriendRequest(ctx, fromPlayerID, toPlayerID)
}

func (s *Service) ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error) {
	return s.friends.ListIncomingFriendRequests(ctx, playerID)
}

func (s *Service) ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*state.FriendRequest, error) {
	return s.friends.ListOutgoingFriendRequests(ctx, playerID)
}

func (s *Service) AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return s.friends.AcceptFriendRequest(ctx, fromPlayerID, toPlayerID)
}

func (s *Service) RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return s.friends.RejectFriendRequest(ctx, fromPlayerID, toPlayerID)
}

func (s *Service) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	return s.friends.ListFriendIDs(ctx, playerID)
}

func (s *Service) DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error {
	return s.friends.DeleteFriend(ctx, playerID, friendPlayerID)
}

func (s *Service) GetGrowth(ctx context.Context, playerID int64) (*state.Growth, error) {
	return s.growth.GetGrowth(ctx, playerID)
}

func (s *Service) UpgradeGrowth(ctx context.Context, input state.UpgradeGrowthInput) (*state.UpgradeGrowthResult, error) {
	return s.growth.UpgradeGrowth(ctx, input)
}

// StoreConfig 聚合 Service 所需的各类存储。
type StoreConfig struct {
	Accounts      accountStore
	Sessions      sessionStore
	Players       playerStore
	Registrations registrationStore
	Presences     presenceStore
	Friends       friendStore
	Realtime      realtimeStore
	Growth        growthStore
	Coins         coinsStore
	Chats         chatStore
	Leaderboards  leaderboardStore
	Metrics       *Metrics
}

// NewService 使用存储实现创建状态服务。
func NewService(storeConfig StoreConfig) *Service {
	return &Service{
		registrations: storeConfig.Registrations,
		accounts:      storeConfig.Accounts,
		sessions:      storeConfig.Sessions,
		players:       storeConfig.Players,
		presences:     storeConfig.Presences,
		friends:       storeConfig.Friends,
		realtime:      storeConfig.Realtime,
		growth:        storeConfig.Growth,
		coins:         storeConfig.Coins,
		chats:         storeConfig.Chats,
		leaderboards:  storeConfig.Leaderboards,
		metrics:       storeConfig.Metrics,
	}
}
