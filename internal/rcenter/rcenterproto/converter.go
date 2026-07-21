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
		KCPAddr:       node.GetKcpAddr(),
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
		KcpAddr:       node.KCPAddr,
		ControlAddr:   node.ControlAddr,
		MaxPlayers:    int32(node.MaxPlayers),
		ActivePlayers: int32(node.ActivePlayers),
		LastSeenUnix:  node.LastSeen.Unix(),
	}
}

// ToProtoMatchResult converts a domain match result to generated protobuf data.
func ToProtoMatchResult(result *rcenter.MatchResult) *rcenterpb.MatchResult {
	return &rcenterpb.MatchResult{
		Status:         string(result.Status),
		RoomName:       result.RoomName,
		Token:          result.Token,
		BattleNodeName: result.BattleNodeName,
		BattleKcpAddr:  result.BattleKCPAddr,
		PlayerIds:      result.PlayerIDs,
	}
}

// FromProtoMatchResult converts generated protobuf data to a domain match result.
func FromProtoMatchResult(result *rcenterpb.MatchResult) *rcenter.MatchResult {
	return &rcenter.MatchResult{
		Status:         mapStatus(result.GetStatus()),
		RoomName:       result.GetRoomName(),
		Token:          result.GetToken(),
		BattleNodeName: result.GetBattleNodeName(),
		BattleKCPAddr:  result.GetBattleKcpAddr(),
		PlayerIDs:      result.GetPlayerIds(),
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
