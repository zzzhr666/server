package growth

import "errors"

var (
	ErrInvalidPlayerID    = errors.New("invalid playerID")
	ErrInvalidUpgradeType = errors.New("invalid upgrade type")
	ErrGrowthNotFound     = errors.New("growth not found")
	ErrInsufficientCoins  = errors.New("insufficient coins")
	ErrMaxLevelReached    = errors.New("max level reached")
	ErrInvalidGrowthLevel = errors.New("invalid growth level")
)
