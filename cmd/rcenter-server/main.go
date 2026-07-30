package main

import (
	"log"
	"net"
	"server/internal/contract/rcenterpb"
	"server/internal/contract/statepb"
	"server/internal/platform/config"
	"server/internal/rcenter"
	"server/internal/rcenter/grpcserver"
	"server/internal/state/grpcclient"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Default()
	stateConn, err := grpc.NewClient(cfg.StateGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("fail to call NewClient: %v", err)
	}
	defer func() {
		if err := stateConn.Close(); err != nil {
			log.Fatalf("fail to close client connection: %v", err)
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
		log.Fatalf("failed to listen: %v", err)
	}
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
