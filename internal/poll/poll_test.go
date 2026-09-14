package poll

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUntil(t *testing.T) {
	t.Parallel()
	calls := 0
	value, err := Until(context.Background(), time.Millisecond, func(context.Context) (int, bool, error) {
		calls++
		return calls, calls == 2, nil
	})
	if err != nil || value != 2 || calls != 2 {
		t.Fatalf("Until() = %d, %v; calls=%d", value, err, calls)
	}
}

func TestUntilError(t *testing.T) {
	t.Parallel()
	want := errors.New("stop")
	_, err := Until(context.Background(), time.Hour, func(context.Context) (int, bool, error) {
		return 0, false, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Until() error = %v, want %v", err, want)
	}
}
