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
