package main

import (
	"log"
	"net"
	"server/internal/contract/rcenterpb"
	"server/internal/platform/config"
	"server/internal/rcenter"
	"server/internal/rcenter/grpcserver"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.Default()

	battleRepo := rcenter.NewBattleRepository()
	centerService := rcenter.NewService(battleRepo)

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
