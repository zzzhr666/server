package rcenter

import (
	"context"
	"server/internal/battle/grpcclient"
	"server/internal/contract/battlepb"
	"server/internal/platform/logging"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type battleControlClient interface {
	CreateRoom(ctx context.Context, input grpcclient.CreateRoomInput) (*grpcclient.CreateRoomResult, error)
}

type battleConn interface {
	Close() error
}

type battleClientFactory func(node BattleNode) (battleConn, battleControlClient, error)

type battleClientEntry struct {
	conn   battleConn
	client battleControlClient
}

// BattleRepository 持有已注册战斗节点的缓存 gRPC 控制客户端。
type BattleRepository struct {
	mu      sync.Mutex
	clients map[string]*battleClientEntry
	factory battleClientFactory
}

// NewBattleRepository 创建通过 gRPC 连接战斗节点的仓储。
func NewBattleRepository() *BattleRepository {
	return newBattleRepositoryWithFactory(newGRPCBattleClient)
}

func newBattleRepositoryWithFactory(factory battleClientFactory) *BattleRepository {
	return &BattleRepository{
		clients: make(map[string]*battleClientEntry),
		factory: factory,
	}
}

// CreateRoom forwards a room creation request to a registered battle node.
func (b *BattleRepository) CreateRoom(ctx context.Context, nodeName string, input CreateBattleRoomInput) error {
	b.mu.Lock()
	entry, ok := b.clients[nodeName]
	b.mu.Unlock()
	if !ok {
		logging.Error("battle node client missing node=%s", nodeName)
		return ErrBattleNodeNotRegistered
	}
	res, err := entry.client.CreateRoom(ctx, grpcclient.CreateRoomInput{
		RoomName:       input.RoomName,
		Token:          input.Token,
		PlayerIDs:      input.PlayerIDs,
		PlayerLoadouts: toBattleGRPCPlayerLoadouts(input.PlayerLoadouts),
	})
	if err != nil {
		logging.Error("create battle room failed node=%s: %v", nodeName, err)
		return err
	}
	if res.Status != grpcclient.CreateRoomStatusOK {
		logging.Error("battle room rejected node=%s status=%s", nodeName, res.Status)
		return ErrCreateBattleRoomFailed
	}
	logging.Debug("battle room created node=%s room=%s players=%d", nodeName, input.RoomName, len(input.PlayerIDs))
	return nil
}

func toBattleGRPCPlayerLoadouts(loadouts []PlayerLoadout) []grpcclient.PlayerLoadout {
	result := make([]grpcclient.PlayerLoadout, 0, len(loadouts))
	for _, loadout := range loadouts {
		result = append(result, grpcclient.PlayerLoadout{
			PlayerID:         loadout.PlayerID,
			Nickname:         loadout.Nickname,
			Hero:             loadout.Hero,
			AttackLevel:      loadout.AttackLevel,
			AttackSpeedLevel: loadout.AttackSpeedLevel,
			HealthLevel:      loadout.HealthLevel,
			MoveSpeedLevel:   loadout.MoveSpeedLevel,
		})
	}
	return result
}

// RegisterNode 为战斗节点创建或替换缓存控制客户端。
func (b *BattleRepository) RegisterNode(ctx context.Context, node BattleNode) error {
	grpcConn, client, err := b.factory(node)
	if err != nil {
		logging.Error("connect battle node failed node=%s: %v", node.Name, err)
		return err
	}
	newEntry := &battleClientEntry{
		conn:   grpcConn,
		client: client,
	}
	b.mu.Lock()
	oldEntry := b.clients[node.Name]
	b.clients[node.Name] = newEntry
	b.mu.Unlock()
	if oldEntry == nil {
		logging.Info("battle node registered node=%s control=%s", node.Name, node.ControlAddr)
	}
	if oldEntry != nil {
		if err := oldEntry.conn.Close(); err != nil {
			logging.Error("close old connection error: %v", err)
		}
	}
	return nil
}

func newGRPCBattleClient(node BattleNode) (battleConn, battleControlClient, error) {
	conn, err := grpc.NewClient(node.ControlAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	battlePBClient := battlepb.NewBattleControlServiceClient(conn)
	client := grpcclient.NewClient(battlePBClient)
	return conn, client, nil
}
