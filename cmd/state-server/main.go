package main

import (
	"context"
	"log"
	"net"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/platform/redisdb"
	"server/internal/state/grpcserver"
	"server/internal/state/redisstore"
	"server/internal/state/service"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	cfg := config.Default()

	//prepare redis
	redisClient := redisdb.NewClient(cfg.Redis)
	defer func(redisClient *redis.Client) {
		if err := redisClient.Close(); err != nil {
			log.Fatalf("redis close failed: %v", err)
		}
	}(redisClient)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	//make store
	store := redisstore.NewStore(redisClient)

	//make service
	stateService := service.NewService(service.StoreConfig{
		Accounts:      store,
		Sessions:      store,
		Players:       store,
		Registrations: store,
		Presences:     store,
		Friends:       store,
		Realtime:      store,
	})

	grpcServer := grpc.NewServer()
	statepb.RegisterStateServiceServer(grpcServer, grpcserver.NewServer(grpcserver.ServerConfig{
		StateClient:    stateService,
		PresenceClient: stateService,
		FriendClient:   stateService,
		RealtimeClient: stateService,
	}))
	listener, err := net.Listen("tcp", cfg.StateGRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve stopped: %v", err)
	}

}
