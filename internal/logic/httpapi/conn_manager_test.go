package httpapi

import (
	"testing"
	"time"
)

func TestConnManagerAddStoresConnection(t *testing.T) {
	manager := newConnManager()

	info := manager.Add(7)

	if info.playerID != 7 {
		t.Fatalf("player id = %d, want 7", info.playerID)
	}
	if info.id != 0 {
		t.Fatalf("connection id = %d, want 0", info.id)
	}
	stored, ok := manager.players[7]
	if !ok {
		t.Fatalf("stored connection missing")
	}
	if stored != info {
		t.Fatalf("stored connection = %+v, want %+v", stored, info)
	}
}

func TestConnManagerTouch(t *testing.T) {
	manager := newConnManager()
	info := manager.Add(7)
	refreshedAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

	if ok := manager.Touch(7, info.id, refreshedAt); !ok {
		t.Fatalf("Touch returned false, want true")
	}
	if got := manager.players[7].lastHeartbeatAt; !got.Equal(refreshedAt) {
		t.Fatalf("last heartbeat at = %v, want %v", got, refreshedAt)
	}
	got, ok := manager.Get(7)
	if !ok {
		t.Fatalf("Get returned false, want true")
	}
	if !got.lastHeartbeatAt.Equal(refreshedAt) {
		t.Fatalf("Get last heartbeat at = %v, want %v", got.lastHeartbeatAt, refreshedAt)
	}
}

func TestConnManagerOldConnectionCannotTouchOrRemoveNewConnection(t *testing.T) {
	manager := newConnManager()
	oldInfo := manager.Add(7)
	newInfo := manager.Add(7)
	refreshedAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

	if oldInfo.id == newInfo.id {
		t.Fatalf("connection ids are equal, want unique ids")
	}
	if ok := manager.Touch(7, oldInfo.id, refreshedAt); ok {
		t.Fatalf("old connection Touch returned true, want false")
	}
	if got := manager.players[7].id; got != newInfo.id {
		t.Fatalf("stored connection id = %d, want %d", got, newInfo.id)
	}

	if removed := manager.Remove(7, oldInfo.id); removed {
		t.Fatalf("old connection Remove returned true, want false")
	}
	if got := manager.players[7].id; got != newInfo.id {
		t.Fatalf("stored connection id after old remove = %d, want %d", got, newInfo.id)
	}

	if removed := manager.Remove(7, newInfo.id); !removed {
		t.Fatalf("current connection Remove returned false, want true")
	}
	if _, ok := manager.players[7]; ok {
		t.Fatalf("stored connection still exists after current remove")
	}
}
