package rcenter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"server/internal/contract/state"
	"slices"
	"sync"
	"time"
)

// BattleNodeController 通过控制面协调已注册的战斗节点。
type BattleNodeController interface {
	CreateRoom(ctx context.Context, nodeName string, input CreateBattleRoomInput) error
	RegisterNode(ctx context.Context, node BattleNode) error
}

type waitingPlayer struct {
	playerID int64
	weapon   string
}

// GameCenterService 持有已注册战斗节点、内存匹配队列和活跃战斗分配。
type GameCenterService struct {
	mu                   sync.Mutex
	battleNodes          map[string]BattleNode
	waitingPlayers       []waitingPlayer
	battleNodeController BattleNodeController
	inGamePlayers        map[int64]struct{}
	coinClient           state.CoinClient
	rewardRule           RewardRule
	growthClient         state.GrowthClient
	activeMatches        map[int64]ActiveMatch
	metrics              *Metrics
}

type ServiceConfig struct {
	BattleNodeController BattleNodeController
	CoinClient           state.CoinClient
	RewardRule           RewardRule
	GrowthClient         state.GrowthClient
	Metrics              *Metrics
}

// NewService 创建空的内存 rcenter 服务。
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
		metrics:              config.Metrics,
	}
}

func validateBattleNode(node BattleNode) error {
	if node.Name == "" || node.UDPAddr == "" || node.ControlAddr == "" || node.MaxPlayers <= 0 {
		return ErrInvalidBattleNode
	}
	return nil
}

