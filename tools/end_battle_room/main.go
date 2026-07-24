package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	battlegrpcclient "server/internal/battle/grpcclient"
	"server/internal/contract/battlepb"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9101", "battle control gRPC address")
	roomName := flag.String("room", "room-1", "battle room name")
	reason := flag.String("reason", "manual_end", "end room reason")
	timeout := flag.Duration("timeout", 3*time.Second, "gRPC request timeout")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("create gRPC client: %v", err)
	}
	defer conn.Close()

	client := battlegrpcclient.NewClient(battlepb.NewBattleControlServiceClient(conn))
	res, err := client.EndRoom(ctx, battlegrpcclient.EndRoomInput{
		RoomName: *roomName,
		Reason:   *reason,
	})
	if err != nil {
		log.Fatalf("end room: %v", err)
	}

	fmt.Printf("end_room status=%s message=%q room=%s reason=%s\n",
		res.Status,
		res.Message,
		*roomName,
		*reason,
	)
}
