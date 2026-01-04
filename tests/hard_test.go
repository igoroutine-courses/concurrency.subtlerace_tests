//go:build hard_test

package cron

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStressCronZeroDuration(t *testing.T) {
	for range 100_000 {
		synctest.Test(t, func(t *testing.T) {
			c := New()

			var n atomic.Int64
			next := func() time.Duration {
				if n.Add(1) <= 5 {
					return 0
				}

				return 1 * time.Second
			}

			var calls atomic.Int64
			ctx, cancel := context.WithCancel(t.Context())

			done := make(chan struct{})
			go func() {
				defer close(done)

				c.Run(ctx, func() {
					calls.Add(1)
				}, next)
			}()

			synctest.Wait()

			require.Greater(t, calls.Load(), int64(0))

			cancel()
			synctest.Wait()

			select {
			case <-done:
			default:
				t.Fatalf("Run did not exit after cancel")
			}
		})
	}
}
