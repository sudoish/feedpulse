package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test database initialization with various paths
func TestStorage_DatabasePaths(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"simple name", "test.db", false},
		{"relative path", "./data/test.db", false},
		{"nested path", "a/b/c/test.db", false},
		{"with spaces", "my database.db", false},
		{"unicode", "数据库.db", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, tt.path)

			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
				t.Fatal(err)
			}

			storage, err := NewStorage(dbPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && storage == nil {
				t.Error("Expected non-nil storage")
			}
		})
	}
}

// Test saving items with special characters
func TestSaveItems_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		title string
		url   string
	}{
		{"HTML entities", "Test &amp; &lt;script&gt;", "https://example.com/1"},
		{"quotes", `Title with "quotes"`, "https://example.com/2"},
		{"single quotes", "Title with 'quotes'", "https://example.com/3"},
		{"backslash", `Title\with\backslashes`, "https://example.com/4"},
		{"newlines", "Title\nwith\nnewlines", "https://example.com/5"},
		{"tabs", "Title\twith\ttabs", "https://example.com/6"},
		{"null bytes", "Title\x00with\x00nulls", "https://example.com/7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use unique source per test to avoid cross-test contamination
			source := "Test-" + tt.name
			item := FeedItem{
				ID:     "test-" + tt.name,
				Title:  tt.title,
				URL:    tt.url,
				Source: source,
			}

			err := storage.SaveItems([]FeedItem{item})
			if err != nil {
				t.Errorf("SaveItems() error = %v", err)
			}

			// Verify item was saved correctly
			count, err := storage.GetItemCount(source)
			if err != nil {
				t.Errorf("GetItemCount() error = %v", err)
			}
			if count != 1 {
				t.Errorf("Expected 1 item saved, got %d", count)
			}
		})
	}
}

// Test saving items with unicode content
func TestSaveItems_Unicode(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		title string
	}{
		{"Chinese", "测试标题"},
		{"Japanese", "テストタイトル"},
		{"Korean", "테스트 제목"},
		{"Arabic RTL", "عنوان الاختبار"},
		{"Hebrew RTL", "כותרת בדיקה"},
		{"Emoji", "Test 🚀 Title 🎉"},
		{"Mixed", "Test 测试 🚀 عنوان"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := FeedItem{
				ID:     "test-" + tt.name,
				Title:  tt.title,
				URL:    "https://example.com/" + tt.name,
				Source: "Unicode",
			}

			err := storage.SaveItems([]FeedItem{item})
			if err != nil {
				t.Errorf("SaveItems() error = %v", err)
			}

			count, err := storage.GetItemCount("Unicode")
			if err != nil {
				t.Errorf("GetItemCount() error = %v", err)
			}
			if count == 0 {
				t.Error("Expected unicode item to be saved")
			}
		})
	}
}

// Test very long field values
func TestSaveItems_VeryLongValues(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	longTitle := strings.Repeat("a", 10000)
	longURL := "https://example.com/" + strings.Repeat("path/", 500)

	item := FeedItem{
		ID:     "test-long",
		Title:  longTitle,
		URL:    longURL,
		Source: "LongTest",
	}

	err = storage.SaveItems([]FeedItem{item})
	if err != nil {
		t.Errorf("SaveItems() with long values error = %v", err)
	}

	count, err := storage.GetItemCount("LongTest")
	if err != nil {
		t.Errorf("GetItemCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item with long values, got %d", count)
	}
}

// Test saving many items at once
func TestSaveItems_LargeBatch(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Create 1000 items
	items := make([]FeedItem, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = FeedItem{
			ID:     fmt.Sprintf("item-%d", i),
			Title:  fmt.Sprintf("Item %d", i),
			URL:    fmt.Sprintf("https://example.com/item%d", i),
			Source: "BatchTest",
		}
	}

	start := time.Now()
	err = storage.SaveItems(items)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("SaveItems() large batch error = %v", err)
	}

	count, err := storage.GetItemCount("BatchTest")
	if err != nil {
		t.Errorf("GetItemCount() error = %v", err)
	}
	if count != 1000 {
		t.Errorf("Expected 1000 items, got %d", count)
	}

	t.Logf("Saved 1000 items in %v", duration)
}

