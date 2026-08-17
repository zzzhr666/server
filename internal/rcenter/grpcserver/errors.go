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
	case errors.Is(err, rcenter.ErrInvalidRoomName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, rcenter.ErrInvalidBattleStats):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, rcenter.ErrNoAvailableBattleNode):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, rcenter.ErrUnavailableCoinClient):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, rcenter.ErrUnavailableGrowthClient):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, rcenter.ErrPlayerNotWaiting):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, rcenter.ErrPlayerInGame):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, rcenter.ErrActiveMatchNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
