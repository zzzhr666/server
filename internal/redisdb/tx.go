package redisdb

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// WithTxRetry retries fn when Redis WATCH detects a transaction conflict.
func WithTxRetry(ctx context.Context, maxRetries int, fn func() error) error {
	for i := 0; i < maxRetries; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn()
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return err
	}
	return redis.TxFailedErr
}
