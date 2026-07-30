package rcenterproto

import (
	"reflect"
	"server/internal/contract/rcenterpb"
	"server/internal/rcenter"
	"testing"
)

func TestPlayerBattleStatsConversion(t *testing.T) {
	domainStats := rcenter.PlayerBattleStats{
		PlayerID:   7,
		TotalKills: 3,
		Kills: []rcenter.MonsterKillCount{
			{MonsterKind: "melee", Count: 2},
			{MonsterKind: "elite", Count: 1},
		},
	}

	protoStats := ToProtoPlayerBattleStats(domainStats)
	if protoStats.GetPlayerId() != 7 {
		t.Fatalf("proto player id = %d, want 7", protoStats.GetPlayerId())
	}
	if protoStats.GetTotalKills() != 3 {
		t.Fatalf("proto total kills = %d, want 3", protoStats.GetTotalKills())
	}
	if len(protoStats.GetKills()) != 2 {
		t.Fatalf("proto kills = %d, want 2", len(protoStats.GetKills()))
	}
	if protoStats.GetKills()[0].GetMonsterKind() != "melee" || protoStats.GetKills()[0].GetCount() != 2 {
		t.Fatalf("proto first kill = %+v, want melee x2", protoStats.GetKills()[0])
	}

	roundTrip := FromProtoPlayerBattleStats(protoStats)
	if !reflect.DeepEqual(roundTrip, domainStats) {
		t.Fatalf("round trip stats = %+v, want %+v", roundTrip, domainStats)
	}
}

func TestFromProtoPlayerBattleStatsNilSafe(t *testing.T) {
	got := FromProtoPlayerBattleStats(nil)
	if !reflect.DeepEqual(got, rcenter.PlayerBattleStats{}) {
		t.Fatalf("nil proto stats = %+v, want zero value", got)
	}
}

func TestMonsterKillCountConversion(t *testing.T) {
	domainKill := rcenter.MonsterKillCount{
		MonsterKind: "ranged",
		Count:       4,
	}

	protoKill := ToProtoMonsterKillCount(domainKill)
	if protoKill.GetMonsterKind() != "ranged" {
		t.Fatalf("proto monster kind = %q, want ranged", protoKill.GetMonsterKind())
	}
	if protoKill.GetCount() != 4 {
		t.Fatalf("proto count = %d, want 4", protoKill.GetCount())
	}

	roundTrip := FromProtoMonsterKillCount(protoKill)
	if roundTrip != domainKill {
		t.Fatalf("round trip kill = %+v, want %+v", roundTrip, domainKill)
	}
}

func TestFromProtoMonsterKillCountNilSafe(t *testing.T) {
	got := FromProtoMonsterKillCount((*rcenterpb.MonsterKillCount)(nil))
	if got != (rcenter.MonsterKillCount{}) {
		t.Fatalf("nil proto kill = %+v, want zero value", got)
	}
}
