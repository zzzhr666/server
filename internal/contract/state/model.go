package state

import (
	"context"
	"time"
)

// Account 保存登录凭据和绑定的玩家 ID。
type Account struct {
	Username     string
	PasswordHash string
	PlayerID     int64
}

// Session 保存已认证玩家的登录会话。
type Session struct {
	Token     string
	PlayerID  int64
	ExpiresAt time.Time
}

// Player 保存玩家对外展示的档案数据。
type Player struct {
	ID       int64
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Coins    int64
}

// Growth 保存玩家局外成长等级。
type Growth struct {
	PlayerID         int64
	AttackLevel      int32
	AttackSpeedLevel int32
	HealthLevel      int32
	MoveSpeedLevel   int32
}

// RegisterAccountInput 聚合注册账号时需要写入的状态数据。
type RegisterAccountInput struct {
	Username         string
	PasswordHash     string
	Nickname         string
	Avatar           string
	Email            string
	Phone            string
	SessionToken     string
	SessionExpiresAt time.Time
}

// RegisterAccountResult 返回注册流程创建的账号、玩家和会话记录。
type RegisterAccountResult struct {
	Account *Account
	Player  *Player
	Session *Session
}

// Presence 记录玩家当前连接所在的 logic-server。
type Presence struct {
	PlayerID   int64
	ServerName string
	Status     string
	UpdatedAt  time.Time
}

// Client 定义其他进程需要调用的 state-server 基础状态操作。
type Client interface {
	CreateAccount(ctx context.Context, account *Account) error
	GetAccount(ctx context.Context, username string) (*Account, error)
	RegisterAccount(ctx context.Context, input RegisterAccountInput) (*RegisterAccountResult, error)

	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, token string) (*Session, error)
	DeleteSession(ctx context.Context, token string) error

	CreatePlayer(ctx context.Context, player *Player) error
	GetPlayer(ctx context.Context, id int64) (*Player, error)
	NextPlayerID(ctx context.Context) (int64, error)
}

// PresenceClient 定义 state-server 提供的玩家在线状态操作。
type PresenceClient interface {
	SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error
}

// FriendRequest 表示一条待处理的好友申请。
type FriendRequest struct {
	FromPlayerID int64
	ToPlayerID   int64
	CreatedAt    time.Time
}

// FriendClient 定义 state-server 提供的好友关系操作。
type FriendClient interface {
	SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*FriendRequest, error)
	ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*FriendRequest, error)
	AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListFriendIDs(ctx context.Context, fromPlayerID int64) ([]int64, error)
	DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error
}

// RealtimeEvent 描述需要投递给在线玩家的实时事件内容。
type RealtimeEvent struct {
	Type                string
	TargetPlayerID      int64
	ActorPlayerID       int64
	ActorPlayerNickname string
	Online              bool
	Status              string
	MatchStatus         string
	RoomName            string
	MatchToken          string
	BattleNodeName      string
	BattleUDPAddr       string
	MatchPlayerIDs      []int64
	ChatMessage         *ChatMessage
}

const (
	// RealtimeEventFriendPresenceChanged 通知好友某个玩家的在线状态变化。
	RealtimeEventFriendPresenceChanged = "friend_presence_changed"
	// RealtimeEventFriendRemoved 通知玩家某个好友关系已被对方删除。
	RealtimeEventFriendRemoved = "friend_removed"

	// RealtimeEventFriendRequestReceived 通知玩家收到新的好友申请。
	RealtimeEventFriendRequestReceived = "friend_request_received"
	// RealtimeEventFriendRequestHandled 通知申请方好友申请已被处理。
	RealtimeEventFriendRequestHandled = "friend_request_handled"

	// RealtimeEventConnectionReplaced 通知旧连接已被新的登录连接替换。
	RealtimeEventConnectionReplaced = "connection_replaced"

	// RealtimeEventChatMessage 通知在线玩家收到新的私聊消息。
	RealtimeEventChatMessage = "chat_message"
)

// RealtimeClient 定义跨 logic-server 的实时投递发布与订阅操作。
type RealtimeClient interface {
	PublishRealtime(ctx context.Context, delivery *RealtimeDelivery) error
	SubscribeRealtime(ctx context.Context, route RealtimeRoute) (<-chan *RealtimeDelivery, error)
}

// RealtimeEventMatchResult 表示跨 logic-server 投递的匹配结果事件。
const RealtimeEventMatchResult = "match_result"

