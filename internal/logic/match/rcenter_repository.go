package match

import (
	"context"
	"server/internal/rcenter"
)

type rCenterClient interface {
	StartMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error)
	CancelMatch(ctx context.Context, playerID int64) error
}

// RCenterRepository adapts the rcenter client to the logic match repository.
type RCenterRepository struct {
	client rCenterClient
}

// NewRCenterRepository creates a match repository backed by rcenter gRPC.
func NewRCenterRepository(client rCenterClient) *RCenterRepository {
	return &RCenterRepository{client: client}
}

// StartMatch forwards match start requests to rcenter.
func (r *RCenterRepository) StartMatch(ctx context.Context, playerID int64) (*rcenter.MatchResult, error) {
	return r.client.StartMatch(ctx, playerID)
}

// CancelMatch forwards match cancellation requests to rcenter.
func (r *RCenterRepository) CancelMatch(ctx context.Context, playerID int64) error {
	return r.client.CancelMatch(ctx, playerID)
}
