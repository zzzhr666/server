package stateproto

import (
	"time"

	"server/internal/contract/state"
	"server/internal/contract/statepb"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToProtoAccount 将账号领域模型转换为 protobuf 消息。
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

// FromProtoAccount 将 protobuf 账号消息转换为领域模型。
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

// FromProtoTime 将 protobuf 时间戳转换为 time.Time。
func FromProtoTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// ToProtoTime 将 time.Time 转换为 protobuf 时间戳。
func ToProtoTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// ToProtoPlayer 将玩家领域模型转换为 protobuf 消息。
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

// FromProtoPlayer 将 protobuf 玩家消息转换为领域模型。
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

// ToProtoSession 将会话领域模型转换为 protobuf 消息。
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

// FromProtoSession 将 protobuf 会话消息转换为领域模型。
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

// ToProtoDuration 将 time.Duration 转换为 protobuf 时长。
func ToProtoDuration(d time.Duration) *durationpb.Duration {
	if d <= 0 {
		return nil
	}
	return durationpb.New(d)
}

// FromProtoDuration 将 protobuf 时长转换为 time.Duration。
func FromProtoDuration(d *durationpb.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return d.AsDuration()
}

// ToProtoPresence 将在线状态领域模型转换为 protobuf 消息。
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

// FromProtoPresence 将 protobuf 在线状态消息转换为领域模型。
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

// FromProtoFriendRequest 将 protobuf 好友申请转换为领域模型。
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

// ToProtoFriendRequest 将好友申请领域模型转换为 protobuf 消息。
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

// ToProtoRealtimeEvent 将实时事件领域模型转换为 protobuf 消息。
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

// FromProtoRealtimeEvent 将 protobuf 实时事件转换为领域模型。
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

// ToProtoRealtimeDelivery 将带路由的实时投递转换为 protobuf 消息。
func ToProtoRealtimeDelivery(delivery *state.RealtimeDelivery) *statepb.RealtimeDelivery {
	if delivery == nil {
		return nil
	}
	return &statepb.RealtimeDelivery{
		Route: ToProtoRealtimeRoute(delivery.Route),
		Event: ToProtoRealtimeEvent(delivery.Event),
	}
}

// FromProtoRealtimeDelivery 将 protobuf 实时投递转换为领域模型。
func FromProtoRealtimeDelivery(delivery *statepb.RealtimeDelivery) *state.RealtimeDelivery {
	if delivery == nil {
		return nil
	}
	return &state.RealtimeDelivery{
		Route: FromProtoRealtimeRoute(delivery.GetRoute()),
		Event: FromProtoRealtimeEvent(delivery.GetEvent()),
	}
}

// ToProtoRealtimeRoute 将实时投递路由转换为 protobuf 消息。
func ToProtoRealtimeRoute(route state.RealtimeRoute) *statepb.RealtimeRoute {
	return &statepb.RealtimeRoute{
		Type:       toProtoRealtimeRouteType(route.Type),
		ServerName: route.ServerName,
	}
}

// FromProtoRealtimeRoute 将 protobuf 实时投递路由转换为领域模型。
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

// ToProtoGrowth 将成长领域模型转换为 protobuf 消息。
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

// FromProtoGrowth 将 protobuf 成长消息转换为领域模型。
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

// ToProtoChatMessage 将聊天消息领域模型转换为持久化协议消息。
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

// FromProtoChatMessage 将持久化协议消息转换为聊天领域模型。
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

// ToProtoChatMessages 批量转换聊天领域消息。
func ToProtoChatMessages(messages []*state.ChatMessage) []*statepb.ChatMessage {
	var result = make([]*statepb.ChatMessage, 0, len(messages))
	for _, message := range messages {
		result = append(result, ToProtoChatMessage(message))
	}
	return result
}

// ToProtoRealtimeChatMessage 将聊天领域消息转换为实时投递消息。
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

// FromProtoRealtimeChatMessage 将实时投递消息转换为聊天领域模型。
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

// ToProtoPlayerLeaderboardRecord 将玩家排行榜结算记录转换为 protobuf 消息。
func ToProtoPlayerLeaderboardRecord(record *state.PlayerLeaderboardRecord) *statepb.PlayerLeaderboardRecord {
	if record == nil {
		return nil
	}
	return &statepb.PlayerLeaderboardRecord{
		PlayerId:   record.PlayerID,
		TotalKills: record.TotalKills,
	}
}

// FromProtoPlayerLeaderboardRecord 将 protobuf 玩家排行榜记录转换为领域模型。
func FromProtoPlayerLeaderboardRecord(record *statepb.PlayerLeaderboardRecord) *state.PlayerLeaderboardRecord {
	if record == nil {
		return nil
	}
	return &state.PlayerLeaderboardRecord{
		PlayerID:   record.GetPlayerId(),
		TotalKills: record.GetTotalKills(),
	}
}

// ToProtoMatchLeaderboardRecord 将对局排行榜结算记录转换为 protobuf 消息。
func ToProtoMatchLeaderboardRecord(record *state.MatchLeaderboardRecord) *statepb.MatchLeaderboardRecord {
	if record == nil {
		return nil
	}
	ret := &statepb.MatchLeaderboardRecord{
		Mode:             record.Mode,
		MapVersion:       record.MapVersion,
		Cleared:          record.Cleared,
		CombatDurationMs: record.CombatDurationMS,
	}
	leaderboardRecords := make([]*statepb.PlayerLeaderboardRecord, 0, len(record.Players))
	for _, p := range record.Players {
		leaderboardRecords = append(leaderboardRecords, ToProtoPlayerLeaderboardRecord(&p))
	}
	ret.Players = leaderboardRecords
	return ret
}

// FromProtoMatchLeaderboardRecord 将 protobuf 对局排行榜记录转换为领域模型。
func FromProtoMatchLeaderboardRecord(record *statepb.MatchLeaderboardRecord) *state.MatchLeaderboardRecord {
	if record == nil {
		return nil
	}
	ret := &state.MatchLeaderboardRecord{
		Mode:             record.GetMode(),
		MapVersion:       record.GetMapVersion(),
		Cleared:          record.GetCleared(),
		CombatDurationMS: record.GetCombatDurationMs(),
	}
	leaderboardRecords := make([]state.PlayerLeaderboardRecord, 0, len(record.Players))
	for _, p := range record.Players {
		if p == nil {
			continue
		}
		leaderboardRecords = append(leaderboardRecords, *FromProtoPlayerLeaderboardRecord(p))
	}
	ret.Players = leaderboardRecords
	return ret
}

// ToProtoLeaderboardType 将领域排行榜类型转换为 protobuf 枚举。
func ToProtoLeaderboardType(leaderboardType state.LeaderboardType) statepb.LeaderboardType {
	switch leaderboardType {
	case state.LeaderboardTypeSoloClearTime:
		return statepb.LeaderboardType_SOLO_CLEAR_TIME
	case state.LeaderboardTypeDuoClearTime:
		return statepb.LeaderboardType_DUO_CLEAR_TIME
	case state.LeaderboardTypeTrioClearTime:
		return statepb.LeaderboardType_TRIO_CLEAR_TIME
	case state.LeaderboardTypeQuadClearTime:
		return statepb.LeaderboardType_QUAD_CLEAR_TIME
	case state.LeaderboardTypeTotalKills:
		return statepb.LeaderboardType_TOTAL_KILLS
	default:
		return statepb.LeaderboardType_UNSPECIFIED
	}
}

// FromProtoLeaderboardType 将 protobuf 排行榜枚举转换为领域类型。
func FromProtoLeaderboardType(leaderboardType statepb.LeaderboardType) state.LeaderboardType {
	switch leaderboardType {
	case statepb.LeaderboardType_SOLO_CLEAR_TIME:
		return state.LeaderboardTypeSoloClearTime
	case statepb.LeaderboardType_DUO_CLEAR_TIME:
		return state.LeaderboardTypeDuoClearTime
	case statepb.LeaderboardType_TRIO_CLEAR_TIME:
		return state.LeaderboardTypeTrioClearTime
	case statepb.LeaderboardType_QUAD_CLEAR_TIME:
		return state.LeaderboardTypeQuadClearTime
	case statepb.LeaderboardType_TOTAL_KILLS:
		return state.LeaderboardTypeTotalKills
	default:
		return ""
	}
}

// ToProtoLeaderboardResult 将领域排行榜结果转换为 protobuf 响应。
func ToProtoLeaderboardResult(result *state.ListLeaderboardResult) *statepb.ListLeaderboardResponse {
	if result == nil {
		return nil
	}
	entries := make([]*statepb.LeaderboardEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		players := make([]*statepb.LeaderboardPlayer, 0, len(entry.Players))
		for _, player := range entry.Players {
			players = append(players, &statepb.LeaderboardPlayer{
				PlayerId: player.PlayerID,
				Nickname: player.Nickname,
				Avatar:   player.Avatar,
			})
		}
		entries = append(entries, &statepb.LeaderboardEntry{
			Rank:    entry.Rank,
			Players: players,
			Score:   entry.Score,
		})
	}
	return &statepb.ListLeaderboardResponse{
		Type:       ToProtoLeaderboardType(result.Type),
		MapVersion: result.MapVersion,
		Entries:    entries,
	}

}

// FromProtoLeaderboardResult 将 protobuf 排行榜响应转换为领域结果。
func FromProtoLeaderboardResult(response *statepb.ListLeaderboardResponse) *state.ListLeaderboardResult {
	if response == nil {
		return nil
	}
	entries := make([]state.LeaderboardEntry, 0, len(response.Entries))
	for _, entry := range response.Entries {
		if entry == nil {
			continue
		}
		players := make([]state.LeaderboardPlayer, 0, len(entry.GetPlayers()))
		for _, player := range entry.GetPlayers() {
			if player == nil {
				continue
			}
			players = append(players, state.LeaderboardPlayer{
				PlayerID: player.GetPlayerId(),
				Nickname: player.GetNickname(),
				Avatar:   player.GetAvatar(),
			})
		}
		entries = append(entries, state.LeaderboardEntry{
			Rank:    entry.GetRank(),
			Players: players,
			Score:   entry.GetScore(),
		})
	}
	return &state.ListLeaderboardResult{
		Type:       FromProtoLeaderboardType(response.GetType()),
		MapVersion: response.GetMapVersion(),
		Entries:    entries,
	}
}
