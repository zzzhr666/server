package friend

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
)

type StateRepository struct {
	stateClient statecontract.FriendClient
}

func (s *StateRepository) SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return mapStateError(s.stateClient.SendFriendRequest(ctx, fromPlayerID, toPlayerID))
}

func (s *StateRepository) ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error) {
	requests, err := s.stateClient.ListIncomingFriendRequests(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	results := make([]*Request, 0, len(requests))
	for _, request := range requests {
		results = append(results, fromStateRequest(request))
	}
	return results, nil
}

func (s *StateRepository) ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error) {
	requests, err := s.stateClient.ListOutgoingFriendRequests(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	results := make([]*Request, 0, len(requests))
	for _, request := range requests {
		results = append(results, fromStateRequest(request))
	}
	return results, nil
}

func (s *StateRepository) AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return mapStateError(s.stateClient.AcceptFriendRequest(ctx, fromPlayerID, toPlayerID))
}

func (s *StateRepository) RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return mapStateError(s.stateClient.RejectFriendRequest(ctx, fromPlayerID, toPlayerID))
}

func (s *StateRepository) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	ids, err := s.stateClient.ListFriendIDs(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	return ids, nil
}

func (s *StateRepository) DeleteFriend(ctx context.Context, playerID, friendID int64) error {
	return mapStateError(s.stateClient.DeleteFriend(ctx, playerID, friendID))
}

func NewStateRepository(client statecontract.FriendClient) *StateRepository {
	return &StateRepository{stateClient: client}
}

func fromStateRequest(req *statecontract.FriendRequest) *Request {
	if req == nil {
		return nil
	}
	return &Request{
		FromPlayerID: req.FromPlayerID,
		ToPlayerID:   req.ToPlayerID,
		CreatedAt:    req.CreatedAt,
	}
}

func mapStateError(err error) error {
	switch {
	case errors.Is(err, statecontract.ErrFriendNotFound):
		return ErrNotFound
	case errors.Is(err, statecontract.ErrFriendRequestNotFound):
		return ErrRequestNotFound
	case errors.Is(err, statecontract.ErrFriendAlreadyExists):
		return ErrAlreadyExists
	case errors.Is(err, statecontract.ErrFriendRequestExists):
		return ErrRequestExists
	case errors.Is(err, statecontract.ErrInvalidFriendRequest):
		return ErrInvalidRequest

	default:
		return err
	}
}
