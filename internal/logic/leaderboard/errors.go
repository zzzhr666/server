package leaderboard

import "errors"

// ErrInvalidQuery 表示排行榜类型、地图版本或返回数量不合法。
var ErrInvalidQuery = errors.New("invalid leaderboard query")
