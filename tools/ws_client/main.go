package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func main() {
	url := flag.String("url", "ws://localhost:8080/ws", "logic WebSocket URL")
	token := flag.String("token", "", "auth session token")
	heartbeatInterval := flag.Duration("heartbeat", 30*time.Second, "heartbeat interval; set 0 to disable")
	flag.Parse()

	if *token == "" {
		log.Fatal("token is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, *url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"token": []string{*token},
		},
	})
	if err != nil {
		log.Fatalf("dial websocket: %v", err)
	}
	defer conn.CloseNow()

	fmt.Printf("connected: %s\n", *url)
	fmt.Println("type JSON messages, for example: {\"type\":\"match_start\"}")

	if *heartbeatInterval > 0 {
		go heartbeatLoop(conn, *heartbeatInterval)
	}
	go readLoop(conn)
	writeLoop(conn)
}

func readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			fmt.Printf("read closed: %v\n", err)
			os.Exit(0)
		}
		fmt.Printf("< %s\n", string(data))
	}
}

func writeLoop(conn *websocket.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := conn.Write(context.Background(), websocket.MessageText, []byte(line)); err != nil {
			log.Fatalf("write websocket: %v", err)
		}
		fmt.Printf("> %s\n", line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read stdin: %v", err)
	}
}

func heartbeatLoop(conn *websocket.Conn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		msg := []byte(`{"type":"heartbeat"}`)
		if err := conn.Write(context.Background(), websocket.MessageText, msg); err != nil {
			fmt.Printf("heartbeat stopped: %v\n", err)
			return
		}
		fmt.Printf("> %s\n", msg)
	}
}
