package friend

import (
	"context"
	"server/internal/platform/logging"
)

// Service 定义 HTTP 和 TCP 层可调用的好友业务操作。
type Service interface {
	// SendRequest 向目标玩家发送好友申请。
	SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	// ListIncomingRequests 返回玩家收到的待处理申请。
	ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	// ListOutgoingRequests 返回玩家发出的待处理申请。
	ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	// AcceptRequest 接受一条好友申请并建立双向好友关系。
	AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	// RejectRequest 拒绝并删除一条好友申请。
	RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	// ListFriendIDs 返回玩家当前的好友 ID 列表。
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
	// DeleteFriend 删除双方的好友关系。
	DeleteFriend(ctx context.Context, playerID, friendID int64) error
}

// Repository 定义好友服务依赖的持久化操作。
type Repository interface {
	// SendRequest 持久化一条好友申请。
	SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	// ListIncomingRequests 读取玩家收到的申请。
	ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	// ListOutgoingRequests 读取玩家发出的申请。
	ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error)
	// AcceptRequest 在持久层接受申请。
	AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	// RejectRequest 在持久层拒绝申请。
	RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error
	// ListFriendIDs 读取好友 ID 列表。
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
	// DeleteFriend 在持久层删除好友关系。
	DeleteFriend(ctx context.Context, playerID, friendID int64) error
}

// GameFriendService 在转交仓储前校验好友领域输入。
type GameFriendService struct {
	friendRepo Repository
}

// SendRequest 校验双方玩家后创建好友申请。
func (g *GameFriendService) SendRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	if err := g.friendRepo.SendRequest(ctx, fromPlayerID, toPlayerID); err != nil {
		logging.Warn("friend request failed from_player_id=%d to_player_id=%d: %v", fromPlayerID, toPlayerID, err)
		return err
	}
	logging.Info("friend request sent from_player_id=%d to_player_id=%d", fromPlayerID, toPlayerID)
	return nil
}

// ListIncomingRequests 返回玩家收到的全部待处理好友申请。
func (g *GameFriendService) ListIncomingRequests(ctx context.Context, playerID int64) ([]*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlayerID(playerID); err != nil {
		return nil, err
	}
	return g.friendRepo.ListIncomingRequests(ctx, playerID)
}

// ListOutgoingRequests 返回玩家发出的全部待处理好友申请。
func (g *GameFriendService) ListOutgoingRequests(ctx context.Context, playerID int64) ([]*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlayerID(playerID); err != nil {
		return nil, err
	}
	return g.friendRepo.ListOutgoingRequests(ctx, playerID)
}

// AcceptRequest 接受由 fromPlayerID 发给 toPlayerID 的申请。
func (g *GameFriendService) AcceptRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	if err := g.friendRepo.AcceptRequest(ctx, fromPlayerID, toPlayerID); err != nil {
		logging.Warn("friend request accept failed from_player_id=%d to_player_id=%d: %v", fromPlayerID, toPlayerID, err)
		return err
	}
	logging.Info("friend request accepted from_player_id=%d to_player_id=%d", fromPlayerID, toPlayerID)
	return nil
}

// RejectRequest 拒绝由 fromPlayerID 发给 toPlayerID 的申请。
func (g *GameFriendService) RejectRequest(ctx context.Context, fromPlayerID, toPlayerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(fromPlayerID, toPlayerID); err != nil {
		return err
	}
	if err := g.friendRepo.RejectRequest(ctx, fromPlayerID, toPlayerID); err != nil {
		logging.Debug("friend request reject failed from_player_id=%d to_player_id=%d: %v", fromPlayerID, toPlayerID, err)
		return err
	}
	logging.Info("friend request rejected from_player_id=%d to_player_id=%d", fromPlayerID, toPlayerID)
	return nil
}

// ListFriendIDs 返回指定玩家的好友 ID。
func (g *GameFriendService) ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlayerID(playerID); err != nil {
		return nil, err
	}
	return g.friendRepo.ListFriendIDs(ctx, playerID)
}

// DeleteFriend 删除指定玩家与其好友的关系。
func (g *GameFriendService) DeleteFriend(ctx context.Context, playerID, friendID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePair(playerID, friendID); err != nil {
		return err
	}
	if err := g.friendRepo.DeleteFriend(ctx, playerID, friendID); err != nil {
		logging.Warn("delete friend failed player_id=%d friend_id=%d: %v", playerID, friendID, err)
		return err
	}
	logging.Info("friend deleted player_id=%d friend_id=%d", playerID, friendID)
	return nil
}

// NewService 使用指定仓储创建好友业务服务。
func NewService(friendRepo Repository) *GameFriendService {
	return &GameFriendService{friendRepo: friendRepo}
}

// validatePair 确保双方 ID 有效且不指向同一玩家。
func validatePair(fromPlayerID, toPlayerID int64) error {
	if fromPlayerID <= 0 || toPlayerID <= 0 || fromPlayerID == toPlayerID {
		return ErrInvalidRequest
	}
	return nil
}

// validatePlayerID 确保玩家 ID 为正数。
func validatePlayerID(playerID int64) error {
	if playerID <= 0 {
		return ErrInvalidPlayerID
	}
	return nil
}
