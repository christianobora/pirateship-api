package pirateship

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type httpClientFunc func(*http.Request) (*http.Response, error)

func (f httpClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClientDo(t *testing.T) {
	t.Parallel()

	var requestPayload struct {
		OperationName string         `json:"operationName"`
		Variables     map[string]any `json:"variables"`
		Query         string         `json:"query"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", request.Method)
		}
		if got := request.URL.Query().Get("opname"); got != "ExampleQuery" {
			t.Errorf("opname = %q, want ExampleQuery", got)
		}
		if got := request.Header.Get("X-Test"); got != "value" {
			t.Errorf("X-Test = %q, want value", got)
		}
		if got := request.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("User-Agent = %q, want %q", got, userAgent)
		}
		if err := json.NewDecoder(request.Body).Decode(&requestPayload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		writer.Header().Set("X-Response", "present")
		writer.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(writer, `{"data":{"answer":42}}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(
		WithEndpoint(server.URL+"?existing=1"),
		WithHeaders(http.Header{"X-Test": {"value"}}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	var result struct {
		Answer int `json:"answer"`
	}
	response, err := client.Do(context.Background(), Operation{
		Name:      "ExampleQuery",
		Query:     "query ExampleQuery($id: ID!) { answer(id: $id) }",
		Variables: map[string]string{"id": "123"},
		Type:      OperationQuery,
	}, &result)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if result.Answer != 42 {
		t.Errorf("answer = %d, want 42", result.Answer)
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("X-Response") != "present" {
		t.Errorf("response = %#v", response)
	}
	if requestPayload.OperationName != "ExampleQuery" || requestPayload.Variables["id"] != "123" {
		t.Errorf("payload = %#v", requestPayload)
	}
}

func TestClientDecodesPartialDataBeforeGraphQLErrors(t *testing.T) {
	t.Parallel()

	server := newGraphQLServer(t, http.StatusOK, `{
  "data":{"value":"partial"},
  "errors":[{"message":"session expired","extensions":{"code":"UNAUTHENTICATED"}}]
}`)
	client := newTestClient(t, server.URL)
	var result struct {
		Value string `json:"value"`
	}
	_, err := client.Do(context.Background(), Operation{
		Name: "PartialQuery", Query: "query PartialQuery { value }", Type: OperationQuery,
	}, &result)
	if result.Value != "partial" {
		t.Errorf("value = %q, want partial", result.Value)
	}
	if !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("error = %v, want ErrUnauthenticated", err)
	}
}

func TestClientRetriesQueriesOnly(t *testing.T) {
	t.Parallel()

	t.Run("query", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) < 3 {
				http.Error(writer, "temporary", http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(writer, `{"data":{"ok":true}}`)
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(
			WithEndpoint(server.URL),
			WithRetryPolicy(RetryPolicy{MaxAttempts: 3}),
		)
		if err != nil {
			t.Fatal(err)
		}
		client.jitter = func(time.Duration) time.Duration { return 0 }
		var data struct {
			OK bool `json:"ok"`
		}
		_, err = client.Do(context.Background(), Operation{
			Name: "RetryQuery", Query: "query RetryQuery { ok }", Type: OperationQuery,
		}, &data)
		if err != nil || !data.OK || calls.Load() != 3 {
			t.Fatalf("Do() data=%#v calls=%d error=%v", data, calls.Load(), err)
		}
	})

	t.Run("mutation", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			http.Error(writer, "temporary", http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)
		client := newTestClient(t, server.URL)
		_, err := client.Do(context.Background(), Operation{
			Name: "BuyMutation", Query: "mutation BuyMutation { buy }", Type: OperationMutation,
		}, nil)
		if err == nil || calls.Load() != 1 {
			t.Fatalf("Do() calls=%d error=%v, want one call and error", calls.Load(), err)
		}
	})
}

func TestClientResponseLimit(t *testing.T) {
	t.Parallel()

	server := newGraphQLServer(t, http.StatusOK, `{"data":{"value":"too long"}}`)
	client, err := NewClient(WithEndpoint(server.URL), WithMaxResponseBytes(8))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(context.Background(), Operation{
		Name: "LimitQuery", Query: "query LimitQuery { value }", Type: OperationQuery,
	}, nil)
	var protocolError *ProtocolError
	if !errors.As(err, &protocolError) {
		t.Fatalf("error = %v, want ProtocolError", err)
	}
}

func TestClientValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation Operation
	}{
		{name: "invalid name", operation: Operation{Name: "not valid", Query: "query X{x}", Type: OperationQuery}},
		{name: "empty query", operation: Operation{Name: "Valid", Type: OperationQuery}},
		{name: "invalid type", operation: Operation{Name: "Valid", Query: "query Valid{x}"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient()
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Do(context.Background(), test.operation, nil)
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("error = %v, want ValidationError", err)
			}
		})
	}
}

func TestClientRejectsNilContext(t *testing.T) {
	t.Parallel()
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	var nilContext context.Context
	_, err = client.Do(nilContext, Operation{
		Name: "Valid", Query: "query Valid { value }", Type: OperationQuery,
	}, nil)
	if err == nil {
		t.Fatal("Do() error = nil, want validation error")
	}
}

func TestNewClientOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		option Option
	}{
		{name: "nil option", option: nil},
		{name: "invalid endpoint", option: WithEndpoint("file:///tmp/api")},
		{name: "nil HTTP client", option: WithHTTPClient((*http.Client)(nil))},
		{name: "managed header", option: WithHeaders(http.Header{"User-Agent": {"test"}})},
		{name: "invalid retries", option: WithRetryPolicy(RetryPolicy{})},
		{name: "invalid response limit", option: WithMaxResponseBytes(0)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewClient(test.option); err == nil {
				t.Fatal("NewClient() error = nil")
			}
		})
	}
}

func TestRetryAfterDuration(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	if got := retryAfterDuration("2", now); got != 2*time.Second {
		t.Errorf("seconds duration = %v", got)
	}
	if got := retryAfterDuration(now.Add(time.Minute).Format(http.TimeFormat), now); got != time.Minute {
		t.Errorf("date duration = %v", got)
	}
	if got := retryAfterDuration("invalid", now); got != 0 {
		t.Errorf("invalid duration = %v", got)
	}
}

func FuzzDecodeGraphQLResponse(f *testing.F) {
	f.Add(`{"data":{"value":"ok"}}`)
	f.Add(`{"errors":[{"message":"bad"}]}`)
	f.Add(`not json`)
	f.Fuzz(func(t *testing.T, body string) {
		var destination any
		_ = decodeGraphQLResponse([]byte(body), &destination)
	})
}

func newGraphQLServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(status)
		_, _ = io.Copy(writer, strings.NewReader(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	client, err := NewClient(
		WithEndpoint(endpoint),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 3}),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client.jitter = func(time.Duration) time.Duration { return 0 }
	return client
}