// Test deduplication with same ID
func TestSaveItems_DeduplicationByID(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	item := FeedItem{
		ID:     "duplicate-test",
		Title:  "Original Title",
		URL:    "https://example.com/original",
		Source: "DedupeTest",
	}

	// Save first time
	err = storage.SaveItems([]FeedItem{item})
	if err != nil {
		t.Errorf("SaveItems() first time error = %v", err)
	}

	// Try to save again with different title/URL but same ID
	item.Title = "Updated Title"
	item.URL = "https://example.com/updated"
	err = storage.SaveItems([]FeedItem{item})
	if err != nil {
		t.Errorf("SaveItems() second time error = %v", err)
	}

	// Should still have only 1 item
	count, err := storage.GetItemCount("DedupeTest")
	if err != nil {
		t.Errorf("GetItemCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item (deduped), got %d", count)
	}
}

// Test saving items with all optional fields
func TestSaveItems_AllFields(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	timestamp := "2024-01-01T12:00:00Z"
	item := FeedItem{
		ID:        "full-item",
		Title:     "Full Item",
		URL:       "https://example.com/full",
		Source:    "FullTest",
		Timestamp: &timestamp,
		Tags:      []string{"tag1", "tag2", "tag3"},
		CreatedAt: time.Now(),
	}

	err = storage.SaveItems([]FeedItem{item})
	if err != nil {
		t.Errorf("SaveItems() with all fields error = %v", err)
	}

	count, err := storage.GetItemCount("FullTest")
	if err != nil {
		t.Errorf("GetItemCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item with all fields, got %d", count)
	}
}

// Test saving items with no tags
func TestSaveItems_NoTags(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	item := FeedItem{
		ID:     "notags-item",
		Title:  "Item Without Tags",
		URL:    "https://example.com/notags",
		Source: "NoTagsTest",
		Tags:   []string{},
	}

	err = storage.SaveItems([]FeedItem{item})
	if err != nil {
		t.Errorf("SaveItems() without tags error = %v", err)
	}
}

// Test logging fetch with various statuses
func TestLogFetch_AllStatuses(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		status string
		count  int
		errMsg string
	}{
		{"success", "success", 10, ""},
		{"empty", "success", 0, ""},
		{"error", "error", 0, "connection timeout"},
		{"network error", "error", 0, "DNS lookup failed"},
		{"parse error", "error", 0, "malformed JSON"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errPtr *string
			if tt.errMsg != "" {
				errPtr = &tt.errMsg
			}
			log := FetchLog{
				Source:       tt.name,
				FetchedAt:    time.Now(),
				Status:       tt.status,
				ItemsCount:   tt.count,
				ErrorMessage: errPtr,
				DurationMs:   0,
			}
			err := storage.LogFetch(log)
			if err != nil {
				t.Errorf("LogFetch() error = %v", err)
			}
		})
	}

	// Verify all fetches were logged
	stats, err := storage.GetFetchStats()
	if err != nil {
		t.Errorf("GetFetchStats() error = %v", err)
	}
	if len(stats) != len(tests) {
		t.Errorf("Expected %d fetch stats, got %d", len(tests), len(stats))
	}
}

// Test concurrent reads
func TestStorage_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Save some test data
	items := make([]FeedItem, 10)
	for i := 0; i < 10; i++ {
		items[i] = FeedItem{
			ID:     "item-" + string(rune(i)),
			Title:  "Item " + string(rune(i)),
			URL:    "https://example.com/" + string(rune(i)),
			Source: "ConcurrentTest",
		}
	}
	storage.SaveItems(items)

	// Perform concurrent reads
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := storage.GetItemCount("ConcurrentTest")
			if err != nil {
				t.Errorf("Concurrent GetItemCount() error = %v", err)
			}
		}()
	}
	wg.Wait()
}

