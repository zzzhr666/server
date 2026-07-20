package friend

import "context"

type Service interface {
	SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
	DeleteFriend(ctx context.Context, playerID, friendID int64) error
}

type Repository interface {
	SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
	DeleteFriend(ctx context.Context, playerID, friendID int64) error
}

type GameFriendService struct {
	friendRepo Repository
}

func (g *GameFriendService) SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	return g.friendRepo.SendRequest(ctx, fromPlayerID, toPlayerID)
}

func (g *GameFriendService) ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlayerID(playerID); err != nil {
		return nil, err
	}
	return g.friendRepo.ListIncomingRequests(ctx, playerID)
}

func (g *GameFriendService) ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlayerID(playerID); err != nil {
		return nil, err
	}
	return g.friendRepo.ListOutgoingRequests(ctx, playerID)
}

func (g *GameFriendService) AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	return g.friendRepo.AcceptRequest(ctx, fromPlayerID, toPlayerID)
}

func (g *GameFriendService) RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	return g.friendRepo.RejectRequest(ctx, fromPlayerID, toPlayerID)
}

func (g *GameFriendService) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlayerID(playerID); err != nil {
		return nil, err
	}
	return g.friendRepo.ListFriendIDs(ctx, playerID)
}

func (g *GameFriendService) DeleteFriend(ctx context.Context, playerID, friendID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(playerID, friendID); err != nil {
		return err
	}
	return g.friendRepo.DeleteFriend(ctx, playerID, friendID)
}

func NewService(friendRepo Repository) *GameFriendService {
	return &GameFriendService{friendRepo: friendRepo}
}

func validatePair(fromPlayerID, toPlayerID int64) error {
	if fromPlayerID <= 0 || toPlayerID <= 0 || fromPlayerID == toPlayerID {
		return ErrInvalidRequest
	}
	return nil
}
func validatePlayerID(playerID int64) error {
	if playerID <= 0 {
		return ErrInvalidPlayerID
	}
	return nil
}
