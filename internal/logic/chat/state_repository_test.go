package chat

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
	"testing"
	"time"
)

func TestStateRepositorySaveMessage(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	expiresAt := createdAt.Add(statecontract.DirectChatRetention)
	stateClient := &fakeStateChatClient{
		savedMessage: &statecontract.ChatMessage{
			MessageKey:       "msg-1",
			ChannelType:      statecontract.ChatChannelDirect,
			ChannelKey:       "direct:7:8",
			SenderID:         7,
			ReceiverID:       8,
			Content:          "hello",
			CreatedAt:        createdAt,
			ExpiresAt:        expiresAt,
			ClientMessageKey: "client-msg-1",
		},
	}
	repo := NewStateRepository(stateClient)

	message, err := repo.SaveMessage(context.Background(), SaveMessageInput{
		ChannelType:      ChannelDirect,
		ChannelKey:       "direct:7:8",
		SenderID:         7,
		ReceiverID:       8,
		Content:          "hello",
		CreatedAt:        createdAt,
		ExpiresAt:        expiresAt,
		MaxMessages:      statecontract.DirectChatMaxMessages,
		ClientMessageKey: "client-msg-1",
	})
	if err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}
	if stateClient.saveInput.ChannelType != statecontract.ChatChannelDirect {
		t.Fatalf("state channel type = %q, want %q", stateClient.saveInput.ChannelType, statecontract.ChatChannelDirect)
	}
	if stateClient.saveInput.MaxMessages != statecontract.DirectChatMaxMessages {
		t.Fatalf("state max messages = %d, want %d", stateClient.saveInput.MaxMessages, statecontract.DirectChatMaxMessages)
	}
	if message.MessageKey != "msg-1" || message.ChannelType != ChannelDirect || message.ClientMessageKey != "client-msg-1" {
		t.Fatalf("message = %+v, want converted direct chat message", message)
	}
}

func TestStateRepositoryListMessages(t *testing.T) {
	createdAt := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	stateClient := &fakeStateChatClient{
		messages: []*statecontract.ChatMessage{
			{
				MessageKey:       "msg-1",
				ChannelType:      statecontract.ChatChannelWorld,
				ChannelKey:       statecontract.WorldChatChannelKey,
				SenderID:         7,
				Content:          "hello",
				CreatedAt:        createdAt,
				ClientMessageKey: "client-msg-1",
			},
		},
	}
	repo := NewStateRepository(stateClient)

	messages, err := repo.ListMessages(context.Background(), ListMessagesInput{
		ChannelType:      ChannelWorld,
		ChannelKey:       statecontract.WorldChatChannelKey,
		Limit:            20,
		BeforeMessageKey: "msg-0",
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if stateClient.listInput.ChannelKey != statecontract.WorldChatChannelKey {
		t.Fatalf("state list channel key = %q, want %q", stateClient.listInput.ChannelKey, statecontract.WorldChatChannelKey)
	}
	if stateClient.listInput.BeforeMessageKey != "msg-0" {
		t.Fatalf("state before message key = %q, want msg-0", stateClient.listInput.BeforeMessageKey)
	}
	if len(messages) != 1 || messages[0].MessageKey != "msg-1" || messages[0].ChannelType != ChannelWorld {
		t.Fatalf("messages = %+v, want one converted world message", messages)
	}
}

func TestMapStateError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "invalid message", err: statecontract.ErrInvalidChatMessage, want: ErrInvalidMessage},
		{name: "invalid channel", err: statecontract.ErrInvalidChatChannel, want: ErrInvalidChannel},
		{name: "message exists", err: statecontract.ErrChatMessageExists, want: ErrMessageExists},
		{name: "message not found", err: statecontract.ErrChatMessageNotFound, want: ErrMessageNotFound},
		{name: "unknown", err: context.Canceled, want: context.Canceled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapStateError(tt.err)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("mapStateError returned %v, want nil", got)
				}
				return
			}
			if !errors.Is(got, tt.want) {
				t.Fatalf("mapStateError returned %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeStateChatClient struct {
	saveInput    statecontract.SaveChatMessageInput
	savedMessage *statecontract.ChatMessage
	listInput    statecontract.ListChatMessagesInput
	messages     []*statecontract.ChatMessage
	err          error
}

func (f *fakeStateChatClient) SaveChatMessage(_ context.Context, input statecontract.SaveChatMessageInput) (*statecontract.ChatMessage, error) {
	f.saveInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.savedMessage, nil
}

func (f *fakeStateChatClient) ListChatMessages(_ context.Context, input statecontract.ListChatMessagesInput) ([]*statecontract.ChatMessage, error) {
	f.listInput = input
	if f.err != nil {
		return nil, f.err
	}
	return f.messages, nil
}

var _ statecontract.ChatClient = (*fakeStateChatClient)(nil)
