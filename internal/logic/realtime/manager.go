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

// Add 注册玩家的新连接，并返回新连接及被替换的旧连接信息。
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

// Remove 仅在连接 ID 匹配时移除玩家连接。
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

// Get 返回玩家当前注册的连接信息。
func (m *connectionManager) Get(playerID int64) (connectionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.connections[playerID]
	return info, ok
}

// Touch 仅在连接 ID 匹配时刷新玩家连接的最后活跃时间。
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

// Close 向玩家发送可选的关闭通知并关闭当前连接。
func (m *connectionManager) Close(playerID int64, envelope *realtimepb.ServerEnvelope) bool {
	info, ok := m.Get(playerID)
	if !ok {
		return false
	}
	writeErr := info.session.Write(envelope)
	closeErr := info.session.Close()
	return writeErr == nil && closeErr == nil
}

// Send 将服务端消息写入玩家当前连接。
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

// Broadcast 向除指定玩家外的全部当前连接发送消息，并返回成功数量。
func (m *connectionManager) Broadcast(envelope *realtimepb.ServerEnvelope, excludedPlayerID int64) int {
	if envelope == nil {
		return 0
	}
	m.mu.RLock()
	sessions := make([]*session, 0, len(m.connections))
	for _, connection := range m.connections {
		if excludedPlayerID > 0 && connection.playerID == excludedPlayerID {
			continue
		}
		sessions = append(sessions, connection.session)
	}
	m.mu.RUnlock()
	count := 0
	for _, session := range sessions {
		if err := session.Write(envelope); err != nil {
			continue
		}
		count += 1
	}
	return count
}
func newConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[int64]connectionInfo),
	}
}
