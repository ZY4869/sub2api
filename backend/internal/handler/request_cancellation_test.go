package handler

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestIsRequestCanceled(t *testing.T) {
	t.Run("nil_context_and_nil_error", func(t *testing.T) {
		if isRequestCanceled(context.TODO(), nil) {
			t.Fatal("expected false")
		}
	})

	t.Run("direct_context_canceled_error", func(t *testing.T) {
		if !isRequestCanceled(context.Background(), context.Canceled) {
			t.Fatal("expected true")
		}
	})

	t.Run("wrapped_deadline_error", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", context.DeadlineExceeded)
		if !isRequestCanceled(context.Background(), err) {
			t.Fatal("expected true")
		}
	})

	t.Run("canceled_context_without_error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if !isRequestCanceled(ctx, nil) {
			t.Fatal("expected true")
		}
	})

	t.Run("deadline_context_without_error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		select {
		case <-ctx.Done():
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected context deadline to fire")
		}
		if !isRequestCanceled(ctx, nil) {
			t.Fatal("expected true")
		}
	})
}
