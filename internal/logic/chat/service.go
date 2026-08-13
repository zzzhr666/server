package chat

import (
	"context"
	"fmt"
	"server/internal/contract/state"
	"server/internal/platform/logging"
	"slices"
	"strings"
	"time"
)

const (
	defaultListLimit = 25
	maxListLimit     = 100
	maxContentLength = 500
)

type Repository interface {
	SaveMessage(ctx context.Context, input SaveMessageInput) (*Message, error)
	ListMessages(ctx context.Context, input ListMessagesInput) ([]*Message, error)
}

type FriendChecker interface {
	ListFriendIDs(ctx context.Context, playerID int64) ([]int64, error)
}

type Service interface {
	SendWorldMessage(ctx context.Context, input SendWorldMessageInput) (*Message, error)
	SendDirectMessage(ctx context.Context, input SendDirectMessageInput) (*Message, error)

	ListWorldMessages(ctx context.Context, input ListWorldMessagesInput) ([]*Message, error)
	ListDirectMessages(ctx context.Context, input ListDirectMessagesInput) ([]*Message, error)
}

type GameChatService struct {
	repo   Repository
	friend FriendChecker
	now    func() time.Time
}

func (g GameChatService) SendWorldMessage(ctx context.Context, input SendWorldMessageInput) (*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.SenderID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	if input.ClientMessageKey == "" {
		return nil, ErrInvalidMessage
	}
	if err := validateContent(input.Content); err != nil {
		return nil, err
	}
	now := g.now()
	message, err := g.repo.SaveMessage(ctx, SaveMessageInput{
		ChannelType:      ChannelWorld,
		ChannelKey:       state.WorldChatChannelKey,
		SenderID:         input.SenderID,
		Content:          input.Content,
		CreatedAt:        now,
		ExpiresAt:        now.Add(state.WorldChatRetention),
		MaxMessages:      state.WorldChatMaxMessages,
		ClientMessageKey: input.ClientMessageKey,
		SenderNickname:   input.SenderNickname,
	})
	if err != nil {
		logging.Error("save world chat failed sender_id=%d: %v", input.SenderID, err)
		return nil, err
	}
	logging.Debug("world chat saved sender_id=%d message_key=%s", input.SenderID, message.MessageKey)
	return message, nil
}

func (g GameChatService) SendDirectMessage(ctx context.Context, input SendDirectMessageInput) (*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.SenderID <= 0 || input.ReceiverID <= 0 || input.SenderID == input.ReceiverID {
		return nil, ErrInvalidPlayerID
	}
	if input.ClientMessageKey == "" {
		return nil, ErrInvalidMessage
	}
	if err := validateContent(input.Content); err != nil {
		return nil, err
	}
	if err := g.requireFriend(ctx, input.SenderID, input.ReceiverID); err != nil {
		return nil, err
	}
	now := g.now()
	message, err := g.repo.SaveMessage(ctx, SaveMessageInput{
		ChannelType:      ChannelDirect,
		ChannelKey:       directChannelKey(input.SenderID, input.ReceiverID),
		SenderID:         input.SenderID,
		ReceiverID:       input.ReceiverID,
		Content:          input.Content,
		CreatedAt:        now,
		ExpiresAt:        now.Add(state.DirectChatRetention),
		MaxMessages:      state.DirectChatMaxMessages,
		ClientMessageKey: input.ClientMessageKey,
		SenderNickname:   input.SenderNickname,
	})
	if err != nil {
		logging.Error("save direct chat failed sender_id=%d receiver_id=%d: %v", input.SenderID, input.ReceiverID, err)
		return nil, err
	}
	logging.Debug("direct chat saved sender_id=%d receiver_id=%d message_key=%s", input.SenderID, input.ReceiverID, message.MessageKey)
	return message, nil
}

func (g GameChatService) ListWorldMessages(ctx context.Context, input ListWorldMessagesInput) ([]*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.PlayerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	messages, err := g.repo.ListMessages(ctx, ListMessagesInput{
		ChannelType:      ChannelWorld,
		ChannelKey:       state.WorldChatChannelKey,
		Limit:            normalizeLimit(input.Limit),
		BeforeMessageKey: input.BeforeMessageKey,
	})
	if err != nil {
		logging.Error("list world chat failed player_id=%d: %v", input.PlayerID, err)
		return nil, err
	}
	logging.Debug("world chat page loaded player_id=%d count=%d before=%s", input.PlayerID, len(messages), input.BeforeMessageKey)
	return messages, nil
}

func (g GameChatService) ListDirectMessages(ctx context.Context, input ListDirectMessagesInput) ([]*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.PlayerID <= 0 || input.FriendID <= 0 || input.FriendID == input.PlayerID {
		return nil, ErrInvalidPlayerID
	}
	if err := g.requireFriend(ctx, input.PlayerID, input.FriendID); err != nil {
		return nil, err
	}
	messages, err := g.repo.ListMessages(ctx, ListMessagesInput{
		ChannelType:      ChannelDirect,
		ChannelKey:       directChannelKey(input.PlayerID, input.FriendID),
		Limit:            normalizeLimit(input.Limit),
		BeforeMessageKey: input.BeforeMessageKey,
	})
	if err != nil {
		logging.Error("list direct chat failed player_id=%d friend_id=%d: %v", input.PlayerID, input.FriendID, err)
		return nil, err
	}
	logging.Debug("direct chat page loaded player_id=%d friend_id=%d count=%d before=%s", input.PlayerID, input.FriendID, len(messages), input.BeforeMessageKey)
	return messages, nil
}

func (g GameChatService) requireFriend(ctx context.Context, playerID, friendID int64) error {
	friendIDs, err := g.friend.ListFriendIDs(ctx, playerID)
	if err != nil {
		return err
	}
	if !containsFriend(friendIDs, friendID) {
		return ErrFriendRequired
	}
	return nil
}

func NewService(repo Repository, friend FriendChecker) *GameChatService {
	return &GameChatService{
		repo:   repo,
		friend: friend,
		now:    time.Now,
	}
}

func directChannelKey(playerID, friendID int64) string {
	if playerID < friendID {
		return fmt.Sprintf("direct:%d:%d", playerID, friendID)
	}
	return fmt.Sprintf("direct:%d:%d", friendID, playerID)
}

func containsFriend(friendIDs []int64, targetID int64) bool {
	return slices.Contains(friendIDs, targetID)
}

func normalizeLimit(limit int64) int64 {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func validateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrInvalidMessage
	}
	if len([]rune(content)) > maxContentLength {
		return ErrInvalidMessage
	}
	return nil
}
