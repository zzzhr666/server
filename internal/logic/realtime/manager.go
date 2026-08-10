package realtime

import (
	"server/internal/contract/realtimepb"
	"sync"
	"time"
)

type connectionID uint64

type connectionInfo struct {
	playerID        int64
	id              connectionID
	session         *session
	connectedAt     time.Time
	lastHeartbeatAt time.Time
}

type connectionManager struct {
	mu          sync.RWMutex
	nextID      connectionID
	connections map[int64]connectionInfo //playerId -> conn
}

func (m *connectionManager) Add(playerID int64, session *session) (connectionInfo, *connectionInfo) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	info := connectionInfo{
		playerID:        playerID,
		id:              m.nextConnectionID(),
		session:         session,
		connectedAt:     now,
		lastHeartbeatAt: now,
	}
	oldInfo, ok := m.connections[info.playerID]
	m.connections[playerID] = info
	if ok {
		return info, &oldInfo
	}
	return info, nil
}

func (m *connectionManager) Remove(playerID int64, id connectionID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	info, ok := m.connections[playerID]
	if !ok || info.id != id {
		return false
	}
	delete(m.connections, playerID)
	return true
}

func (m *connectionManager) Get(playerID int64) (connectionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.connections[playerID]
	return info, ok
}

func (m *connectionManager) Touch(playerID int64, id connectionID, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	info, ok := m.connections[playerID]
	if !ok {
		return false
	}
	if info.id != id {
		return false
	}
	info.lastHeartbeatAt = now
	m.connections[playerID] = info
	return true
}

func (m *connectionManager) Close(playerID int64, envelope *realtimepb.ServerEnvelope) bool {
	info, ok := m.Get(playerID)
	if !ok {
		return false
	}
	writeErr := info.session.Write(envelope)
	closeErr := info.session.Close()
	return writeErr == nil && closeErr == nil
}

func (m *connectionManager) Send(playerID int64, envelope *realtimepb.ServerEnvelope) bool {
	info, ok := m.Get(playerID)
	if !ok {
		return false
	}
	if err := info.session.Write(envelope); err != nil {
		return false
	}
	return true
}

func (m *connectionManager) nextConnectionID() connectionID {
	m.nextID++
	return m.nextID
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[int64]connectionInfo),
	}
}
