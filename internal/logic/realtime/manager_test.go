package realtime

import (
	"testing"
	"time"
)

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
