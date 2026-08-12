package mongostore

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct {
	db *mongo.Database
}

func NewStore(db *mongo.Database) *Store {
	return &Store{db: db}
}

func (s *Store) chatMessages() *mongo.Collection {
	return s.db.Collection("chat_messages")
}

func (s *Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.chatMessages().Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("chat_message_expires_at_ttl").SetExpireAfterSeconds(0),
		},
		{
			Keys: bson.D{
				{Key: "channel_type", Value: 1},
				{Key: "channel_key", Value: 1},
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			},
			Options: options.Index().SetName("chat_message_channel_page"),
		},
		{
			Keys: bson.D{
				{Key: "channel_key", Value: 1},
				{Key: "sender_id", Value: 1},
				{Key: "client_message_key", Value: 1},
			},
			Options: options.Index().SetName("chat_message_client_dedupe").SetUnique(true),
		},
	})
	return err
}
