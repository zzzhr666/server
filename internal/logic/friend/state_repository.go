package friend

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
)

// StateRepository 将好友领域操作适配到 state-server 客户端。
type StateRepository struct {
	stateClient statecontract.FriendClient
}

// SendRequest 通过 state-server 创建好友申请。
func (s *StateRepository) SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return mapStateError(s.stateClient.SendFriendRequest(ctx, fromPlayerID, toPlayerID))
}

// ListIncomingRequests 从 state-server 读取收到的申请并转换为领域模型。
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

// ListOutgoingRequests 从 state-server 读取发出的申请并转换为领域模型。
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

// AcceptRequest 通过 state-server 接受好友申请。
func (s *StateRepository) AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return mapStateError(s.stateClient.AcceptFriendRequest(ctx, fromPlayerID, toPlayerID))
}

// RejectRequest 通过 state-server 拒绝好友申请。
func (s *StateRepository) RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	return mapStateError(s.stateClient.RejectFriendRequest(ctx, fromPlayerID, toPlayerID))
}

// ListFriendIDs 从 state-server 读取好友 ID 列表。
func (s *StateRepository) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	ids, err := s.stateClient.ListFriendIDs(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	return ids, nil
}

// DeleteFriend 通过 state-server 删除好友关系。
func (s *StateRepository) DeleteFriend(ctx context.Context, playerID, friendID int64) error {
	return mapStateError(s.stateClient.DeleteFriend(ctx, playerID, friendID))
}

// NewStateRepository 使用 state-server 客户端创建好友仓储。
func NewStateRepository(client statecontract.FriendClient) *StateRepository {
	return &StateRepository{stateClient: client}
}

// fromStateRequest 将状态层好友申请转换为领域模型。
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

// mapStateError 将状态层错误收敛为好友领域错误。
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
