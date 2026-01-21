//go:build hard_test

package cron

import (
	"context"
	"math"
	"runtime"
	"runtime/debug"
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
	for range 100 {
		c := New()

		block := make(chan struct{})
		unblock := make(chan struct{})
		once := sync.Once{}
		wg := sync.WaitGroup{}

		gNum := inspectNumGoroutines(t, func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			action := func() {
				once.Do(func() {
					close(block)
				})

				<-unblock
			}
			next := func() time.Duration {
				return 1 * time.Millisecond
			}

			wg.Go(func() {
				c.Run(ctx, action, next)

			})

			<-block
			close(unblock)
			<-ctx.Done()
		})

		wg.Wait()
		require.LessOrEqual(t, gNum, 2)

		runtime.GC()

		// hint: time.AfterFunc создаёт дополнительную горутину, чтобы случайно не заблокировать структуры рантайма
		// при выполнении таймера:

		// func goFunc(arg any, seq uintptr, delta int64) {
		//	go arg.(func())()
		//}

		// Используйте time.NewTimer(d)
	}
}

func inspectNumGoroutines(t *testing.T, f func()) int {
	debug.SetGCPercent(-1)
	defer debug.SetGCPercent(100)

	cur := debug.SetMemoryLimit(-1)
	defer debug.SetMemoryLimit(cur)

	debug.SetMemoryLimit(math.MaxInt)

	t.Helper()

	start := runtime.NumGoroutine()

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
	return int(result.Load()) - start - 2
}
