package metrics

import "errors"

// ErrInvalidConfig 表示 Metrics Server 缺少必要配置。
var ErrInvalidConfig = errors.New("invalid configuration")
