package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"server/internal/contract/rcenterpb"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/platform/lifecycle"
	"server/internal/platform/logging"
	platformmetrics "server/internal/platform/metrics"
	"server/internal/rcenter"
	"server/internal/rcenter/grpcserver"
	"server/internal/state/grpcclient"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const shutdownTimeout = 5 * time.Second

type application struct {
	config        config.Config
	stateConn     *grpc.ClientConn
	grpcListener  net.Listener
	grpcServer    *grpc.Server
	metricsServer *platformmetrics.Server
}

func newApplication(cfg config.Config) (*application, error) {
	metricsRegistry := platformmetrics.NewRegistry()
	metricsServer, err := platformmetrics.NewServer(platformmetrics.ServerConfig{
		Addr:     cfg.MetricsAddr,
		Gatherer: metricsRegistry.Gatherer(),
	})
	if err != nil {
		return nil, fmt.Errorf("create metrics server: %w", err)
	}
	rcenterMetrics := rcenter.NewMetrics(metricsRegistry.Registerer())
	stateConn, err := grpc.NewClient(cfg.StateGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create state client connection: %w", err)
	}
	stateClient := grpcclient.NewClient(statepb.NewStateServiceClient(stateConn))
	battleRepo := rcenter.NewBattleRepository()
	centerService := rcenter.NewService(rcenter.ServiceConfig{
		BattleNodeController: battleRepo,
		CoinClient:           stateClient,
		RewardRule:           rcenter.DefaultRewardRule(),
		GrowthClient:         stateClient,
		Metrics:              rcenterMetrics,
	})

	grpcServer := grpc.NewServer()
	rcenterpb.RegisterRCenterServiceServer(grpcServer, grpcserver.NewServer(centerService))
	grpcListener, err := net.Listen("tcp", cfg.RCenterGRPCAddr)
	if err != nil {
		_ = stateConn.Close()
		return nil, fmt.Errorf("listen for rcenter gRPC: %w", err)
	}
	return &application{
		config:        cfg,
		stateConn:     stateConn,
		grpcListener:  grpcListener,
		grpcServer:    grpcServer,
		metricsServer: metricsServer,
	}, nil
}

func (app *application) Run(ctx context.Context) error {
	services := lifecycle.NewGroup(ctx, 2)
	services.Go("metrics server", func(context.Context) error {
		return app.metricsServer.Serve()
	})
	services.Go("rcenter gRPC server", func(context.Context) error {
		return app.serveGRPC()
	})

	logging.Info("rcenter-server listening grpc=%s metrics=%s", app.config.RCenterGRPCAddr, app.config.MetricsAddr)
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
		logging.Info("rcenter-server stopped")
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

func (app *application) Close() error {
	app.grpcServer.Stop()
	listenerErr := app.grpcListener.Close()
	if errors.Is(listenerErr, net.ErrClosed) {
		listenerErr = nil
	}
	return errors.Join(
		listenerErr,
		app.stateConn.Close(),
	)
}
