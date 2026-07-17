package main

import (
	"context"
	"log"
	"net"
	"net/rpc"
	"server/internal/platform/config"
	"server/internal/platform/redisdb"
	"server/internal/state/redisstore"
	"server/internal/state/rpcserver"
	"server/internal/state/service"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()
	cfg := config.Default()

	//prepare redis
	redisClient := redisdb.NewClient(cfg.Redis)
	defer func(redisClient *redis.Client) {
		if err := redisClient.Close(); err != nil {
			log.Fatalf("redis close failed: %v", err)
		}
	}(redisClient)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	//make store
	store := redisstore.NewStore(redisClient)

	//make service
	stateService := service.NewService(store, store, store)
	server := rpc.NewServer()
	if err := server.RegisterName(rpcserver.ServiceName, rpcserver.NewServer(stateService)); err != nil {
		log.Fatalf("register state rpc server failed: %v", err)
	}

	listener, err := net.Listen("tcp", cfg.StateRPCAddr)
	if err != nil {
		log.Fatalf("state rpc listen failed: %v", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Fatalf("state rpc listener close failed: %v", err)
		}
	}(listener)

	log.Printf("state rpc server listening at %v", listener.Addr())
	server.Accept(listener)

}
