package service

import (
	"context"
	statecontract "server/internal/contract/state"
	"time"
)

type accountStore interface {
	CreateAccount(ctx context.Context, account *statecontract.Account) error
	GetAccount(ctx context.Context, username string) (*statecontract.Account, error)
}

type sessionStore interface {
	CreateSession(ctx context.Context, account *statecontract.Session) error
	GetSession(ctx context.Context, token string) (*statecontract.Session, error)
	DeleteSession(ctx context.Context, token string) error
}

type playerStore interface {
	CreatePlayer(ctx context.Context, player *statecontract.Player) error
	GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error)
	NextPlayerID(ctx context.Context) (int64, error)
}

type presenceStore interface {
	SetPresence(ctx context.Context, presence *statecontract.Presence, ttl time.Duration) error
	GetPresence(ctx context.Context, playerID int64) (*statecontract.Presence, error)
	ClearPresence(ctx context.Context, playerID int64, serverName string) error
	RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error
}

type registrationStore interface {
	RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error)
}

type friendStore interface {
	SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*statecontract.FriendRequest, error)
	ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*statecontract.FriendRequest, error)
	AcceptFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	RejectFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
	DeleteFriend(ctx context.Context, playerID, friendPlayerID int64) error
}

// Service coordinates state operations across the configured stores.
type Service struct {
	registrations registrationStore
	accounts      accountStore
	sessions      sessionStore
	players       playerStore
	presences     presenceStore
	friends       friendStore
}

// SetPresence records a player's online state.
func (s *Service) SetPresence(ctx context.Context, presence *statecontract.Presence, ttl time.Duration) error {
	return s.presences.SetPresence(ctx, presence, ttl)
}

// GetPresence loads a player's online state.
func (s *Service) GetPresence(ctx context.Context, playerID int64) (*statecontract.Presence, error) {
	return s.presences.GetPresence(ctx, playerID)
}

// ClearPresence removes online state still owned by serverName.
func (s *Service) ClearPresence(ctx context.Context, playerID int64, serverName string) error {
	return s.presences.ClearPresence(ctx, playerID, serverName)
}

// RefreshPresence extends online state still owned by serverName.
func (s *Service) RefreshPresence(ctx context.Context, playerID int64, serverName string, updatedAt time.Time, ttl time.Duration) error {
	return s.presences.RefreshPresence(ctx, playerID, serverName, updatedAt, ttl)
}

// CreatePlayer stores a player profile.
func (s *Service) CreatePlayer(ctx context.Context, player *statecontract.Player) error {
	return s.players.CreatePlayer(ctx, player)
}

// GetPlayer loads a player profile by ID.
func (s *Service) GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error) {
	return s.players.GetPlayer(ctx, id)
}

// NextPlayerID allocates the next player ID.
func (s *Service) NextPlayerID(ctx context.Context) (int64, error) {
	return s.players.NextPlayerID(ctx)
}

// CreateSession stores a login session.
func (s *Service) CreateSession(ctx context.Context, session *statecontract.Session) error {
	return s.sessions.CreateSession(ctx, session)
}

// GetSession loads a login session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*statecontract.Session, error) {
	return s.sessions.GetSession(ctx, token)
}

// DeleteSession removes a login session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.sessions.DeleteSession(ctx, token)
}

// CreateAccount stores account credentials.
func (s *Service) CreateAccount(ctx context.Context, account *statecontract.Account) error {
	return s.accounts.CreateAccount(ctx, account)
}

// GetAccount loads account credentials by username.
func (s *Service) GetAccount(ctx context.Context, username string) (*statecontract.Account, error) {
	return s.accounts.GetAccount(ctx, username)
}

// RegisterAccount creates account, player, and session state together.
func (s *Service) RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	return s.registrations.RegisterAccount(ctx, input)
}

func (s *Service) SendFriendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return s.friends.SendFriendRequest(ctx, fromPlayerID, toPlayerID)
}

func (s *Service) ListIncomingFriendRequests(ctx context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	return s.friends.ListIncomingFriendRequests(ctx, playerID)
}

func (s *Service) ListOutgoingFriendRequests(ctx context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
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

// StoreConfig groups the stores required by Service.
type StoreConfig struct {
	Accounts      accountStore
	Sessions      sessionStore
	Players       playerStore
	Registrations registrationStore
	Presences     presenceStore
	Friends       friendStore
}

// NewService creates a state service from store implementations.
func NewService(storeConfig StoreConfig) *Service {
	return &Service{
		accounts:      storeConfig.Accounts,
		sessions:      storeConfig.Sessions,
		players:       storeConfig.Players,
		registrations: storeConfig.Registrations,
		presences:     storeConfig.Presences,
		friends:       storeConfig.Friends,
	}
}
