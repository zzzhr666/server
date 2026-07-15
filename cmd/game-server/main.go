package main

import (
	"context"
	"learning/internal/config"
	"learning/internal/httpapi"
	"learning/internal/player"
	"learning/internal/redisdb"
	"learning/internal/room"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	cfg := config.Default()

	redisClient := redisdb.NewClient(cfg.Redis)
	defer func(redisClient *redis.Client) {
		err := redisClient.Close()
		if err != nil {
			log.Fatalf("redisClient.Close: %v", err)
		}
	}(redisClient)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	playerRepo := player.NewRedisRepository(redisClient)
	roomRepo := room.NewRedisRepository(redisClient)
	playerService := player.NewService(playerRepo)
	roomService := room.NewService(playerRepo, roomRepo)

	handler := httpapi.NewHandler(playerService, roomService)

	log.Printf("Listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handler.Routes()); err != nil {
		log.Fatalf("Error stopped: %v", err)
	}

}
