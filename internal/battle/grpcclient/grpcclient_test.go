package grpcclient

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"server/internal/contract/battlepb"

	"google.golang.org/grpc"
)

func TestClientCreateRoom(t *testing.T) {
	grpcBattle := &fakeBattleControlServiceClient{
		createRoomResponse: &battlepb.CreateRoomResponse{
			Status:  battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_OK,
			Message: "room created",
		},
	}
	client := NewClient(grpcBattle)

	result, err := client.CreateRoom(context.Background(), CreateRoomInput{
		RoomName:  "room-1",
		Token:     "token-1",
		PlayerIDs: []int64{7, 8},
	})
	if err != nil {
		t.Fatalf("CreateRoom returned error: %v", err)
	}
	if grpcBattle.createRoomRequest.GetRoomName() != "room-1" {
		t.Fatalf("room name = %q, want room-1", grpcBattle.createRoomRequest.GetRoomName())
	}
	if grpcBattle.createRoomRequest.GetToken() != "token-1" {
		t.Fatalf("token = %q, want token-1", grpcBattle.createRoomRequest.GetToken())
	}
	if !reflect.DeepEqual(grpcBattle.createRoomRequest.GetPlayerIds(), []int64{7, 8}) {
		t.Fatalf("player ids = %v, want [7 8]", grpcBattle.createRoomRequest.GetPlayerIds())
	}
	if result.Status != CreateRoomStatusOK {
		t.Fatalf("status = %q, want %q", result.Status, CreateRoomStatusOK)
	}
	if result.Message != "room created" {
		t.Fatalf("message = %q, want room created", result.Message)
	}
}

func TestClientCreateRoomMapsStatuses(t *testing.T) {
	tests := []struct {
		name string
		in   battlepb.CreateRoomStatus
		want CreateRoomStatus
	}{
		{name: "invalid request", in: battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_INVALID_REQUEST, want: CreateRoomStatusInvalidRequest},
		{name: "already exists", in: battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_ALREADY_EXISTS, want: CreateRoomStatusAlreadyExists},
		{name: "internal error", in: battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_INTERNAL_ERROR, want: CreateRoomStatusInternalError},
		{name: "unexpected", in: battlepb.CreateRoomStatus_CREATE_ROOM_STATUS_UNSPECIFIED, want: CreateRoomStatusUnexpected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&fakeBattleControlServiceClient{
				createRoomResponse: &battlepb.CreateRoomResponse{Status: tt.in},
			})

			result, err := client.CreateRoom(context.Background(), CreateRoomInput{})
			if err != nil {
				t.Fatalf("CreateRoom returned error: %v", err)
			}
			if result.Status != tt.want {
				t.Fatalf("status = %q, want %q", result.Status, tt.want)
			}
		})
	}
}

func TestClientCreateRoomReturnsGRPCError(t *testing.T) {
	wantErr := errors.New("battle unavailable")
	client := NewClient(&fakeBattleControlServiceClient{err: wantErr})

	_, err := client.CreateRoom(context.Background(), CreateRoomInput{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("CreateRoom error = %v, want %v", err, wantErr)
	}
}

type fakeBattleControlServiceClient struct {
	createRoomRequest  *battlepb.CreateRoomRequest
	createRoomResponse *battlepb.CreateRoomResponse
	err                error
}

func (f *fakeBattleControlServiceClient) CreateRoom(ctx context.Context, req *battlepb.CreateRoomRequest, opts ...grpc.CallOption) (*battlepb.CreateRoomResponse, error) {
	f.createRoomRequest = req
	if f.err != nil {
		return nil, f.err
	}
	return f.createRoomResponse, nil
}

func (f *fakeBattleControlServiceClient) JoinRoom(ctx context.Context, req *battlepb.JoinRoomRequest, opts ...grpc.CallOption) (*battlepb.JoinRoomResponse, error) {
	return nil, errors.New("JoinRoom is not used by these tests")
}
