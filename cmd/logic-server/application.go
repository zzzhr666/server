package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"server/internal/contract/rcenterpb"
	"server/internal/contract/statepb"
	"server/internal/logic/auth"
	"server/internal/logic/chat"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/httpapi"
	logicmatch "server/internal/logic/match"
	"server/internal/logic/player"
	"server/internal/logic/presence"
	"server/internal/logic/realtime"
	"server/internal/platform/config"
	"server/internal/platform/lifecycle"
	"server/internal/platform/logging"
	platformmetrics "server/internal/platform/metrics"
	rcentergrpcclient "server/internal/rcenter/grpcclient"
	stategrpcclient "server/internal/state/grpcclient"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const shutdownTimeout = 5 * time.Second

type application struct {
	config             config.Config
	stateConn          *grpc.ClientConn
	rcenterConn        *grpc.ClientConn
	metricsServer      *platformmetrics.Server
	businessHTTPServer *http.Server
	realtimeHandler    *realtime.Handler
	realtimeServer     *realtime.Server
}

func newApplication(cfg config.Config, serverName string) (*application, error) {
	metricsRegistry := platformmetrics.NewRegistry()
	metricsServer, err := platformmetrics.NewServer(platformmetrics.ServerConfig{
		Addr:     cfg.MetricsAddr,
		Gatherer: metricsRegistry.Gatherer(),
	})
	if err != nil {
		return nil, fmt.Errorf("create metrics server: %w", err)
	}

	stateConn, err := grpc.NewClient(cfg.StateGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create state client connection: %w", err)
	}
	stateService := stategrpcclient.NewClient(statepb.NewStateServiceClient(stateConn))

	rcenterConn, err := grpc.NewClient(cfg.RCenterGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = stateConn.Close()
		return nil, fmt.Errorf("create rcenter client connection: %w", err)
	}
	rcenterService := rcentergrpcclient.NewClient(rcenterpb.NewRCenterServiceClient(rcenterConn))

	playerService := player.NewService(player.NewStateRepository(stateService))
	authService := auth.NewService(auth.NewStateRepository(stateService), playerService, 10*time.Minute)
	presenceService := presence.NewService(presence.NewStateRepository(stateService))
	friendService := friend.NewService(friend.NewStateRepository(stateService))
	chatService := chat.NewService(chat.NewStateRepository(stateService), friendService)
	growthService := growth.NewService(growth.NewStateRepository(stateService), growth.DefaultUpgradeRules())
	matchService := logicmatch.NewService(logicmatch.NewRCenterRepository(rcenterService))

	httpHandler := httpapi.NewHandler(httpapi.HandlerConfig{
		AuthService: authService,
		ServerName:  serverName,
	})
	businessHTTPServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	realtimeHandler := realtime.NewHandler(realtime.HandlerConfig{
		AuthService:     authService,
		PresenceService: presenceService,
		MatchService:    matchService,
		FriendService:   friendService,
		PlayerService:   playerService,
		GrowthService:   growthService,
		ServerName:      serverName,
		RealtimeClient:  stateService,
		ChatService:     chatService,
	})
	realtimeListener, err := net.Listen("tcp", cfg.TCPAddr)
	if err != nil {
		_ = rcenterConn.Close()
		_ = stateConn.Close()
		return nil, fmt.Errorf("listen for realtime connections: %w", err)
	}

	return &application{
		config:             cfg,
		stateConn:          stateConn,
		rcenterConn:        rcenterConn,
		metricsServer:      metricsServer,
		businessHTTPServer: businessHTTPServer,
		realtimeHandler:    realtimeHandler,
		realtimeServer:     realtime.NewServer(realtimeListener, realtimeHandler),
	}, nil
}

func (app *application) Run(ctx context.Context) error {
	services := lifecycle.NewGroup(ctx, 4)
	app.startServices(services)

	logging.Info(
		"logic-server listening http=%s tcp=%s metrics=%s",
		app.config.HTTPAddr,
		app.config.TCPAddr,
		app.config.MetricsAddr,
	)
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
		logging.Info("logic-server stopped")
	}
	return errors.Join(runErr, shutdownErr, waitErr)
}

func (app *application) startServices(services *lifecycle.Group) {
	services.Go("metrics server", func(context.Context) error {
		return app.metricsServer.Serve()
	})
	services.Go("realtime subscriber", func(ctx context.Context) error {
		return app.realtimeHandler.RunRealtimeSubscriber(ctx)
	})
	services.Go("realtime tcp server", func(ctx context.Context) error {
		return app.realtimeServer.Serve(ctx)
	})
	services.Go("business http server", func(context.Context) error {
		return app.serveBusinessHTTP()
	})
}

func (app *application) serveBusinessHTTP() error {
	err := app.businessHTTPServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (app *application) shutdownServers(ctx context.Context) error {
	shutdownErrors := make(chan error, 2)
	go func() {
		shutdownErrors <- app.businessHTTPServer.Shutdown(ctx)
	}()
	go func() {
		shutdownErrors <- app.metricsServer.Shutdown(ctx)
	}()

	var errs []error
	for range 2 {
		if err := <-shutdownErrors; err != nil {
			errs = append(errs, fmt.Errorf("shutdown HTTP server: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (app *application) Close() error {
	app.realtimeServer.Close()
	return errors.Join(
		closeClientConnection("rcenter", app.rcenterConn),
		closeClientConnection("state", app.stateConn),
	)
}

func closeClientConnection(name string, conn *grpc.ClientConn) error {
	if conn == nil {
		return nil
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close %s client connection: %w", name, err)
	}
	return nil
}