const RealtimeBroadcastChannelName = "broadcast"

// UpgradeGrowthInput 描述一次成长升级需要的校验和扣费参数。
type UpgradeGrowthInput struct {
	PlayerID     int64
	UpgradeField string
	Cost         int64
	MaxLevel     int32
}

// UpgradeGrowthResult 返回成长升级后的等级和剩余金币。
type UpgradeGrowthResult struct {
	Growth         *Growth
	RemainingCoins int64
}

// GrowthClient 定义 state-server 提供的成长状态操作。
type GrowthClient interface {
	GetGrowth(ctx context.Context, playerID int64) (*Growth, error)
	UpgradeGrowth(ctx context.Context, input UpgradeGrowthInput) (*UpgradeGrowthResult, error)
}

// AddPlayerCoinsInput 描述一次玩家金币变更。
type AddPlayerCoinsInput struct {
	PlayerID int64
	Amount   int64
}

// AddPlayerCoinsResult 返回金币变更后的玩家金币数量。
type AddPlayerCoinsResult struct {
	PlayerID int64
	Coins    int64
}

// CoinClient 定义 state-server 提供的金币状态操作。
type CoinClient interface {
	AddPlayerCoins(ctx context.Context, input AddPlayerCoinsInput) (*AddPlayerCoinsResult, error)
}

// ChatClient 定义 state-server 提供的聊天消息持久化操作。
type ChatClient interface {
	// SaveChatMessage 保存一条聊天消息并按频道保留上限清理旧记录。
	SaveChatMessage(ctx context.Context, input SaveChatMessageInput) (*ChatMessage, error)
	// ListChatMessages 返回指定频道的持久化聊天记录。
	ListChatMessages(ctx context.Context, input ListChatMessagesInput) ([]*ChatMessage, error)
}

// ChatChannelType 标识聊天频道的业务类型。
type ChatChannelType string

const (
	// ChatChannelWorld 表示全局世界聊天频道。
	ChatChannelWorld ChatChannelType = "world"
	// ChatChannelDirect 表示两个好友之间的私聊频道。
	ChatChannelDirect ChatChannelType = "direct"
)

const (
	// WorldChatChannelKey 是全局世界频道的稳定存储键。
	WorldChatChannelKey = "world:global"

	// WorldChatMaxMessages 限制世界频道最多保留的消息条数。
	WorldChatMaxMessages = 100
	// WorldChatRetention 定义世界频道消息的最长可读时间。
	WorldChatRetention = 24 * time.Hour

	// DirectChatMaxMessages 限制每个私聊会话最多保留的消息条数。
	DirectChatMaxMessages = 50
	// DirectChatRetention 定义私聊消息的最长可读时间。
	DirectChatRetention = 24 * time.Hour
)

// ChatMessage 表示 state-server 返回的一条持久化聊天记录。
type ChatMessage struct {
	MessageKey       string
	ChannelType      ChatChannelType
	ChannelKey       string
	SenderID         int64
	ReceiverID       int64
	Content          string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	ClientMessageKey string
	SenderNickname   string
}

// SaveChatMessageInput 描述一次聊天消息写入及其保留规则。
type SaveChatMessageInput struct {
	ChannelType      ChatChannelType
	ChannelKey       string
	SenderID         int64
	ReceiverID       int64
	Content          string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	MaxMessages      int64
	ClientMessageKey string
	SenderNickname   string
}

// ListChatMessagesInput 描述指定频道的分页聊天历史查询。
type ListChatMessagesInput struct {
	ChannelType      ChatChannelType
	ChannelKey       string
	Limit            int64
	BeforeMessageKey string
}

// RealtimeRouteType 表示实时投递的目标范围。
type RealtimeRouteType string

const (
	// RealtimeRouteServer 将事件投递到指定 logic-server。
	RealtimeRouteServer RealtimeRouteType = "server"
	// RealtimeRouteBroadcast 将事件投递到所有 logic-server。
	RealtimeRouteBroadcast RealtimeRouteType = "broadcast"
)

// RealtimeRoute 描述实时事件的投递目标。
type RealtimeRoute struct {
	Type       RealtimeRouteType
	ServerName string
}

// RealtimeDelivery 将实时事件及其投递目标组合为完整传输单元。
type RealtimeDelivery struct {
	Route RealtimeRoute
	Event *RealtimeEvent
}
