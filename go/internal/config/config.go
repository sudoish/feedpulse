package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Settings Settings `yaml:"settings"`
	Feeds    []Feed   `yaml:"feeds"`
}

// Settings contains global configuration
type Settings struct {
	MaxConcurrency     int    `yaml:"max_concurrency"`
	DefaultTimeoutSecs int    `yaml:"default_timeout_secs"`
	RetryMax           int    `yaml:"retry_max"`
	RetryBaseDelayMs   int    `yaml:"retry_base_delay_ms"`
	DatabasePath       string `yaml:"database_path"`
}

// Feed represents a single feed source
type Feed struct {
	Name                 string            `yaml:"name"`
	URL                  string            `yaml:"url"`
	FeedType             string            `yaml:"feed_type"`
	RefreshIntervalSecs  int               `yaml:"refresh_interval_secs"`
	Headers              map[string]string `yaml:"headers"`
}

// LoadConfig loads and validates the configuration file
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Apply defaults
	if cfg.Settings.MaxConcurrency == 0 {
		cfg.Settings.MaxConcurrency = 5
	}
	if cfg.Settings.DefaultTimeoutSecs == 0 {
		cfg.Settings.DefaultTimeoutSecs = 10
	}
	if cfg.Settings.RetryMax == 0 {
		cfg.Settings.RetryMax = 3
	}
	if cfg.Settings.RetryBaseDelayMs == 0 {
		cfg.Settings.RetryBaseDelayMs = 500
	}
	if cfg.Settings.DatabasePath == "" {
		cfg.Settings.DatabasePath = "feedpulse.db"
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate performs validation on the configuration
func (c *Config) Validate() error {
	// Validate settings
	if c.Settings.MaxConcurrency < 1 || c.Settings.MaxConcurrency > 50 {
		return fmt.Errorf("max_concurrency must be between 1 and 50, got %d", c.Settings.MaxConcurrency)
	}
	if c.Settings.DefaultTimeoutSecs < 1 {
		return fmt.Errorf("default_timeout_secs must be positive, got %d", c.Settings.DefaultTimeoutSecs)
	}
	if c.Settings.RetryMax < 0 {
		return fmt.Errorf("retry_max must be non-negative, got %d", c.Settings.RetryMax)
	}
	if c.Settings.RetryBaseDelayMs < 0 {
		return fmt.Errorf("retry_base_delay_ms must be non-negative, got %d", c.Settings.RetryBaseDelayMs)
	}

	// Validate feeds
	if len(c.Feeds) == 0 {
		return fmt.Errorf("no feeds configured")
	}

	for i, feed := range c.Feeds {
		if err := feed.Validate(); err != nil {
			return fmt.Errorf("feed %d: %w", i, err)
		}
	}

	return nil
}

// Validate performs validation on a single feed
func (f *Feed) Validate() error {
	// Name required
	if f.Name == "" {
		return fmt.Errorf("missing field 'name'")
	}

	// URL required and valid
	if f.URL == "" {
		return fmt.Errorf("feed '%s': missing field 'url'", f.Name)
	}

	parsedURL, err := url.ParseRequestURI(f.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("feed '%s': invalid URL '%s'", f.Name, f.URL)
	}

	// Feed type required
	if f.FeedType == "" {
		return fmt.Errorf("feed '%s': missing field 'feed_type'", f.Name)
	}

	// Feed type must be valid
	validTypes := map[string]bool{"json": true, "rss": true, "atom": true}
	if !validTypes[f.FeedType] {
		return fmt.Errorf("feed '%s': feed_type must be one of: json, rss, atom, got '%s'", f.Name, f.FeedType)
	}

	// Refresh interval must be positive if set
	if f.RefreshIntervalSecs < 0 {
		return fmt.Errorf("feed '%s': refresh_interval_secs must be non-negative, got %d", f.Name, f.RefreshIntervalSecs)
	}

	// Apply default
	if f.RefreshIntervalSecs == 0 {
		f.RefreshIntervalSecs = 300
	}

	return nil
}
