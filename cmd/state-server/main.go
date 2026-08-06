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

	// 创建 Redis 客户端，状态服务是 Go 侧唯一直接访问 Redis 的进程。
	redisClient := redisdb.NewClient(cfg.Redis)
	defer func(redisClient *redis.Client) {
		if err := redisClient.Close(); err != nil {
			log.Fatalf("redis close failed: %v", err)
		}
	}(redisClient)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	// 将 Redis 访问封装为状态存储实现。
	store := redisstore.NewStore(redisClient)

	// 组装状态领域服务与其 gRPC 适配器。
	stateService := service.NewService(service.StoreConfig{
		Accounts:      store,
		Sessions:      store,
		Players:       store,
		Registrations: store,
		Presences:     store,
		Friends:       store,
		Realtime:      store,
		Growth:        store,
		Coins:         store,
	})

	grpcServer := grpc.NewServer()
	statepb.RegisterStateServiceServer(grpcServer, grpcserver.NewServer(grpcserver.ServerConfig{
		StateClient:    stateService,
		PresenceClient: stateService,
		FriendClient:   stateService,
		RealtimeClient: stateService,
		GrowthClient:   stateService,
		CoinClient:     stateService,
	}))
	listener, err := net.Listen("tcp", cfg.StateGRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve stopped: %v", err)
	}

}
