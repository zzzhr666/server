package growth

type Growth struct {
	PlayerID         int64
	AttackLevel      int32
	AttackSpeedLevel int32
	HealthLevel      int32
	MoveSpeedLevel   int32
}

type UpgradeType int32

type UpgradeResult struct {
	Growth         *Growth
	RemainingCoins int64
	Cost           int64
}

// UpgradeOption describes the next purchasable level for one growth attribute.
// NextCost is zero when the attribute has reached MaxLevel.
type UpgradeOption struct {
	Type         UpgradeType
	CurrentLevel int32
	NextCost     int64
	MaxLevel     int32
}

const (
	UpgradeAttack      UpgradeType = iota
	UpgradeAttackSpeed UpgradeType = iota
	UpgradeHealth      UpgradeType = iota
	UpgradeMoveSpeed   UpgradeType = iota
)

type UpgradeRule struct {
	Type     UpgradeType
	BaseCost int64
	CostStep int64
	MaxLevel int32
}
type UpgradePersistInput struct {
	PlayerID    int64
	UpgradeType UpgradeType
	Cost        int64
	MaxLevel    int32
}

type UpgradePersistResult struct {
	Growth         *Growth
	RemainingCoins int64
}

func (r UpgradeRule) CostForLevel(currentLevel int32) (int64, error) {
	if currentLevel < 1 {
		return 0, ErrInvalidGrowthLevel
	}
	return r.BaseCost + r.CostStep*int64(currentLevel-1), nil
}

func (g *Growth) LevelOf(upgradeType UpgradeType) (int32, error) {
	switch upgradeType {
	case UpgradeAttackSpeed:
		return g.AttackSpeedLevel, nil
	case UpgradeAttack:
		return g.AttackLevel, nil
	case UpgradeHealth:
		return g.HealthLevel, nil
	case UpgradeMoveSpeed:
		return g.MoveSpeedLevel, nil
	default:
		return 0, ErrInvalidUpgradeType
	}
}

func NewInitialGrowth(playerID int64) *Growth {
	return &Growth{
		PlayerID:         playerID,
		AttackLevel:      1,
		AttackSpeedLevel: 1,
		HealthLevel:      1,
		MoveSpeedLevel:   1,
	}
}
