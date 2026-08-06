package httpapi

import (
	"context"
	statecontract "server/internal/contract/state"
)

type realtimeSubscriber struct {
	serverName string
	client     statecontract.RealtimeClient
	pusher     *localRealtimePusher
}

func newRealtimeSubscriber(serverName string, client statecontract.RealtimeClient, pusher *localRealtimePusher) *realtimeSubscriber {
	return &realtimeSubscriber{
		serverName: serverName,
		client:     client,
		pusher:     pusher,
	}
}

// Run 订阅当前 logic-server 的实时频道并在本地转发事件。
func (r *realtimeSubscriber) Run(ctx context.Context) error {
	events, err := r.client.SubscribeRealtime(ctx, r.serverName)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if event == nil {
				continue
			}
			// state-server 已按 serverName 路由事件；这里仅负责将跨实例事件落到
			// 当前进程持有的 WebSocket，不再查询在线状态以避免转发循环。
			r.pusher.Push(ctx, *event)

		}
	}
}
