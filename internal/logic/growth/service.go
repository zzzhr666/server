package growth

import (
	"context"
)

type Service interface {
	Upgrade(ctx context.Context, playerID int64, upgradeType UpgradeType) (*UpgradeResult, error)
	Get(ctx context.Context, playerID int64) (*Growth, error)
	UpgradeOptions(growth *Growth) ([]UpgradeOption, error)
}

type Repository interface {
	Get(ctx context.Context, playerID int64) (*Growth, error)
	Upgrade(ctx context.Context, input UpgradePersistInput) (*UpgradePersistResult, error)
}

type GameGrowthService struct {
	repo  Repository
	rules map[UpgradeType]UpgradeRule
}

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

func (g *GameGrowthService) Get(ctx context.Context, playerID int64) (*Growth, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if playerID <= 0 {
		return nil, ErrInvalidPlayerID
	}
	return g.repo.Get(ctx, playerID)

}

// UpgradeOptions returns the current level, next price, and level cap for every growth attribute.
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
