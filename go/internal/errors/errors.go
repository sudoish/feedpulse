// Package errors provides domain-specific error types for feedpulse.
//
// These error types provide structured error information with context,
// making it easier to handle and report errors in a user-friendly way.
package errors

import "fmt"

// ConfigError represents configuration-related errors
type ConfigError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ConfigError) Error() string {
	if e.Field != "" && e.Value != nil {
		return fmt.Sprintf("config error: %s=%v: %s", e.Field, e.Value, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("config error: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

// NewConfigError creates a new configuration error
func NewConfigError(field string, value interface{}, message string) *ConfigError {
	return &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NetworkError represents network-related errors (timeouts, connection failures, etc.)
type NetworkError struct {
	URL     string
	Op      string // operation: "fetch", "connect", "timeout"
	Message string
	Cause   error
}

func (e *NetworkError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("network error (%s) at %s: %s: %v", e.Op, e.URL, e.Message, e.Cause)
	}
	return fmt.Sprintf("network error (%s) at %s: %s", e.Op, e.URL, e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.Cause
}

// NewNetworkError creates a new network error
func NewNetworkError(url, op, message string, cause error) *NetworkError {
	return &NetworkError{
		URL:     url,
		Op:      op,
		Message: message,
		Cause:   cause,
	}
}

// ParseError represents feed parsing errors
type ParseError struct {
	Source   string
	FeedType string
	Message  string
	Line     int // optional: line number where error occurred
	Cause    error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("parse error in %s (%s) at line %d: %s", e.Source, e.FeedType, e.Line, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("parse error in %s (%s): %s: %v", e.Source, e.FeedType, e.Message, e.Cause)
	}
	return fmt.Sprintf("parse error in %s (%s): %s", e.Source, e.FeedType, e.Message)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}

// NewParseError creates a new parse error
func NewParseError(source, feedType, message string, cause error) *ParseError {
	return &ParseError{
		Source:   source,
		FeedType: feedType,
		Message:  message,
		Cause:    cause,
	}
}

// StorageError represents database/storage errors
type StorageError struct {
	Op      string // operation: "save", "query", "delete", "init"
	Message string
	Cause   error
}

func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("storage error (%s): %s: %v", e.Op, e.Message, e.Cause)
	}
	return fmt.Sprintf("storage error (%s): %s", e.Op, e.Message)
}

func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewStorageError creates a new storage error
func NewStorageError(op, message string, cause error) *StorageError {
	return &StorageError{
		Op:      op,
		Message: message,
		Cause:   cause,
	}
}

// ValidationError represents data validation errors
type ValidationError struct {
	Field   string
	Value   interface{}
	Rule    string // validation rule that failed: "required", "format", "range"
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation error: %s=%v failed %s: %s", e.Field, e.Value, e.Rule, e.Message)
	}
	return fmt.Sprintf("validation error: %s failed %s: %s", e.Field, e.Rule, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, rule, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	}
}
