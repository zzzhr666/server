package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"server/internal/contract/battlepb"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9101", "battle control gRPC address")
	roomName := flag.String("room", "room-1", "battle room name")
	token := flag.String("token", "token-1", "battle room token")
	players := flag.String("players", "1001,1002", "comma-separated player IDs")
	timeout := flag.Duration("timeout", 3*time.Second, "gRPC request timeout")
	flag.Parse()

	playerIDs, err := parsePlayerIDs(*players)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("create gRPC client: %v", err)
	}
	defer conn.Close()

	client := battlepb.NewBattleControlServiceClient(conn)
	res, err := client.CreateRoom(ctx, &battlepb.CreateRoomRequest{
		RoomName:  *roomName,
		Token:     *token,
		PlayerIds: playerIDs,
	})
	if err != nil {
		log.Fatalf("create room: %v", err)
	}

	fmt.Printf("create_room status=%s message=%q room=%s players=%v\n",
		res.GetStatus().String(),
		res.GetMessage(),
		*roomName,
		playerIDs,
	)
}

func parsePlayerIDs(raw string) ([]int64, error) {
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid player id %q", part)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("missing player IDs")
	}
	return ids, nil
}
