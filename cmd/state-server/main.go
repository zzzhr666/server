package main

import (
	"context"
	"log"
	"net"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/platform/mongodb"
	"server/internal/platform/redisdb"
	"server/internal/state/grpcserver"
	"server/internal/state/mongostore"
	"server/internal/state/redisstore"
	"server/internal/state/service"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
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

	mongoClient, err := mongo.Connect(mongodb.ClientOptions(cfg.Mongo))
	if err != nil {
		log.Fatalf("connect mongodb failed: %v", err)
	}
	defer func(client *mongo.Client) {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatalf("disconnect mongodb failed: %v", err)
		}
	}(mongoClient)

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("ping mongodb failed: %v", err)
	}

	chatStore := mongostore.NewStore(mongoClient.Database(cfg.Mongo.Database))
	if err := chatStore.EnsureIndexes(ctx); err != nil {
		log.Fatalf("ensure indexes failed: %v", err)
	}

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
		Chats:         chatStore,
	})

	grpcServer := grpc.NewServer()
	statepb.RegisterStateServiceServer(grpcServer, grpcserver.NewServer(grpcserver.ServerConfig{
		StateClient:    stateService,
		PresenceClient: stateService,
		FriendClient:   stateService,
		RealtimeClient: stateService,
		GrowthClient:   stateService,
		CoinClient:     stateService,
		ChatClient:     stateService,
	}))
	listener, err := net.Listen("tcp", cfg.StateGRPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpc serve stopped: %v", err)
	}

}
