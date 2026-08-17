package rcenter

import "testing"

func TestCalculateCoinReward(t *testing.T) {
	rule := RewardRule{
		BaseReward:    50,
		VictoryReward: 100,
		MonsterKillReward: map[string]int64{
			"melee":  10,
			"ranged": 15,
			"elite":  35,
		},
	}

	tests := []struct {
		name   string
		stat   PlayerBattleStats
		reason string
		want   int64
	}{
		{
			name: "participation reward only",
			stat: PlayerBattleStats{
				PlayerID: 7,
			},
			reason: "defeat",
			want:   50,
		},
		{
			name: "known monster rewards",
			stat: PlayerBattleStats{
				PlayerID: 7,
				Kills: []MonsterKillCount{
					{MonsterKind: "melee", Count: 3},
					{MonsterKind: "elite", Count: 1},
				},
			},
			reason: "defeat",
			want:   115,
		},
		{
			name: "unknown monster uses default reward",
			stat: PlayerBattleStats{
				PlayerID: 7,
				Kills: []MonsterKillCount{
					{MonsterKind: "unknown", Count: 2},
				},
			},
			reason: "defeat",
			want:   54,
		},
		{
			name: "victory adds reward",
			stat: PlayerBattleStats{
				PlayerID: 7,
				Kills: []MonsterKillCount{
					{MonsterKind: "ranged", Count: 2},
				},
			},
			reason: "victory",
			want:   180,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateCoinReward(tt.stat, tt.reason, rule)
			if err != nil {
				t.Fatalf("CalculateCoinReward returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("reward = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDefaultRewardRuleLimitsFullSoloClearReward(t *testing.T) {
	stat := PlayerBattleStats{
		PlayerID:   7,
		TotalKills: 135,
		Kills: []MonsterKillCount{
			{MonsterKind: "melee", Count: 95},
			{MonsterKind: "ranged", Count: 40},
		},
	}

	got, err := CalculateCoinReward(stat, BattleFinishReasonVictory, DefaultRewardRule())
	if err != nil {
		t.Fatalf("CalculateCoinReward returned error: %v", err)
	}
	if got != 460 {
		t.Fatalf("reward = %d, want 460", got)
	}
}
