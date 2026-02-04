// Package testutil provides shared test utilities and helpers.
//
// This package contains helper functions for creating test fixtures,
// temporary databases, mock servers, and other testing infrastructure.
package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"feedpulse/internal/storage"
)

// NewTestDB creates a temporary database for testing.
// The database is automatically cleaned up when the test completes.
func NewTestDB(t *testing.T) *storage.Storage {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return db
}

// MockServer creates a test HTTP server that returns the specified response.
// The server is automatically closed when the test completes.
func MockServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// MockServerWithHeaders creates a test HTTP server with custom headers.
func MockServerWithHeaders(t *testing.T, statusCode int, headers map[string]string, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// MockServerFunc creates a test HTTP server with a custom handler function.
func MockServerFunc(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// CreateTempConfig creates a temporary config file for testing.
// The file is automatically cleaned up when the test completes.
func CreateTempConfig(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	return configPath
}

// MustMarshalJSON marshals a value to JSON or fails the test.
func MustMarshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	return data
}

// AssertError checks that an error occurred and optionally contains expected text.
func AssertError(t *testing.T, err error, wantErrText string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if wantErrText != "" && !contains(err.Error(), wantErrText) {
		t.Errorf("error %q does not contain %q", err.Error(), wantErrText)
	}
}

// AssertNoError checks that no error occurred.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertEqual checks that two values are equal.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// AssertNotEqual checks that two values are not equal.
func AssertNotEqual(t *testing.T, got, notWant interface{}) {
	t.Helper()

	if got == notWant {
		t.Errorf("got %v, expected different value", got)
	}
}

// AssertTrue checks that a condition is true.
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()

	if !condition {
		t.Errorf("expected true: %s", message)
	}
}

// AssertFalse checks that a condition is false.
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()

	if condition {
		t.Errorf("expected false: %s", message)
	}
}

// contains is a helper to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SampleHackerNewsJSON returns a sample HackerNews API response for testing.
func SampleHackerNewsJSON() string {
	return `[1, 2, 3, 4, 5]`
}

// SampleGitHubJSON returns a sample GitHub API response for testing.
func SampleGitHubJSON() string {
	return `{
		"items": [
			{
				"full_name": "test/repo",
				"html_url": "https://github.com/test/repo",
				"updated_at": "2024-01-01T00:00:00Z",
				"topics": ["golang", "testing"]
			}
		]
	}`
}

// SampleRedditJSON returns a sample Reddit API response for testing.
func SampleRedditJSON() string {
	return `{
		"data": {
			"children": [
				{
					"data": {
						"title": "Test Post",
						"url": "https://reddit.com/r/test/123",
						"created_utc": 1609459200,
						"link_flair_text": "Discussion"
					}
				}
			]
		}
	}`
}

// SampleLobstersJSON returns a sample Lobsters API response for testing.
func SampleLobstersJSON() string {
	return `[
		{
			"title": "Test Story",
			"url": "https://example.com/story",
			"comments_url": "https://lobste.rs/s/abc123",
			"created_at": "2024-01-01T00:00:00Z",
			"tags": ["go", "programming"]
		}
	]`
}
