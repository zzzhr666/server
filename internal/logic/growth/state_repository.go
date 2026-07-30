package growth

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
)

type StateRepository struct {
	client statecontract.GrowthClient
}

func (s *StateRepository) Upgrade(ctx context.Context, input UpgradePersistInput) (*UpgradePersistResult, error) {
	res, err := s.client.UpgradeGrowth(ctx, statecontract.UpgradeGrowthInput{
		PlayerID:     input.PlayerID,
		UpgradeField: mappingTypeToStr(input.UpgradeType),
		Cost:         input.Cost,
		MaxLevel:     input.MaxLevel,
	})
	if err != nil {
		return nil, mapStateError(err)
	}
	return &UpgradePersistResult{
		Growth:         fromStateGrowth(res.Growth),
		RemainingCoins: res.RemainingCoins,
	}, nil
}

func (s *StateRepository) Get(ctx context.Context, playerID int64) (*Growth, error) {
	growth, err := s.client.GetGrowth(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateGrowth(growth), nil
}

func NewStateRepository(client statecontract.GrowthClient) *StateRepository {
	return &StateRepository{
		client: client,
	}
}

func mapStateError(err error) error {
	switch {
	case errors.Is(err, statecontract.ErrGrowthNotFound):
		return ErrGrowthNotFound
	case errors.Is(err, statecontract.ErrInsufficientCoins):
		return ErrInsufficientCoins
	case errors.Is(err, statecontract.ErrMaxGrowthLevel):
		return ErrMaxLevelReached
	case errors.Is(err, statecontract.ErrInvalidGrowth), errors.Is(err,
		statecontract.ErrInvalidGrowthField):
		return ErrInvalidGrowthLevel
	default:
		return err
	}
}

func fromStateGrowth(growth *statecontract.Growth) *Growth {
	return &Growth{
		PlayerID:         growth.PlayerID,
		AttackLevel:      growth.AttackLevel,
		AttackSpeedLevel: growth.AttackSpeedLevel,
		HealthLevel:      growth.HealthLevel,
		MoveSpeedLevel:   growth.MoveSpeedLevel,
	}
}

func mappingTypeToStr(upgradeType UpgradeType) string {
	switch upgradeType {
	case UpgradeAttackSpeed:
		return "AttackSpeed"
	case UpgradeAttack:
		return "Attack"
	case UpgradeHealth:
		return "Health"
	case UpgradeMoveSpeed:
		return "MoveSpeed"
	default:
		return "unknown"
	}
}
