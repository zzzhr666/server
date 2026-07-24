package grpcclient

import (
	"context"
	"server/internal/contract/battlepb"
)

// CreateRoomInput carries the room payload sent from rcenter to a battle node.
type CreateRoomInput struct {
	RoomName  string
	Token     string
	PlayerIDs []int64
}

// CreateRoomStatus is the battle client status normalized for rcenter business code.
type CreateRoomStatus string

const (
	// CreateRoomStatusOK means the battle node created the room successfully.
	CreateRoomStatusOK CreateRoomStatus = "ok"
	// CreateRoomStatusInvalidRequest means the create request missed required data.
	CreateRoomStatusInvalidRequest CreateRoomStatus = "invalid_request"
	// CreateRoomStatusAlreadyExists means the requested room already exists.
	CreateRoomStatusAlreadyExists CreateRoomStatus = "already_exists"
	// CreateRoomStatusInternalError means the battle node failed internally.
	CreateRoomStatusInternalError CreateRoomStatus = "internal_error"
	// CreateRoomStatusUnexpected preserves unknown protobuf status values.
	CreateRoomStatusUnexpected CreateRoomStatus = "unexpected"
)

// CreateRoomResult contains the normalized battle room creation response.
type CreateRoomResult struct {
	Status  CreateRoomStatus
	Message string
}

// EndRoomInput carries a control-plane request to end a running battle room.
type EndRoomInput struct {
	RoomName string
	Reason   string
}

// EndRoomStatus is the battle room ending status normalized for tools and callers.
type EndRoomStatus string

const (
	// EndRoomStatusOK means the battle node ended the room successfully.
	EndRoomStatusOK EndRoomStatus = "ok"
	// EndRoomStatusInvalidRequest means the end request missed required data.
	EndRoomStatusInvalidRequest EndRoomStatus = "invalid_request"
	// EndRoomStatusRoomNotFound means the battle node has no running instance for the room.
	EndRoomStatusRoomNotFound EndRoomStatus = "room_not_found"
	// EndRoomStatusInternalError means the battle node failed internally.
	EndRoomStatusInternalError EndRoomStatus = "internal_error"
	// EndRoomStatusUnexpected preserves unknown protobuf status values.
	EndRoomStatusUnexpected EndRoomStatus = "unexpected"
)

// EndRoomResult contains the normalized battle room ending response.
type EndRoomResult struct {
	Status  EndRoomStatus
	Message string
}

// Client adapts the generated BattleControlService gRPC client for rcenter.
type Client struct {
	client battlepb.BattleControlServiceClient
}

// NewClient wraps a generated BattleControlService client.
func NewClient(client battlepb.BattleControlServiceClient) *Client {
	return &Client{client: client}
}

// CreateRoom asks a battle node to reserve a room for matched players.
func (c *Client) CreateRoom(ctx context.Context, input CreateRoomInput) (*CreateRoomResult, error) {
	res, err := c.client.CreateRoom(ctx, &battlepb.CreateRoomRequest{
		RoomName:  input.RoomName,
		Token:     input.Token,
		PlayerIds: input.PlayerIDs,
	})
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

// EndRoom asks a battle node to end a running room.
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
