package growth

// DefaultUpgradeRules 返回本地演示环境的局外成长经济规则。
func DefaultUpgradeRules() []UpgradeRule {
	return []UpgradeRule{
		{
			Type:     UpgradeAttack,
			BaseCost: 100,
			CostStep: 50,
			MaxLevel: 10,
		},
		{
			Type:     UpgradeAttackSpeed,
			BaseCost: 120,
			CostStep: 60,
			MaxLevel: 10,
		},
		{
			Type:     UpgradeHealth,
			BaseCost: 90,
			CostStep: 45,
			MaxLevel: 10,
		},
		{
			Type:     UpgradeMoveSpeed,
			BaseCost: 110,
			CostStep: 55,
			MaxLevel: 10,
		},
	}
}
