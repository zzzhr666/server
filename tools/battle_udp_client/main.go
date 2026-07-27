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
	attack := flag.Bool("attack", false, "request attack in input packet")
	dash := flag.Bool("dash", false, "request dash in input packet")
	inputEvery := flag.Duration("move-after", 500*time.Millisecond, "input send interval after game_start")
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

	buffer := make([]byte, 4096)
	inputLoopStarted := false
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
		if _, ok := packet.GetPayload().(*battlepb.ServerPacket_GameStart); ok && !inputLoopStarted {
			inputLoopStarted = true
			go sendInputLoop(conn, *roomName, *playerID, *moveX, *moveY, *attack, *dash, *inputEvery)
		}
		if printServerPacket(&packet) && *exitOnGameOver {
			return
		}
	}
}

func sendInputLoop(conn *net.UDPConn, roomName string, playerID int64, moveX, moveY float64, attack, dash bool, interval time.Duration) {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := sendInput(conn, roomName, playerID, moveX, moveY, attack, dash); err != nil {
			log.Printf("send input: %v", err)
			return
		}
		<-ticker.C
	}
}

func sendInput(conn *net.UDPConn, roomName string, playerID int64, moveX, moveY float64, attack, dash bool) error {
	input := &battlepb.ClientPacket{
		Payload: &battlepb.ClientPacket_Input{
			Input: &battlepb.ClientInput{
				RoomName:        roomName,
				PlayerId:        playerID,
				X:               float32(moveX),
				Y:               float32(moveY),
				AttackRequested: attack,
				DashRequested:   dash,
			},
		},
	}
	bytes, err := proto.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}
	if _, err := conn.Write(bytes); err != nil {
		return fmt.Errorf("write input: %w", err)
	}
	fmt.Printf("sent input player=%d x=%.2f y=%.2f attack=%v dash=%v\n", playerID, moveX, moveY, attack, dash)
	return nil
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
