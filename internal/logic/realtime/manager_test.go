package realtime

import (
	"fmt"
	"net"
	"server/internal/contract/realtimepb"
	"testing"
	"time"
)

func TestConnectionManagerBroadcastsToAllExceptExcludedPlayer(t *testing.T) {
	manager := newConnectionManager()
	connections := make([]net.Conn, 0, 3)
	for _, playerID := range []int64{7, 8, 9} {
		serverConn, clientConn := net.Pipe()
		connections = append(connections, clientConn)
		manager.Add(playerID, newSession(serverConn))
	}
	defer func() {
		for _, conn := range connections {
			_ = conn.Close()
		}
	}()

	envelope := &realtimepb.ServerEnvelope{
		Payload: &realtimepb.ServerEnvelope_FriendRemoved{
			FriendRemoved: &realtimepb.FriendRemoved{PlayerId: 7},
		},
	}
	done := make(chan int, 1)
	go func() { done <- manager.Broadcast(envelope, 8) }()

	results := make(chan error, 2)
	for _, clientConn := range []net.Conn{connections[0], connections[2]} {
		go func(conn net.Conn) {
			got, err := readServerEnvelope(conn)
			if err != nil {
				results <- err
				return
			}
			if got.GetFriendRemoved().GetPlayerId() != 7 {
				results <- fmt.Errorf("friend removed player = %d, want 7", got.GetFriendRemoved().GetPlayerId())
				return
			}
			results <- nil
		}(clientConn)
	}
	for i := 0; i < 2; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("broadcast readers timed out")
		}
	}
	if count := <-done; count != 2 {
		t.Fatalf("Broadcast() count = %d, want 2", count)
	}
}

func TestConnectionManagerBroadcastContinuesAfterWriteFailure(t *testing.T) {
	manager := newConnectionManager()
	failedServer, failedClient := net.Pipe()
	workingServer, workingClient := net.Pipe()
	defer failedClient.Close()
	defer workingClient.Close()
	manager.Add(7, newSession(failedServer))
	manager.Add(8, newSession(workingServer))
	_ = failedClient.Close()

	envelope := &realtimepb.ServerEnvelope{
		Payload: &realtimepb.ServerEnvelope_FriendRemoved{
			FriendRemoved: &realtimepb.FriendRemoved{PlayerId: 7},
		},
	}
	done := make(chan int, 1)
	go func() { done <- manager.Broadcast(envelope, 0) }()
	if err := workingClient.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	got := readServerEnvelopeOrFail(t, workingClient)
	if got.GetFriendRemoved().GetPlayerId() != 7 {
		t.Fatalf("broadcast envelope = %v, want friend removed for player 7", got)
	}
	if count := <-done; count != 1 {
		t.Fatalf("Broadcast() count = %d, want 1", count)
	}
}

func TestConnectionManagerOldConnectionCannotAffectReplacement(t *testing.T) {
	manager := newConnectionManager()
	oldSession := &session{}
	newSession := &session{}

	oldInfo, replaced := manager.Add(7, oldSession)
	if replaced != nil {
		t.Fatalf("first Add() replacement = %+v, want nil", replaced)
	}
	newInfo, replaced := manager.Add(7, newSession)
	if replaced == nil || replaced.id != oldInfo.id || replaced.session != oldSession {
		t.Fatalf("replacement = %+v, want old connection", replaced)
	}
	if oldInfo.id == newInfo.id {
		t.Fatalf("connection IDs are equal: %d", oldInfo.id)
	}

	if manager.Touch(7, oldInfo.id, time.Now()) {
		t.Fatal("Touch() with the replaced connection = true, want false")
	}
	if manager.Remove(7, oldInfo.id) {
		t.Fatal("Remove() with the replaced connection = true, want false")
	}

	current, ok := manager.Get(7)
	if !ok {
		t.Fatal("Get() after replacement = not found")
	}
	if current.id != newInfo.id || current.session != newSession {
		t.Fatalf("current connection = %+v, want ID %d and replacement session", current, newInfo.id)
	}

	heartbeatAt := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	if !manager.Touch(7, newInfo.id, heartbeatAt) {
		t.Fatal("Touch() with the current connection = false, want true")
	}
	current, ok = manager.Get(7)
	if !ok {
		t.Fatal("Get() after touch = not found")
	}
	if !current.lastHeartbeatAt.Equal(heartbeatAt) {
		t.Fatalf("last heartbeat = %v, want %v", current.lastHeartbeatAt, heartbeatAt)
	}

	if !manager.Remove(7, newInfo.id) {
		t.Fatal("Remove() with the current connection = false, want true")
	}
	if _, ok := manager.Get(7); ok {
		t.Fatal("Get() after removing the current connection = found, want not found")
	}
}
