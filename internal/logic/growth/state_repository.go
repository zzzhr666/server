package growth

import (
	"context"
	"errors"
	statecontract "server/internal/contract/state"
)

// StateRepository 将成长领域的读写操作适配到 state-server。
type StateRepository struct {
	client statecontract.GrowthClient
}

// Upgrade 调用 state-server 原子地校验货币、等级上限并持久化升级结果。
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

// Get 从 state-server 读取玩家成长数据。
func (s *StateRepository) Get(ctx context.Context, playerID int64) (*Growth, error) {
	growth, err := s.client.GetGrowth(ctx, playerID)
	if err != nil {
		return nil, mapStateError(err)
	}
	return fromStateGrowth(growth), nil
}

// NewStateRepository 使用 state-server 成长客户端创建仓储。
func NewStateRepository(client statecontract.GrowthClient) *StateRepository {
	return &StateRepository{
		client: client,
	}
}

// mapStateError 将状态层成长错误映射为领域错误。
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

// fromStateGrowth 将状态层成长数据转换为领域模型。
func fromStateGrowth(growth *statecontract.Growth) *Growth {
	return &Growth{
		PlayerID:         growth.PlayerID,
		AttackLevel:      growth.AttackLevel,
		AttackSpeedLevel: growth.AttackSpeedLevel,
		HealthLevel:      growth.HealthLevel,
		MoveSpeedLevel:   growth.MoveSpeedLevel,
	}
}

// mappingTypeToStr 将领域升级类型转换为状态层字段名。
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
