package main

import (
	"context"
	"log"
	"net"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/platform/logging"
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
	level, err := logging.ParseLevel(cfg.LogConfig.LogLevel)
	if err != nil {
		log.Fatalf("parse log level failed: %v", err)
	}
	mode, err := logging.ParseMode(cfg.LogConfig.LogMode)
	if err != nil {
		log.Fatalf("parse log mode failed: %v", err)
	}
	if err := logging.ConfigureDefault(logging.DefaultLoggerOptions{
		Level:       level,
		Mode:        mode,
		ServiceName: "state-server",
		LogDir:      "./logs",
	}); err != nil {
		log.Fatalf("configure logger failed: %v", err)
	}
	defer func() {
		if err := logging.CloseDefault(); err != nil {
			log.Printf("close logger failed: %v", err)
		}
	}()

	// 创建 Redis 客户端，状态服务是 Go 侧唯一直接访问 Redis 的进程。
	redisClient := redisdb.NewClient(cfg.Redis)
	defer func(redisClient *redis.Client) {
		if err := redisClient.Close(); err != nil {
			logging.Error("redis close failed: %v", err)
		}
	}(redisClient)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logging.Fatal("redis ping failed: %v", err)
	}

	// 将 Redis 访问封装为状态存储实现。
	store := redisstore.NewStore(redisClient)

	mongoClient, err := mongo.Connect(mongodb.ClientOptions(cfg.Mongo))
	if err != nil {
		logging.Fatal("connect mongodb failed: %v", err)
	}
	defer func(client *mongo.Client) {
		if err := client.Disconnect(ctx); err != nil {
			logging.Error("disconnect mongodb failed: %v", err)
		}
	}(mongoClient)

	if err := mongoClient.Ping(ctx, nil); err != nil {
		logging.Fatal("ping mongodb failed: %v", err)
	}

	chatStore := mongostore.NewStore(mongoClient.Database(cfg.Mongo.Database))
	if err := chatStore.EnsureIndexes(ctx); err != nil {
		logging.Fatal("ensure indexes failed: %v", err)
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
		logging.Fatal("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		logging.Fatal("grpc serve stopped: %v", err)
	}

}
