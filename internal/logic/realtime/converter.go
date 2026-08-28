package realtime

import (
	"server/internal/contract/realtimepb"
	"server/internal/contract/state"
	"server/internal/logic/chat"
	"server/internal/logic/friend"
	"server/internal/logic/growth"
	"server/internal/logic/leaderboard"
	"server/internal/logic/player"
	"server/internal/logic/presence"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// toProtoPlayer 将玩家领域模型转换为实时协议模型。
func toProtoPlayer(value *player.Player) *realtimepb.Player {
	if value == nil {
		return nil
	}
	return &realtimepb.Player{
		Id:       value.ID,
		Nickname: value.Nickname,
		Avatar:   value.Avatar,
		Email:    value.Email,
		Phone:    value.Phone,
		Coins:    value.Coins,
	}
}

// toProtoGrowth 将成长领域模型及其可升级选项转换为实时协议模型。
func toProtoGrowth(value *growth.Growth, options []growth.UpgradeOption) *realtimepb.Growth {
	if value == nil {
		return nil
	}
	protoGrowth := &realtimepb.Growth{
		PlayerId:         value.PlayerID,
		AttackLevel:      value.AttackLevel,
		AttackSpeedLevel: value.AttackSpeedLevel,
		HealthLevel:      value.HealthLevel,
		MoveSpeedLevel:   value.MoveSpeedLevel,
		UpgradeOptions:   make([]*realtimepb.GrowthUpgradeOption, 0, len(options)),
	}
	for _, option := range options {
		protoGrowth.UpgradeOptions = append(protoGrowth.UpgradeOptions, toProtoGrowthUpgradeOption(option))
	}
	return protoGrowth
}

// toProtoGrowthUpgradeOption 将成长升级选项转换为实时协议模型。
func toProtoGrowthUpgradeOption(value growth.UpgradeOption) *realtimepb.GrowthUpgradeOption {
	return &realtimepb.GrowthUpgradeOption{
		Type:         fromUpgradeTypeName(value.Type),
		CurrentLevel: value.CurrentLevel,
		NextCost:     value.NextCost,
		MaxLevel:     value.MaxLevel,
	}
}

// toProtoUpgradeGrowthResponse 将成长升级结果转换为实时协议响应。
func toProtoUpgradeGrowthResponse(value *growth.UpgradeResult, options []growth.UpgradeOption) *realtimepb.UpgradeGrowthResponse {
	if value == nil {
		return nil
	}
	return &realtimepb.UpgradeGrowthResponse{
		Growth:         toProtoGrowth(value.Growth, options),
		RemainingCoins: value.RemainingCoins,
		Cost:           value.Cost,
	}
}

// toProtoFriendRequest 将好友申请领域模型转换为实时协议模型。
func toProtoFriendRequest(value *friend.Request) *realtimepb.FriendRequest {
	if value == nil {
		return nil
	}
	return &realtimepb.FriendRequest{
		FromPlayerId: value.FromPlayerID,
		ToPlayerId:   value.ToPlayerID,
		CreatedAt:    timestamppb.New(value.CreatedAt),
	}
}

func toProtoFriendRequestWithNicknames(value *friend.Request, fromNickname, toNickname string) *realtimepb.FriendRequest {
	request := toProtoFriendRequest(value)
	if request != nil {
		request.FromNickname = fromNickname
		request.ToNickname = toNickname
	}
	return request
}

// toProtoFriendRequests 将好友申请列表转换为实时协议模型列表。
func toProtoFriendRequests(values []*friend.Request) []*realtimepb.FriendRequest {
	requests := make([]*realtimepb.FriendRequest, 0, len(values))
	for _, value := range values {
		if request := toProtoFriendRequest(value); request != nil {
			requests = append(requests, request)
		}
	}
	return requests
}

// toProtoFriendSummary 将好友玩家档案与在线状态转换为实时协议模型。
func toProtoFriendSummary(value *player.Player, valuePresence *presence.Presence) *realtimepb.FriendSummary {
	if value == nil {
		return nil
	}
	summary := &realtimepb.FriendSummary{
		PlayerId: value.ID,
		Nickname: value.Nickname,
		Avatar:   value.Avatar,
		Status:   presence.StatusOffline,
	}
	if valuePresence != nil {
		summary.Online = valuePresence.Status == presence.StatusOnline
		summary.Status = valuePresence.Status
		summary.UpdatedAt = timestamppb.New(valuePresence.UpdatedAt)
	}
	return summary
}

func fromUpgradeTypeName(upgradeType growth.UpgradeType) string {
	switch upgradeType {
	case growth.UpgradeAttack:
		return "attack"
	case growth.UpgradeAttackSpeed:
		return "attack_speed"
	case growth.UpgradeHealth:
		return "health"
	case growth.UpgradeMoveSpeed:
		return "move_speed"
	default:
		return "unknown"
	}
}

func toUpgradeTypeName(upgradeType string) growth.UpgradeType {
	switch upgradeType {
	case "attack":
		return growth.UpgradeAttack
	case "attack_speed":
		return growth.UpgradeAttackSpeed
	case "health":
		return growth.UpgradeHealth
	case "move_speed":
		return growth.UpgradeMoveSpeed
	}
	return growth.UpgradeUnknown
}

func toProtoMessage(value *chat.Message) *realtimepb.ChatMessage {
	if value == nil {
		return nil
	}
	return &realtimepb.ChatMessage{
		MessageKey:       value.MessageKey,
		ChannelType:      string(value.ChannelType),
		ChannelKey:       value.ChannelKey,
		SenderId:         value.SenderID,
		ReceiverId:       value.ReceiverID,
		Content:          value.Content,
		CreatedAt:        timestamppb.New(value.CreatedAt),
		ExpiresAt:        timestamppb.New(value.ExpiresAt),
		ClientMessageKey: value.ClientMessageKey,
		SenderNickname:   value.SenderNickname,
	}
}

func toProtoMessages(values []*chat.Message) []*realtimepb.ChatMessage {
	messages := make([]*realtimepb.ChatMessage, 0, len(values))
	for _, value := range values {
		if message := toProtoMessage(value); message != nil {
			messages = append(messages, message)
		}
	}
	return messages
}

func toStateChatMessage(message *chat.Message) *state.ChatMessage {
	if message == nil {
		return nil
	}
	return &state.ChatMessage{
		MessageKey:       message.MessageKey,
		ChannelType:      state.ChatChannelType(message.ChannelType),
		ChannelKey:       message.ChannelKey,
		SenderID:         message.SenderID,
		ReceiverID:       message.ReceiverID,
		Content:          message.Content,
		CreatedAt:        message.CreatedAt,
		ExpiresAt:        message.ExpiresAt,
		ClientMessageKey: message.ClientMessageKey,
		SenderNickname:   message.SenderNickname,
	}
}

func toLeaderboardType(value realtimepb.LeaderboardType) leaderboard.Type {
	switch value {
	case realtimepb.LeaderboardType_LEADERBOARD_TYPE_SOLO_CLEAR_TIME:
		return leaderboard.TypeSoloClearTime
	case realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME:
		return leaderboard.TypeDuoClearTime
	case realtimepb.LeaderboardType_LEADERBOARD_TYPE_TRIO_CLEAR_TIME:
		return leaderboard.TypeTrioClearTime
	case realtimepb.LeaderboardType_LEADERBOARD_TYPE_QUAD_CLEAR_TIME:
		return leaderboard.TypeQuadClearTime
	case realtimepb.LeaderboardType_LEADERBOARD_TYPE_TOTAL_KILLS:
		return leaderboard.TypeTotalKills
	default:
		return ""
	}
}

func toProtoLeaderboardType(value leaderboard.Type) realtimepb.LeaderboardType {
	switch value {
	case leaderboard.TypeSoloClearTime:
		return realtimepb.LeaderboardType_LEADERBOARD_TYPE_SOLO_CLEAR_TIME
	case leaderboard.TypeDuoClearTime:
		return realtimepb.LeaderboardType_LEADERBOARD_TYPE_DUO_CLEAR_TIME
	case leaderboard.TypeTrioClearTime:
		return realtimepb.LeaderboardType_LEADERBOARD_TYPE_TRIO_CLEAR_TIME
	case leaderboard.TypeQuadClearTime:
		return realtimepb.LeaderboardType_LEADERBOARD_TYPE_QUAD_CLEAR_TIME
	case leaderboard.TypeTotalKills:
		return realtimepb.LeaderboardType_LEADERBOARD_TYPE_TOTAL_KILLS
	default:
		return realtimepb.LeaderboardType_LEADERBOARD_TYPE_UNSPECIFIED
	}
}

func toLeaderboardListInput(request *realtimepb.ListLeaderboardRequest) leaderboard.ListInput {
	if request == nil {
		return leaderboard.ListInput{}
	}
	return leaderboard.ListInput{
		Type:       toLeaderboardType(request.GetType()),
		MapVersion: request.GetMapVersion(),
		Limit:      request.GetLimit(),
	}
}

func toProtoLeaderboardResult(result *leaderboard.Result) *realtimepb.LeaderboardResponse {
	if result == nil {
		return nil
	}
	entries := make([]*realtimepb.LeaderboardEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		players := make([]*realtimepb.LeaderboardPlayer, 0, len(entry.Players))
		for _, curPlayer := range entry.Players {
			players = append(players, &realtimepb.LeaderboardPlayer{
				PlayerId: curPlayer.PlayerID,
				Nickname: curPlayer.Nickname,
				Avatar:   curPlayer.Avatar,
			})
		}
		entries = append(entries, &realtimepb.LeaderboardEntry{
			Rank:    entry.Rank,
			Players: players,
			Score:   entry.Score,
		})
	}
	return &realtimepb.LeaderboardResponse{
		Type:       toProtoLeaderboardType(result.Type),
		MapVersion: result.MapVersion,
		Entries:    entries,
	}
}
