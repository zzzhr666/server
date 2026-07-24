package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/protobuf/proto"

	"server/internal/contract/battlepb"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7001", "battle UDP address")
	roomName := flag.String("room", "", "battle room name")
	token := flag.String("token", "", "battle room token")
	playerID := flag.Int64("player", 0, "player id")
	moveX := flag.Float64("move-x", 0, "move input x")
	moveY := flag.Float64("move-y", 0, "move input y")
	moveAfter := flag.Duration("move-after", time.Second, "delay before sending move input")
	timeout := flag.Duration("timeout", 10*time.Second, "read timeout")
	exitOnTimeout := flag.Bool("exit-on-timeout", false, "exit when no packet is received before timeout")
	exitOnGameOver := flag.Bool("exit-on-game-over", true, "exit after receiving a game_over packet")
	flag.Parse()

	if *roomName == "" || *token == "" || *playerID <= 0 {
		log.Fatal("room, token, and positive player are required")
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalf("resolve UDP address: %v", err)
	}
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Fatalf("dial UDP: %v", err)
	}
	defer conn.Close()

	hello := &battlepb.ClientPacket{
		Payload: &battlepb.ClientPacket_Hello{
			Hello: &battlepb.ClientHello{
				RoomName: *roomName,
				PlayerId: *playerID,
				Token:    *token,
			},
		},
	}
	bytes, err := proto.Marshal(hello)
	if err != nil {
		log.Fatalf("marshal hello: %v", err)
	}
	if _, err := conn.Write(bytes); err != nil {
		log.Fatalf("send hello: %v", err)
	}
	fmt.Printf("sent hello player=%d room=%s addr=%s\n", *playerID, *roomName, *addr)

	go func() {
		time.Sleep(*moveAfter)
		moveInput := &battlepb.ClientPacket{
			Payload: &battlepb.ClientPacket_MoveInput{
				MoveInput: &battlepb.ClientMoveInput{
					RoomName: *roomName,
					PlayerId: *playerID,
					X:        float32(*moveX),
					Y:        float32(*moveY),
				},
			},
		}
		bytes, err := proto.Marshal(moveInput)
		if err != nil {
			log.Printf("marshal move input: %v", err)
			return
		}
		if _, err := conn.Write(bytes); err != nil {
			log.Printf("send move input: %v", err)
			return
		}
		fmt.Printf("sent move_input player=%d x=%.2f y=%.2f\n", *playerID, *moveX, *moveY)
	}()

	buffer := make([]byte, 4096)
	for {
		if err := conn.SetReadDeadline(time.Now().Add(*timeout)); err != nil {
			log.Fatalf("set read deadline: %v", err)
		}
		n, err := conn.Read(buffer)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				if *exitOnTimeout {
					log.Fatalf("read packet: %v", err)
				}
				fmt.Printf("waiting for packets: %v\n", err)
				continue
			}
			log.Fatalf("read packet: %v", err)
		}

		var packet battlepb.ServerPacket
		if err := proto.Unmarshal(buffer[:n], &packet); err != nil {
			fmt.Printf("received undecodable packet len=%d err=%v\n", n, err)
			continue
		}
		if printServerPacket(&packet) && *exitOnGameOver {
			return
		}
	}
}

func printServerPacket(packet *battlepb.ServerPacket) bool {
	switch payload := packet.GetPayload().(type) {
	case *battlepb.ServerPacket_Hello:
		fmt.Printf("server_hello conv=%d message=%q\n", payload.Hello.GetConv(), payload.Hello.GetMessage())
	case *battlepb.ServerPacket_GameStart:
		fmt.Printf("game_start room=%s players=%v\n", payload.GameStart.GetRoomName(), payload.GameStart.GetPlayerIds())
	case *battlepb.ServerPacket_GameOver:
		fmt.Printf("game_over room=%s players=%v reason=%s\n", payload.GameOver.GetRoomName(), payload.GameOver.GetPlayerIds(), payload.GameOver.GetReason())
		return true
	case *battlepb.ServerPacket_Snapshot:
		fmt.Printf("snapshot room=%s entities=%d\n", payload.Snapshot.GetRoomName(), len(payload.Snapshot.GetEntities()))
		for _, entity := range payload.Snapshot.GetEntities() {
			fmt.Printf("  entity=%d pos=(%.2f, %.2f) dir=(%.2f, %.2f) hp=%d/%d\n",
				entity.GetEntity(),
				entity.GetXPosition(),
				entity.GetYPosition(),
				entity.GetXDirection(),
				entity.GetYDirection(),
				entity.GetCurrentHealth(),
				entity.GetMaxHealth(),
			)
		}
	case *battlepb.ServerPacket_Error:
		fmt.Printf("error code=%s message=%q\n", payload.Error.GetCode(), payload.Error.GetMessage())
	default:
		fmt.Printf("unknown packet: %T\n", payload)
	}
	return false
}
