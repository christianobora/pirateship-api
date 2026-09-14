package pirateship

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForBatch(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		status := BatchStatusRating
		if calls.Add(1) >= 2 {
			status = BatchStatusRated
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"data": map[string]any{"batch": map[string]any{"id": "batch", "status": status}},
		})
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL)
	batch, err := client.WaitForBatch(context.Background(), "batch", time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForBatch() error = %v", err)
	}
	if batch.Status != BatchStatusRated || calls.Load() != 2 {
		t.Errorf("batch=%#v calls=%d", batch, calls.Load())
	}
}

func TestWaitForBatchContext(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, `{"data":{"batch":{"id":"batch","status":"RATING"}}}`)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := client.WaitForBatch(ctx, "batch", time.Millisecond)
	if err == nil {
		t.Fatal("WaitForBatch() error = nil")
	}
}
