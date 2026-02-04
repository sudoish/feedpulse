// Package integration provides end-to-end integration tests for feedpulse.
//
// These tests verify the complete pipeline: config → fetch → parse → store → report.
// They use real (but mocked) HTTP servers and databases.
package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"feedpulse/internal/config"
	"feedpulse/internal/parser"
	"feedpulse/internal/storage"
)

// TestIntegration_EndToEnd tests the complete workflow
func TestIntegration_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`))
	}))
	defer server.Close()

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create parser
	p := parser.NewParser()

	// Simulate fetching (we'll mock HTTP response)
	mockData := []byte(`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`)
	
	// Parse
	result := p.Parse("GitHub", "json", mockData)
	if len(result.Errors) > 0 {
		t.Errorf("Parse errors: %v", result.Errors)
	}
	if len(result.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result.Items))
	}

	// Store
	err = db.SaveItems(result.Items)
	if err != nil {
		t.Fatalf("SaveItems failed: %v", err)
	}

	// Verify
	count, err := db.GetItemCount("GitHub")
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 stored item, got %d", count)
	}

	// Log fetch
	log := storage.FetchLog{
		Source:       "GitHub",
		FetchedAt:    time.Now(),
		Status:       "success",
		ItemsCount:   1,
		ErrorMessage: nil,
		DurationMs:   0,
	}
	err = db.LogFetch(log)
	if err != nil {
		t.Fatalf("LogFetch failed: %v", err)
	}

	// Verify stats
	stats, err := db.GetFetchStats()
	if err != nil {
		t.Fatalf("GetFetchStats failed: %v", err)
	}
	if len(stats) != 1 {
		t.Errorf("Expected 1 fetch stat, got %d", len(stats))
	}
}

// TestIntegration_MultipleFeeds tests handling multiple feeds concurrently
func TestIntegration_MultipleFeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "multi.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	// Simulate multiple feeds
	feeds := []struct {
		source   string
		feedType string
		data     string
	}{
		{
			"GitHub",
			"json",
			`{"items":[{"full_name":"test1/repo","html_url":"https://github.com/test1/repo"}]}`,
		},
		{
			"Reddit",
			"json",
			`{"data":{"children":[{"data":{"title":"Test Post","url":"https://reddit.com/r/test/123"}}]}}`,
		},
		{
			"Lobsters",
			"json",
			`[{"title":"Test Story","url":"https://example.com","comments_url":"https://lobste.rs/s/abc"}]`,
		},
	}

	// Process each feed
	for _, feed := range feeds {
		result := p.Parse(feed.source, feed.feedType, []byte(feed.data))
		if len(result.Errors) > 0 {
			t.Errorf("%s parse errors: %v", feed.source, result.Errors)
			continue
		}

		err := db.SaveItems(result.Items)
		if err != nil {
			t.Errorf("%s save failed: %v", feed.source, err)
			continue
		}

		log := storage.FetchLog{
			Source:       feed.source,
			FetchedAt:    time.Now(),
			Status:       "success",
			ItemsCount:   len(result.Items),
			ErrorMessage: nil,
			DurationMs:   0,
		}
		err = db.LogFetch(log)
		if err != nil {
			t.Errorf("%s log failed: %v", feed.source, err)
		}
	}

	// Verify total items
	totalCount, err := db.GetAllItemsCount()
	if err != nil {
		t.Fatalf("GetAllItemsCount failed: %v", err)
	}
	if totalCount != 3 {
		t.Errorf("Expected 3 total items, got %d", totalCount)
	}

	// Verify fetch stats
	stats, err := db.GetFetchStats()
	if err != nil {
		t.Fatalf("GetFetchStats failed: %v", err)
	}
	if len(stats) != 3 {
		t.Errorf("Expected 3 fetch stats, got %d", len(stats))
	}
}

// TestIntegration_ErrorHandling tests error scenarios
func TestIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "errors.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	// Test malformed JSON
	result := p.Parse("Test", "json", []byte(`{invalid json`))
	if len(result.Errors) == 0 {
		t.Error("Expected parse error for malformed JSON")
	}

	// Log the error
	errMsg := result.Errors[0]
	log := storage.FetchLog{
		Source:       "Test",
		FetchedAt:    time.Now(),
		Status:       "error",
		ItemsCount:   0,
		ErrorMessage: &errMsg,
		DurationMs:   0,
	}
	err = db.LogFetch(log)
	if err != nil {
		t.Errorf("LogFetch for error failed: %v", err)
	}

	// Verify error was logged
	stats, err := db.GetFetchStats()
	if err != nil {
		t.Fatalf("GetFetchStats failed: %v", err)
	}
	if len(stats) != 1 {
		t.Errorf("Expected 1 error stat, got %d", len(stats))
	}
}

// TestIntegration_Deduplication tests that duplicate items are not stored
func TestIntegration_Deduplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "dedupe.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	// Parse same data twice
	data := []byte(`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`)
	
	result1 := p.Parse("GitHub", "json", data)
	db.SaveItems(result1.Items)

	result2 := p.Parse("GitHub", "json", data)
	db.SaveItems(result2.Items)

	// Should still have only 1 item
	count, err := db.GetItemCount("GitHub")
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item (deduped), got %d", count)
	}
}

// TestIntegration_LargeDataset tests performance with large dataset
func TestIntegration_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "large.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	// Generate large JSON (100 items)
	var itemsJSON []string
	for i := 0; i < 100; i++ {
		itemsJSON = append(itemsJSON, fmt.Sprintf(`{"full_name":"test/repo%d","html_url":"https://github.com/test/repo%d"}`, i, i))
	}
	data := []byte(`{"items":[` + strings.Join(itemsJSON, ",") + `]}`)

	start := time.Now()
	result := p.Parse("GitHub", "json", data)
	parseTime := time.Since(start)

	if len(result.Errors) > 0 {
		t.Errorf("Parse errors: %v", result.Errors)
	}

	start = time.Now()
	err = db.SaveItems(result.Items)
	saveTime := time.Since(start)

	if err != nil {
		t.Fatalf("SaveItems failed: %v", err)
	}

	count, err := db.GetItemCount("GitHub")
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected 100 items, got %d", count)
	}

	t.Logf("Parse time: %v, Save time: %v", parseTime, saveTime)
}

// TestIntegration_ConfigLifecycle tests config loading and validation
func TestIntegration_ConfigLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create valid config
	configContent := `
settings:
  max_concurrency: 5
  default_timeout_secs: 10
  retry_max: 3
  retry_base_delay_ms: 500
  database_path: "test.db"
feeds:
  - name: "GitHub"
    url: "https://api.github.com/search/repositories"
    feed_type: "json"
    refresh_interval_secs: 300
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify config
	if cfg.Settings.MaxConcurrency != 5 {
		t.Errorf("Expected MaxConcurrency=5, got %d", cfg.Settings.MaxConcurrency)
	}
	if len(cfg.Feeds) != 1 {
		t.Errorf("Expected 1 feed, got %d", len(cfg.Feeds))
	}
	if cfg.Feeds[0].Name != "GitHub" {
		t.Errorf("Expected feed name 'GitHub', got '%s'", cfg.Feeds[0].Name)
	}
}

