package grpcserver

import (
	"errors"
	statecontract "server/internal/contract/state"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapStateError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, statecontract.ErrAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, statecontract.ErrAccountExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, statecontract.ErrSessionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, statecontract.ErrPlayerNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, statecontract.ErrPresenceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, statecontract.ErrInvalidPresence):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, statecontract.ErrFriendRequestNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, statecontract.ErrFriendNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, statecontract.ErrFriendRequestExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, statecontract.ErrFriendAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, statecontract.ErrInvalidFriendRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