// Test concurrent writes to different sources
func TestStorage_ConcurrentWrites_DifferentSources(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			source := fmt.Sprintf("Source-%d", id)
			items := []FeedItem{
				{
					ID:     fmt.Sprintf("%s-item1", source),
					Title:  fmt.Sprintf("Item 1 from %s", source),
					URL:    fmt.Sprintf("https://example.com/%s/1", source),
					Source: source,
				},
				{
					ID:     fmt.Sprintf("%s-item2", source),
					Title:  fmt.Sprintf("Item 2 from %s", source),
					URL:    fmt.Sprintf("https://example.com/%s/2", source),
					Source: source,
				},
			}

			err := storage.SaveItems(items)
			if err != nil {
				t.Errorf("Concurrent SaveItems() for %s error = %v", source, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all items were saved
	totalCount, err := storage.GetAllItemsCount()
	if err != nil {
		t.Errorf("GetAllItemsCount() error = %v", err)
	}
	expectedCount := numGoroutines * 2
	if totalCount != expectedCount {
		t.Errorf("Expected %d total items, got %d", expectedCount, totalCount)
	}
}

// Test empty database queries
func TestStorage_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Query empty database
	count, err := storage.GetItemCount("NonExistent")
	if err != nil {
		t.Errorf("GetItemCount() on empty db error = %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 items in empty db, got %d", count)
	}

	totalCount, err := storage.GetAllItemsCount()
	if err != nil {
		t.Errorf("GetAllItemsCount() on empty db error = %v", err)
	}
	if totalCount != 0 {
		t.Errorf("Expected 0 total items, got %d", totalCount)
	}

	stats, err := storage.GetFetchStats()
	if err != nil {
		t.Errorf("GetFetchStats() on empty db error = %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("Expected 0 fetch stats, got %d", len(stats))
	}
}

// Test database file permissions
func TestStorage_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewStorage(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	// Check database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Save some data to ensure file is written
	item := FeedItem{
		ID:     "perm-test",
		Title:  "Permission Test",
		URL:    "https://example.com/perm",
		Source: "PermTest",
	}
	storage.SaveItems([]FeedItem{item})

	// File should be readable
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Errorf("Failed to stat database file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Database file is empty")
	}
}

// Test saving items with nil timestamp
func TestSaveItems_NilTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	item := FeedItem{
		ID:        "nil-timestamp",
		Title:     "Item with nil timestamp",
		URL:       "https://example.com/nil",
		Source:    "NilTest",
		Timestamp: nil,
	}

	err = storage.SaveItems([]FeedItem{item})
	if err != nil {
		t.Errorf("SaveItems() with nil timestamp error = %v", err)
	}
}

// Test multiple fetch logs for same source
func TestLogFetch_MultipleForSameSource(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Log multiple fetches
	for i := 0; i < 5; i++ {
		log := FetchLog{
			Source:       "TestSource",
			FetchedAt:    time.Now(),
			Status:       "success",
			ItemsCount:   i * 10,
			ErrorMessage: nil,
			DurationMs:   0,
		}
		err := storage.LogFetch(log)
		if err != nil {
			t.Errorf("LogFetch() iteration %d error = %v", i, err)
		}
		time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	}

	stats, err := storage.GetFetchStats()
	if err != nil {
		t.Errorf("GetFetchStats() error = %v", err)
	}

	// GetFetchStats returns aggregated stats by source, so we expect 1 entry for "TestSource"
	if len(stats) != 1 {
		t.Errorf("Expected 1 aggregated stat, got %d", len(stats))
	}
	
	// Verify the stat has correct count
	if len(stats) > 0 && stats[0].TotalFetches != 5 {
		t.Errorf("Expected 5 total fetches, got %d", stats[0].TotalFetches)
	}
}

// Test item count by source accuracy
func TestGetItemCount_Accuracy(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Save items from multiple sources
	sources := []string{"Source1", "Source2", "Source3"}
	itemsPerSource := []int{5, 10, 15}

	for i, source := range sources {
		items := make([]FeedItem, itemsPerSource[i])
		for j := 0; j < itemsPerSource[i]; j++ {
			items[j] = FeedItem{
				ID:     fmt.Sprintf("%s-item-%d", source, j),
				Title:  fmt.Sprintf("Item %d", j),
				URL:    fmt.Sprintf("https://example.com/%s/%d", source, j),
				Source: source,
			}
		}
		storage.SaveItems(items)
	}

	// Verify counts for each source
	for i, source := range sources {
		count, err := storage.GetItemCount(source)
		if err != nil {
			t.Errorf("GetItemCount(%s) error = %v", source, err)
		}
		if count != itemsPerSource[i] {
			t.Errorf("Expected %d items for %s, got %d", itemsPerSource[i], source, count)
		}
	}

	// Verify total count
	totalExpected := 5 + 10 + 15
	totalActual, err := storage.GetAllItemsCount()
	if err != nil {
		t.Errorf("GetAllItemsCount() error = %v", err)
	}
	if totalActual != totalExpected {
		t.Errorf("Expected %d total items, got %d", totalExpected, totalActual)
	}
}

// Test saving same item multiple times (idempotency)
func TestSaveItems_Idempotency(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	item := FeedItem{
		ID:     "idempotent-test",
		Title:  "Idempotent Item",
		URL:    "https://example.com/idempotent",
		Source: "IdempotentTest",
	}

	// Save same item 10 times
	for i := 0; i < 10; i++ {
		err := storage.SaveItems([]FeedItem{item})
		if err != nil {
			t.Errorf("SaveItems() iteration %d error = %v", i, err)
		}
	}

	// Should still have only 1 item
	count, err := storage.GetItemCount("IdempotentTest")
	if err != nil {
		t.Errorf("GetItemCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 item (idempotent), got %d", count)
	}
}
