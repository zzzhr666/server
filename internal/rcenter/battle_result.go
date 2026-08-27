package rcenter

// MonsterKillCount 记录一种怪物的击杀数量。
type MonsterKillCount struct {
	// MonsterKind 是怪物类型标识。
	MonsterKind string
	// Count 是该类型怪物的击杀次数。
	Count int32
}

// PlayerBattleStats 汇总玩家在一场战斗中的击杀数据。
type PlayerBattleStats struct {
	// PlayerID 是统计数据所属的玩家 ID。
	PlayerID int64
	// TotalKills 是全部怪物击杀总数。
	TotalKills int32
	// Kills 按怪物类型列出击杀明细。
	Kills []MonsterKillCount
}

// RewardRule 定义一局战斗的金币结算规则。
type RewardRule struct {
	// BaseReward 是每局战斗的基础奖励。
	BaseReward int64
	// VictoryReward 是获胜时额外给予的奖励。
	VictoryReward int64
	// MonsterKillReward 定义各怪物类型每次击杀的奖励。
	MonsterKillReward map[string]int64
}

// CalculateCoinReward 根据战斗结果和奖励规则计算应发放的金币。
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
			reward = DefaultMonsterKillReward
			if killCount.MonsterKind == "boss" {
				reward = BossMonsterKillReward
			}
		}
		finalReward += reward * int64(killCount.Count)
	}
	return finalReward, nil
}

const (
	// BattleFinishReasonVictory 表示以胜利结束战斗。
	BattleFinishReasonVictory string = "victory"
	// DefaultMonsterKillReward 是未知怪物类型的默认单次击杀奖励。
	DefaultMonsterKillReward int64 = 2
	// BossMonsterKillReward 是 Boss 每次击杀的金币奖励。
	BossMonsterKillReward int64 = 100
)

// DefaultRewardRule 返回本地默认的战斗金币奖励规则。
func DefaultRewardRule() RewardRule {
	return RewardRule{
		BaseReward:    50,
		VictoryReward: 100,
		MonsterKillReward: map[string]int64{
			"melee":  2,
			"ranged": 3,
			"boss":   BossMonsterKillReward,
		},
	}
}
