package growth

import (
	"context"
	"testing"
)

func TestDefaultUpgradeRulesCostCurve(t *testing.T) {
	rules := DefaultUpgradeRules()
	ruleByType := make(map[UpgradeType]UpgradeRule, len(rules))
	for _, rule := range rules {
		ruleByType[rule.Type] = rule
	}

	tests := []struct {
		name         string
		upgradeType  UpgradeType
		currentLevel int32
		wantCost     int64
	}{
		{
			name:         "attack level one",
			upgradeType:  UpgradeAttack,
			currentLevel: 1,
			wantCost:     100,
		},
		{
			name:         "attack level two",
			upgradeType:  UpgradeAttack,
			currentLevel: 2,
			wantCost:     150,
		},
		{
			name:         "attack speed level one",
			upgradeType:  UpgradeAttackSpeed,
			currentLevel: 1,
			wantCost:     120,
		},
		{
			name:         "health level three",
			upgradeType:  UpgradeHealth,
			currentLevel: 3,
			wantCost:     180,
		},
		{
			name:         "move speed level two",
			upgradeType:  UpgradeMoveSpeed,
			currentLevel: 2,
			wantCost:     165,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, ok := ruleByType[tt.upgradeType]
			if !ok {
				t.Fatalf("rule for type %d not found", tt.upgradeType)
			}
			got, err := rule.CostForLevel(tt.currentLevel)
			if err != nil {
				t.Fatalf("CostForLevel returned error: %v", err)
			}
			if got != tt.wantCost {
				t.Fatalf("cost = %d, want %d", got, tt.wantCost)
			}
		})
	}
}

func TestUpgradeUsesConfiguredRule(t *testing.T) {
	repo := &fakeGrowthRepository{
		growth: &Growth{
			PlayerID:         7,
			AttackLevel:      2,
			AttackSpeedLevel: 1,
			HealthLevel:      1,
			MoveSpeedLevel:   1,
		},
		upgradeResult: &UpgradePersistResult{
			Growth: &Growth{
				PlayerID:         7,
				AttackLevel:      3,
				AttackSpeedLevel: 1,
				HealthLevel:      1,
				MoveSpeedLevel:   1,
			},
			RemainingCoins: 850,
		},
	}
	service := NewService(repo, DefaultUpgradeRules())

	result, err := service.Upgrade(context.Background(), 7, UpgradeAttack)
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}
	if repo.upgradeInput.PlayerID != 7 {
		t.Fatalf("repo player id = %d, want 7", repo.upgradeInput.PlayerID)
	}
	if repo.upgradeInput.UpgradeType != UpgradeAttack {
		t.Fatalf("repo upgrade type = %d, want %d", repo.upgradeInput.UpgradeType, UpgradeAttack)
	}
	if repo.upgradeInput.Cost != 150 {
		t.Fatalf("repo cost = %d, want 150", repo.upgradeInput.Cost)
	}
	if repo.upgradeInput.MaxLevel != 10 {
		t.Fatalf("repo max level = %d, want 10", repo.upgradeInput.MaxLevel)
	}
	if result.Cost != 150 {
		t.Fatalf("result cost = %d, want 150", result.Cost)
	}
	if result.RemainingCoins != 850 {
		t.Fatalf("remaining coins = %d, want 850", result.RemainingCoins)
	}
}

func TestUpgradeOptionsReturnsNextCostsAndCaps(t *testing.T) {
	service := NewService(&fakeGrowthRepository{}, DefaultUpgradeRules())

	options, err := service.UpgradeOptions(&Growth{
		PlayerID:         7,
		AttackLevel:      2,
		AttackSpeedLevel: 10,
		HealthLevel:      3,
		MoveSpeedLevel:   1,
	})
	if err != nil {
		t.Fatalf("UpgradeOptions returned error: %v", err)
	}
	if len(options) != 4 {
		t.Fatalf("options length = %d, want 4", len(options))
	}
	if options[0].Type != UpgradeAttack || options[0].CurrentLevel != 2 || options[0].NextCost != 150 || options[0].MaxLevel != 10 {
		t.Fatalf("attack option = %+v, want level=2 next cost=150 max=10", options[0])
	}
	if options[1].Type != UpgradeAttackSpeed || options[1].CurrentLevel != 10 || options[1].NextCost != 0 || options[1].MaxLevel != 10 {
		t.Fatalf("attack speed option = %+v, want maxed option", options[1])
	}
}

type fakeGrowthRepository struct {
	growth        *Growth
	upgradeInput  UpgradePersistInput
	upgradeResult *UpgradePersistResult
}

func (f *fakeGrowthRepository) Get(context.Context, int64) (*Growth, error) {
	return f.growth, nil
}

func (f *fakeGrowthRepository) Upgrade(_ context.Context, input UpgradePersistInput) (*UpgradePersistResult, error) {
	f.upgradeInput = input
	return f.upgradeResult, nil
}
