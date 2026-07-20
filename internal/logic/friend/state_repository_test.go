package friend

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestStateRepositoryForwardsFriendOperations(t *testing.T) {
	createdAt := time.Unix(100, 0)
	state := &fakeFriendClient{
		incomingRequests: []*statecontract.FriendRequest{
			{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: createdAt},
		},
		outgoingRequests: []*statecontract.FriendRequest{
			{FromPlayerID: 8, ToPlayerID: 9, CreatedAt: createdAt.Add(time.Minute)},
		},
		friendIDs: []int64{10, 11},
	}
	repo := NewStateRepository(state)

	if err := repo.SendRequest(context.Background(), 7, 8); err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if state.sentFromPlayerID != 7 || state.sentToPlayerID != 8 {
		t.Fatalf("send request got from=%d to=%d, want from=7 to=8", state.sentFromPlayerID, state.sentToPlayerID)
	}

	incoming, err := repo.ListIncomingRequests(context.Background(), 8)
	if err != nil {
		t.Fatalf("ListIncomingRequests returned error: %v", err)
	}
	if state.listIncomingPlayerID != 8 {
		t.Fatalf("list incoming player id = %d, want 8", state.listIncomingPlayerID)
	}
	if len(incoming) != 1 || incoming[0].FromPlayerID != 7 {
		t.Fatalf("incoming requests = %+v, want one request from 7", incoming)
	}
	if !incoming[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("incoming created at = %v, want %v", incoming[0].CreatedAt, createdAt)
	}

	outgoing, err := repo.ListOutgoingRequests(context.Background(), 8)
	if err != nil {
		t.Fatalf("ListOutgoingRequests returned error: %v", err)
	}
	if state.listOutgoingPlayerID != 8 {
		t.Fatalf("list outgoing player id = %d, want 8", state.listOutgoingPlayerID)
	}
	if len(outgoing) != 1 || outgoing[0].ToPlayerID != 9 {
		t.Fatalf("outgoing requests = %+v, want one request to 9", outgoing)
	}

	if err := repo.AcceptRequest(context.Background(), 7, 8); err != nil {
		t.Fatalf("AcceptRequest returned error: %v", err)
	}
	if state.acceptedFromPlayerID != 7 || state.acceptedToPlayerID != 8 {
		t.Fatalf("accept request got from=%d to=%d, want from=7 to=8", state.acceptedFromPlayerID, state.acceptedToPlayerID)
	}

	if err := repo.RejectRequest(context.Background(), 9, 8); err != nil {
		t.Fatalf("RejectRequest returned error: %v", err)
	}
	if state.rejectedFromPlayerID != 9 || state.rejectedToPlayerID != 8 {
		t.Fatalf("reject request got from=%d to=%d, want from=9 to=8", state.rejectedFromPlayerID, state.rejectedToPlayerID)
	}

	friendIDs, err := repo.ListFriendIDs(context.Background(), 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	if state.listFriendIDsPlayerID != 7 {
		t.Fatalf("list friend ids player id = %d, want 7", state.listFriendIDsPlayerID)
	}
	if len(friendIDs) != 2 || friendIDs[0] != 10 || friendIDs[1] != 11 {
		t.Fatalf("friend ids = %v, want [10 11]", friendIDs)
	}

	if err := repo.DeleteFriend(context.Background(), 7, 10); err != nil {
		t.Fatalf("DeleteFriend returned error: %v", err)
	}
	if state.deletedPlayerID != 7 || state.deletedFriendID != 10 {
		t.Fatalf("delete friend got player=%d friend=%d, want player=7 friend=10", state.deletedPlayerID, state.deletedFriendID)
	}
}

func TestStateRepositoryMapsErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "invalid request",
			err:  statecontract.ErrInvalidFriendRequest,
			want: ErrInvalidRequest,
		},
		{
			name: "request exists",
			err:  statecontract.ErrFriendRequestExists,
			want: ErrRequestExists,
		},
		{
			name: "request not found",
			err:  statecontract.ErrFriendRequestNotFound,
			want: ErrRequestNotFound,
		},
		{
			name: "friend already exists",
			err:  statecontract.ErrFriendAlreadyExists,
			want: ErrAlreadyExists,
		},
		{
			name: "friend not found",
			err:  statecontract.ErrFriendNotFound,
			want: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewStateRepository(&fakeFriendClient{err: tt.err})

			err := repo.SendRequest(context.Background(), 7, 8)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

type fakeFriendClient struct {
	err                   error
	sentFromPlayerID      int64
	sentToPlayerID        int64
	listIncomingPlayerID  int64
	listOutgoingPlayerID  int64
	incomingRequests      []*statecontract.FriendRequest
	outgoingRequests      []*statecontract.FriendRequest
	acceptedFromPlayerID  int64
	acceptedToPlayerID    int64
	rejectedFromPlayerID  int64
	rejectedToPlayerID    int64
	listFriendIDsPlayerID int64
	friendIDs             []int64
	deletedPlayerID       int64
	deletedFriendID       int64
}

func (f *fakeFriendClient) SendFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.sentFromPlayerID = fromPlayerID
	f.sentToPlayerID = toPlayerID
	return f.err
}

func (f *fakeFriendClient) ListIncomingFriendRequests(_ context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	f.listIncomingPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.incomingRequests, nil
}

func (f *fakeFriendClient) ListOutgoingFriendRequests(_ context.Context, playerID int64) ([]*statecontract.FriendRequest, error) {
	f.listOutgoingPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.outgoingRequests, nil
}

func (f *fakeFriendClient) AcceptFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.acceptedFromPlayerID = fromPlayerID
	f.acceptedToPlayerID = toPlayerID
	return f.err
}

func (f *fakeFriendClient) RejectFriendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.rejectedFromPlayerID = fromPlayerID
	f.rejectedToPlayerID = toPlayerID
	return f.err
}

func (f *fakeFriendClient) ListFriendIDs(_ context.Context, playerID int64) ([]int64, error) {
	f.listFriendIDsPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.friendIDs, nil
}

func (f *fakeFriendClient) DeleteFriend(_ context.Context, playerID, friendPlayerID int64) error {
	f.deletedPlayerID = playerID
	f.deletedFriendID = friendPlayerID
	return f.err
}

var _ statecontract.FriendClient = (*fakeFriendClient)(nil)
