//go:build hard_test

package cron

import (
	"context"
	"runtime"
	"sync"
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

func TestNumGoroutines(t *testing.T) {
	c := New()

	gNum := inspectNumGoroutines(t, func() {
		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond*2)
		defer cancel()

		c.Run(ctx,
			func() {},
			func() time.Duration {
				return time.Millisecond
			})
	})

	require.Zero(t, gNum)
}

func inspectNumGoroutines(t *testing.T, f func()) int {
	t.Helper()

	wg := new(sync.WaitGroup)

	result := atomic.Int64{}
	result.Store(int64(runtime.NumGoroutine()))

	done := atomic.Bool{}
	wg.Go(func() {
		f()
		done.Store(true)
	})

	wg.Go(func() {
		for !done.Load() {
			result.Store(max(result.Load(), int64(runtime.NumGoroutine())))
		}
	})

	wg.Wait()
	return max(0, int(result.Load())-2-3)
}
