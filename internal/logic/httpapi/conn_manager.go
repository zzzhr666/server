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
	// 每次连接使用单调 ID 覆盖同一玩家的旧记录。延迟执行的旧连接 defer 会通过
	// ID 校验发现自己已过期，从而不会错误清除新连接的在线状态。
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
	// 仅当前连接可刷新心跳；被新登录替换的旧 WebSocket 不得继续维持玩家在线。
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
	// WebSocket 关闭顺序不确定，使用 ID 防止旧连接关闭时删掉新连接。
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
	// 读取记录后立即释放 map 锁，网络写入不能阻塞其他连接的心跳、替换和清理。
	if err := wsjson.Write(ctx, info.conn, msg); err != nil {
		return false
	}
	return true
}

func (m *connManager) Close(ctx context.Context, playerID int64, msg any, status websocket.StatusCode, reason string) bool {
	m.mu.RLock()
	info, ok := m.players[playerID]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	_ = wsjson.Write(ctx, info.conn, msg)
	if err := info.conn.Close(status, reason); err != nil {
		return false
	}
	return true
}
