package main

import (
	"context"
	"flag"
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
	demoBattleNode := flag.Bool("demo-battle-node", false, "register one in-memory demo battle node on startup")
	demoBattleName := flag.String("demo-battle-name", "battle-demo", "demo battle node name")
	demoBattleKCPAddr := flag.String("demo-battle-kcp-addr", "127.0.0.1:7001", "demo battle node KCP address")
	demoBattleControlAddr := flag.String("demo-battle-control-addr", "127.0.0.1:9101", "demo battle node control address")
	demoBattleMaxPlayers := flag.Int("demo-battle-max-players", 100, "demo battle node max players")
	demoBattleActivePlayers := flag.Int("demo-battle-active-players", 0, "demo battle node active players")
	flag.Parse()

	centerService := rcenter.NewService()
	if *demoBattleNode {
		if err := centerService.RegisterBattleNode(context.Background(), rcenter.BattleNode{
			Name:          *demoBattleName,
			KCPAddr:       *demoBattleKCPAddr,
			ControlAddr:   *demoBattleControlAddr,
			MaxPlayers:    *demoBattleMaxPlayers,
			ActivePlayers: *demoBattleActivePlayers,
		}); err != nil {
			log.Fatalf("register demo battle node failed: %v", err)
		}
		log.Printf("registered demo battle node name=%s kcp_addr=%s control_addr=%s", *demoBattleName, *demoBattleKCPAddr, *demoBattleControlAddr)
	}

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
