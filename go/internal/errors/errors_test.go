package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigError(t *testing.T) {
	tests := []struct {
		name    string
		err     *ConfigError
		wantMsg string
	}{
		{
			name: "with field and value",
			err:  NewConfigError("timeout", 0, "must be positive"),
			wantMsg: "config error: timeout=0: must be positive",
		},
		{
			name: "with field only",
			err:  NewConfigError("database_path", nil, "required"),
			wantMsg: "config error: database_path: required",
		},
		{
			name: "message only",
			err:  &ConfigError{Message: "invalid configuration"},
			wantMsg: "config error: invalid configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("ConfigError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestNetworkError(t *testing.T) {
	cause := errors.New("connection refused")

	tests := []struct {
		name    string
		err     *NetworkError
		wantMsg string
	}{
		{
			name: "with cause",
			err:  NewNetworkError("http://example.com", "fetch", "failed to connect", cause),
			wantMsg: "network error (fetch) at http://example.com: failed to connect: connection refused",
		},
		{
			name: "without cause",
			err:  NewNetworkError("http://example.com", "timeout", "request timed out", nil),
			wantMsg: "network error (timeout) at http://example.com: request timed out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("NetworkError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	cause := errors.New("connection refused")
	err := NewNetworkError("http://example.com", "connect", "failed", cause)

	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("NetworkError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestParseError(t *testing.T) {
	cause := errors.New("unexpected token")

	tests := []struct {
		name    string
		err     *ParseError
		wantMsg string
	}{
		{
			name: "with line number",
			err:  &ParseError{Source: "HackerNews", FeedType: "json", Message: "invalid syntax", Line: 42},
			wantMsg: "parse error in HackerNews (json) at line 42: invalid syntax",
		},
		{
			name: "with cause",
			err:  NewParseError("GitHub", "json", "malformed JSON", cause),
			wantMsg: "parse error in GitHub (json): malformed JSON: unexpected token",
		},
		{
			name: "basic error",
			err:  NewParseError("Reddit", "json", "missing required field", nil),
			wantMsg: "parse error in Reddit (json): missing required field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("ParseError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestParseError_Unwrap(t *testing.T) {
	cause := errors.New("unexpected EOF")
	err := NewParseError("Test", "json", "failed", cause)

	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("ParseError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestStorageError(t *testing.T) {
	cause := errors.New("disk full")

	tests := []struct {
		name    string
		err     *StorageError
		wantMsg string
	}{
		{
			name: "with cause",
			err:  NewStorageError("save", "failed to write", cause),
			wantMsg: "storage error (save): failed to write: disk full",
		},
		{
			name: "without cause",
			err:  NewStorageError("init", "database locked", nil),
			wantMsg: "storage error (init): database locked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("StorageError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

func TestStorageError_Unwrap(t *testing.T) {
	cause := errors.New("constraint violation")
	err := NewStorageError("save", "failed", cause)

	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("StorageError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		wantMsg string
	}{
		{
			name: "with value",
			err:  NewValidationError("port", -1, "range", "must be between 1 and 65535"),
			wantMsg: "validation error: port=-1 failed range: must be between 1 and 65535",
		},
		{
			name: "without value",
			err:  NewValidationError("email", nil, "required", "field is required"),
			wantMsg: "validation error: email failed required: field is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

// Test that errors can be used with errors.Is and errors.As
func TestErrorWrapping(t *testing.T) {
	cause := errors.New("root cause")
	netErr := NewNetworkError("http://example.com", "fetch", "failed", cause)

	if !errors.Is(netErr, cause) {
		t.Error("NetworkError should wrap cause error")
	}

	var ne *NetworkError
	if !errors.As(netErr, &ne) {
		t.Error("should be able to unwrap NetworkError")
	}
	if ne.URL != "http://example.com" {
		t.Errorf("unwrapped error URL = %v, want %v", ne.URL, "http://example.com")
	}
}

func TestErrorMessages_ContainContext(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantText string
	}{
		{
			name:     "config error includes field",
			err:      NewConfigError("max_concurrency", 100, "out of range"),
			wantText: "max_concurrency",
		},
		{
			name:     "network error includes URL",
			err:      NewNetworkError("http://example.com/feed", "timeout", "timed out", nil),
			wantText: "http://example.com/feed",
		},
		{
			name:     "parse error includes source",
			err:      NewParseError("HackerNews", "json", "invalid", nil),
			wantText: "HackerNews",
		},
		{
			name:     "storage error includes operation",
			err:      NewStorageError("query", "failed", nil),
			wantText: "query",
		},
		{
			name:     "validation error includes rule",
			err:      NewValidationError("url", "ftp://bad", "format", "invalid scheme"),
			wantText: "format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if !strings.Contains(msg, tt.wantText) {
				t.Errorf("error message %q should contain %q", msg, tt.wantText)
			}
		})
	}
}
