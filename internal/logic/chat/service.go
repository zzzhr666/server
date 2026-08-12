package chat

import (
	"context"
	"fmt"
	"server/internal/contract/state"
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
	return g.repo.SaveMessage(ctx, SaveMessageInput{
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
	return g.repo.SaveMessage(ctx, SaveMessageInput{
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
}

func (g GameChatService) ListWorldMessages(ctx context.Context, input ListWorldMessagesInput) ([]*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.PlayerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	return g.repo.ListMessages(ctx, ListMessagesInput{
		ChannelType:      ChannelWorld,
		ChannelKey:       state.WorldChatChannelKey,
		Limit:            normalizeLimit(input.Limit),
		BeforeMessageKey: input.BeforeMessageKey,
	})
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
	return g.repo.ListMessages(ctx, ListMessagesInput{
		ChannelType:      ChannelDirect,
		ChannelKey:       directChannelKey(input.PlayerID, input.FriendID),
		Limit:            normalizeLimit(input.Limit),
		BeforeMessageKey: input.BeforeMessageKey,
	})
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
