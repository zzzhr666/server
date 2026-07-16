package httpapi

import (
	playerpkg "learning/internal/player"
	roompkg "learning/internal/room"
)

type Handler struct {
	playersService playerpkg.Service
	roomsService   roompkg.Service
}

// NewHandler creates an HTTP handler with player and room services.
func NewHandler(players playerpkg.Service, rooms roompkg.Service) *Handler {
	return &Handler{playersService: players, roomsService: rooms}
}
