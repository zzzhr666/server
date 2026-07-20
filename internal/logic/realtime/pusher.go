package realtime

import (
	"context"
	"errors"
	"server/internal/logic/presence"
)

type Pusher struct {
	presenceService presence.Service
	bus             EventBus
}

func NewPusher(presenceService presence.Service, bus EventBus) *Pusher {
	return &Pusher{
		presenceService: presenceService,
		bus:             bus,
	}
}

func (p *Pusher) PushToPlayer(ctx context.Context, targetPlayerID int64, event Event) error {
	pres, err := p.presenceService.Get(ctx, targetPlayerID)
	if errors.Is(err, presence.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	event.TargetPlayerID = targetPlayerID
	return p.bus.Publish(ctx, pres.ServerName, event)
}
