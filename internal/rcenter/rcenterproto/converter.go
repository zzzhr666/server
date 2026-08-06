package rcenterproto

import (
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"time"
)

// FromProtoBattleNode converts generated protobuf data to a domain battle node.
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

// ToProtoBattleNode converts a domain battle node to generated protobuf data.
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

// ToProtoMatchResult converts a domain match result to generated protobuf data.
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
			Weapon:   loadout.Weapon,
		})
	}
	return protoResult
}

// FromProtoMatchResult converts generated protobuf data to a domain match result.
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
			Weapon:   loadout.GetWeapon(),
		})
	}
	return matchResult
}

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

func FromProtoMonsterKillCount(kill *rcenterpb.MonsterKillCount) rcenter.MonsterKillCount {
	return rcenter.MonsterKillCount{
		MonsterKind: kill.GetMonsterKind(),
		Count:       kill.GetCount(),
	}

}

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
