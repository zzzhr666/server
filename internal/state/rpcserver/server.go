package rpcserver

import (
	"context"
	statecontract "server/internal/contract/state"
)

const ServiceName = "StateService"

type Server struct {
	state statecontract.Client
}

func NewServer(state statecontract.Client) *Server {
	return &Server{state: state}
}

type GetAccountArgs struct {
	Username string
}

type GetAccountReply struct {
	Account *statecontract.Account
}

func (s *Server) GetAccount(args GetAccountArgs, reply *GetAccountReply) error {
	account, err := s.state.GetAccount(context.Background(), args.Username)
	if err != nil {
		return err
	}
	reply.Account = account
	return nil
}

type CreateAccountArgs struct {
	Account *statecontract.Account
}
type CreateAccountReply struct{}

func (s *Server) CreateAccount(args CreateAccountArgs, reply *CreateAccountReply) error {
	return s.state.CreateAccount(context.Background(), args.Account)
}

type RegisterAccountArgs struct {
	Input statecontract.RegisterAccountInput
}
type RegisterAccountReply struct {
	Result *statecontract.RegisterAccountResult
}

func (s *Server) RegisterAccount(args RegisterAccountArgs, reply *RegisterAccountReply) error {
	res, err := s.state.RegisterAccount(context.Background(), args.Input)
	if err != nil {
		return err
	}
	reply.Result = res
	return nil
}

type CreateSessionArgs struct {
	Session *statecontract.Session
}

type CreateSessionReply struct{}

func (s *Server) CreateSession(args CreateSessionArgs, reply *CreateSessionReply) error {
	return s.state.CreateSession(context.Background(), args.Session)
}

type GetSessionArgs struct {
	Token string
}
type GetSessionReply struct {
	Session *statecontract.Session
}

func (s *Server) GetSession(args GetSessionArgs, reply *GetSessionReply) error {
	session, err := s.state.GetSession(context.Background(), args.Token)
	if err != nil {
		return err
	}
	reply.Session = session
	return nil
}

type DeleteSessionArgs struct {
	Token string
}

type DeleteSessionReply struct{}

func (s *Server) DeleteSession(args DeleteSessionArgs, reply *DeleteSessionReply) error {
	return s.state.DeleteSession(context.Background(), args.Token)
}

type CreatePlayerArgs struct {
	Player *statecontract.Player
}
type CreatePlayerReply struct{}

func (s *Server) CreatePlayer(args CreatePlayerArgs, reply *CreatePlayerReply) error {
	return s.state.CreatePlayer(context.Background(), args.Player)
}

type GetPlayerArgs struct {
	ID int64
}

type GetPlayerReply struct {
	Player *statecontract.Player
}

func (s *Server) GetPlayer(args GetPlayerArgs, reply *GetPlayerReply) error {
	player, err := s.state.GetPlayer(context.Background(), args.ID)
	if err != nil {
		return err
	}
	reply.Player = player
	return nil
}

type NextPlayerIDArgs struct{}
type NextPlayerIDReply struct {
	ID int64
}

func (s *Server) NextPlayerID(args NextPlayerIDArgs, reply *NextPlayerIDReply) error {
	id, err := s.state.NextPlayerID(context.Background())
	if err != nil {
		return err
	}
	reply.ID = id
	return nil
}