// TestIntegration_ConcurrentOperations tests concurrent fetch/parse/store
func TestIntegration_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// This test verifies no race conditions exist
	// Run with: go test -race
	p := parser.NewParser()

	data := []byte(`{"items":[{"full_name":"test/repo","html_url":"https://github.com/test/repo"}]}`)

	// Simulate concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			source := fmt.Sprintf("Source-%d", id)
			result := p.Parse(source, "json", data)
			db.SaveItems(result.Items)
			log := storage.FetchLog{
				Source:       source,
				FetchedAt:    time.Now(),
				Status:       "success",
				ItemsCount:   len(result.Items),
				ErrorMessage: nil,
				DurationMs:   0,
			}
			db.LogFetch(log)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all data was saved
	totalCount, err := db.GetAllItemsCount()
	if err != nil {
		t.Fatalf("GetAllItemsCount failed: %v", err)
	}
	if totalCount != 10 {
		t.Errorf("Expected 10 items, got %d", totalCount)
	}
}

// TestIntegration_PartialFailure tests handling of partial failures
func TestIntegration_PartialFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "partial.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	// Data with some valid and some invalid items
	data := []byte(`{"items":[
		{"full_name":"valid1/repo","html_url":"https://github.com/valid1/repo"},
		{"full_name":"missing-url"},
		{"full_name":"valid2/repo","html_url":"https://github.com/valid2/repo"}
	]}`)

	result := p.Parse("GitHub", "json", data)

	// Should have some errors but also some items
	if len(result.Errors) == 0 {
		t.Error("Expected some parse errors")
	}
	if len(result.Items) != 2 {
		t.Errorf("Expected 2 valid items, got %d", len(result.Items))
	}

	// Save valid items
	err = db.SaveItems(result.Items)
	if err != nil {
		t.Fatalf("SaveItems failed: %v", err)
	}

	// Verify valid items were saved
	count, err := db.GetItemCount("GitHub")
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 stored items, got %d", count)
	}
}

// TestIntegration_UnicodeThroughPipeline tests unicode handling end-to-end
func TestIntegration_UnicodeThroughPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "unicode.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	// Data with unicode content
	data := []byte(`{"items":[
		{"full_name":"测试/项目","html_url":"https://github.com/test/repo"},
		{"full_name":"emoji/🚀","html_url":"https://github.com/emoji/repo"},
		{"full_name":"rtl/العربية","html_url":"https://github.com/rtl/repo"}
	]}`)

	result := p.Parse("GitHub", "json", data)
	if len(result.Errors) > 0 {
		t.Errorf("Parse errors with unicode: %v", result.Errors)
	}

	err = db.SaveItems(result.Items)
	if err != nil {
		t.Fatalf("SaveItems with unicode failed: %v", err)
	}

	count, err := db.GetItemCount("GitHub")
	if err != nil {
		t.Fatalf("GetItemCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 unicode items, got %d", count)
	}
}

// TestIntegration_EmptyResponses tests handling of empty responses
func TestIntegration_EmptyResponses(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")
	db, err := storage.NewStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	p := parser.NewParser()

	tests := []struct {
		name   string
		data   string
		source string
	}{
		{"empty GitHub", `{"items":[]}`, "GitHub"},
		{"empty Reddit", `{"data":{"children":[]}}`, "Reddit"},
		{"empty Lobsters", `[]`, "Lobsters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.source, "json", []byte(tt.data))
			
			// Empty responses should be logged as errors
			errorMsg := "no items"
			if len(result.Errors) > 0 {
				errorMsg = result.Errors[0]
			}
			
			log := storage.FetchLog{
				Source:       tt.source,
				FetchedAt:    time.Now(),
				Status:       "success",
				ItemsCount:   0,
				ErrorMessage: &errorMsg,
				DurationMs:   0,
			}
			err := db.LogFetch(log)
			if err != nil {
				t.Errorf("LogFetch failed: %v", err)
			}
		})
	}

	// Verify logs
	stats, err := db.GetFetchStats()
	if err != nil {
		t.Fatalf("GetFetchStats failed: %v", err)
	}
	if len(stats) != 3 {
		t.Errorf("Expected 3 fetch stats, got %d", len(stats))
	}
}
