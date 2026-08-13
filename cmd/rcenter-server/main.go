package main

import (
	"log"
	"net"
	"server/internal/contract/rcenterpb"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/platform/logging"
	"server/internal/rcenter"
	"server/internal/rcenter/grpcserver"
	"server/internal/state/grpcclient"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
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
		ServiceName: "rcenter-server",
		LogDir:      "./logs",
	}); err != nil {
		log.Fatalf("configure logger failed: %v", err)
	}
	defer func() {
		if err := logging.CloseDefault(); err != nil {
			log.Printf("close logger failed: %v", err)
		}
	}()
	stateConn, err := grpc.NewClient(cfg.StateGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logging.Fatal("fail to call NewClient: %v", err)
	}
	defer func() {
		if err := stateConn.Close(); err != nil {
			logging.Error("fail to close client connection: %v", err)
		}
	}()
	statePBClient := statepb.NewStateServiceClient(stateConn)
	stateClient := grpcclient.NewClient(statePBClient)
	battleRepo := rcenter.NewBattleRepository()
	centerService := rcenter.NewService(
		rcenter.ServiceConfig{
			BattleNodeController: battleRepo,
			CoinClient:           stateClient,
			RewardRule:           rcenter.DefaultRewardRule(),
			GrowthClient:         stateClient,
		})

	grpcServer := grpc.NewServer()
	rcenterpb.RegisterRCenterServiceServer(grpcServer, grpcserver.NewServer(centerService))
	listener, err := net.Listen("tcp", cfg.RCenterGRPCAddr)
	if err != nil {
		logging.Fatal("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		logging.Fatal("failed to serve: %v", err)
	}
}
