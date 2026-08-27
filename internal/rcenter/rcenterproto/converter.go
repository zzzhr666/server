package rcenterproto

import (
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"time"
)

// FromProtoBattleNode 将生成的 protobuf 数据转换为领域战斗节点。
func FromProtoBattleNode(node *rcenterpb.BattleNode) rcenter.BattleNode {
	return rcenter.BattleNode{
		Name:          node.GetName(),
		UDPAddr:       node.GetUdpAddr(),
		ControlAddr:   node.GetControlAddr(),
		MaxPlayers:    int(node.GetMaxPlayers()),
		ActivePlayers: int(node.GetActivePlayers()),
		LastSeen:      time.Unix(node.GetLastSeenUnix(), 0),
	}
}

// ToProtoBattleNode 将领域战斗节点转换为生成的 protobuf 数据。
func ToProtoBattleNode(node rcenter.BattleNode) *rcenterpb.BattleNode {
	return &rcenterpb.BattleNode{
		Name:          node.Name,
		UdpAddr:       node.UDPAddr,
		ControlAddr:   node.ControlAddr,
		MaxPlayers:    int32(node.MaxPlayers),
		ActivePlayers: int32(node.ActivePlayers),
		LastSeenUnix:  node.LastSeen.Unix(),
	}
}

// ToProtoMatchResult 将领域匹配结果转换为生成的 protobuf 数据。
func ToProtoMatchResult(result *rcenter.MatchResult) *rcenterpb.MatchResult {
	protoResult := &rcenterpb.MatchResult{
		Status:         string(result.Status),
		RoomName:       result.RoomName,
		Token:          result.Token,
		BattleNodeName: result.BattleNodeName,
		BattleUdpAddr:  result.BattleUDPAddr,
		PlayerIds:      result.PlayerIDs,
	}
	protoResult.PlayerLoadouts = make([]*rcenterpb.PlayerLoadout, 0, len(result.PlayerLoadouts))
	for _, loadout := range result.PlayerLoadouts {
		protoResult.PlayerLoadouts = append(protoResult.PlayerLoadouts, &rcenterpb.PlayerLoadout{
			PlayerId: loadout.PlayerID,
			Nickname: loadout.Nickname,
			Hero:     loadout.Hero,
		})
	}
	return protoResult
}

// FromProtoMatchResult 将生成的 protobuf 数据转换为领域匹配结果。
func FromProtoMatchResult(result *rcenterpb.MatchResult) *rcenter.MatchResult {
	matchResult := &rcenter.MatchResult{
		Status:         mapStatus(result.GetStatus()),
		RoomName:       result.GetRoomName(),
		Token:          result.GetToken(),
		BattleNodeName: result.GetBattleNodeName(),
		BattleUDPAddr:  result.GetBattleUdpAddr(),
		PlayerIDs:      result.GetPlayerIds(),
	}
	matchResult.PlayerLoadouts = make([]rcenter.PlayerLoadout, 0, len(result.GetPlayerLoadouts()))
	for _, loadout := range result.GetPlayerLoadouts() {
		matchResult.PlayerLoadouts = append(matchResult.PlayerLoadouts, rcenter.PlayerLoadout{
			PlayerID: loadout.GetPlayerId(),
			Nickname: loadout.GetNickname(),
			Hero:     loadout.GetHero(),
		})
	}
	return matchResult
}

// FromProtoPlayerBattleStats 将 protobuf 玩家战斗统计转换为 rcenter 领域模型。
func FromProtoPlayerBattleStats(stat *rcenterpb.PlayerBattleStats) rcenter.PlayerBattleStats {
	res := rcenter.PlayerBattleStats{
		PlayerID:   stat.GetPlayerId(),
		TotalKills: stat.GetTotalKills(),
	}
	for _, kill := range stat.GetKills() {
		res.Kills = append(res.Kills, FromProtoMonsterKillCount(kill))
	}
	return res
}

// ToProtoPlayerBattleStats 将 rcenter 玩家战斗统计转换为 protobuf 消息。
func ToProtoPlayerBattleStats(stat rcenter.PlayerBattleStats) *rcenterpb.PlayerBattleStats {
	res := &rcenterpb.PlayerBattleStats{
		PlayerId:   stat.PlayerID,
		TotalKills: stat.TotalKills,
	}
	for _, kill := range stat.Kills {
		res.Kills = append(res.Kills, ToProtoMonsterKillCount(kill))
	}
	return res
}

// FromProtoMonsterKillCount 将 protobuf 怪物击杀统计转换为 rcenter 领域模型。
func FromProtoMonsterKillCount(kill *rcenterpb.MonsterKillCount) rcenter.MonsterKillCount {
	return rcenter.MonsterKillCount{
		MonsterKind: kill.GetMonsterKind(),
		Count:       kill.GetCount(),
	}

}

// ToProtoMonsterKillCount 将 rcenter 怪物击杀统计转换为 protobuf 消息。
func ToProtoMonsterKillCount(kill rcenter.MonsterKillCount) *rcenterpb.MonsterKillCount {
	return &rcenterpb.MonsterKillCount{
		MonsterKind: kill.MonsterKind,
		Count:       kill.Count,
	}
}

func mapStatus(statusStr string) rcenter.MatchStatus {
	switch statusStr {
	case string(rcenter.MatchStatusWaiting):
		return rcenter.MatchStatusWaiting
	case string(rcenter.MatchStatusMatched):
		return rcenter.MatchStatusMatched
	default:
		return rcenter.MatchStatusUnexpected
	}
}
