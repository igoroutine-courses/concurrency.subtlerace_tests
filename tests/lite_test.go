//go:build lite_test

package cron

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestCronFiresDeterministically(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New()

		next := func() time.Duration {
			return 1 * time.Second
		}

		var calls atomic.Int64
		ctx, cancel := context.WithCancel(t.Context())

		t.Cleanup(func() {
			cancel()
		})

		done := make(chan struct{})
		go func() {
			defer close(done)

			c.Run(ctx, func() {
				calls.Add(1)
			}, next)
		}()

		const iters = 1_000_000

		time.Sleep(iters*time.Second + time.Nanosecond)
		synctest.Wait()

		require.EqualValues(t, iters, calls.Load())

		cancel()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatalf("Run did not exit after cancel")
		}
	})
}

func TestCronStopsAfterCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New()
		next := func() time.Duration {
			return 1 * time.Second
		}

		var calls atomic.Int64
		ctx, cancel := context.WithCancel(t.Context())

		t.Cleanup(func() {
			cancel()
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Run(ctx, func() {
				calls.Add(1)
			}, next)
		}()

		time.Sleep(2*time.Second + time.Nanosecond)
		synctest.Wait()

		require.EqualValues(t, 2, calls.Load())

		cancel()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatalf("Run did not exit after cancel")
		}

		before := calls.Load()
		time.Sleep(1_000_000 * time.Second)
		synctest.Wait()

		after := calls.Load()
		require.Equal(t, before, after, "calls changed after cancel")
	})
}

func TestCronZeroDuration(t *testing.T) {
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

		t.Cleanup(func() {
			cancel()
		})

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

func TestContextAlreadyCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := New()

		next := func() time.Duration {
			return 0
		}

		var calls atomic.Int64
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)

			c.Run(ctx, func() {
				calls.Add(1)
			}, next)
		}()

		synctest.Wait()

		require.Zero(t, calls.Load())
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatalf("Run did not exit after cancel")
		}
	})
}

func TestNoInternalState(t *testing.T) {
	const size = unsafe.Sizeof(*New())
	require.Zero(t, size)
}