// ResumeMatch 返回分配给玩家的活跃战斗连接数据。
func (g *GameCenterService) ResumeMatch(ctx context.Context, playerID int64) (*MatchResult, error) {
	if err := ctx.Err(); err != nil {
		g.observeMatchOperation("resume", "error")
		return nil, err
	}
	if playerID <= 0 {
		g.observeMatchOperation("resume", "error")
		return nil, ErrInvalidPlayerID
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	// ActiveMatch 独立于 TCP 连接保存。返回副本时克隆切片，避免调用方修改
	// 内存中的共享对局数据，破坏其他玩家的恢复结果。
	res, ok := g.activeMatches[playerID]
	if !ok {
		g.observeMatchOperation("resume", "error")
		return nil, ErrActiveMatchNotFound
	}
	g.observeMatchOperation("resume", "success")
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

// RegisterBattleNode 记录或刷新一个可承载房间的战斗节点。
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
	if g.metrics != nil {
		g.metrics.BattleNodes.Set(float64(len(g.battleNodes)))
	}

	return nil
}

// ListBattleNodes 返回已注册战斗节点的快照。
func (g *GameCenterService) ListBattleNodes() []BattleNode {
	g.mu.Lock()
	defer g.mu.Unlock()
	nodes := make([]BattleNode, 0, len(g.battleNodes))
	for _, node := range g.battleNodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// StartMatch 将玩家入队，或与等待时间最长的玩家配对。
func (g *GameCenterService) StartMatch(ctx context.Context, playerID int64, weapon string) (*MatchResult, error) {
	if err := ctx.Err(); err != nil {
		g.observeMatchOperation("start", "error")
		return nil, err
	}
	if playerID <= 0 {
		g.observeMatchOperation("start", "error")
		return nil, ErrInvalidPlayerID
	}

	// 锁内只处理内存匹配状态：检查已在局、选节点、FIFO 队列和预占 inGame 标记。
	// 外部 gRPC 读取成长和创建房间必须在锁外执行，否则慢节点会阻塞所有匹配请求。
	g.mu.Lock()
	_, ok := g.inGamePlayers[playerID]
	if ok {
		g.mu.Unlock()
		g.observeMatchOperation("start", "error")
		return nil, ErrPlayerInGame
	}
	node, ok := g.selectBattleNode()
	if !ok {
		g.mu.Unlock()
		g.observeMatchOperation("start", "error")
		return nil, ErrNoAvailableBattleNode
	}
	if g.isWaiting(playerID) {
		g.mu.Unlock()
		g.observeMatchOperation("start", "success")
		return &MatchResult{
			Status: MatchStatusWaiting,
		}, nil
	}
	if len(g.waitingPlayers) == 0 {
		// 首位玩家仅入 FIFO 队列，不创建房间也不占用 battle 节点容量。
		g.waitingPlayers = append(g.waitingPlayers, waitingPlayer{
			playerID: playerID,
			weapon:   weapon,
		})
		if g.metrics != nil {
			g.metrics.MatchQueue.Set(float64(len(g.waitingPlayers)))
		}
		g.mu.Unlock()
		g.observeMatchOperation("start", "success")
		return &MatchResult{
			Status: MatchStatusWaiting,
		}, nil
	}

	waitingPlayer := g.waitingPlayers[0]
	g.waitingPlayers = g.waitingPlayers[1:]
	if g.metrics != nil {
		g.metrics.MatchQueue.Set(float64(len(g.waitingPlayers)))
	}

	roomName := newRandomName("room")
	token := newRandomName("token")
	playerIDs := []int64{waitingPlayer.playerID, playerID}

	// 从队列配对成功后立即预占两名玩家，防止其在创建房间的异步窗口内再次发起匹配。
	// 任一后续步骤失败都会通过 clearGameContext 回滚此预占。
	for _, inGamePlayerID := range playerIDs {
		g.inGamePlayers[inGamePlayerID] = struct{}{}
	}
	g.mu.Unlock()
	// 局外成长在房间创建前读取并随 loadout 下发，战斗服后续无需访问局外状态。
	if g.growthClient == nil {
		g.clearGameContext(playerIDs)
		g.observeMatchOperation("start", "error")
		return nil, ErrUnavailableGrowthClient
	}
	waitingGrowth, err := g.growthClient.GetGrowth(ctx, waitingPlayer.playerID)
	if err != nil {
		g.clearGameContext(playerIDs)
		g.observeMatchOperation("start", "error")
		return nil, err
	}
	currentGrowth, err := g.growthClient.GetGrowth(ctx, playerID)
	if err != nil {
		g.clearGameContext(playerIDs)
		g.observeMatchOperation("start", "error")
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

	// CreateRoom 成功才发布 ActiveMatch。这样 TCP ResumeMatch 永远不会给客户端
	// 返回一个 battle-server 尚未预留的房间与 token。
	if err := g.battleNodeController.CreateRoom(ctx, node.Name, CreateBattleRoomInput{
		RoomName:       roomName,
		Token:          token,
		PlayerIDs:      playerIDs,
		PlayerLoadouts: playerLoadouts,
	}); err != nil {
		g.clearGameContext(playerIDs)
		g.observeMatchOperation("start", "error")
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
	// 再次加锁后为双方写入同一份 ActiveMatch；连接断开并不删除它，直到 battle-server
	// 调用 FinishMatch 或创建流程失败回滚，确保任意一方均能恢复到同一对局。
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, player := range playerIDs {
		g.activeMatches[player] = activeMatch
	}
	if g.metrics != nil {
		g.metrics.ActiveMatchPlayers.Set(float64(len(g.activeMatches)))
	}
	g.observeMatchOperation("start", "success")
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

// FinishMatch 原子发放对局奖励，并仅释放属于上报房间的玩家状态。
func (g *GameCenterService) FinishMatch(ctx context.Context, input FinishMatchInput) error {
	if err := ctx.Err(); err != nil {
		g.observeMatchOperation("finish", "error")
		return err
	}
	if input.RoomName == "" {
		g.observeMatchOperation("finish", "error")
		return ErrInvalidRoomName
	}
	if len(input.PlayerIDs) == 0 {
		g.observeMatchOperation("finish", "error")
		return ErrInvalidPlayerID
	}
	playerIDs := make(map[int64]struct{}, len(input.PlayerIDs))
	for _, playerID := range input.PlayerIDs {
		if playerID <= 0 {
			g.observeMatchOperation("finish", "error")
			return ErrInvalidPlayerID
		}
		if _, exists := playerIDs[playerID]; exists {
			g.observeMatchOperation("finish", "error")
			return ErrInvalidBattleStats
		}
		playerIDs[playerID] = struct{}{}
	}
	if len(input.PlayerStats) > 0 && len(input.PlayerStats) != len(playerIDs) {
		g.observeMatchOperation("finish", "error")
		return ErrInvalidBattleStats
	}
	if len(input.PlayerStats) > 0 && g.coinClient == nil {
		g.observeMatchOperation("finish", "error")
		return ErrUnavailableCoinClient
	}
	// 先计算完全部奖励，再以房间名作为稳定结算 ID 一次提交，避免部分玩家已到账、
	// 其余玩家失败后重试导致重复发放。断线超时没有统计，不写金币。
	rewards := make([]state.PlayerCoinReward, 0, len(input.PlayerStats))
	statPlayerIDs := make(map[int64]struct{}, len(input.PlayerStats))
	for _, stat := range input.PlayerStats {
		if _, belongsToMatch := playerIDs[stat.PlayerID]; !belongsToMatch {
			g.observeMatchOperation("finish", "error")
			return ErrInvalidBattleStats
		}
		if _, duplicate := statPlayerIDs[stat.PlayerID]; duplicate {
			g.observeMatchOperation("finish", "error")
			return ErrInvalidBattleStats
		}
		statPlayerIDs[stat.PlayerID] = struct{}{}
		reward, err := CalculateCoinReward(stat, input.Reason, g.rewardRule)
		if err != nil {
			g.observeMatchOperation("finish", "error")
			return err
		}
		rewards = append(rewards, state.PlayerCoinReward{
			PlayerID: stat.PlayerID,
			Amount:   reward,
		})
	}
	if len(rewards) > 0 {
		_, err := g.coinClient.SettleMatchRewards(ctx, state.SettleMatchRewardsInput{
			SettlementID: input.RoomName,
			Rewards:      rewards,
		})
		if err != nil {
			g.observeMatchOperation("finish", "error")
			return err
		}
	}
	// 结算成功或已经结算过后才释放对局；房间校验阻止迟到的旧回调误删新对局。
	g.clearFinishedMatch(input.RoomName, input.PlayerIDs)
	g.observeMatchOperation("finish", "success")
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

// CancelMatch 将等待中的玩家移出匹配队列。
func (g *GameCenterService) CancelMatch(ctx context.Context, playerID int64) error {
	if err := ctx.Err(); err != nil {
		g.observeMatchOperation("cancel", "error")
		return err
	}
	if playerID <= 0 {
		g.observeMatchOperation("cancel", "error")
		return ErrInvalidPlayerID
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, waitingPlayer := range g.waitingPlayers {
		if waitingPlayer.playerID == playerID {
			g.waitingPlayers = append(g.waitingPlayers[:i], g.waitingPlayers[i+1:]...)
			if g.metrics != nil {
				g.metrics.MatchQueue.Set(float64(len(g.waitingPlayers)))
			}
			g.observeMatchOperation("cancel", "success")
			return nil
		}
	}
	g.observeMatchOperation("cancel", "error")
	return ErrPlayerNotWaiting
}

// newRandomName 为房间名和令牌创建可读前缀加随机后缀。
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
	// 创建房间失败和正常结算都走这里，保证 inGamePlayers 与 activeMatches 始终成对释放，
	// 否则玩家会永久收到 ErrPlayerInGame 或过期的 ResumeMatch。
	for _, playerID := range playIDs {
		delete(g.inGamePlayers, playerID)
		delete(g.activeMatches, playerID)
	}
	if g.metrics != nil {
		g.metrics.ActiveMatchPlayers.Set(float64(len(g.activeMatches)))
	}
}

func (g *GameCenterService) clearFinishedMatch(roomName string, playerIDs []int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, playerID := range playerIDs {
		activeMatch, ok := g.activeMatches[playerID]
		if !ok || activeMatch.RoomName != roomName {
			continue
		}
		delete(g.inGamePlayers, playerID)
		delete(g.activeMatches, playerID)
	}
	if g.metrics != nil {
		g.metrics.ActiveMatchPlayers.Set(float64(len(g.activeMatches)))
	}
}

func (g *GameCenterService) observeMatchOperation(operation, result string) {
	if g.metrics == nil {
		return
	}
	g.metrics.MatchOperations.WithLabelValues(operation, result).Inc()
}
