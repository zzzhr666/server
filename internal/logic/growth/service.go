package growth

import (
	"context"
)

type Service interface {
	// Upgrade 提升玩家指定成长属性。
	Upgrade(ctx context.Context, playerID int64, upgradeType UpgradeType) (*UpgradeResult, error)
	// Get 返回玩家当前成长数据。
	Get(ctx context.Context, playerID int64) (*Growth, error)
	// UpgradeOptions 返回全部成长属性的当前升级选项。
	UpgradeOptions(growth *Growth) ([]UpgradeOption, error)
}

type Repository interface {
	// Get 从状态层读取玩家成长数据。
	Get(ctx context.Context, playerID int64) (*Growth, error)
	// Upgrade 在状态层原子执行扣费与等级更新。
	Upgrade(ctx context.Context, input UpgradePersistInput) (*UpgradePersistResult, error)
}

type GameGrowthService struct {
	repo  Repository
	rules map[UpgradeType]UpgradeRule
}

// NewService 使用成长仓储与升级规则创建局外成长服务。
func NewService(repo Repository, rules []UpgradeRule) *GameGrowthService {
	ruleMap := make(map[UpgradeType]UpgradeRule, len(rules))

	for _, rule := range rules {
		ruleMap[rule.Type] = rule
	}
	return &GameGrowthService{
		repo:  repo,
		rules: ruleMap,
	}

}

// Upgrade 校验升级规则并原子扣除货币、提升指定成长属性。
func (g *GameGrowthService) Upgrade(ctx context.Context, playerID int64, upgradeType UpgradeType) (*UpgradeResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	rule, ok := g.rules[upgradeType]
	if !ok {
		return nil, ErrInvalidUpgradeType
	}

	growth, err := g.repo.Get(ctx, playerID)
	if err != nil {
		return nil, err
	}
	currentLevel, err := growth.LevelOf(upgradeType)
	if err != nil {
		return nil, err
	}

	if currentLevel >= rule.MaxLevel {
		return nil, ErrMaxLevelReached
	}

	cost, err := rule.CostForLevel(currentLevel)
	if err != nil {
		return nil, err
	}
	upgradeResult, err := g.repo.Upgrade(ctx, UpgradePersistInput{
		PlayerID:    playerID,
		UpgradeType: upgradeType,
		Cost:        cost,
		MaxLevel:    rule.MaxLevel,
	})
	if err != nil {
		return nil, err
	}
	return &UpgradeResult{
		Growth:         upgradeResult.Growth,
		RemainingCoins: upgradeResult.RemainingCoins,
		Cost:           cost,
	}, nil
}

// Get 返回玩家当前的局外成长数据。
func (g *GameGrowthService) Get(ctx context.Context, playerID int64) (*Growth, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	return g.repo.Get(ctx, playerID)

}

// UpgradeOptions 返回每项成长属性的当前等级、下次价格和等级上限。
func (g *GameGrowthService) UpgradeOptions(current *Growth) ([]UpgradeOption, error) {
	if current == nil {
		return nil, ErrInvalidGrowthLevel
	}

	types := []UpgradeType{
		UpgradeAttack,
		UpgradeAttackSpeed,
		UpgradeHealth,
		UpgradeMoveSpeed,
	}
	options := make([]UpgradeOption, 0, len(types))
	for _, upgradeType := range types {
		rule, ok := g.rules[upgradeType]
		if !ok {
			return nil, ErrInvalidUpgradeType
		}
		level, err := current.LevelOf(upgradeType)
		if err != nil {
			return nil, err
		}
		option := UpgradeOption{
			Type:         upgradeType,
			CurrentLevel: level,
			MaxLevel:     rule.MaxLevel,
		}
		if level < rule.MaxLevel {
			option.NextCost, err = rule.CostForLevel(level)
			if err != nil {
				return nil, err
			}
		}
		options = append(options, option)
	}
	return options, nil
}
