package rcenter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	statecontract "server/internal/contract/state"
	"slices"
	"sync"
	"time"
)

// BattleNodeController coordinates registered battle nodes over the control plane.
type BattleNodeController interface {
	CreateRoom(ctx context.Context, nodeName string, input CreateBattleRoomInput) error
	RegisterNode(ctx context.Context, node BattleNode) error
}

type waitingPlayer struct {
	playerID int64
	weapon   string
}

// GameCenterService keeps registered battle nodes, the in-memory match queue, and active battle assignments.
type GameCenterService struct {
	mu                   sync.Mutex
	battleNodes          map[string]BattleNode
	waitingPlayers       []waitingPlayer
	battleNodeController BattleNodeController
	inGamePlayers        map[int64]struct{}
	coinClient           statecontract.CoinClient
	rewardRule           RewardRule
	growthClient         statecontract.GrowthClient
	activeMatches        map[int64]ActiveMatch
}

type ServiceConfig struct {
	BattleNodeController BattleNodeController
	CoinClient           statecontract.CoinClient
	RewardRule           RewardRule
	GrowthClient         statecontract.GrowthClient
}

// NewService creates an empty in-memory rcenter service.
func NewService(config ServiceConfig) *GameCenterService {
	rewardRule := config.RewardRule
	if rewardRule.MonsterKillReward == nil {
		rewardRule = DefaultRewardRule()
	}
	return &GameCenterService{
		battleNodes:          make(map[string]BattleNode),
		battleNodeController: config.BattleNodeController,
		inGamePlayers:        make(map[int64]struct{}),
		coinClient:           config.CoinClient,
		rewardRule:           rewardRule,
		growthClient:         config.GrowthClient,
		activeMatches:        make(map[int64]ActiveMatch),
	}
}

func validateBattleNode(node BattleNode) error {
	if node.Name == "" || node.UDPAddr == "" || node.ControlAddr == "" || node.MaxPlayers <= 0 {
		return ErrInvalidBattleNode
	}
	return nil
}

// ResumeMatch returns the active battle connection data assigned to a player.
func (g *GameCenterService) ResumeMatch(ctx context.Context, playerID int64) (*MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	res, ok := g.activeMatches[playerID]
	if !ok {
		return nil, ErrActiveMatchNotFound
	}

	return &MatchResult{
		Status:         MatchStatusMatched,
		RoomName:       res.RoomName,
		Token:          res.Token,
		BattleNodeName: res.BattleNodeName,
		BattleUDPAddr:  res.BattleUDPAddr,
		PlayerIDs:      slices.Clone(res.PlayerIDs),
		PlayerLoadouts: slices.Clone(res.PlayerLoadouts),
	}, nil
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
func (g *GameCenterService) StartMatch(ctx context.Context, playerID int64, weapon string) (*MatchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}

	g.mu.Lock()
	_, ok := g.inGamePlayers[playerID]
	if ok {
		g.mu.Unlock()
		return nil, ErrPlayerInGame
	}
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
		g.waitingPlayers = append(g.waitingPlayers, waitingPlayer{
			playerID: playerID,
			weapon:   weapon,
		})
		g.mu.Unlock()
		return &MatchResult{
			Status: MatchStatusWaiting,
		}, nil
	}

	waitingPlayer := g.waitingPlayers[0]
	g.waitingPlayers = g.waitingPlayers[1:]

	roomName := newRandomName("room")
	token := newRandomName("token")
	playerIDs := []int64{waitingPlayer.playerID, playerID}

	for _, inGamePlayerID := range playerIDs {
		g.inGamePlayers[inGamePlayerID] = struct{}{}
	}
	g.mu.Unlock()
	if g.growthClient == nil {
		g.clearGameContext(playerIDs)
		return nil, ErrUnavailableGrowthClient
	}
	waitingGrowth, err := g.growthClient.GetGrowth(ctx, waitingPlayer.playerID)
	if err != nil {
		g.clearGameContext(playerIDs)
		return nil, err
	}
	currentGrowth, err := g.growthClient.GetGrowth(ctx, playerID)
	if err != nil {
		g.clearGameContext(playerIDs)
		return nil, err
	}

	playerLoadouts := []PlayerLoadout{
		{
			PlayerID:         waitingPlayer.playerID,
			Weapon:           waitingPlayer.weapon,
			AttackLevel:      waitingGrowth.AttackLevel,
			AttackSpeedLevel: waitingGrowth.AttackSpeedLevel,
			HealthLevel:      waitingGrowth.HealthLevel,
			MoveSpeedLevel:   waitingGrowth.MoveSpeedLevel,
		},
		{
			PlayerID:         playerID,
			Weapon:           weapon,
			AttackLevel:      currentGrowth.AttackLevel,
			AttackSpeedLevel: currentGrowth.AttackSpeedLevel,
			HealthLevel:      currentGrowth.HealthLevel,
			MoveSpeedLevel:   currentGrowth.MoveSpeedLevel,
		},
	}

	if err := g.battleNodeController.CreateRoom(ctx, node.Name, CreateBattleRoomInput{
		RoomName:       roomName,
		Token:          token,
		PlayerIDs:      playerIDs,
		PlayerLoadouts: playerLoadouts,
	}); err != nil {
		g.clearGameContext(playerIDs)
		return nil, err
	}

	activeMatch := ActiveMatch{
		RoomName:       roomName,
		Token:          token,
		BattleNodeName: node.Name,
		BattleUDPAddr:  node.UDPAddr,
		PlayerIDs:      slices.Clone(playerIDs),
		PlayerLoadouts: slices.Clone(playerLoadouts),
		CreatedAt:      time.Now(),
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, player := range playerIDs {
		g.activeMatches[player] = activeMatch
	}

	return &MatchResult{
		Status:         MatchStatusMatched,
		RoomName:       roomName,
		Token:          token,
		BattleNodeName: node.Name,
		BattleUDPAddr:  node.UDPAddr,
		PlayerIDs:      playerIDs,
		PlayerLoadouts: playerLoadouts,
	}, nil
}

// FinishMatch releases matched players so they can enter matchmaking again.
func (g *GameCenterService) FinishMatch(ctx context.Context, input FinishMatchInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, playerID := range input.PlayerIDs {
		if playerID <= 0 {
			return ErrInvalidPlayerID
		}
	}
	if len(input.PlayerStats) > 0 && g.coinClient == nil {
		return ErrUnavailableCoinClient
	}
	for _, stat := range input.PlayerStats {
		reward, err := CalculateCoinReward(stat, input.Reason, g.rewardRule)
		if err != nil {
			return err
		}
		_, err = g.coinClient.AddPlayerCoins(ctx, statecontract.AddPlayerCoinsInput{
			PlayerID: stat.PlayerID,
			Amount:   reward,
		})
		if err != nil {
			return err
		}
	}
	g.clearGameContext(input.PlayerIDs)
	return nil
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
		if playerID == waitingPlayer.playerID {
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
	for i, waitingPlayer := range g.waitingPlayers {
		if waitingPlayer.playerID == playerID {
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

func (g *GameCenterService) clearGameContext(playIDs []int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, playerID := range playIDs {
		delete(g.inGamePlayers, playerID)
		delete(g.activeMatches, playerID)
	}
}
