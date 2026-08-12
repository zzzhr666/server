package grpcserver

import (
	"errors"
	"server/internal/contract/state"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapStateError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, state.ErrAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrAccountExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, state.ErrSessionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrPlayerNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrPresenceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrInvalidPresence):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrInvalidRealtimeRoute):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrFriendRequestNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrFriendNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrFriendRequestExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, state.ErrFriendAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, state.ErrInvalidFriendRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrGrowthNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrInvalidGrowth):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrInvalidGrowthField):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrInsufficientCoins):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, state.ErrMaxGrowthLevel):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, state.ErrInvalidPlayer):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrChatMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, state.ErrInvalidChatMessage):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrInvalidChatChannel):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, state.ErrChatMessageExists):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
