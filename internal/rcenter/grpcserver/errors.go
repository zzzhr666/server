package grpcserver

import (
	"errors"
	"server/internal/rcenter"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapRCenterError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, rcenter.ErrInvalidBattleNode):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, rcenter.ErrInvalidPlayerID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, rcenter.ErrNoAvailableBattleNode):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, rcenter.ErrPlayerNotWaiting):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
