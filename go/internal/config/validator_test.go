package config

import (
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errText string
	}{
		// Valid URLs
		{"valid http", "http://example.com", false, ""},
		{"valid https", "https://example.com", false, ""},
		{"with path", "https://example.com/path/to/feed", false, ""},
		{"with query", "https://example.com?param=value", false, ""},
		{"with port", "http://example.com:8080", false, ""},
		{"with auth", "https://user:pass@example.com", false, ""},

		// Invalid URLs
		{"empty", "", true, "cannot be empty"},
		{"no scheme", "example.com", true, "invalid URL format"},
		{"invalid scheme ftp", "ftp://example.com", true, "must use HTTP or HTTPS"},
		{"invalid scheme file", "file:///path/to/file", true, "must use HTTP or HTTPS"},
		{"no host", "http://", true, "must have a host"},
		{"malformed", "http://[invalid", true, "invalid URL format"},
		{"only scheme", "http:", true, "must have a host"},
		{"spaces", "http://exam ple.com", true, "invalid URL format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateURL() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		wantErr bool
		errText string
	}{
		{"valid 1", 1, false, ""},
		{"valid 30", 30, false, ""},
		{"valid 300", 300, false, ""},
		{"valid max", 3600, false, ""},
		{"zero", 0, true, "must be positive"},
		{"negative", -1, true, "must be positive"},
		{"too large", 3601, true, "too large"},
		{"way too large", 100000, true, "too large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateTimeout() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
		errText     string
	}{
		{"valid 1", 1, false, ""},
		{"valid 5", 5, false, ""},
		{"valid 25", 25, false, ""},
		{"valid max", 50, false, ""},
		{"zero", 0, true, "must be at least 1"},
		{"negative", -1, true, "must be at least 1"},
		{"too high", 51, true, "too high"},
		{"way too high", 1000, true, "too high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConcurrency(tt.concurrency)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConcurrency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateConcurrency() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateFeedName(t *testing.T) {
	longName := strings.Repeat("a", 101)

	tests := []struct {
		name     string
		feedName string
		wantErr  bool
		errText  string
	}{
		{"valid simple", "MyFeed", false, ""},
		{"valid with spaces", "My Feed Name", false, ""},
		{"valid with unicode", "测试Feed🚀", false, ""},
		{"valid with dashes", "my-feed-name", false, ""},
		{"valid with underscores", "my_feed_name", false, ""},
		{"empty", "", true, "cannot be empty"},
		{"too long", longName, true, "too long"},
		{"control char null", "feed\x00name", true, "invalid control characters"},
		{"control char newline", "feed\nname", true, "invalid control characters"},
		{"control char del", "feed\x7fname", true, "invalid control characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeedName(tt.feedName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFeedName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateFeedName() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateFeedType(t *testing.T) {
	tests := []struct {
		name     string
		feedType string
		wantErr  bool
		errText  string
	}{
		{"valid json", "json", false, ""},
		{"valid rss", "rss", false, ""},
		{"valid atom", "atom", false, ""},
		{"empty", "", true, "cannot be empty"},
		{"invalid xml", "xml", true, "must be one of"},
		{"invalid html", "html", true, "must be one of"},
		{"case sensitive", "JSON", true, "must be one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeedType(tt.feedType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFeedType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateFeedType() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateRefreshInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		wantErr  bool
		errText  string
	}{
		{"valid 60", 60, false, ""},
		{"valid 300", 300, false, ""},
		{"valid 3600", 3600, false, ""},
		{"valid max", 86400, false, ""},
		{"zero (default)", 0, false, ""},
		{"negative", -1, true, "must be non-negative"},
		{"too short", 30, true, "too short"},
		{"too short 59", 59, true, "too short"},
		{"too long", 86401, true, "too long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRefreshInterval(tt.interval)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRefreshInterval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateRefreshInterval() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateRetryMax(t *testing.T) {
	tests := []struct {
		name     string
		retryMax int
		wantErr  bool
		errText  string
	}{
		{"valid 0", 0, false, ""},
		{"valid 3", 3, false, ""},
		{"valid 10", 10, false, ""},
		{"negative", -1, true, "must be non-negative"},
		{"too high", 11, true, "too high"},
		{"way too high", 100, true, "too high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRetryMax(tt.retryMax)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRetryMax() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateRetryMax() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateRetryDelay(t *testing.T) {
	tests := []struct {
		name    string
		delay   int
		wantErr bool
		errText string
	}{
		{"valid 0", 0, false, ""},
		{"valid 100", 100, false, ""},
		{"valid 1000", 1000, false, ""},
		{"valid max", 60000, false, ""},
		{"negative", -1, true, "must be non-negative"},
		{"too high", 60001, true, "too high"},
		{"way too high", 1000000, true, "too high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRetryDelay(tt.delay)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRetryDelay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateRetryDelay() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateDatabasePath(t *testing.T) {
	longPath := strings.Repeat("a", 4097)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errText string
	}{
		{"valid simple", "feedpulse.db", false, ""},
		{"valid absolute", "/var/lib/feedpulse.db", false, ""},
		{"valid relative", "./data/feedpulse.db", false, ""},
		{"valid with spaces", "my database.db", false, ""},
		{"empty", "", true, "cannot be empty"},
		{"too long", longPath, true, "too long"},
		{"null byte", "feed\x00pulse.db", true, "invalid characters"},
		{"newline", "feedpulse\n.db", true, "invalid characters"},
		{"carriage return", "feedpulse\r.db", true, "invalid characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDatabasePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDatabasePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateDatabasePath() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateFeedConfig(t *testing.T) {
	tests := []struct {
		name    string
		feed    *Feed
		wantErr bool
		errText string
	}{
		{
			name: "valid feed",
			feed: &Feed{
				Name:                "TestFeed",
				URL:                 "https://example.com/feed",
				FeedType:            "json",
				RefreshIntervalSecs: 300,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			feed: &Feed{
				URL:      "https://example.com/feed",
				FeedType: "json",
			},
			wantErr: true,
			errText: "cannot be empty",
		},
		{
			name: "invalid URL",
			feed: &Feed{
				Name:     "TestFeed",
				URL:      "ftp://example.com",
				FeedType: "json",
			},
			wantErr: true,
			errText: "must use HTTP or HTTPS",
		},
		{
			name: "invalid feed type",
			feed: &Feed{
				Name:     "TestFeed",
				URL:      "https://example.com/feed",
				FeedType: "invalid",
			},
			wantErr: true,
			errText: "must be one of",
		},
		{
			name: "invalid refresh interval",
			feed: &Feed{
				Name:                "TestFeed",
				URL:                 "https://example.com/feed",
				FeedType:            "json",
				RefreshIntervalSecs: 30,
			},
			wantErr: true,
			errText: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeedConfig(tt.feed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFeedConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateFeedConfig() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
		wantErr  bool
		errText  string
	}{
		{
			name: "valid settings",
			settings: &Settings{
				MaxConcurrency:     5,
				DefaultTimeoutSecs: 30,
				RetryMax:           3,
				RetryBaseDelayMs:   500,
				DatabasePath:       "feedpulse.db",
			},
			wantErr: false,
		},
		{
			name: "invalid concurrency",
			settings: &Settings{
				MaxConcurrency:     0,
				DefaultTimeoutSecs: 30,
				RetryMax:           3,
				RetryBaseDelayMs:   500,
				DatabasePath:       "feedpulse.db",
			},
			wantErr: true,
			errText: "must be at least 1",
		},
		{
			name: "invalid timeout",
			settings: &Settings{
				MaxConcurrency:     5,
				DefaultTimeoutSecs: 0,
				RetryMax:           3,
				RetryBaseDelayMs:   500,
				DatabasePath:       "feedpulse.db",
			},
			wantErr: true,
			errText: "must be positive",
		},
		{
			name: "invalid retry max",
			settings: &Settings{
				MaxConcurrency:     5,
				DefaultTimeoutSecs: 30,
				RetryMax:           -1,
				RetryBaseDelayMs:   500,
				DatabasePath:       "feedpulse.db",
			},
			wantErr: true,
			errText: "must be non-negative",
		},
		{
			name: "invalid retry delay",
			settings: &Settings{
				MaxConcurrency:     5,
				DefaultTimeoutSecs: 30,
				RetryMax:           3,
				RetryBaseDelayMs:   -1,
				DatabasePath:       "feedpulse.db",
			},
			wantErr: true,
			errText: "must be non-negative",
		},
		{
			name: "empty database path",
			settings: &Settings{
				MaxConcurrency:     5,
				DefaultTimeoutSecs: 30,
				RetryMax:           3,
				RetryBaseDelayMs:   500,
				DatabasePath:       "",
			},
			wantErr: true,
			errText: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSettings(tt.settings)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSettings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("ValidateSettings() error = %v, want error containing %q", err, tt.errText)
			}
		})
	}
}

func TestValidateURL_Unicode(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"unicode domain", "https://测试.com", false},
		{"unicode path", "https://example.com/测试", false},
		{"emoji in path", "https://example.com/🚀", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"ipv4", "http://192.168.1.1", false},
		{"ipv6", "http://[2001:db8::1]", false},
		{"localhost", "http://localhost", false},
		{"very long domain", "https://" + strings.Repeat("a", 200) + ".com", false},
		// Note: fragment-only URLs fail because they have no host
		{"with fragment", "https://example.com/path#fragment", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
