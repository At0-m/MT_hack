package inference

import (
	"context"
	"errors"
	"testing"
)

func TestCancellationTerminatesAndJoinsRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started, terminated := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- runCancelable(ctx, func() error { close(started); <-terminated; return errors.New("native terminated") }, func() error { close(terminated); return nil })
	}()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-terminated:
	default:
		t.Fatal("termination callback not joined")
	}
}
