package pirateship

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForLabels(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		status := LabelStatusPending
		if calls.Add(1) >= 2 {
			status = LabelStatusFinished
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"data": map[string]any{"labels": []map[string]any{{
				"id": "label", "status": status, "fileFormat": "PDF", "pageLayout": "LAYOUT_4x6",
				"url": "https://example.com/force/0/label.pdf",
			}}},
		})
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL)
	labels, err := client.WaitForLabels(
		context.Background(), LabelsRequest{DownloadID: "download"}, time.Millisecond,
	)
	if err != nil {
		t.Fatalf("WaitForLabels() error = %v", err)
	}
	if len(labels) != 1 || labels[0].Status != LabelStatusFinished || calls.Load() != 2 {
		t.Errorf("labels=%#v calls=%d", labels, calls.Load())
	}
}

func TestWaitForLabelsError(t *testing.T) {
	t.Parallel()
	server := newGraphQLServer(t, http.StatusOK, `{"data":{"labels":[{"id":"label","status":"ERROR"}]}}`)
	client := newTestClient(t, server.URL)
	_, err := client.WaitForLabels(context.Background(), LabelsRequest{DownloadID: "download"}, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "label creation failed") {
		t.Fatalf("WaitForLabels() error = %v", err)
	}
}

func TestDownloadLabel(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.Contains(request.URL.Path, "/force/1/") {
			t.Errorf("path = %q, want force/1", request.URL.Path)
		}
		if request.Header.Get("X-GraphQL-Secret") != "" {
			t.Error("custom GraphQL header forwarded to artifact download")
		}
		_, _ = io.WriteString(writer, "label bytes")
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(
		WithEndpoint("https://example.com/graphql"),
		WithHTTPClient(server.Client()),
		WithHeaders(http.Header{"X-GraphQL-Secret": {"secret"}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	var destination bytes.Buffer
	response, err := client.DownloadLabel(context.Background(), Label{
		Status: LabelStatusFinished,
		URL:    server.URL + "/force/0/label.pdf",
	}, &destination)
	if err != nil {
		t.Fatalf("DownloadLabel() error = %v", err)
	}
	if response.StatusCode != http.StatusOK || destination.String() != "label bytes" {
		t.Errorf("response=%#v body=%q", response, destination.String())
	}
}

func TestDownloadLabelValidation(t *testing.T) {
	t.Parallel()
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	tests := []Label{
		{Status: LabelStatusPending, URL: "https://example.com/label"},
		{Status: LabelStatusFinished, URL: "http://example.com/label"},
		{Status: LabelStatusFinished, URL: "https://user@example.com/label"},
	}
	for _, label := range tests {
		if _, err := client.DownloadLabel(context.Background(), label, &bytes.Buffer{}); err == nil {
			t.Errorf("DownloadLabel(%#v) error = nil", label)
		}
	}
	var nilDestination *bytes.Buffer
	if _, err := client.DownloadLabel(context.Background(), Label{
		Status: LabelStatusFinished, URL: "https://example.com/label",
	}, nilDestination); err == nil {
		t.Error("DownloadLabel() error = nil for typed nil destination")
	}
}
