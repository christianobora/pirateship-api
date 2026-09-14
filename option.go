package pirateship

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

const (
	defaultMaxResponseBytes = 4 << 20
	defaultTimeout          = 30 * time.Second
)

// HTTPClient performs HTTP requests. *http.Client satisfies this interface.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// RetryPolicy controls retries for read-only GraphQL queries. Mutations are
// never retried automatically, regardless of this policy.
type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// Option configures a Client.
type Option interface {
	apply(*clientConfig) error
}

type optionFunc func(*clientConfig) error

func (f optionFunc) apply(config *clientConfig) error {
	return f(config)
}

type clientConfig struct {
	endpoint         *url.URL
	httpClient       HTTPClient
	headers          http.Header
	retryPolicy      RetryPolicy
	maxResponseBytes int64
}

func defaultClientConfig() *clientConfig {
	endpoint := &url.URL{
		Scheme: "https",
		Host:   "ship.pirateship.com",
		Path:   "/api/graphql",
	}

	return &clientConfig{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		headers: make(http.Header),
		retryPolicy: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 250 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
		maxResponseBytes: defaultMaxResponseBytes,
	}
}

// WithEndpoint overrides the GraphQL endpoint. It is primarily useful for
// tests and compatible proxies.
func WithEndpoint(rawURL string) Option {
	return optionFunc(func(config *clientConfig) error {
		endpoint, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("parse endpoint: %w", err)
		}
		if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
			return fmt.Errorf("endpoint scheme must be http or https, got %q", endpoint.Scheme)
		}
		if endpoint.Host == "" {
			return errors.New("endpoint host is required")
		}
		if endpoint.User != nil {
			return errors.New("endpoint must not contain user information")
		}
		if endpoint.Fragment != "" {
			return errors.New("endpoint must not contain a fragment")
		}

		copy := *endpoint
		config.endpoint = &copy
		return nil
	})
}

// WithHTTPClient sets the HTTP client used for all requests. Use a client with
// a cookie jar containing an authenticated Pirate Ship session for operations
// that require authentication.
func WithHTTPClient(client HTTPClient) Option {
	return optionFunc(func(config *clientConfig) error {
		if isNilInterface(client) {
			return errors.New("HTTP client must not be nil")
		}
		config.httpClient = client
		return nil
	})
}

// WithHeaders adds headers to every GraphQL request. Values are copied when the
// client is constructed. Content-Type, Content-Length, Host, and User-Agent are
// managed by the client and cannot be overridden here.
func WithHeaders(headers http.Header) Option {
	return optionFunc(func(config *clientConfig) error {
		for name, values := range headers {
			canonicalName := http.CanonicalHeaderKey(name)
			switch canonicalName {
			case "Content-Type", "Content-Length", "Host", "User-Agent":
				return fmt.Errorf("header %q is managed by the client", canonicalName)
			}
			if strings.TrimSpace(canonicalName) == "" {
				return errors.New("header name must not be empty")
			}
			config.headers[canonicalName] = append([]string(nil), values...)
		}
		return nil
	})
}

// WithRetryPolicy sets the retry policy used for queries.
func WithRetryPolicy(policy RetryPolicy) Option {
	return optionFunc(func(config *clientConfig) error {
		if policy.MaxAttempts < 1 {
			return errors.New("retry max attempts must be at least 1")
		}
		if policy.InitialBackoff < 0 {
			return errors.New("retry initial backoff must not be negative")
		}
		if policy.MaxBackoff < policy.InitialBackoff {
			return errors.New("retry max backoff must not be less than initial backoff")
		}
		config.retryPolicy = policy
		return nil
	})
}

// WithMaxResponseBytes limits buffered GraphQL response bodies. The default is
// 4 MiB. Label files are streamed and are not subject to this limit.
func WithMaxResponseBytes(limit int64) Option {
	return optionFunc(func(config *clientConfig) error {
		if limit < 1 {
			return errors.New("maximum response size must be positive")
		}
		config.maxResponseBytes = limit
		return nil
	})
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
