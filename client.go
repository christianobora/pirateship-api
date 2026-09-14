package pirateship

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const userAgent = "github.com/christianobora/pirateship-api"

var operationNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// OperationType identifies whether a GraphQL operation is read-only.
type OperationType uint8

const (
	// OperationQuery identifies a read-only GraphQL query.
	OperationQuery OperationType = iota + 1
	// OperationMutation identifies a state-changing GraphQL mutation.
	OperationMutation
)

// Operation describes one GraphQL operation. Custom operations provide an
// escape hatch for API fields that are newer than this module.
type Operation struct {
	Name      string
	Query     string
	Variables any
	Type      OperationType
}

// Response contains HTTP metadata for an executed GraphQL operation.
type Response struct {
	StatusCode int
	Header     http.Header
}

type httpResponse struct {
	statusCode int
	status     string
	header     http.Header
}

// Client is a concurrency-safe Pirate Ship API client.
type Client struct {
	endpoint         url.URL
	httpClient       HTTPClient
	headers          http.Header
	retryPolicy      RetryPolicy
	maxResponseBytes int64
	jitter           func(time.Duration) time.Duration
}

// NewClient constructs a Pirate Ship client.
func NewClient(options ...Option) (*Client, error) {
	config := defaultClientConfig()
	for index, option := range options {
		if option == nil {
			return nil, fmt.Errorf("apply option %d: option must not be nil", index)
		}
		if err := option.apply(config); err != nil {
			return nil, fmt.Errorf("apply option %d: %w", index, err)
		}
	}

	return &Client{
		endpoint:         *config.endpoint,
		httpClient:       config.httpClient,
		headers:          config.headers.Clone(),
		retryPolicy:      config.retryPolicy,
		maxResponseBytes: config.maxResponseBytes,
		jitter:           fullJitter,
	}, nil
}

// Do executes a GraphQL operation and decodes its data object into result.
// GraphQL partial data is decoded before GraphQLErrors is returned.
func (c *Client) Do(
	ctx context.Context,
	operation Operation,
	result any,
) (*Response, error) {
	if ctx == nil {
		return nil, &ValidationError{Field: "context", Problem: "must not be nil"}
	}
	if err := validateOperation(operation); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(struct {
		OperationName string `json:"operationName"`
		Variables     any    `json:"variables"`
		Query         string `json:"query"`
	}{
		OperationName: operation.Name,
		Variables:     operation.Variables,
		Query:         operation.Query,
	})
	if err != nil {
		return nil, fmt.Errorf("pirateship: encode %s variables: %w", operation.Name, err)
	}

	var lastErr error
	for attempt := 1; attempt <= c.retryPolicy.MaxAttempts; attempt++ {
		response, body, truncated, requestErr := c.send(ctx, operation.Name, payload)
		if requestErr != nil {
			lastErr = fmt.Errorf("pirateship: execute %s: %w", operation.Name, requestErr)
			if !c.canRetry(operation, attempt) || !isRetryableRequestError(requestErr) {
				return nil, lastErr
			}
			if err := c.waitForRetry(ctx, attempt, ""); err != nil {
				return nil, err
			}
			continue
		}

		metadata := &Response{
			StatusCode: response.statusCode,
			Header:     response.header.Clone(),
		}
		if response.statusCode < http.StatusOK || response.statusCode >= http.StatusMultipleChoices {
			lastErr = &HTTPError{
				StatusCode: response.statusCode,
				Status:     response.status,
				Header:     response.header.Clone(),
				Body:       body,
				Truncated:  truncated,
			}
			if !c.canRetry(operation, attempt) || !isRetryableStatus(response.statusCode) {
				return metadata, lastErr
			}
			if err := c.waitForRetry(ctx, attempt, response.header.Get("Retry-After")); err != nil {
				return metadata, err
			}
			continue
		}

		if truncated {
			return metadata, &ProtocolError{
				Problem: fmt.Sprintf("response exceeds %d bytes", c.maxResponseBytes),
			}
		}
		return metadata, decodeGraphQLResponse(body, result)
	}

	return nil, lastErr
}

func (c *Client) send(
	ctx context.Context,
	operationName string,
	payload []byte,
) (httpResponse, []byte, bool, error) {
	endpoint := c.endpoint
	query := endpoint.Query()
	query.Set("opname", operationName)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		bytes.NewReader(payload),
	)
	if err != nil {
		return httpResponse{}, nil, false, fmt.Errorf("build HTTP request: %w", err)
	}
	request.Header = c.headers.Clone()
	request.Header.Set("Accept", "application/graphql-response+json, application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", userAgent)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return httpResponse{}, nil, false, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, truncated, err := readLimited(response.Body, c.maxResponseBytes)
	if err != nil {
		return httpResponse{}, nil, false, fmt.Errorf("read HTTP response: %w", err)
	}
	return httpResponse{
		statusCode: response.StatusCode,
		status:     response.Status,
		header:     response.Header.Clone(),
	}, body, truncated, nil
}

func (c *Client) canRetry(operation Operation, attempt int) bool {
	return operation.Type == OperationQuery && attempt < c.retryPolicy.MaxAttempts
}

func (c *Client) waitForRetry(
	ctx context.Context,
	attempt int,
	retryAfter string,
) error {
	delay := retryAfterDuration(retryAfter, time.Now())
	if delay <= 0 {
		delay = c.retryPolicy.InitialBackoff
		for exponent := 1; exponent < attempt; exponent++ {
			if delay >= c.retryPolicy.MaxBackoff/2 {
				delay = c.retryPolicy.MaxBackoff
				break
			}
			delay *= 2
		}
		delay = min(delay, c.retryPolicy.MaxBackoff)
		delay = c.jitter(delay)
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("pirateship: wait to retry: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func validateOperation(operation Operation) error {
	if !operationNamePattern.MatchString(operation.Name) {
		return &ValidationError{
			Field:   "operation name",
			Problem: "must be a valid GraphQL operation name",
		}
	}
	if strings.TrimSpace(operation.Query) == "" {
		return &ValidationError{Field: "operation query", Problem: "must not be empty"}
	}
	switch operation.Type {
	case OperationQuery, OperationMutation:
		return nil
	default:
		return &ValidationError{Field: "operation type", Problem: "must be query or mutation"}
	}
}

func decodeGraphQLResponse(body []byte, result any) error {
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors GraphQLErrors   `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return &ProtocolError{Problem: "decode GraphQL envelope", Err: err}
	}

	hasData := len(envelope.Data) > 0 && !bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null"))
	if hasData && result != nil {
		if err := json.Unmarshal(envelope.Data, result); err != nil {
			return &ProtocolError{Problem: "decode GraphQL data", Err: err}
		}
	}
	if len(envelope.Errors) > 0 {
		return envelope.Errors
	}
	if !hasData {
		return &ProtocolError{Problem: "response contains neither data nor errors"}
	}
	return nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, bool, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > limit {
		return body[:limit], true, nil
	}
	return body, false, nil
}

func isRetryableRequestError(err error) bool {
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func retryAfterDuration(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 32); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil || !when.After(now) {
		return 0
	}
	return when.Sub(now)
}

func fullJitter(delay time.Duration) time.Duration {
	if delay <= 1 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(delay)))
}
