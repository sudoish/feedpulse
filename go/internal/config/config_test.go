package config

import (
	"os"
	"testing"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
	if err.Error() != "config file not found: /nonexistent/config.yaml" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create temp file with invalid YAML
	tmpfile, err := os.CreateTemp("", "invalid-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
settings:
  max_concurrency: "not a number"
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}

func TestValidate_MaxConcurrencyOutOfRange(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     100,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:                "Test",
				URL:                 "https://example.com",
				FeedType:            "json",
				RefreshIntervalSecs: 300,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for max_concurrency > 50")
	}
}

func TestValidate_MissingFeedName(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "",
				URL:      "https://example.com",
				FeedType: "json",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for missing feed name")
	}
}

func TestValidate_InvalidURL(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "Test",
				URL:      "not a url",
				FeedType: "json",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

func TestValidate_InvalidFeedType(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "Test",
				URL:      "https://example.com",
				FeedType: "invalid",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid feed_type")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:                "Test",
				URL:                 "https://example.com",
				FeedType:            "json",
				RefreshIntervalSecs: 300,
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

// Additional comprehensive tests

func TestLoadConfig_EmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "empty-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		t.Fatal("Expected error for empty config")
	}
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "defaults-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
settings: {}
feeds:
  - name: "TestFeed"
    url: "https://example.com"
    feed_type: "json"
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check defaults were applied
	if cfg.Settings.MaxConcurrency != 5 {
		t.Errorf("Expected default MaxConcurrency=5, got %d", cfg.Settings.MaxConcurrency)
	}
	if cfg.Settings.DefaultTimeoutSecs != 10 {
		t.Errorf("Expected default DefaultTimeoutSecs=10, got %d", cfg.Settings.DefaultTimeoutSecs)
	}
	if cfg.Settings.RetryMax != 3 {
		t.Errorf("Expected default RetryMax=3, got %d", cfg.Settings.RetryMax)
	}
	if cfg.Settings.RetryBaseDelayMs != 500 {
		t.Errorf("Expected default RetryBaseDelayMs=500, got %d", cfg.Settings.RetryBaseDelayMs)
	}
	if cfg.Settings.DatabasePath != "feedpulse.db" {
		t.Errorf("Expected default DatabasePath='feedpulse.db', got %s", cfg.Settings.DatabasePath)
	}
	// Note: RefreshIntervalSecs default is only applied during validation, not in LoadConfig
	// So we can't test it here without calling Validate()
}

func TestLoadConfig_WithHeaders(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "headers-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
settings:
  max_concurrency: 5
  default_timeout_secs: 10
  retry_max: 3
  retry_base_delay_ms: 500
  database_path: "test.db"
feeds:
  - name: "TestFeed"
    url: "https://example.com"
    feed_type: "json"
    headers:
      Authorization: "Bearer token123"
      User-Agent: "FeedPulse/1.0"
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(cfg.Feeds[0].Headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(cfg.Feeds[0].Headers))
	}
	if cfg.Feeds[0].Headers["Authorization"] != "Bearer token123" {
		t.Errorf("Unexpected Authorization header: %s", cfg.Feeds[0].Headers["Authorization"])
	}
}

func TestLoadConfig_MultipleFeeds(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "multifeeds-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
settings:
  max_concurrency: 10
  database_path: "feeds.db"
feeds:
  - name: "Feed1"
    url: "https://example.com/1"
    feed_type: "json"
  - name: "Feed2"
    url: "https://example.com/2"
    feed_type: "rss"
  - name: "Feed3"
    url: "https://example.com/3"
    feed_type: "atom"
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(cfg.Feeds) != 3 {
		t.Errorf("Expected 3 feeds, got %d", len(cfg.Feeds))
	}
}

func TestValidate_NoFeeds(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for no feeds")
	}
	if err.Error() != "no feeds configured" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestValidate_NegativeTimeout(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: -1,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "Test",
				URL:      "https://example.com",
				FeedType: "json",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for negative timeout")
	}
}

func TestValidate_NegativeRetryMax(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           -1,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "Test",
				URL:      "https://example.com",
				FeedType: "json",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for negative retry_max")
	}
}

func TestValidate_NegativeRetryDelay(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   -1,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "Test",
				URL:      "https://example.com",
				FeedType: "json",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for negative retry_base_delay_ms")
	}
}

