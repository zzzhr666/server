package rcenter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// BattleNodeController coordinates registered battle nodes over the control plane.
type BattleNodeController interface {
	CreateRoom(ctx context.Context, nodeName string, input CreateBattleRoomInput) error
	RegisterNode(ctx context.Context, node BattleNode) error
}

// GameCenterService keeps registered battle nodes and the in-memory match queue.
type GameCenterService struct {
	mu                   sync.Mutex
	battleNodes          map[string]BattleNode
	waitingPlayers       []int64
	battleNodeController BattleNodeController
}

// NewService creates an empty in-memory rcenter service.
func NewService(battleNodeController BattleNodeController) *GameCenterService {
	return &GameCenterService{
		battleNodes:          make(map[string]BattleNode),
		battleNodeController: battleNodeController,
	}
}

func validateBattleNode(node BattleNode) error {
	if node.Name == "" || node.KCPAddr == "" || node.ControlAddr == "" || node.MaxPlayers <= 0 {
		return ErrInvalidBattleNode
	}
	return nil
}

// RegisterBattleNode records or refreshes a battle node that can host rooms.
func (g *GameCenterService) RegisterBattleNode(ctx context.Context, node BattleNode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateBattleNode(node); err != nil {
		return err
	}
	if err := g.battleNodeController.RegisterNode(ctx, node); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	node.LastSeen = time.Now()
	g.battleNodes[node.Name] = node

	return nil
}

// ListBattleNodes returns a snapshot of registered battle nodes.
func (g *GameCenterService) ListBattleNodes() []BattleNode {
	g.mu.Lock()
	defer g.mu.Unlock()
	nodes := make([]BattleNode, 0, len(g.battleNodes))
	for _, node := range g.battleNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// StartMatch queues a player or pairs them with the oldest waiting player.
func (g *GameCenterService) StartMatch(ctx context.Context, playerID int64) (*MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	g.mu.Lock()
	node, ok := g.selectBattleNode()
	if !ok {
		g.mu.Unlock()
		return nil, ErrNoAvailableBattleNode
	}
	if g.isWaiting(playerID) {
		g.mu.Unlock()
		return &MatchResult{
			Status: MatchStatusWaiting,
		}, nil
	}
	if len(g.waitingPlayers) == 0 {
		g.waitingPlayers = append(g.waitingPlayers, playerID)
		g.mu.Unlock()
		return &MatchResult{
			Status: MatchStatusWaiting,
		}, nil
	}

	waitingPlayerID := g.waitingPlayers[0]
	g.waitingPlayers = g.waitingPlayers[1:]
	g.mu.Unlock()

	roomName := newRandomName("room")
	token := newRandomName("token")
	playerIDs := []int64{waitingPlayerID, playerID}

	if err := g.battleNodeController.CreateRoom(ctx, node.Name, CreateBattleRoomInput{
		RoomName:  roomName,
		Token:     token,
		PlayerIDs: playerIDs,
	}); err != nil {
		return nil, err
	}

	return &MatchResult{
		Status:         MatchStatusMatched,
		RoomName:       roomName,
		Token:          token,
		BattleNodeName: node.Name,
		BattleKCPAddr:  node.KCPAddr,
		PlayerIDs:      playerIDs,
	}, nil
}

func (g *GameCenterService) selectBattleNode() (BattleNode, bool) {
	var selected BattleNode
	found := false
	for _, node := range g.battleNodes {
		if node.ActivePlayers >= node.MaxPlayers {
			continue
		}
		if !found || node.ActivePlayers < selected.ActivePlayers {
			selected = node
			found = true
		}
	}
	return selected, found
}

func (g *GameCenterService) isWaiting(playerID int64) bool {
	for _, waitingPlayer := range g.waitingPlayers {
		if playerID == waitingPlayer {
			return true
		}
	}
	return false
}

// CancelMatch removes a waiting player from the match queue.
func (g *GameCenterService) CancelMatch(ctx context.Context, playerID int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if playerID <= 0 {
		return ErrInvalidPlayerID
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, id := range g.waitingPlayers {
		if id == playerID {
			g.waitingPlayers = append(g.waitingPlayers[:i], g.waitingPlayers[i+1:]...)
			return nil
		}
	}
	return ErrPlayerNotWaiting
}

// newRandomName creates a readable prefix plus random suffix for room names and tokens.
func newRandomName(prefix string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return prefix + "-" + time.Now().Format("2006-01-02 15:04:05")
	}
	return prefix + "-" + hex.EncodeToString(bytes)
}
