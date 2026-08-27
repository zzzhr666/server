package grpcclient

import (
	"context"
	"server/internal/contract/battlepb"
)

// CreateRoomInput 携带 rcenter 发送给战斗节点的房间数据。
type CreateRoomInput struct {
	RoomName       string
	Token          string
	PlayerIDs      []int64
	PlayerLoadouts []PlayerLoadout
}

// PlayerLoadout 携带一名玩家选择的战斗配置。
type PlayerLoadout struct {
	PlayerID         int64
	Nickname         string
	Hero             string
	AttackLevel      int32
	AttackSpeedLevel int32
	HealthLevel      int32
	MoveSpeedLevel   int32
}

// CreateRoomStatus 是为 rcenter 业务代码归一化的战斗客户端状态。
type CreateRoomStatus string

const (
	// CreateRoomStatusOK 表示战斗节点成功创建了房间。
	CreateRoomStatusOK CreateRoomStatus = "ok"
	// CreateRoomStatusInvalidRequest 表示创建请求缺少必填数据。
	CreateRoomStatusInvalidRequest CreateRoomStatus = "invalid_request"
	// CreateRoomStatusAlreadyExists 表示请求的房间已经存在。
	CreateRoomStatusAlreadyExists CreateRoomStatus = "already_exists"
	// CreateRoomStatusInternalError 表示战斗节点发生内部错误。
	CreateRoomStatusInternalError CreateRoomStatus = "internal_error"
	// CreateRoomStatusUnexpected 保留未知的 protobuf 状态值。
	CreateRoomStatusUnexpected CreateRoomStatus = "unexpected"
)

// CreateRoomResult 包含归一化后的战斗房间创建响应。
type CreateRoomResult struct {
	Status  CreateRoomStatus
	Message string
}

// EndRoomInput 携带用于结束运行中战斗房间的控制面请求。
type EndRoomInput struct {
	RoomName string
	Reason   string
}

// EndRoomStatus 是为工具和调用方归一化的战斗房间结束状态。
type EndRoomStatus string

const (
	// EndRoomStatusOK 表示战斗节点成功结束了房间。
	EndRoomStatusOK EndRoomStatus = "ok"
	// EndRoomStatusInvalidRequest 表示结束请求缺少必填数据。
	EndRoomStatusInvalidRequest EndRoomStatus = "invalid_request"
	// EndRoomStatusRoomNotFound 表示战斗节点没有该房间的运行实例。
	EndRoomStatusRoomNotFound EndRoomStatus = "room_not_found"
	// EndRoomStatusInternalError 表示战斗节点发生内部错误。
	EndRoomStatusInternalError EndRoomStatus = "internal_error"
	// EndRoomStatusUnexpected 保留未知的 protobuf 状态值。
	EndRoomStatusUnexpected EndRoomStatus = "unexpected"
)

// EndRoomResult 包含归一化后的战斗房间结束响应。
type EndRoomResult struct {
	Status  EndRoomStatus
	Message string
}

// Client 将生成的 BattleControlService gRPC 客户端适配给 rcenter。
type Client struct {
	client battlepb.BattleControlServiceClient
}

// NewClient 包装生成的 BattleControlService 客户端。
func NewClient(client battlepb.BattleControlServiceClient) *Client {
	return &Client{client: client}
}

// CreateRoom 请求战斗节点为已匹配玩家预留房间。
func (c *Client) CreateRoom(ctx context.Context, input CreateRoomInput) (*CreateRoomResult, error) {
	req := &battlepb.CreateRoomRequest{
		RoomName:  input.RoomName,
		Token:     input.Token,
		PlayerIds: input.PlayerIDs,
	}
	req.PlayerLoadouts = make([]*battlepb.PlayerLoadout, 0, len(input.PlayerLoadouts))
	for _, loadout := range input.PlayerLoadouts {
		req.PlayerLoadouts = append(req.PlayerLoadouts, &battlepb.PlayerLoadout{
			PlayerId:         loadout.PlayerID,
			Nickname:         loadout.Nickname,
			Hero:             loadout.Hero,
			AttackLevel:      loadout.AttackLevel,
			AttackSpeedLevel: loadout.AttackSpeedLevel,
			HealthLevel:      loadout.HealthLevel,
			MoveSpeedLevel:   loadout.MoveSpeedLevel,
		})
	}

	res, err := c.client.CreateRoom(ctx, req)
	if err != nil {
		return nil, err
	}
	return &CreateRoomResult{
		Status:  fromProtoCreateRoomStatus(res.GetStatus()),
		Message: res.GetMessage(),
	}, nil
}

func fromProtoCreateRoomStatus(status battlepb.CreateRoomStatus) CreateRoomStatus {
	switch status {
	case battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_OK:
		return CreateRoomStatusOK
	case battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_INVALID_REQUEST:
		return CreateRoomStatusInvalidRequest
	case battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_ALREADY_EXISTS:
		return CreateRoomStatusAlreadyExists
	case battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_INTERNAL_ERROR:
		return CreateRoomStatusInternalError
	default:
		return CreateRoomStatusUnexpected
	}
}

// EndRoom 请求战斗节点结束运行中的房间。
func (c *Client) EndRoom(ctx context.Context, input EndRoomInput) (*EndRoomResult, error) {
	res, err := c.client.EndRoom(ctx, &battlepb.EndRoomRequest{
		RoomName: input.RoomName,
		Reason:   input.Reason,
	})
	if err != nil {
		return nil, err
	}
	return &EndRoomResult{
		Status:  fromProtoEndRoomStatus(res.GetStatus()),
		Message: res.GetMessage(),
	}, nil
}

func fromProtoEndRoomStatus(status battlepb.EndRoomStatus) EndRoomStatus {
	switch status {
	case battlepb.EndRoomStatus_END_ROOM_STATUS_OK:
		return EndRoomStatusOK
	case battlepb.EndRoomStatus_END_ROOM_STATUS_INVALID_REQUEST:
		return EndRoomStatusInvalidRequest
	case battlepb.EndRoomStatus_END_ROOM_STATUS_ROOM_NOT_FOUND:
		return EndRoomStatusRoomNotFound
	case battlepb.EndRoomStatus_END_ROOM_STATUS_INTERNAL_ERROR:
		return EndRoomStatusInternalError
	default:
		return EndRoomStatusUnexpected
	}
}
