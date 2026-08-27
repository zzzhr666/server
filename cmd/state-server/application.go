package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/platform/lifecycle"
	"server/internal/platform/logging"
	platformmetrics "server/internal/platform/metrics"
	"server/internal/platform/mongodb"
	"server/internal/platform/redisdb"
	"server/internal/state/grpcserver"
	"server/internal/state/mongostore"
	"server/internal/state/redisstore"
	"server/internal/state/service"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc"
)

const shutdownTimeout = 5 * time.Second

type application struct {
	config        config.Config
	redisClient   *redis.Client
	mongoClient   *mongo.Client
	grpcListener  net.Listener
	grpcServer    *grpc.Server
	metricsServer *platformmetrics.Server
}

func newApplication(ctx context.Context, cfg config.Config) (*application, error) {
	metricsRegistry := platformmetrics.NewRegistry()
	metricsServer, err := platformmetrics.NewServer(platformmetrics.ServerConfig{
		Addr:     cfg.MetricsAddr,
		Gatherer: metricsRegistry.Gatherer(),
	})
	if err != nil {
		return nil, fmt.Errorf("create metrics server: %w", err)
	}
	grpcMetrics := grpcserver.NewMetrics(metricsRegistry.Registerer())
	stateMetrics := service.NewMetrics(metricsRegistry.Registerer())

	redisClient := redisdb.NewClient(cfg.Redis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	redisStore := redisstore.NewStore(redisClient)

	mongoClient, err := mongo.Connect(mongodb.ClientOptions(cfg.Mongo))
	if err != nil {
		_ = redisClient.Close()
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}
	if err := mongoClient.Ping(ctx, nil); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		_ = redisClient.Close()
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	mongoStore := mongostore.NewStore(mongoClient.Database(cfg.Mongo.Database))
	if err := mongoStore.EnsureIndexes(ctx); err != nil {
		_ = mongoClient.Disconnect(context.Background())
		_ = redisClient.Close()
		return nil, fmt.Errorf("ensure mongodb indexes: %w", err)
	}

	stateService := service.NewService(service.StoreConfig{
		Accounts:      redisStore,
		Sessions:      redisStore,
		Players:       redisStore,
		Registrations: redisStore,
		Presences:     redisStore,
		Friends:       redisStore,
		Realtime:      redisStore,
		Growth:        redisStore,
		Coins:         redisStore,
		Chats:         mongoStore,
		Leaderboards:  redisStore,
		Metrics:       stateMetrics,
	})
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcserver.UnaryMetricsInterceptor(grpcMetrics)))
	statepb.RegisterStateServiceServer(grpcServer, grpcserver.NewServer(grpcserver.ServerConfig{
		StateClient:       stateService,
		PresenceClient:    stateService,
		FriendClient:      stateService,
		RealtimeClient:    stateService,
		GrowthClient:      stateService,
		CoinClient:        stateService,
		ChatClient:        stateService,
		LeaderboardClient: stateService,
	}))
	grpcListener, err := net.Listen("tcp", cfg.StateGRPCAddr)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())
		_ = redisClient.Close()
		return nil, fmt.Errorf("listen for state gRPC: %w", err)
	}

	return &application{
		config:        cfg,
		redisClient:   redisClient,
		mongoClient:   mongoClient,
		grpcListener:  grpcListener,
		grpcServer:    grpcServer,
		metricsServer: metricsServer,
	}, nil
}

// Run 启动 state-server 的 gRPC 服务并阻塞至服务退出。
func (app *application) Run(ctx context.Context) error {
	services := lifecycle.NewGroup(ctx, 2)
	services.Go("metrics server", func(context.Context) error {
		return app.metricsServer.Serve()
	})
	services.Go("state gRPC server", func(context.Context) error {
		return app.serveGRPC()
	})

	logging.Info("state-server listening grpc=%s metrics=%s", app.config.StateGRPCAddr, app.config.MetricsAddr)
	runErr := services.WaitForStop()
	if runErr == nil {
		logging.Info("shutdown signal received")
	}
	services.Stop()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	shutdownErr := app.shutdownServers(shutdownCtx)
	waitErr := services.Wait(shutdownCtx)
	if waitErr == nil {
		logging.Info("state-server stopped")
	}
	return errors.Join(runErr, shutdownErr, waitErr)
}

func (app *application) serveGRPC() error {
	err := app.grpcServer.Serve(app.grpcListener)
	if errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return err
}

func (app *application) shutdownServers(ctx context.Context) error {
	metricsErr := make(chan error, 1)
	go func() {
		metricsErr <- app.metricsServer.Shutdown(ctx)
	}()
	grpcErr := lifecycle.GracefulStopGRPC(ctx, app.grpcServer)
	return errors.Join(grpcErr, <-metricsErr)
}

// Close 在给定 Context 内关闭 state-server 及其存储连接。
func (app *application) Close(ctx context.Context) error {
	app.grpcServer.Stop()
	listenerErr := app.grpcListener.Close()
	if errors.Is(listenerErr, net.ErrClosed) {
		listenerErr = nil
	}
	return errors.Join(
		listenerErr,
		app.mongoClient.Disconnect(ctx),
		app.redisClient.Close(),
	)
}
