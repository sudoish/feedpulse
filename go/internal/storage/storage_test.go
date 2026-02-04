package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	store, err := NewStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestSaveItems_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	items := []FeedItem{
		{
			ID:        "test123",
			Title:     "Test Item",
			URL:       "https://example.com",
			Source:    "TestSource",
			CreatedAt: time.Now(),
		},
	}
	
	err = store.SaveItems(items)
	if err != nil {
		t.Fatalf("failed to save items: %v", err)
	}
	
	// Verify item was saved
	count, err := store.GetItemCount("TestSource")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 item, got %d", count)
	}
}

func TestSaveItems_Deduplication(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	item := FeedItem{
		ID:        "test123",
		Title:     "Test Item",
		URL:       "https://example.com",
		Source:    "TestSource",
		CreatedAt: time.Now(),
	}
	
	// Save same item twice
	err = store.SaveItems([]FeedItem{item})
	if err != nil {
		t.Fatalf("failed to save items: %v", err)
	}
	
	item.Title = "Updated Title"
	err = store.SaveItems([]FeedItem{item})
	if err != nil {
		t.Fatalf("failed to save items: %v", err)
	}
	
	// Should still be only 1 item (deduplication)
	count, err := store.GetItemCount("TestSource")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 item (deduplicated), got %d", count)
	}
}

func TestSaveItems_WithTags(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	items := []FeedItem{
		{
			ID:        "test123",
			Title:     "Test Item",
			URL:       "https://example.com",
			Source:    "TestSource",
			Tags:      []string{"tag1", "tag2"},
			CreatedAt: time.Now(),
		},
	}
	
	err = store.SaveItems(items)
	if err != nil {
		t.Fatalf("failed to save items with tags: %v", err)
	}
}

func TestSaveItems_WithTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	timestamp := "2025-01-01T00:00:00Z"
	items := []FeedItem{
		{
			ID:        "test123",
			Title:     "Test Item",
			URL:       "https://example.com",
			Source:    "TestSource",
			Timestamp: &timestamp,
			CreatedAt: time.Now(),
		},
	}
	
	err = store.SaveItems(items)
	if err != nil {
		t.Fatalf("failed to save items with timestamp: %v", err)
	}
}

func TestLogFetch_Success(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	log := FetchLog{
		Source:     "TestSource",
		FetchedAt:  time.Now(),
		Status:     "success",
		ItemsCount: 10,
		DurationMs: 500,
	}
	
	err = store.LogFetch(log)
	if err != nil {
		t.Fatalf("failed to log fetch: %v", err)
	}
}

func TestLogFetch_Error(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	errMsg := "HTTP 500"
	log := FetchLog{
		Source:       "TestSource",
		FetchedAt:    time.Now(),
		Status:       "error",
		ErrorMessage: &errMsg,
		DurationMs:   1000,
	}
	
	err = store.LogFetch(log)
	if err != nil {
		t.Fatalf("failed to log fetch error: %v", err)
	}
}

func TestGetItemCount(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	// Initially zero
	count, err := store.GetItemCount("TestSource")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 items, got %d", count)
	}
	
	// Add items
	items := []FeedItem{
		{
			ID:        "test1",
			Title:     "Test 1",
			URL:       "https://example.com/1",
			Source:    "TestSource",
			CreatedAt: time.Now(),
		},
		{
			ID:        "test2",
			Title:     "Test 2",
			URL:       "https://example.com/2",
			Source:    "TestSource",
			CreatedAt: time.Now(),
		},
		{
			ID:        "test3",
			Title:     "Test 3",
			URL:       "https://example.com/3",
			Source:    "OtherSource",
			CreatedAt: time.Now(),
		},
	}
	store.SaveItems(items)
	
	// Check TestSource count
	count, err = store.GetItemCount("TestSource")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 items for TestSource, got %d", count)
	}
	
	// Check OtherSource count
	count, err = store.GetItemCount("OtherSource")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 item for OtherSource, got %d", count)
	}
}

func TestGetAllItemsCount(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	items := []FeedItem{
		{ID: "1", Title: "T1", URL: "u1", Source: "S1", CreatedAt: time.Now()},
		{ID: "2", Title: "T2", URL: "u2", Source: "S1", CreatedAt: time.Now()},
		{ID: "3", Title: "T3", URL: "u3", Source: "S2", CreatedAt: time.Now()},
	}
	store.SaveItems(items)
	
	count, err := store.GetAllItemsCount()
	if err != nil {
		t.Fatalf("failed to get total count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 total items, got %d", count)
	}
}

func TestGetFetchStats(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	// Add items
	items := []FeedItem{
		{ID: "1", Title: "T1", URL: "u1", Source: "Source1", CreatedAt: time.Now()},
		{ID: "2", Title: "T2", URL: "u2", Source: "Source1", CreatedAt: time.Now()},
	}
	store.SaveItems(items)
	
	// Log fetches
	store.LogFetch(FetchLog{
		Source:     "Source1",
		FetchedAt:  time.Now(),
		Status:     "success",
		ItemsCount: 2,
		DurationMs: 100,
	})
	
	errMsg := "test error"
	store.LogFetch(FetchLog{
		Source:       "Source2",
		FetchedAt:    time.Now(),
		Status:       "error",
		ErrorMessage: &errMsg,
		DurationMs:   200,
	})
	
	// Get stats
	stats, err := store.GetFetchStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	
	if len(stats) < 1 {
		t.Fatalf("expected at least 1 stat, got %d", len(stats))
	}
	
	// Find Source1 stats
	var source1Stats *FetchStats
	for _, stat := range stats {
		if stat.Source == "Source1" {
			source1Stats = &stat
			break
		}
	}
	
	if source1Stats == nil {
		t.Fatal("Source1 not found in stats")
	}
	
	if source1Stats.ItemsCount != 2 {
		t.Errorf("expected 2 items for Source1, got %d", source1Stats.ItemsCount)
	}
	if source1Stats.ErrorCount != 0 {
		t.Errorf("expected 0 errors for Source1, got %d", source1Stats.ErrorCount)
	}
	if source1Stats.LastSuccess == nil {
		t.Error("expected LastSuccess to be set")
	}
}

func TestSaveItems_EmptySlice(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	// Should handle empty slice gracefully
	err = store.SaveItems([]FeedItem{})
	if err != nil {
		t.Errorf("failed to handle empty items: %v", err)
	}
}

func TestConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewStorage(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()
	
	// Test concurrent writes (should be handled by WAL mode)
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			items := []FeedItem{
				{
					ID:        string(rune('A' + n)),
					Title:     "Test",
					URL:       "https://example.com",
					Source:    "TestSource",
					CreatedAt: time.Now(),
				},
			}
			store.SaveItems(items)
			done <- true
		}(i)
	}
	
	for i := 0; i < 10; i++ {
		<-done
	}
	
	count, _ := store.GetAllItemsCount()
	if count != 10 {
		t.Errorf("expected 10 items after concurrent writes, got %d", count)
	}
}
