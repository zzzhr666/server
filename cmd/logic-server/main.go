package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"server/internal/contract/statepb"
	"server/internal/logic/auth"
	"server/internal/logic/friend"
	"server/internal/logic/httpapi"
	"server/internal/logic/player"
	"server/internal/logic/presence"
	"server/internal/platform/config"
	"server/internal/state/grpcclient"
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

	conn, err := grpc.NewClient(cfg.StateGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials())) // 本地开发环境不启用 TLS.
	if err != nil {
		log.Fatalf("grpc.NewClient failed: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Fatalf("close client connection: %v", err)
		}
	}()

	statePBClient := statepb.NewStateServiceClient(conn)
	stateService := grpcclient.NewClient(statePBClient)

	playerRepo := player.NewStateRepository(stateService)
	playerService := player.NewService(playerRepo)

	authRepo := auth.NewStateRepository(stateService)
	authService := auth.NewService(authRepo, playerService, time.Minute*10)

	presenceRepo := presence.NewStateRepository(stateService)
	presenceService := presence.NewService(presenceRepo)

	friendRepo := friend.NewStateRepository(stateService)
	friendService := friend.NewService(friendRepo)

	handler := httpapi.NewHandler(httpapi.HandlerConfig{
		AuthService:     authService,
		ServerName:      serverName,
		PresenceService: presenceService,
		FriendService:   friendService,
		PlayerService:   playerService,
		RealtimeClient:  stateService,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := handler.RunRealtimeSubscriber(ctx); err != nil && ctx.Err() == nil {
			log.Printf("realtime subscriber stopped: %v", err)
		}
	}()

	log.Printf("logic-server listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handler.Routes()); err != nil {
		log.Fatalf("logic-server stopped: %v", err)
	}
}
