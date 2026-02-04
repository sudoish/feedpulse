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
