package stateproto

import (
	"time"

	"server/internal/contract/state"
	"server/internal/contract/statepb"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToProtoAccount(account *state.Account) *statepb.Account {
	if account == nil {
		return nil
	}
	return &statepb.Account{
		Username:     account.Username,
		PasswordHash: account.PasswordHash,
		PlayerId:     account.PlayerID,
	}
}

func FromProtoAccount(account *statepb.Account) *state.Account {
	if account == nil {
		return nil
	}
	return &state.Account{
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

func ToProtoPlayer(player *state.Player) *statepb.Player {
	if player == nil {
		return nil
	}
	return &statepb.Player{
		Id:       player.ID,
		Nickname: player.Nickname,
		Avatar:   player.Avatar,
		Email:    player.Email,
		Phone:    player.Phone,
		Coins:    player.Coins,
	}
}

func FromProtoPlayer(player *statepb.Player) *state.Player {
	if player == nil {
		return nil
	}
	return &state.Player{
		ID:       player.GetId(),
		Nickname: player.GetNickname(),
		Avatar:   player.GetAvatar(),
		Email:    player.GetEmail(),
		Phone:    player.GetPhone(),
		Coins:    player.GetCoins(),
	}
}

func ToProtoSession(session *state.Session) *statepb.Session {
	if session == nil {
		return nil
	}
	return &statepb.Session{
		Token:     session.Token,
		PlayerId:  session.PlayerID,
		ExpiresAt: ToProtoTime(session.ExpiresAt),
	}
}

func FromProtoSession(session *statepb.Session) *state.Session {
	if session == nil {
		return nil
	}
	return &state.Session{
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

func ToProtoPresence(presence *state.Presence) *statepb.Presence {
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

func FromProtoPresence(presence *statepb.Presence) *state.Presence {
	if presence == nil {
		return nil
	}
	return &state.Presence{
		PlayerID:   presence.GetPlayerId(),
		ServerName: presence.GetServerName(),
		Status:     presence.GetStatus(),
		UpdatedAt:  FromProtoTime(presence.GetUpdatedAt()),
	}
}

func FromProtoFriendRequest(friendRequest *statepb.FriendRequest) *state.FriendRequest {
	if friendRequest == nil {
		return nil
	}
	return &state.FriendRequest{
		FromPlayerID: friendRequest.GetFromPlayer(),
		ToPlayerID:   friendRequest.GetToPlayer(),
		CreatedAt:    FromProtoTime(friendRequest.GetCreatedAt()),
	}
}

func ToProtoFriendRequest(friendRequest *state.FriendRequest) *statepb.FriendRequest {
	if friendRequest == nil {
		return nil
	}
	return &statepb.FriendRequest{
		FromPlayer: friendRequest.FromPlayerID,
		ToPlayer:   friendRequest.ToPlayerID,
		CreatedAt:  ToProtoTime(friendRequest.CreatedAt),
	}
}

func ToProtoRealtimeEvent(event *state.RealtimeEvent) *statepb.RealtimeEvent {
	if event == nil {
		return nil
	}
	return &statepb.RealtimeEvent{
		Type:           event.Type,
		TargetPlayerId: event.TargetPlayerID,
		ActorPlayerId:  event.ActorPlayerID,
		Online:         event.Online,
		Status:         event.Status,
		MatchStatus:    event.MatchStatus,
		RoomName:       event.RoomName,
		MatchToken:     event.MatchToken,
		BattleUdpAddr:  event.BattleUDPAddr,
		BattleNodeName: event.BattleNodeName,
		MatchPlayerIds: event.MatchPlayerIDs,
		ChatMessage:    ToProtoRealtimeChatMessage(event.ChatMessage),
	}
}

func FromProtoRealtimeEvent(event *statepb.RealtimeEvent) *state.RealtimeEvent {
	if event == nil {
		return nil
	}
	return &state.RealtimeEvent{
		Type:           event.GetType(),
		TargetPlayerID: event.GetTargetPlayerId(),
		ActorPlayerID:  event.GetActorPlayerId(),
		Online:         event.GetOnline(),
		Status:         event.GetStatus(),
		MatchStatus:    event.GetMatchStatus(),
		RoomName:       event.GetRoomName(),
		MatchToken:     event.GetMatchToken(),
		BattleUDPAddr:  event.GetBattleUdpAddr(),
		BattleNodeName: event.GetBattleNodeName(),
		MatchPlayerIDs: event.GetMatchPlayerIds(),
		ChatMessage:    FromProtoRealtimeChatMessage(event.GetChatMessage()),
	}
}

func ToProtoRealtimeDelivery(delivery *state.RealtimeDelivery) *statepb.RealtimeDelivery {
	if delivery == nil {
		return nil
	}
	return &statepb.RealtimeDelivery{
		Route: ToProtoRealtimeRoute(delivery.Route),
		Event: ToProtoRealtimeEvent(delivery.Event),
	}
}

func FromProtoRealtimeDelivery(delivery *statepb.RealtimeDelivery) *state.RealtimeDelivery {
	if delivery == nil {
		return nil
	}
	return &state.RealtimeDelivery{
		Route: FromProtoRealtimeRoute(delivery.GetRoute()),
		Event: FromProtoRealtimeEvent(delivery.GetEvent()),
	}
}

func ToProtoRealtimeRoute(route state.RealtimeRoute) *statepb.RealtimeRoute {
	return &statepb.RealtimeRoute{
		Type:       toProtoRealtimeRouteType(route.Type),
		ServerName: route.ServerName,
	}
}

func FromProtoRealtimeRoute(route *statepb.RealtimeRoute) state.RealtimeRoute {
	if route == nil {
		return state.RealtimeRoute{}
	}
	return state.RealtimeRoute{
		Type:       fromProtoRealtimeRouteType(route.GetType()),
		ServerName: route.GetServerName(),
	}
}

func toProtoRealtimeRouteType(routeType state.RealtimeRouteType) statepb.RealtimeRouteType {
	switch routeType {
	case state.RealtimeRouteServer:
		return statepb.RealtimeRouteType_REALTIME_ROUTE_TYPE_SERVER
	case state.RealtimeRouteBroadcast:
		return statepb.RealtimeRouteType_REALTIME_ROUTE_TYPE_BROADCAST
	default:
		return statepb.RealtimeRouteType_REALTIME_ROUTE_TYPE_UNSPECIFIED
	}
}

// ToProtoRealtimeRouteType 将内部实时路由类型转换为 protobuf 枚举。
func ToProtoRealtimeRouteType(routeType state.RealtimeRouteType) statepb.RealtimeRouteType {
	return toProtoRealtimeRouteType(routeType)
}

func fromProtoRealtimeRouteType(routeType statepb.RealtimeRouteType) state.RealtimeRouteType {
	switch routeType {
	case statepb.RealtimeRouteType_REALTIME_ROUTE_TYPE_SERVER:
		return state.RealtimeRouteServer
	case statepb.RealtimeRouteType_REALTIME_ROUTE_TYPE_BROADCAST:
		return state.RealtimeRouteBroadcast
	default:
		return ""
	}
}

// FromProtoRealtimeRouteType 将 protobuf 实时路由枚举转换为内部类型。
func FromProtoRealtimeRouteType(routeType statepb.RealtimeRouteType) state.RealtimeRouteType {
	return fromProtoRealtimeRouteType(routeType)
}

func ToProtoGrowth(growth *state.Growth) *statepb.Growth {
	if growth == nil {
		return nil
	}
	return &statepb.Growth{
		PlayerId:         growth.PlayerID,
		AttackLevel:      growth.AttackLevel,
		AttackSpeedLevel: growth.AttackSpeedLevel,
		HealthLevel:      growth.HealthLevel,
		MoveSpeedLevel:   growth.MoveSpeedLevel,
	}
}

func FromProtoGrowth(growth *statepb.Growth) *state.Growth {
	if growth == nil {
		return nil
	}
	return &state.Growth{
		PlayerID:         growth.GetPlayerId(),
		AttackLevel:      growth.GetAttackLevel(),
		AttackSpeedLevel: growth.AttackSpeedLevel,
		HealthLevel:      growth.GetHealthLevel(),
		MoveSpeedLevel:   growth.GetMoveSpeedLevel(),
	}
}

func ToProtoChatMessage(message *state.ChatMessage) *statepb.ChatMessage {
	if message == nil {
		return nil
	}
	return &statepb.ChatMessage{
		MessageKey:       message.MessageKey,
		ChannelType:      string(message.ChannelType),
		ChannelKey:       message.ChannelKey,
		SenderId:         message.SenderID,
		ReceiverId:       message.ReceiverID,
		Content:          message.Content,
		CreatedAt:        ToProtoTime(message.CreatedAt),
		ExpiresAt:        ToProtoTime(message.ExpiresAt),
		ClientMessageKey: message.ClientMessageKey,
		SenderNickname:   message.SenderNickname,
	}
}

func FromProtoChatMessage(message *statepb.ChatMessage) *state.ChatMessage {
	if message == nil {
		return nil
	}
	return &state.ChatMessage{
		MessageKey:       message.GetMessageKey(),
		ChannelType:      state.ChatChannelType(message.GetChannelType()),
		ChannelKey:       message.GetChannelKey(),
		SenderID:         message.GetSenderId(),
		ReceiverID:       message.GetReceiverId(),
		Content:          message.GetContent(),
		CreatedAt:        FromProtoTime(message.GetCreatedAt()),
		ExpiresAt:        FromProtoTime(message.GetExpiresAt()),
		ClientMessageKey: message.GetClientMessageKey(),
		SenderNickname:   message.GetSenderNickname(),
	}
}

func ToProtoChatMessages(messages []*state.ChatMessage) []*statepb.ChatMessage {
	var result = make([]*statepb.ChatMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, ToProtoChatMessage(message))
	}
	return result
}

func ToProtoRealtimeChatMessage(message *state.ChatMessage) *statepb.RealtimeChatMessage {
	if message == nil {
		return nil
	}
	return &statepb.RealtimeChatMessage{
		MessageKey:       message.MessageKey,
		ChannelType:      string(message.ChannelType),
		ChannelKey:       message.ChannelKey,
		SenderId:         message.SenderID,
		ReceiverId:       message.ReceiverID,
		Content:          message.Content,
		CreatedAt:        ToProtoTime(message.CreatedAt),
		ExpiresAt:        ToProtoTime(message.ExpiresAt),
		ClientMessageKey: message.ClientMessageKey,
		SenderNickname:   message.SenderNickname,
	}
}

func FromProtoRealtimeChatMessage(message *statepb.RealtimeChatMessage) *state.ChatMessage {
	if message == nil {
		return nil
	}
	return &state.ChatMessage{
		MessageKey:       message.GetMessageKey(),
		ChannelType:      state.ChatChannelType(message.GetChannelType()),
		ChannelKey:       message.GetChannelKey(),
		SenderID:         message.GetSenderId(),
		ReceiverID:       message.GetReceiverId(),
		Content:          message.GetContent(),
		CreatedAt:        FromProtoTime(message.GetCreatedAt()),
		ExpiresAt:        FromProtoTime(message.GetExpiresAt()),
		ClientMessageKey: message.GetClientMessageKey(),
		SenderNickname:   message.GetSenderNickname(),
	}
}

// ToProtoSettleMatchReward 将一条领域奖励转换为 protobuf 奖励。
func ToProtoSettleMatchReward(reward *state.PlayerCoinReward) *statepb.PlayerCoinReward {
	if reward == nil {
		return nil
	}
	return &statepb.PlayerCoinReward{
		PlayerId: reward.PlayerID,
		Amount:   reward.Amount,
	}
}

// ToProtoSettleMatchRewards 将领域奖励列表转换为 protobuf 奖励列表。
func ToProtoSettleMatchRewards(rewards []state.PlayerCoinReward) []*statepb.PlayerCoinReward {
	if rewards == nil {
		return nil
	}
	result := make([]*statepb.PlayerCoinReward, 0, len(rewards))
	for i := range rewards {
		result = append(result, ToProtoSettleMatchReward(&rewards[i]))
	}
	return result
}

// FromProtoSettleMatchRewards 将 protobuf 奖励列表转换为领域奖励列表。
func FromProtoSettleMatchRewards(rewards []*statepb.PlayerCoinReward) []state.PlayerCoinReward {
	if rewards == nil {
		return nil
	}
	result := make([]state.PlayerCoinReward, 0, len(rewards))
	for _, reward := range rewards {
		if reward == nil {
			continue
		}
		result = append(result, state.PlayerCoinReward{
			PlayerID: reward.GetPlayerId(),
			Amount:   reward.GetAmount(),
		})
	}
	return result
}