func TestValidate_URLSchemes(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"http valid", "http://example.com", false},
		{"https valid", "https://example.com", false},
		{"ftp invalid", "ftp://example.com", true},
		{"file invalid", "file:///path", true},
		{"no scheme", "example.com", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Settings: Settings{
					MaxConcurrency:     5,
					DefaultTimeoutSecs: 10,
					RetryMax:           3,
					RetryBaseDelayMs:   500,
					DatabasePath:       "test.db",
				},
				Feeds: []Feed{
					{
						Name:     "Test",
						URL:      tt.url,
						FeedType: "json",
					},
				},
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("URL %s: got error=%v, wantErr=%v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_FeedTypes(t *testing.T) {
	tests := []struct {
		name     string
		feedType string
		wantErr  bool
	}{
		{"json valid", "json", false},
		{"rss valid", "rss", false},
		{"atom valid", "atom", false},
		{"xml invalid", "xml", true},
		{"html invalid", "html", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Settings: Settings{
					MaxConcurrency:     5,
					DefaultTimeoutSecs: 10,
					RetryMax:           3,
					RetryBaseDelayMs:   500,
					DatabasePath:       "test.db",
				},
				Feeds: []Feed{
					{
						Name:     "Test",
						URL:      "https://example.com",
						FeedType: tt.feedType,
					},
				},
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FeedType %s: got error=%v, wantErr=%v", tt.feedType, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ConcurrencyBounds(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
	}{
		{"min valid", 1, false},
		{"mid valid", 25, false},
		{"max valid", 50, false},
		{"zero invalid", 0, true},
		{"negative invalid", -1, true},
		{"too high", 51, true},
		{"way too high", 1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Settings: Settings{
					MaxConcurrency:     tt.concurrency,
					DefaultTimeoutSecs: 10,
					RetryMax:           3,
					RetryBaseDelayMs:   500,
					DatabasePath:       "test.db",
				},
				Feeds: []Feed{
					{
						Name:     "Test",
						URL:      "https://example.com",
						FeedType: "json",
					},
				},
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Concurrency %d: got error=%v, wantErr=%v", tt.concurrency, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_RefreshInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		wantErr  bool
	}{
		{"valid 300", 300, false},
		{"valid 0 (default)", 0, false},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed := &Feed{
				Name:                "Test",
				URL:                 "https://example.com",
				FeedType:            "json",
				RefreshIntervalSecs: tt.interval,
			}

			err := feed.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshInterval %d: got error=%v, wantErr=%v", tt.interval, err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_UnicodeContent(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "unicode-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
settings:
  database_path: "测试.db"
feeds:
  - name: "中文Feed"
    url: "https://example.com/测试"
    feed_type: "json"
  - name: "Emoji Feed 🚀"
    url: "https://example.com/feed"
    feed_type: "json"
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Unexpected error with unicode: %v", err)
	}

	if cfg.Settings.DatabasePath != "测试.db" {
		t.Errorf("Expected unicode database path, got: %s", cfg.Settings.DatabasePath)
	}
	if cfg.Feeds[0].Name != "中文Feed" {
		t.Errorf("Expected unicode feed name, got: %s", cfg.Feeds[0].Name)
	}
	if cfg.Feeds[1].Name != "Emoji Feed 🚀" {
		t.Errorf("Expected emoji in feed name, got: %s", cfg.Feeds[1].Name)
	}
}

func TestLoadConfig_VeryLongValues(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "long-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	longURL := "https://example.com/" + string(make([]byte, 1000))
	for i := range longURL[19:] {
		longURL = longURL[:19+i] + "a" + longURL[20+i:]
	}

	content := `
settings:
  database_path: "test.db"
feeds:
  - name: "TestFeed"
    url: "` + longURL + `"
    feed_type: "json"
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Unexpected error with long URL: %v", err)
	}

	if len(cfg.Feeds[0].URL) < 1000 {
		t.Errorf("Expected long URL to be preserved, got length: %d", len(cfg.Feeds[0].URL))
	}
}

func TestValidate_AllFeedTypes(t *testing.T) {
	cfg := &Config{
		Settings: Settings{
			MaxConcurrency:     5,
			DefaultTimeoutSecs: 10,
			RetryMax:           3,
			RetryBaseDelayMs:   500,
			DatabasePath:       "test.db",
		},
		Feeds: []Feed{
			{
				Name:     "JSONFeed",
				URL:      "https://example.com/json",
				FeedType: "json",
			},
			{
				Name:     "RSSFeed",
				URL:      "https://example.com/rss",
				FeedType: "rss",
			},
			{
				Name:     "AtomFeed",
				URL:      "https://example.com/atom",
				FeedType: "atom",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected all feed types to be valid, got error: %v", err)
	}
}

func TestValidate_EdgeCaseMaxConcurrency(t *testing.T) {
	// Test boundary conditions
	tests := []struct {
		value   int
		wantErr bool
	}{
		{1, false},   // min valid
		{50, false},  // max valid
		{0, true},    // just below min
		{51, true},   // just above max
		{-1, true},   // negative
		{100, true},  // way too high
	}

	for _, tt := range tests {
		cfg := &Config{
			Settings: Settings{
				MaxConcurrency:     tt.value,
				DefaultTimeoutSecs: 10,
				RetryMax:           3,
				RetryBaseDelayMs:   500,
				DatabasePath:       "test.db",
			},
			Feeds: []Feed{{Name: "Test", URL: "https://example.com", FeedType: "json"}},
		}

		err := cfg.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("MaxConcurrency=%d: got error=%v, wantErr=%v", tt.value, err, tt.wantErr)
		}
	}
}
