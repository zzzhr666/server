package main

import (
	"context"
	"flag"
	"log"
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
	"server/internal/platform/logging"
	rcentergrpcclient "server/internal/rcenter/grpcclient"
	stategrpcclient "server/internal/state/grpcclient"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func listenAddrFromPort(port string) string {
	if port == "" {
		return ""
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

func main() {

	cfg := config.Default()

	port := flag.String("port", "", "HTTP listen port")
	shortPort := flag.String("p", "", "HTTP listen port")
	serverName := "logic-default"
	flag.StringVar(&serverName, "name", "logic-default", "logic server instance name")
	flag.Parse()

	if addr := listenAddrFromPort(*port); addr != "" {
		cfg.HTTPAddr = addr
	}
	if addr := listenAddrFromPort(*shortPort); addr != "" {
		cfg.HTTPAddr = addr
	}
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
		ServiceName: serverName,
		LogDir:      "./logs",
	}); err != nil {
		log.Fatalf("configure logger failed: %v", err)
	}
	defer func() {
		if err := logging.CloseDefault(); err != nil {
			log.Printf("close logger failed: %v", err)
		}
	}()

	conn, err := grpc.NewClient(cfg.StateGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials())) // 本地开发环境不启用 TLS.
	if err != nil {
		logging.Fatal("grpc.NewClient failed: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logging.Error("close client connection: %v", err)
		}
	}()

	statePBClient := statepb.NewStateServiceClient(conn)
	stateService := stategrpcclient.NewClient(statePBClient)

	playerRepo := player.NewStateRepository(stateService)
	playerService := player.NewService(playerRepo)

	authRepo := auth.NewStateRepository(stateService)
	authService := auth.NewService(authRepo, playerService, time.Minute*10)

	presenceRepo := presence.NewStateRepository(stateService)
	presenceService := presence.NewService(presenceRepo)

	friendRepo := friend.NewStateRepository(stateService)
	friendService := friend.NewService(friendRepo)

	chatRepo := chat.NewStateRepository(stateService)
	chatService := chat.NewService(chatRepo, friendService)

	growthRepo := growth.NewStateRepository(stateService)
	growthService := growth.NewService(growthRepo, growth.DefaultUpgradeRules())

	rCenterConn, err := grpc.NewClient(cfg.RCenterGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logging.Fatal("rcenter grpc.NewClient failed: %v", err)
	}
	defer func() {
		if err := rCenterConn.Close(); err != nil {
			logging.Error("close rcenter client connection: %v", err)
		}
	}()
	rCenterPBClient := rcenterpb.NewRCenterServiceClient(rCenterConn)
	rCenterService := rcentergrpcclient.NewClient(rCenterPBClient)
	matchRepo := logicmatch.NewRCenterRepository(rCenterService)
	matchService := logicmatch.NewService(matchRepo)
	httpHandler := httpapi.NewHandler(httpapi.HandlerConfig{
		AuthService: authService,
		ServerName:  serverName,
	})
	tcpHandler := realtime.NewHandler(realtime.HandlerConfig{
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
	tcpListener, err := net.Listen("tcp", cfg.TCPAddr)
	if err != nil {
		logging.Fatal("tcp listener failed: %v", err)
	}
	tcpServer := realtime.NewServer(tcpListener, tcpHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := tcpHandler.RunRealtimeSubscriber(ctx); err != nil && ctx.Err() == nil {
			logging.Fatal("realtime subscriber failed: %v", err)
		}
	}()
	go func() {
		if err := tcpServer.Serve(ctx); err != nil && ctx.Err() == nil {
			logging.Fatal("tcp server failed: %v", err)
		}
	}()
	logging.Info("logic-server listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, httpHandler.Routes()); err != nil {
		logging.Fatal("logic-server stopped: %v", err)
	}
}
