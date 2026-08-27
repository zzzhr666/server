package chat

import (
	"context"
	"errors"
	"server/internal/contract/state"
)

type StateRepository struct {
	stateClient state.ChatClient
}

// SaveMessage 通过 state-server 持久化聊天消息并转换为领域模型。
func (s *StateRepository) SaveMessage(ctx context.Context, input SaveMessageInput) (*Message, error) {
	message, err := s.stateClient.SaveChatMessage(ctx, state.SaveChatMessageInput{
		ChannelType:      toStateChannelType(input.ChannelType),
		ChannelKey:       input.ChannelKey,
		SenderID:         input.SenderID,
		ReceiverID:       input.ReceiverID,
		Content:          input.Content,
		CreatedAt:        input.CreatedAt,
		ExpiresAt:        input.ExpiresAt,
		MaxMessages:      input.MaxMessages,
		ClientMessageKey: input.ClientMessageKey,
		SenderNickname:   input.SenderNickname,
	})
	if err != nil {
		return nil, mapStateError(err)
	}

	return fromStateMessage(message), nil
}

// ListMessages 从 state-server 读取聊天历史并转换为领域模型。
func (s *StateRepository) ListMessages(ctx context.Context, input ListMessagesInput) ([]*Message, error) {
	messages, err := s.stateClient.ListChatMessages(ctx, state.ListChatMessagesInput{
		ChannelType:      toStateChannelType(input.ChannelType),
		ChannelKey:       input.ChannelKey,
		Limit:            input.Limit,
		BeforeMessageKey: input.BeforeMessageKey,
	})
	if err != nil {
		return nil, mapStateError(err)
	}
	result := make([]*Message, 0, len(messages))
	for _, message := range messages {
		result = append(result, fromStateMessage(message))
	}
	return result, nil
}

// NewStateRepository 使用 state-server 聊天客户端创建仓储。
func NewStateRepository(client state.ChatClient) *StateRepository {
	return &StateRepository{
		stateClient: client,
	}
}

func mapStateError(err error) error {
	switch {
	case errors.Is(err, state.ErrInvalidChatMessage):
		return ErrInvalidMessage
	case errors.Is(err, state.ErrInvalidChatChannel):
		return ErrInvalidChannel
	case errors.Is(err, state.ErrChatMessageExists):
		return ErrMessageExists
	case errors.Is(err, state.ErrChatMessageNotFound):
		return ErrMessageNotFound
	}
	return err
}

func toStateChannelType(channelType ChannelType) state.ChatChannelType {
	return state.ChatChannelType(channelType)
}
func fromStateChannelType(channelType state.ChatChannelType) ChannelType {
	return ChannelType(channelType)
}

func fromStateMessage(value *state.ChatMessage) *Message {
	if value == nil {
		return nil
	}
	return &Message{
		MessageKey:       value.MessageKey,
		ChannelType:      fromStateChannelType(value.ChannelType),
		ChannelKey:       value.ChannelKey,
		SenderID:         value.SenderID,
		ReceiverID:       value.ReceiverID,
		Content:          value.Content,
		CreatedAt:        value.CreatedAt,
		ExpiresAt:        value.ExpiresAt,
		ClientMessageKey: value.ClientMessageKey,
		SenderNickname:   value.SenderNickname,
	}
}
