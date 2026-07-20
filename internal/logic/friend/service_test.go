package friend

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceForwardsFriendOperations(t *testing.T) {
	repo := &fakeRepository{
		incomingRequests: []*Request{
			{FromPlayerID: 7, ToPlayerID: 8, CreatedAt: time.Unix(100, 0)},
		},
		outgoingRequests: []*Request{
			{FromPlayerID: 8, ToPlayerID: 9, CreatedAt: time.Unix(200, 0)},
		},
		friendIDs: []int64{10, 11},
	}
	svc := NewService(repo)

	if err := svc.SendRequest(context.Background(), 7, 8); err != nil {
		t.Fatalf("SendRequest returned error: %v", err)
	}
	if repo.sentFromPlayerID != 7 || repo.sentToPlayerID != 8 {
		t.Fatalf("send request got from=%d to=%d, want from=7 to=8", repo.sentFromPlayerID, repo.sentToPlayerID)
	}

	incoming, err := svc.ListIncomingRequests(context.Background(), 8)
	if err != nil {
		t.Fatalf("ListIncomingRequests returned error: %v", err)
	}
	if repo.listIncomingPlayerID != 8 {
		t.Fatalf("list incoming player id = %d, want 8", repo.listIncomingPlayerID)
	}
	if len(incoming) != 1 || incoming[0].FromPlayerID != 7 {
		t.Fatalf("incoming requests = %+v, want one request from 7", incoming)
	}

	outgoing, err := svc.ListOutgoingRequests(context.Background(), 8)
	if err != nil {
		t.Fatalf("ListOutgoingRequests returned error: %v", err)
	}
	if repo.listOutgoingPlayerID != 8 {
		t.Fatalf("list outgoing player id = %d, want 8", repo.listOutgoingPlayerID)
	}
	if len(outgoing) != 1 || outgoing[0].ToPlayerID != 9 {
		t.Fatalf("outgoing requests = %+v, want one request to 9", outgoing)
	}

	if err := svc.AcceptRequest(context.Background(), 7, 8); err != nil {
		t.Fatalf("AcceptRequest returned error: %v", err)
	}
	if repo.acceptedFromPlayerID != 7 || repo.acceptedToPlayerID != 8 {
		t.Fatalf("accept request got from=%d to=%d, want from=7 to=8", repo.acceptedFromPlayerID, repo.acceptedToPlayerID)
	}

	if err := svc.RejectRequest(context.Background(), 9, 8); err != nil {
		t.Fatalf("RejectRequest returned error: %v", err)
	}
	if repo.rejectedFromPlayerID != 9 || repo.rejectedToPlayerID != 8 {
		t.Fatalf("reject request got from=%d to=%d, want from=9 to=8", repo.rejectedFromPlayerID, repo.rejectedToPlayerID)
	}

	friendIDs, err := svc.ListFriendIDs(context.Background(), 7)
	if err != nil {
		t.Fatalf("ListFriendIDs returned error: %v", err)
	}
	if repo.listFriendIDsPlayerID != 7 {
		t.Fatalf("list friend ids player id = %d, want 7", repo.listFriendIDsPlayerID)
	}
	if len(friendIDs) != 2 || friendIDs[0] != 10 || friendIDs[1] != 11 {
		t.Fatalf("friend ids = %v, want [10 11]", friendIDs)
	}

	if err := svc.DeleteFriend(context.Background(), 7, 10); err != nil {
		t.Fatalf("DeleteFriend returned error: %v", err)
	}
	if repo.deletedPlayerID != 7 || repo.deletedFriendID != 10 {
		t.Fatalf("delete friend got player=%d friend=%d, want player=7 friend=10", repo.deletedPlayerID, repo.deletedFriendID)
	}
}

