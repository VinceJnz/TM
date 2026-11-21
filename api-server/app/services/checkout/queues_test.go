package checkout

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRecoveryQueuesRunIndependentlyAndStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	blocked := make(chan struct{})
	started := make(chan struct{})
	progress := make(chan struct{}, 4)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		runQueue(ctx, time.Millisecond, func(ctx context.Context) error {
			close(started)
			select {
			case <-blocked:
			case <-ctx.Done():
			}
			return nil
		})
	}()
	<-started
	go func() {
		defer wg.Done()
		runQueue(ctx, time.Millisecond, func(context.Context) error {
			select {
			case progress <- struct{}{}:
			default:
			}
			return nil
		})
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-progress:
		case <-time.After(time.Second):
			t.Fatal("slow queue blocked independent recovery")
		}
	}
	cancel()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("workers failed to stop")
	}
}
