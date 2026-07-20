package httpapi

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type connID uint64

type connectionInfo struct {
	playerID        int64
	id              connID
	conn            *websocket.Conn
	connectedAt     time.Time
	lastHeartbeatAt time.Time
}

type connManager struct {
	mu      sync.RWMutex
	nextID  connID
	players map[int64]connectionInfo
}

func newConnManager() *connManager {
	return &connManager{
		nextID:  0,
		players: make(map[int64]connectionInfo),
	}
}

func (m *connManager) Add(playerID int64, conn *websocket.Conn) connectionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	connectionID := m.nextID
	m.nextID++

	info := connectionInfo{
		playerID:        playerID,
		id:              connectionID,
		conn:            conn,
		connectedAt:     now,
		lastHeartbeatAt: now,
	}
	m.players[playerID] = info
	return info
}

func (m *connManager) Touch(playerID int64, id connID, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.players[playerID]
	if !ok {
		return false
	}
	if info.id != id {
		return false
	}
	info.lastHeartbeatAt = now
	m.players[playerID] = info
	return true
}

func (m *connManager) Get(playerID int64) (connectionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.players[playerID]
	return info, ok
}

func (m *connManager) Remove(playerID int64, id connID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.players[playerID]
	if !ok || info.id != id {
		return false
	}
	delete(m.players, playerID)
	return true
}

func (m *connManager) SendJSON(ctx context.Context, playerID int64, msg any) bool {
	m.mu.RLock()
	info, ok := m.players[playerID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	if err := wsjson.Write(ctx, info.conn, msg); err != nil {
		return false
	}
	return true
}
