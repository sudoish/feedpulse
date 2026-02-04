package parser

import (
	"encoding/json"
	"testing"
)

func TestParse_HackerNews(t *testing.T) {
	p := NewParser()
	data := []byte(`[1, 2, 3, 4, 5]`)
	
	result := p.Parse("HackerNews", "json", data)
	
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
	if len(result.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(result.Items))
	}
	if result.Items[0].Title != "HN Story 1" {
		t.Errorf("unexpected title: %s", result.Items[0].Title)
	}
	if result.Items[0].URL != "https://news.ycombinator.com/item?id=1" {
		t.Errorf("unexpected URL: %s", result.Items[0].URL)
	}
}

func TestParse_GitHub(t *testing.T) {
	p := NewParser()
	data := []byte(`{
		"items": [
			{
				"full_name": "test/repo",
				"html_url": "https://github.com/test/repo",
				"updated_at": "2025-01-01T00:00:00Z",
				"topics": ["go", "cli"]
			}
		]
	}`)
	
	result := p.Parse("GitHub", "json", data)
	
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Title != "test/repo" {
		t.Errorf("unexpected title: %s", result.Items[0].Title)
	}
	if len(result.Items[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Items[0].Tags))
	}
}

func TestParse_Reddit(t *testing.T) {
	p := NewParser()
	data := []byte(`{
		"data": {
			"children": [
				{
					"data": {
						"title": "Test Post",
						"url": "https://example.com",
						"created_utc": 1704067200,
						"link_flair_text": "discussion"
					}
				}
			]
		}
	}`)
	
	result := p.Parse("Reddit", "json", data)
	
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Title != "Test Post" {
		t.Errorf("unexpected title: %s", result.Items[0].Title)
	}
	if len(result.Items[0].Tags) != 1 || result.Items[0].Tags[0] != "discussion" {
		t.Errorf("unexpected tags: %v", result.Items[0].Tags)
	}
}

func TestParse_Lobsters(t *testing.T) {
	p := NewParser()
	data := []byte(`[
		{
			"title": "Test Story",
			"url": "https://example.com/story",
			"created_at": "2025-01-01T00:00:00Z",
			"tags": ["programming", "go"]
		}
	]`)
	
	result := p.Parse("Lobsters", "json", data)
	
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Title != "Test Story" {
		t.Errorf("unexpected title: %s", result.Items[0].Title)
	}
	if len(result.Items[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Items[0].Tags))
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	p := NewParser()
	data := []byte(`{invalid json`)
	
	result := p.Parse("Test", "json", data)
	
	if len(result.Errors) == 0 {
		t.Error("expected error for malformed JSON")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestParse_MissingRequiredFields(t *testing.T) {
	p := NewParser()
	
	// GitHub with missing html_url
	data := []byte(`{
		"items": [
			{
				"full_name": "test/repo"
			}
		]
	}`)
	
	result := p.Parse("GitHub", "json", data)
	
	if len(result.Errors) == 0 {
		t.Error("expected error for missing required field")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestParse_EmptyResponse(t *testing.T) {
	p := NewParser()
	data := []byte(`[]`)
	
	result := p.Parse("Test", "json", data)
	
	// Empty array should not produce errors, just no items
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestParse_TypeCoercion(t *testing.T) {
	p := NewParser()
	// Test that getString handles different types
	data := []byte(`{
		"items": [
			{
				"full_name": 12345,
				"html_url": "https://github.com/test"
			}
		]
	}`)
	
	result := p.Parse("GitHub", "json", data)
	
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item (with type coercion), got %d", len(result.Items))
	}
	if result.Items[0].Title != "12345" {
		t.Errorf("expected coerced title '12345', got '%s'", result.Items[0].Title)
	}
}

func TestGenerateID_Deterministic(t *testing.T) {
	p := NewParser()
	
	id1 := p.generateID("source1", "https://example.com")
	id2 := p.generateID("source1", "https://example.com")
	id3 := p.generateID("source2", "https://example.com")
	
	if id1 != id2 {
		t.Error("IDs should be deterministic for same input")
	}
	if id1 == id3 {
		t.Error("IDs should differ for different sources")
	}
	if len(id1) != 64 { // SHA256 hex = 64 chars
		t.Errorf("expected 64-char hex ID, got %d chars", len(id1))
	}
}

func TestParse_UnicodeHandling(t *testing.T) {
	p := NewParser()
	data := []byte(`{
		"items": [
			{
				"full_name": "test/repo-中文",
				"html_url": "https://github.com/test/repo"
			}
		]
	}`)
	
	result := p.Parse("GitHub", "json", data)
	
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors with unicode: %v", result.Errors)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
}

func TestParse_LobstersFallbackURL(t *testing.T) {
	p := NewParser()
	// Lobsters should fall back to comments_url if url is missing
	data := []byte(`[
		{
			"title": "Test Story",
			"comments_url": "https://lobste.rs/s/abc123"
		}
	]`)
	
	result := p.Parse("Lobsters", "json", data)
	
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].URL != "https://lobste.rs/s/abc123" {
		t.Errorf("expected fallback URL, got %s", result.Items[0].URL)
	}
}

func TestParse_UnknownFeedType(t *testing.T) {
	p := NewParser()
	data := []byte(`{"test": "data"}`)
	
	result := p.Parse("Test", "unknown", data)
	
	if len(result.Errors) == 0 {
		t.Error("expected error for unknown feed type")
	}
}

func TestParse_RSSNotImplemented(t *testing.T) {
	p := NewParser()
	data := []byte(`<rss></rss>`)
	
	result := p.Parse("Test", "rss", data)
	
	if len(result.Errors) == 0 {
		t.Error("expected error for RSS (not implemented)")
	}
}

// Benchmark tests
func BenchmarkParse_HackerNews(b *testing.B) {
	p := NewParser()
	var ids []int
	for i := 0; i < 500; i++ {
		ids = append(ids, i)
	}
	data, _ := json.Marshal(ids)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse("HN", "json", data)
	}
}

func BenchmarkGenerateID(b *testing.B) {
	p := NewParser()
	for i := 0; i < b.N; i++ {
		p.generateID("source", "https://example.com/test")
	}
}
