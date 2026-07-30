package rcenter

type MonsterKillCount struct {
	MonsterKind string
	Count       int32
}

type PlayerBattleStats struct {
	PlayerID   int64
	TotalKills int32
	Kills      []MonsterKillCount
}

type RewardRule struct {
	BaseReward        int64
	VictoryReward     int64
	MonsterKillReward map[string]int64
}

func CalculateCoinReward(stat PlayerBattleStats, reason string, rule RewardRule) (int64, error) {
	if stat.PlayerID <= 0 {
		return 0, ErrInvalidPlayerID
	}
	if stat.TotalKills < 0 {
		return 0, ErrInvalidBattleStats
	}
	if rule.BaseReward < 0 || rule.VictoryReward < 0 {
		return 0, ErrInvalidRewardRules
	}
	finalReward := rule.BaseReward
	if reason == "victory" {
		finalReward += rule.VictoryReward
	}
	for _, killCount := range stat.Kills {
		if killCount.Count < 0 {
			return 0, ErrInvalidBattleStats
		}
		reward, ok := rule.MonsterKillReward[killCount.MonsterKind]
		if !ok {
			reward = 10
		}
		finalReward += reward * int64(killCount.Count)
	}
	return finalReward, nil
}

const (
	BattleFinishReasonVictory string = "victory"
	DefaultMonsterKillReward  int64  = 10
)

func DefaultRewardRule() RewardRule {
	return RewardRule{
		BaseReward:    50,
		VictoryReward: 100,
		MonsterKillReward: map[string]int64{
			"melee": 10,
		},
	}
}
