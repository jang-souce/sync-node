package utils

import (
	"context"
	"math/rand"
	"time"
)

func Retry(ctx context.Context, attempts int, base time.Duration, max time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		// 记录重试日志
		if Logger != nil {
			Logger.Warnf("Operation failed (attempt %d/%d): %v. Retrying...", i+1, attempts, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		d := base << i
		if d > max {
			d = max
		}
		j := time.Duration(rand.Int63n(int64(d/10) + 1))
		time.Sleep(d + j)
	}
	return err
}
