package mongostore

import (
	"context"
	"errors"
	"server/internal/contract/state"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type chatMessageDocument struct {
	ID               bson.ObjectID         `bson:"_id,omitempty"`
	ChannelType      state.ChatChannelType `bson:"channel_type"`
	ChannelKey       string                `bson:"channel_key"`
	SenderID         int64                 `bson:"sender_id"`
	ReceiverID       int64                 `bson:"receiver_id"`
	Content          string                `bson:"content"`
	CreatedAt        time.Time             `bson:"created_at"`
	ExpiresAt        time.Time             `bson:"expires_at"`
	ClientMessageKey string                `bson:"client_message_key"`
	SenderNickname   string                `bson:"sender_nickname"`
}

func toContractChatMessage(doc *chatMessageDocument) *state.ChatMessage {
	if doc == nil {
		return nil
	}
	return &state.ChatMessage{
		MessageKey:       doc.ID.Hex(),
		ChannelType:      doc.ChannelType,
		ChannelKey:       doc.ChannelKey,
		SenderID:         doc.SenderID,
		ReceiverID:       doc.ReceiverID,
		Content:          doc.Content,
		CreatedAt:        doc.CreatedAt,
		ExpiresAt:        doc.ExpiresAt,
		ClientMessageKey: doc.ClientMessageKey,
		SenderNickname:   doc.SenderNickname,
	}
}

func fromSaveChatMessageInput(input state.SaveChatMessageInput) *chatMessageDocument {
	return &chatMessageDocument{
		ID:               bson.NewObjectIDFromTimestamp(input.CreatedAt),
		ChannelType:      input.ChannelType,
		ChannelKey:       input.ChannelKey,
		SenderID:         input.SenderID,
		ReceiverID:       input.ReceiverID,
		Content:          input.Content,
		CreatedAt:        input.CreatedAt,
		ExpiresAt:        input.ExpiresAt,
		ClientMessageKey: input.ClientMessageKey,
		SenderNickname:   input.SenderNickname,
	}
}

func validateSaveChatMessageInput(input state.SaveChatMessageInput) error {
	if input.ChannelType != state.ChatChannelDirect && input.ChannelType != state.ChatChannelWorld {
		return state.ErrInvalidChatChannel
	}
	if input.ChannelKey == "" {
		return state.ErrInvalidChatChannel
	}
	if input.SenderID <= 0 {
		return state.ErrInvalidChatMessage
	}
	if input.ChannelType == state.ChatChannelDirect && input.ReceiverID <= 0 {
		return state.ErrInvalidChatMessage
	}
	if input.Content == "" || input.ClientMessageKey == "" || input.MaxMessages <= 0 {
		return state.ErrInvalidChatMessage
	}
	if input.ExpiresAt.IsZero() || input.CreatedAt.IsZero() || !input.ExpiresAt.After(input.CreatedAt) {
		return state.ErrInvalidChatMessage
	}

	return nil
}

func validateListChatMessagesInput(input state.ListChatMessagesInput) error {
	if input.ChannelType != state.ChatChannelWorld && input.ChannelType != state.ChatChannelDirect {
		return state.ErrInvalidChatChannel
	}
	if input.ChannelKey == "" {
		return state.ErrInvalidChatChannel
	}
	if input.Limit <= 0 {
		return state.ErrInvalidChatMessage
	}
	return nil
}

// SaveChatMessage 校验并持久化聊天消息，客户端消息键重复时返回已有记录。
func (s *Store) SaveChatMessage(ctx context.Context, input state.SaveChatMessageInput) (*state.ChatMessage, error) {
	if err := validateSaveChatMessageInput(input); err != nil {
		return nil, err
	}
	doc := fromSaveChatMessageInput(input)
	_, err := s.chatMessages().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return s.findChatMessageByClientKey(ctx, input.ChannelKey, input.SenderID, input.ClientMessageKey)
		}
		return nil, err
	}
	if err := s.trimChatMessages(ctx, input.ChannelKey, input.MaxMessages); err != nil {
		return nil, err
	}
	return toContractChatMessage(doc), nil

}

// ListChatMessages 按频道与游标倒序读取聊天历史。
func (s *Store) ListChatMessages(ctx context.Context, input state.ListChatMessagesInput) ([]*state.ChatMessage, error) {
	if err := validateListChatMessagesInput(input); err != nil {
		return nil, err
	}
	filter := bson.M{
		"channel_type": input.ChannelType,
		"channel_key":  input.ChannelKey,
		"expires_at":   bson.M{"$gt": time.Now()},
	}
	if input.BeforeMessageKey != "" {
		before, err := s.findChatMessageDocumentByKey(ctx, input.BeforeMessageKey)
		if err != nil {
			return nil, err
		}
		filter["$or"] = bson.A{
			bson.M{"created_at": bson.M{"$lt": before.CreatedAt}},
			bson.M{"created_at": before.CreatedAt, "_id": bson.M{"$lt": before.ID}},
		}
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).SetLimit(input.Limit)
	cursor, err := s.chatMessages().Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	var docs []chatMessageDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	messages := make([]*state.ChatMessage, 0, len(docs))
	for i := len(docs) - 1; i >= 0; i-- {
		messages = append(messages, toContractChatMessage(&docs[i]))
	}
	return messages, nil
}

func (s *Store) findChatMessageByClientKey(ctx context.Context, channelKey string, senderID int64, clientMessageKey string) (*state.ChatMessage, error) {
	filter := bson.M{
		"channel_key":        channelKey,
		"sender_id":          senderID,
		"client_message_key": clientMessageKey,
	}
	var doc chatMessageDocument
	if err := s.chatMessages().FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, state.ErrChatMessageNotFound
		}
		return nil, err
	}
	return toContractChatMessage(&doc), nil
}

func (s *Store) trimChatMessages(ctx context.Context, channelKey string, maxMessages int64) error {
	if maxMessages <= 0 {
		return state.ErrInvalidChatMessage
	}
	findOptions := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).SetSkip(maxMessages)
	var boundary chatMessageDocument
	err := s.chatMessages().FindOne(ctx, bson.M{"channel_key": channelKey}, findOptions).Decode(&boundary)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}
	filter := bson.M{
		"channel_key": channelKey,
		"$or": bson.A{
			bson.M{"created_at": bson.M{"$lt": boundary.CreatedAt}}, bson.M{"created_at": boundary.CreatedAt, "_id": bson.M{"$lte": boundary.ID}},
		},
	}
	_, err = s.chatMessages().DeleteMany(ctx, filter)
	return err
}

func (s *Store) findChatMessageDocumentByKey(ctx context.Context, messageKey string) (*chatMessageDocument, error) {
	id, err := bson.ObjectIDFromHex(messageKey)
	if err != nil {
		return nil, state.ErrInvalidChatMessage
	}
	var doc chatMessageDocument
	if err := s.chatMessages().FindOne(ctx, bson.M{"_id": id}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, state.ErrChatMessageNotFound
		}
		return nil, err
	}
	return &doc, nil
}
