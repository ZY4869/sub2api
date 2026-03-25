package service

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestIsContextDoneError(t *testing.T) {
	t.Run("nil_context_and_nil_error", func(t *testing.T) {
		if isContextDoneError(nil, nil) {
			t.Fatal("expected false")
		}
	})

	t.Run("direct_context_canceled_error", func(t *testing.T) {
		if !isContextDoneError(context.Background(), context.Canceled) {
			t.Fatal("expected true")
		}
	})

	t.Run("wrapped_deadline_error", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", context.DeadlineExceeded)
		if !isContextDoneError(context.Background(), err) {
			t.Fatal("expected true")
		}
	})

	t.Run("canceled_context_without_error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if !isContextDoneError(ctx, nil) {
			t.Fatal("expected true")
		}
	})

	t.Run("deadline_context_without_error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()
		time.Sleep(5 * time.Millisecond)
		if !isContextDoneError(ctx, nil) {
			t.Fatal("expected true")
		}
	})
}
