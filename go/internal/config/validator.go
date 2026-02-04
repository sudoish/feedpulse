// Package config validator provides validation functions for configuration values.
package config

import (
	"fmt"
	"net/url"
	"strings"

	"feedpulse/internal/errors"
)

// ValidateURL validates that a URL is well-formed and uses HTTP/HTTPS.
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return errors.NewValidationError("url", urlStr, "required", "URL cannot be empty")
	}

	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return errors.NewValidationError("url", urlStr, "format", fmt.Sprintf("invalid URL format: %v", err))
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.NewValidationError("url", urlStr, "format",
			fmt.Sprintf("URL must use HTTP or HTTPS, got: %s", parsedURL.Scheme))
	}

	if parsedURL.Host == "" {
		return errors.NewValidationError("url", urlStr, "format", "URL must have a host")
	}

	return nil
}

// ValidateTimeout validates that a timeout value is positive.
func ValidateTimeout(timeout int) error {
	if timeout <= 0 {
		return errors.NewValidationError("timeout", timeout, "range",
			fmt.Sprintf("timeout must be positive, got: %d", timeout))
	}

	if timeout > 3600 {
		return errors.NewValidationError("timeout", timeout, "range",
			fmt.Sprintf("timeout too large (max 3600 seconds), got: %d", timeout))
	}

	return nil
}

// ValidateConcurrency validates that a concurrency value is within acceptable range.
func ValidateConcurrency(maxConcurrency int) error {
	if maxConcurrency < 1 {
		return errors.NewValidationError("max_concurrency", maxConcurrency, "range",
			fmt.Sprintf("max_concurrency must be at least 1, got: %d", maxConcurrency))
	}

	if maxConcurrency > 50 {
		return errors.NewValidationError("max_concurrency", maxConcurrency, "range",
			fmt.Sprintf("max_concurrency too high (max 50), got: %d", maxConcurrency))
	}

	return nil
}

// ValidateFeedName validates that a feed name is not empty and contains valid characters.
func ValidateFeedName(name string) error {
	if name == "" {
		return errors.NewValidationError("name", name, "required", "feed name cannot be empty")
	}

	if len(name) > 100 {
		return errors.NewValidationError("name", name, "length",
			fmt.Sprintf("feed name too long (max 100 chars), got: %d", len(name)))
	}

	// Check for invalid characters (control characters, etc.)
	for _, r := range name {
		if r < 32 || r == 127 {
			return errors.NewValidationError("name", name, "format",
				"feed name contains invalid control characters")
		}
	}

	return nil
}

// ValidateFeedType validates that a feed type is one of the supported types.
func ValidateFeedType(feedType string) error {
	if feedType == "" {
		return errors.NewValidationError("feed_type", feedType, "required", "feed_type cannot be empty")
	}

	validTypes := []string{"json", "rss", "atom"}
	for _, validType := range validTypes {
		if feedType == validType {
			return nil
		}
	}

	return errors.NewValidationError("feed_type", feedType, "format",
		fmt.Sprintf("feed_type must be one of: %s, got: %s", strings.Join(validTypes, ", "), feedType))
}

// ValidateRefreshInterval validates that a refresh interval is reasonable.
func ValidateRefreshInterval(interval int) error {
	if interval < 0 {
		return errors.NewValidationError("refresh_interval_secs", interval, "range",
			fmt.Sprintf("refresh_interval_secs must be non-negative, got: %d", interval))
	}

	if interval > 0 && interval < 60 {
		return errors.NewValidationError("refresh_interval_secs", interval, "range",
			fmt.Sprintf("refresh_interval_secs too short (min 60 seconds), got: %d", interval))
	}

	if interval > 86400 {
		return errors.NewValidationError("refresh_interval_secs", interval, "range",
			fmt.Sprintf("refresh_interval_secs too long (max 24 hours), got: %d", interval))
	}

	return nil
}

// ValidateRetryMax validates that retry count is reasonable.
func ValidateRetryMax(retryMax int) error {
	if retryMax < 0 {
		return errors.NewValidationError("retry_max", retryMax, "range",
			fmt.Sprintf("retry_max must be non-negative, got: %d", retryMax))
	}

	if retryMax > 10 {
		return errors.NewValidationError("retry_max", retryMax, "range",
			fmt.Sprintf("retry_max too high (max 10), got: %d", retryMax))
	}

	return nil
}

// ValidateRetryDelay validates that retry delay is reasonable.
func ValidateRetryDelay(delayMs int) error {
	if delayMs < 0 {
		return errors.NewValidationError("retry_base_delay_ms", delayMs, "range",
			fmt.Sprintf("retry_base_delay_ms must be non-negative, got: %d", delayMs))
	}

	if delayMs > 60000 {
		return errors.NewValidationError("retry_base_delay_ms", delayMs, "range",
			fmt.Sprintf("retry_base_delay_ms too high (max 60000ms), got: %d", delayMs))
	}

	return nil
}

// ValidateDatabasePath validates that a database path is not empty.
func ValidateDatabasePath(path string) error {
	if path == "" {
		return errors.NewValidationError("database_path", path, "required", "database_path cannot be empty")
	}

	if len(path) > 4096 {
		return errors.NewValidationError("database_path", path, "length",
			fmt.Sprintf("database_path too long (max 4096 chars), got: %d", len(path)))
	}

	// Check for invalid characters
	invalidChars := []string{"\x00", "\n", "\r"}
	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return errors.NewValidationError("database_path", path, "format",
				"database_path contains invalid characters")
		}
	}

	return nil
}

// ValidateFeedConfig validates all aspects of a feed configuration.
func ValidateFeedConfig(feed *Feed) error {
	if err := ValidateFeedName(feed.Name); err != nil {
		return fmt.Errorf("feed validation failed: %w", err)
	}

	if err := ValidateURL(feed.URL); err != nil {
		return fmt.Errorf("feed '%s': %w", feed.Name, err)
	}

	if err := ValidateFeedType(feed.FeedType); err != nil {
		return fmt.Errorf("feed '%s': %w", feed.Name, err)
	}

	if feed.RefreshIntervalSecs != 0 {
		if err := ValidateRefreshInterval(feed.RefreshIntervalSecs); err != nil {
			return fmt.Errorf("feed '%s': %w", feed.Name, err)
		}
	}

	return nil
}

// ValidateSettings validates all settings in the configuration.
func ValidateSettings(settings *Settings) error {
	if err := ValidateConcurrency(settings.MaxConcurrency); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	if err := ValidateTimeout(settings.DefaultTimeoutSecs); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	if err := ValidateRetryMax(settings.RetryMax); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	if err := ValidateRetryDelay(settings.RetryBaseDelayMs); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	if err := ValidateDatabasePath(settings.DatabasePath); err != nil {
		return fmt.Errorf("settings validation failed: %w", err)
	}

	return nil
}
