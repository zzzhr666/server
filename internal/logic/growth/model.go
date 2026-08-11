package growth

// Growth 保存玩家的局外成长等级。
type Growth struct {
	// PlayerID 是成长数据所属的玩家 ID。
	PlayerID int64
	// AttackLevel 是攻击力等级。
	AttackLevel int32
	// AttackSpeedLevel 是攻击速度等级。
	AttackSpeedLevel int32
	// HealthLevel 是生命值等级。
	HealthLevel int32
	// MoveSpeedLevel 是移动速度等级。
	MoveSpeedLevel int32
}

// UpgradeType 标识可升级的成长属性。
type UpgradeType int32

// UpgradeResult 是一次升级完成后返回的成长数据与货币余额。
type UpgradeResult struct {
	Growth         *Growth
	RemainingCoins int64
	Cost           int64
}

// UpgradeOption 描述一项成长属性的下一次可购买升级。
// NextCost 在属性已到 MaxLevel 时为零。
type UpgradeOption struct {
	Type         UpgradeType
	CurrentLevel int32
	NextCost     int64
	MaxLevel     int32
}

const (
	// UpgradeAttack 表示攻击力升级。
	UpgradeAttack UpgradeType = iota
	// UpgradeAttackSpeed 表示攻击速度升级。
	UpgradeAttackSpeed UpgradeType = iota
	// UpgradeHealth 表示生命值升级。
	UpgradeHealth UpgradeType = iota
	// UpgradeMoveSpeed 表示移动速度升级。
	UpgradeMoveSpeed UpgradeType = iota

	UpgradeUnknown = iota
)

// UpgradeRule 定义一种属性的升级价格和等级上限。
type UpgradeRule struct {
	Type     UpgradeType
	BaseCost int64
	CostStep int64
	MaxLevel int32
}

// UpgradePersistInput 是提交给状态层的原子升级请求。
type UpgradePersistInput struct {
	PlayerID    int64
	UpgradeType UpgradeType
	Cost        int64
	MaxLevel    int32
}

// UpgradePersistResult 是状态层完成升级后返回的结果。
type UpgradePersistResult struct {
	Growth         *Growth
	RemainingCoins int64
}

// CostForLevel 返回从当前等级升级到下一等级所需的货币数。
func (r UpgradeRule) CostForLevel(currentLevel int32) (int64, error) {
	if currentLevel < 1 {
		return 0, ErrInvalidGrowthLevel
	}
	return r.BaseCost + r.CostStep*int64(currentLevel-1), nil
}

// LevelOf 返回指定成长属性的当前等级。
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

// NewInitialGrowth 创建所有属性均为一级的初始成长数据。
func NewInitialGrowth(playerID int64) *Growth {
	return &Growth{
		PlayerID:         playerID,
		AttackLevel:      1,
		AttackSpeedLevel: 1,
		HealthLevel:      1,
		MoveSpeedLevel:   1,
	}
}
