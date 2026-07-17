package rpcclient

import (
	"context"
	"net/rpc"
	statecontract "server/internal/contract/state"
	"server/internal/state/rpcserver"
)

type Client struct {
	rpc *rpc.Client
}

func NewClient(rpcClient *rpc.Client) *Client {
	return &Client{rpc: rpcClient}
}

func mapRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch err.Error() {
	case statecontract.ErrAccountExists.Error():
		return statecontract.ErrAccountExists
	case statecontract.ErrAccountNotFound.Error():
		return statecontract.ErrAccountNotFound
	case statecontract.ErrSessionNotFound.Error():
		return statecontract.ErrSessionNotFound
	case statecontract.ErrPlayerNotFound.Error():
		return statecontract.ErrPlayerNotFound
	default:
		return err
	}
}

func (c *Client) call(method string, args any, reply any) error {
	return mapRPCError(c.rpc.Call(rpcserver.ServiceName+"."+method, args, reply))
}

func (c *Client) GetAccount(ctx context.Context, username string) (*statecontract.Account, error) {
	_ = ctx
	var reply rpcserver.GetAccountReply
	err := c.call("GetAccount", rpcserver.GetAccountArgs{
		Username: username,
	}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Account, nil
}

func (c *Client) RegisterAccount(ctx context.Context, input statecontract.RegisterAccountInput) (*statecontract.RegisterAccountResult, error) {
	_ = ctx
	var reply rpcserver.RegisterAccountReply
	err := c.call("RegisterAccount", rpcserver.RegisterAccountArgs{Input: input}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Result, nil
}

func (c *Client) CreateSession(ctx context.Context, session *statecontract.Session) error {
	_ = ctx
	var reply rpcserver.CreateSessionReply
	err := c.call("CreateSession", rpcserver.CreateSessionArgs{Session: session}, &reply)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) GetSession(ctx context.Context, token string) (*statecontract.Session, error) {
	_ = ctx
	var reply rpcserver.GetSessionReply
	err := c.call("GetSession", rpcserver.GetSessionArgs{Token: token}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Session, nil
}

func (c *Client) DeleteSession(ctx context.Context, token string) error {
	_ = ctx
	var reply rpcserver.DeleteSessionReply
	return c.call("DeleteSession", rpcserver.DeleteSessionArgs{Token: token}, &reply)
}

func (c *Client) CreatePlayer(ctx context.Context, player *statecontract.Player) error {
	_ = ctx
	var reply rpcserver.CreatePlayerReply
	return c.call("CreatePlayer", rpcserver.CreatePlayerArgs{Player: player}, &reply)

}

func (c *Client) GetPlayer(ctx context.Context, id int64) (*statecontract.Player, error) {
	_ = ctx
	var reply rpcserver.GetPlayerReply
	err := c.call("GetPlayer", rpcserver.GetPlayerArgs{ID: id}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Player, nil
}

func (c *Client) NextPlayerID(ctx context.Context) (int64, error) {
	_ = ctx
	var reply rpcserver.NextPlayerIDReply
	err := c.call("NextPlayerID", rpcserver.NextPlayerIDArgs{}, &reply)
	if err != nil {
		return 0, err
	}
	return reply.ID, nil
}

func (c *Client) CreateAccount(ctx context.Context, account *statecontract.Account) error {
	_ = ctx
	var reply rpcserver.CreateAccountReply
	return c.call("CreateAccount", rpcserver.CreateAccountArgs{Account: account}, &reply)
}
