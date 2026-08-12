package chat

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestSendWorldMessageSavesWithWorldRetention(t *testing.T) {
	now := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		saved: &Message{MessageKey: "msg-1"},
	}
	svc := NewService(repo, &fakeFriendChecker{})
	svc.now = func() time.Time { return now }

	message, err := svc.SendWorldMessage(context.Background(), SendWorldMessageInput{
		SenderID:         7,
		Content:          "hello",
		ClientMessageKey: "client-msg-1",
	})
	if err != nil {
		t.Fatalf("SendWorldMessage returned error: %v", err)
	}
	if message.MessageKey != "msg-1" {
		t.Fatalf("message key = %q, want msg-1", message.MessageKey)
	}
	if repo.saveInput.ChannelType != ChannelWorld {
		t.Fatalf("channel type = %q, want %q", repo.saveInput.ChannelType, ChannelWorld)
	}
	if repo.saveInput.ChannelKey != statecontract.WorldChatChannelKey {
		t.Fatalf("channel key = %q, want %q", repo.saveInput.ChannelKey, statecontract.WorldChatChannelKey)
	}
	if repo.saveInput.MaxMessages != statecontract.WorldChatMaxMessages {
		t.Fatalf("max messages = %d, want %d", repo.saveInput.MaxMessages, statecontract.WorldChatMaxMessages)
	}
	if !repo.saveInput.CreatedAt.Equal(now) {
		t.Fatalf("created at = %v, want %v", repo.saveInput.CreatedAt, now)
	}
	if !repo.saveInput.ExpiresAt.Equal(now.Add(statecontract.WorldChatRetention)) {
		t.Fatalf("expires at = %v, want %v", repo.saveInput.ExpiresAt, now.Add(statecontract.WorldChatRetention))
	}
}

func TestSendDirectMessageRequiresFriend(t *testing.T) {
	svc := NewService(&fakeRepository{}, &fakeFriendChecker{friendIDs: []int64{9}})

	_, err := svc.SendDirectMessage(context.Background(), SendDirectMessageInput{
		SenderID:         7,
		ReceiverID:       8,
		Content:          "hello",
		ClientMessageKey: "client-msg-1",
	})
	if !errors.Is(err, ErrFriendRequired) {
		t.Fatalf("SendDirectMessage error = %v, want %v", err, ErrFriendRequired)
	}
}

func TestSendDirectMessageSavesStableChannelKey(t *testing.T) {
	now := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	repo := &fakeRepository{
		saved: &Message{MessageKey: "msg-1"},
	}
	friends := &fakeFriendChecker{friendIDs: []int64{7}}
	svc := NewService(repo, friends)
	svc.now = func() time.Time { return now }

	_, err := svc.SendDirectMessage(context.Background(), SendDirectMessageInput{
		SenderID:         8,
		ReceiverID:       7,
		Content:          "hello",
		ClientMessageKey: "client-msg-1",
	})
	if err != nil {
		t.Fatalf("SendDirectMessage returned error: %v", err)
	}
	if friends.playerID != 8 {
		t.Fatalf("friend check player id = %d, want 8", friends.playerID)
	}
	if repo.saveInput.ChannelKey != "direct:7:8" {
		t.Fatalf("channel key = %q, want direct:7:8", repo.saveInput.ChannelKey)
	}
	if repo.saveInput.MaxMessages != statecontract.DirectChatMaxMessages {
		t.Fatalf("max messages = %d, want %d", repo.saveInput.MaxMessages, statecontract.DirectChatMaxMessages)
	}
	if !repo.saveInput.ExpiresAt.Equal(now.Add(statecontract.DirectChatRetention)) {
		t.Fatalf("expires at = %v, want %v", repo.saveInput.ExpiresAt, now.Add(statecontract.DirectChatRetention))
	}
}

func TestListWorldMessagesNormalizesLimit(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, &fakeFriendChecker{})

	_, err := svc.ListWorldMessages(context.Background(), ListWorldMessagesInput{
		PlayerID:         7,
		Limit:            maxListLimit + 1,
		BeforeMessageKey: "msg-0",
	})
	if err != nil {
		t.Fatalf("ListWorldMessages returned error: %v", err)
	}
	if repo.listInput.Limit != maxListLimit {
		t.Fatalf("limit = %d, want %d", repo.listInput.Limit, maxListLimit)
	}
	if repo.listInput.BeforeMessageKey != "msg-0" {
		t.Fatalf("before message key = %q, want msg-0", repo.listInput.BeforeMessageKey)
	}
}

func TestListDirectMessagesRequiresFriend(t *testing.T) {
	svc := NewService(&fakeRepository{}, &fakeFriendChecker{friendIDs: []int64{9}})

	_, err := svc.ListDirectMessages(context.Background(), ListDirectMessagesInput{
		PlayerID: 7,
		FriendID: 8,
	})
	if !errors.Is(err, ErrFriendRequired) {
		t.Fatalf("ListDirectMessages error = %v, want %v", err, ErrFriendRequired)
	}
}

func TestValidateContentRejectsInvalidContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "empty"},
		{name: "blank", content: "   "},
		{name: "too long", content: string(make([]rune, maxContentLength+1))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateContent(tt.content); !errors.Is(err, ErrInvalidMessage) {
				t.Fatalf("validateContent error = %v, want %v", err, ErrInvalidMessage)
			}
		})
	}
}

type fakeRepository struct {
	saveInput SaveMessageInput
	listInput ListMessagesInput
	saved     *Message
	messages  []*Message
	err       error
}

func (f *fakeRepository) SaveMessage(_ context.Context, input SaveMessageInput) (*Message, error) {
	f.saveInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.saved, nil
}

func (f *fakeRepository) ListMessages(_ context.Context, input ListMessagesInput) ([]*Message, error) {
	f.listInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.messages, nil
}

type fakeFriendChecker struct {
	playerID  int64
	friendIDs []int64
	err       error
}

func (f *fakeFriendChecker) ListFriendIDs(_ context.Context, playerID int64) ([]int64, error) {
	f.playerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.friendIDs, nil
}
