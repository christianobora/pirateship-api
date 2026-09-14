// Package poll provides context-aware polling for asynchronous Pirate Ship jobs.
package poll

import (
	"context"
	"time"
)

// Until calls fetch immediately and then at interval until fetch reports done.
func Until[T any](
	ctx context.Context,
	interval time.Duration,
	fetch func(context.Context) (value T, done bool, err error),
) (T, error) {
	for {
		value, done, err := fetch(ctx)
		if err != nil || done {
			return value, err
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			var zero T
			return zero, ctx.Err()
		case <-timer.C:
		}
	}
}
