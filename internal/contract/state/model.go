package state

import (
	"context"
	"time"
)

// Account stores login credentials and the bound player ID.
type Account struct {
	Username     string
	PasswordHash string
	PlayerID     int64
}

// Session stores an authenticated player session.
type Session struct {
	Token     string
	PlayerID  int64
	ExpiresAt time.Time
}

// Player stores user-visible player profile data.
type Player struct {
	ID       int64
	Nickname string
	Avatar   string
	Email    string
	Phone    string
}

// RegisterAccountInput groups the state data needed for account registration.
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

// RegisterAccountResult returns all records created during registration.
type RegisterAccountResult struct {
	Account *Account
	Player  *Player
	Session *Session
}

// Presence records where a player is currently connected.
type Presence struct {
	PlayerID   int64
	ServerName string
	Status     string
	UpdatedAt  time.Time
}

// Client defines state-server operations needed by other processes.
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

// PresenceClient defines state-server operations for player online state.
type PresenceClient interface {
	SetPresence(ctx context.Context, presence *Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error
}

type FriendRequest struct {
	FromPlayerID int64
	ToPlayerID   int64
	CreatedAt    time.Time
}

// FriendClient defines state-server operations for friend relationships.
type FriendClient interface {
	SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*FriendRequest, error)
	ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*FriendRequest, error)
	AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListFriendIDs(ctx context.Context, fromPlayerID int64) ([]int64, error)
	DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error
}

// RealtimeEvent describes a message routed to one connected player.
type RealtimeEvent struct {
	Type           string
	TargetPlayerID int64
	ActorPlayerID  int64
	Online         bool
	Status         string
	MatchStatus    string
	RoomName       string
	MatchToken     string
	BattleNodeName string
	BattleKCPAddr  string
	MatchPlayerIDs []int64
}

const (
	// RealtimeEventFriendPresenceChanged notifies friends about online-state changes.
	RealtimeEventFriendPresenceChanged = "friend_presence_changed"
	// RealtimeEventFriendRemoved notifies a player that another player removed the friendship.
	RealtimeEventFriendRemoved = "friend_removed"

	// RealtimeEventFriendRequestReceived notifies a player about a new incoming friend request.
	RealtimeEventFriendRequestReceived = "friend_request_received"
	// RealtimeEventFriendRequestHandled notifies a requester that a friend request was handled.
	RealtimeEventFriendRequestHandled = "friend_request_handled"

	// RealtimeEventConnectionReplaced tells an old connection that a newer login replaced it.
	RealtimeEventConnectionReplaced = "connection_replaced"
)

// RealtimeClient defines cross-logic-server realtime message routing.
type RealtimeClient interface {
	PublishRealtimeToServer(ctx context.Context, serverName string, event *RealtimeEvent) error
	SubscribeRealtime(ctx context.Context, serverName string) (<-chan *RealtimeEvent, error)
}

// RealtimeEventMatchResult delivers a cross-logic-server match result.
const RealtimeEventMatchResult = "match_result"
