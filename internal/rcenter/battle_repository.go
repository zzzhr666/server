package rcenter

import (
	"context"
	"log"
	"server/internal/battle/grpcclient"
	"server/internal/contract/battlepb"
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

// BattleRepository owns cached gRPC control clients for registered battle nodes.
type BattleRepository struct {
	mu      sync.Mutex
	clients map[string]*battleClientEntry
	factory battleClientFactory
}

// NewBattleRepository creates a repository that dials battle nodes with gRPC.
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
		return ErrBattleNodeNotRegistered
	}
	res, err := entry.client.CreateRoom(ctx, grpcclient.CreateRoomInput{
		RoomName:  input.RoomName,
		Token:     input.Token,
		PlayerIDs: input.PlayerIDs,
	})
	if err != nil {
		return err
	}
	if res.Status != grpcclient.CreateRoomStatusOK {
		return ErrCreateBattleRoomFailed
	}
	return nil
}

// RegisterNode creates or replaces the cached control client for a battle node.
func (b *BattleRepository) RegisterNode(ctx context.Context, node BattleNode) error {
	grpcConn, client, err := b.factory(node)
	if err != nil {
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
	if oldEntry != nil {
		if err := oldEntry.conn.Close(); err != nil {
			log.Println("close old connection error:", err)
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
