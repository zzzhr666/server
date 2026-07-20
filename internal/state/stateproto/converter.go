package stateproto

import (
	"time"

	statecontract "server/internal/contract/state"
	"server/internal/contract/statepb"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToProtoAccount(account *statecontract.Account) *statepb.Account {
	if account == nil {
		return nil
	}
	return &statepb.Account{
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		PlayerId:     account.PlayerID,
	}
}

func FromProtoAccount(account *statepb.Account) *statecontract.Account {
	if account == nil {
		return nil
	}
	return &statecontract.Account{
		Username:     account.GetUsername(),
		PasswordHash: account.GetPasswordHash(),
		PlayerID:     account.GetPlayerId(),
	}
}

func FromProtoTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func ToProtoTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func ToProtoPlayer(player *statecontract.Player) *statepb.Player {
	if player == nil {
		return nil
	}
	return &statepb.Player{
		Id:       player.ID,
		Nickname: player.Nickname,
		Avatar:   player.Avatar,
		Email:    player.Email,
		Phone:    player.Phone,
	}
}

func FromProtoPlayer(player *statepb.Player) *statecontract.Player {
	if player == nil {
		return nil
	}
	return &statecontract.Player{
		ID:       player.GetId(),
		Nickname: player.GetNickname(),
		Avatar:   player.GetAvatar(),
		Email:    player.GetEmail(),
		Phone:    player.GetPhone(),
	}
}

func ToProtoSession(session *statecontract.Session) *statepb.Session {
	if session == nil {
		return nil
	}
	return &statepb.Session{
		Token:     session.Token,
		PlayerId:  session.PlayerID,
		ExpiresAt: ToProtoTime(session.ExpiresAt),
	}
}

func FromProtoSession(session *statepb.Session) *statecontract.Session {
	if session == nil {
		return nil
	}
	return &statecontract.Session{
		Token:     session.GetToken(),
		PlayerID:  session.GetPlayerId(),
		ExpiresAt: FromProtoTime(session.GetExpiresAt()),
	}
}

func ToProtoDuration(d time.Duration) *durationpb.Duration {
	if d <= 0 {
		return nil
	}
	return durationpb.New(d)
}

func FromProtoDuration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return d.AsDuration()
}

func ToProtoPresence(presence *statecontract.Presence) *statepb.Presence {
	if presence == nil {
		return nil
	}
	return &statepb.Presence{
		PlayerId:   presence.PlayerID,
		ServerName: presence.ServerName,
		Status:     presence.Status,
		UpdatedAt:  ToProtoTime(presence.UpdatedAt),
	}
}

func FromProtoPresence(presence *statepb.Presence) *statecontract.Presence {
	if presence == nil {
		return nil
	}
	return &statecontract.Presence{
		PlayerID:   presence.GetPlayerId(),
		ServerName: presence.GetServerName(),
		Status:     presence.GetStatus(),
		UpdatedAt:  FromProtoTime(presence.GetUpdatedAt()),
	}
}

func FromProtoFriendRequest(friendRequest *statepb.FriendRequest) *statecontract.FriendRequest {
	if friendRequest == nil {
		return nil
	}
	return &statecontract.FriendRequest{
		FromPlayerID: friendRequest.GetFromPlayer(),
		ToPlayerID:   friendRequest.GetToPlayer(),
		CreatedAt:    FromProtoTime(friendRequest.GetCreatedAt()),
	}
}

func ToProtoFriendRequest(friendRequest *statecontract.FriendRequest) *statepb.FriendRequest {
	if friendRequest == nil {
		return nil
	}
	return &statepb.FriendRequest{
		FromPlayer: friendRequest.FromPlayerID,
		ToPlayer:   friendRequest.ToPlayerID,
		CreatedAt:  ToProtoTime(friendRequest.CreatedAt),
	}
}
