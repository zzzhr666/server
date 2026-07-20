package realtime

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type EventBus interface {
	Publish(ctx context.Context, serverName string, event Event) error
	Subscribe(ctx context.Context, serverName string) (<-chan Event, error)
}

func channelName(serverName string) string {
	return "game:realtime:" + serverName
}

type RedisEventBus struct {
	client *redis.Client
}

func NewRedisEventBus(client *redis.Client) *RedisEventBus {
	return &RedisEventBus{client: client}
}

func (b *RedisEventBus) Publish(ctx context.Context, serverName string, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, channelName(serverName), payload).Err()
}

func (b *RedisEventBus) Subscribe(ctx context.Context, serverName string) (<-chan Event, error) {
	pubsub := b.client.Subscribe(ctx, channelName(serverName))
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}

	events := make(chan Event, 16)
	go func() {
		defer close(events)
		defer func() {
			_ = pubsub.Close()
		}()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var event Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					continue
				}
				select {
				case events <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return events, nil
}
