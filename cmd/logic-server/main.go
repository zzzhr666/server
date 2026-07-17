package main

import (
	"log"
	"net/http"
	"net/rpc"
	"server/internal/logic/auth"
	"server/internal/logic/httpapi"
	"server/internal/logic/player"
	"server/internal/platform/config"
	"server/internal/state/rpcclient"
	"time"
)

func main() {

	cfg := config.Default()

	rpcClient, err := rpc.Dial("tcp", cfg.StateRPCAddr)
	if err != nil {
		log.Fatalf("state rpc dial %s failed: %v", cfg.StateRPCAddr, err)
	}
	defer func(rpcConn *rpc.Client) {
		if err := rpcConn.Close(); err != nil {
			log.Fatalf("state rpc close failed: %v", err)
		}
	}(rpcClient)

	stateService := rpcclient.NewClient(rpcClient)
	playerRepo := player.NewStateRepository(stateService)
	playerService := player.NewService(playerRepo)

	authRepo := auth.NewStateRepository(stateService)
	authService := auth.NewService(authRepo, playerService, time.Minute)

	handler := httpapi.NewHandler(httpapi.HandlerConfig{
		AuthService: authService,
	})

	log.Printf("logic-server listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handler.Routes()); err != nil {
		log.Fatalf("logic-server stopped: %v", err)
	}
}
