package pirateship

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	// ErrUnauthenticated indicates that the session is missing or no longer valid.
	ErrUnauthenticated = errors.New("pirateship: unauthenticated")
	// ErrRateLimited indicates that Pirate Ship rate-limited the request.
	ErrRateLimited = errors.New("pirateship: rate limited")
)

// ValidationError reports an invalid caller-supplied field.
type ValidationError struct {
	Field   string
	Problem string
}

// Error implements error.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("pirateship: invalid %s: %s", e.Field, e.Problem)
}

// HTTPError reports a non-success HTTP response. Body is intentionally omitted
// from Error so server responses cannot be logged accidentally.
type HTTPError struct {
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
	Truncated  bool
}

// Error implements error.
func (e *HTTPError) Error() string {
	return "pirateship: HTTP request failed: " + e.Status
}

// GraphQLError is one error returned in a GraphQL response.
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []any                  `json:"path,omitempty"`
	Extensions map[string]any         `json:"extensions,omitempty"`
}

// GraphQLErrorLocation identifies a position in a GraphQL document.
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// GraphQLErrors contains errors returned by the GraphQL server. Partial data,
// when present, is still decoded before this error is returned.
type GraphQLErrors []GraphQLError

// Error implements error.
func (e GraphQLErrors) Error() string {
	messages := make([]string, 0, len(e))
	for _, graphqlError := range e {
		if message := strings.TrimSpace(graphqlError.Message); message != "" {
			messages = append(messages, message)
		}
	}
	if len(messages) == 0 {
		return "pirateship: GraphQL request failed"
	}
	return "pirateship: GraphQL request failed: " + strings.Join(messages, "; ")
}

// Is supports errors.Is for ErrUnauthenticated and ErrRateLimited.
func (e GraphQLErrors) Is(target error) bool {
	for _, graphqlError := range e {
		code, _ := graphqlError.Extensions["code"].(string)
		failureReason, _ := graphqlError.Extensions["failureReason"].(string)
		switch target {
		case ErrUnauthenticated:
			if code == "UNAUTHENTICATED" {
				return true
			}
		case ErrRateLimited:
			if code == "RATE_LIMITED" || failureReason == "rate_limited" {
				return true
			}
		}
	}
	return false
}

// ProtocolError reports a malformed or unexpectedly large server response.
type ProtocolError struct {
	Problem string
	Err     error
}

// LabelCreationError reports a label-rendering job that ended in an error.
type LabelCreationError struct {
	DownloadID ID
}

// Error implements error.
func (e *LabelCreationError) Error() string {
	return fmt.Sprintf("pirateship: label creation failed for download %q", e.DownloadID)
}

// Error implements error.
func (e *ProtocolError) Error() string {
	if e.Err == nil {
		return "pirateship: invalid API response: " + e.Problem
	}
	return "pirateship: invalid API response: " + e.Problem + ": " + e.Err.Error()
}

// Unwrap returns the underlying decoding error, if any.
func (e *ProtocolError) Unwrap() error {
	return e.Err
}