func TestServiceValidatesFriendPairOperations(t *testing.T) {
	tests := []struct {
		name string
		run  func(Service) error
	}{
		{
			name: "send self request",
			run: func(s Service) error {
				return s.SendRequest(context.Background(), 7, 7)
			},
		},
		{
			name: "send request with zero from",
			run: func(s Service) error {
				return s.SendRequest(context.Background(), 0, 8)
			},
		},
		{
			name: "accept self request",
			run: func(s Service) error {
				return s.AcceptRequest(context.Background(), 7, 7)
			},
		},
		{
			name: "reject self request",
			run: func(s Service) error {
				return s.RejectRequest(context.Background(), 7, 7)
			},
		},
		{
			name: "delete self friend",
			run: func(s Service) error {
				return s.DeleteFriend(context.Background(), 7, 7)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := NewService(repo)

			err := tt.run(svc)
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want %v", err, ErrInvalidRequest)
			}
			if repo.called {
				t.Fatalf("repo was called for invalid input")
			}
		})
	}
}

func TestServiceValidatesSinglePlayerOperations(t *testing.T) {
	tests := []struct {
		name string
		run  func(Service) error
	}{
		{
			name: "list incoming zero player",
			run: func(s Service) error {
				_, err := s.ListIncomingRequests(context.Background(), 0)
				return err
			},
		},
		{
			name: "list outgoing zero player",
			run: func(s Service) error {
				_, err := s.ListOutgoingRequests(context.Background(), 0)
				return err
			},
		},
		{
			name: "list friend ids zero player",
			run: func(s Service) error {
				_, err := s.ListFriendIDs(context.Background(), 0)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			svc := NewService(repo)

			err := tt.run(svc)
			if !errors.Is(err, ErrInvalidPlayerID) {
				t.Fatalf("error = %v, want %v", err, ErrInvalidPlayerID)
			}
			if repo.called {
				t.Fatalf("repo was called for invalid input")
			}
		})
	}
}

func TestServiceReturnsContextErrorBeforeRepository(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.SendRequest(ctx, 7, 8)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendRequest error = %v, want %v", err, context.Canceled)
	}
	if repo.called {
		t.Fatalf("repo was called for canceled context")
	}
}

type fakeRepository struct {
	called                bool
	err                   error
	sentFromPlayerID      int64
	sentToPlayerID        int64
	listIncomingPlayerID  int64
	listOutgoingPlayerID  int64
	incomingRequests      []*Request
	outgoingRequests      []*Request
	acceptedFromPlayerID  int64
	acceptedToPlayerID    int64
	rejectedFromPlayerID  int64
	rejectedToPlayerID    int64
	listFriendIDsPlayerID int64
	friendIDs             []int64
	deletedPlayerID       int64
	deletedFriendID       int64
}

func (f *fakeRepository) SendRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.called = true
	f.sentFromPlayerID = fromPlayerID
	f.sentToPlayerID = toPlayerID
	return f.err
}

func (f *fakeRepository) ListIncomingRequests(_ context.Context, playerID int64) ([]*Request, error) {
	f.called = true
	f.listIncomingPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.incomingRequests, nil
}

func (f *fakeRepository) ListOutgoingRequests(_ context.Context, playerID int64) ([]*Request, error) {
	f.called = true
	f.listOutgoingPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.outgoingRequests, nil
}

func (f *fakeRepository) AcceptRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.called = true
	f.acceptedFromPlayerID = fromPlayerID
	f.acceptedToPlayerID = toPlayerID
	return f.err
}

func (f *fakeRepository) RejectRequest(_ context.Context, fromPlayerID, toPlayerID int64) error {
	f.called = true
	f.rejectedFromPlayerID = fromPlayerID
	f.rejectedToPlayerID = toPlayerID
	return f.err
}

func (f *fakeRepository) ListFriendIDs(_ context.Context, playerID int64) ([]int64, error) {
	f.called = true
	f.listFriendIDsPlayerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.friendIDs, nil
}

func (f *fakeRepository) DeleteFriend(_ context.Context, playerID, friendID int64) error {
	f.called = true
	f.deletedPlayerID = playerID
	f.deletedFriendID = friendID
	return f.err
}

var _ Repository = (*fakeRepository)(nil)
