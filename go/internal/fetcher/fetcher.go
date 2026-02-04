package fetcher

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"feedpulse/internal/config"
	"feedpulse/internal/parser"
	"feedpulse/internal/storage"
)

// FetchResult represents the result of fetching a single feed
type FetchResult struct {
	Source       string
	Success      bool
	ItemsCount   int
	NewItems     int
	Error        string
	DurationMs   int64
	Items        []storage.FeedItem
}

// Fetcher handles concurrent feed fetching
type Fetcher struct {
	config *config.Config
	parser *parser.Parser
	client *http.Client
}

// NewFetcher creates a new fetcher instance
func NewFetcher(cfg *config.Config) *Fetcher {
	return &Fetcher{
		config: cfg,
		parser: parser.NewParser(),
		client: &http.Client{
			Timeout: time.Duration(cfg.Settings.DefaultTimeoutSecs) * time.Second,
		},
	}
}

// FetchAll fetches all configured feeds concurrently
func (f *Fetcher) FetchAll(ctx context.Context) []FetchResult {
	// Create a semaphore to limit concurrency
	sem := make(chan struct{}, f.config.Settings.MaxConcurrency)
	
	var wg sync.WaitGroup
	results := make([]FetchResult, len(f.config.Feeds))

	for i, feed := range f.config.Feeds {
		wg.Add(1)
		go func(index int, feed config.Feed) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = FetchResult{
					Source:  feed.Name,
					Success: false,
					Error:   "cancelled",
				}
				return
			}

			// Fetch the feed
			results[index] = f.fetchFeed(ctx, feed)
		}(i, feed)
	}

	wg.Wait()
	return results
}

// fetchFeed fetches a single feed with retries
func (f *Fetcher) fetchFeed(ctx context.Context, feed config.Feed) FetchResult {
	start := time.Now()

	var lastErr error
	for attempt := 0; attempt <= f.config.Settings.RetryMax; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff with jitter
			delay := f.calculateBackoff(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return FetchResult{
					Source:     feed.Name,
					Success:    false,
					Error:      "cancelled during retry",
					DurationMs: time.Since(start).Milliseconds(),
				}
			}
		}

		// Attempt to fetch
		data, err := f.fetchURL(ctx, feed)
		if err != nil {
			lastErr = err
			// Don't retry on 404 or client errors
			if isClientError(err) {
				break
			}
			continue
		}

		// Parse the feed
		parseResult := f.parser.Parse(feed.Name, feed.FeedType, data)
		
		// Log parse errors but don't fail
		if len(parseResult.Errors) > 0 {
			for _, parseErr := range parseResult.Errors {
				fmt.Fprintf(io.Discard, "warning: %s: %s\n", feed.Name, parseErr)
			}
		}

		duration := time.Since(start).Milliseconds()
		return FetchResult{
			Source:     feed.Name,
			Success:    true,
			ItemsCount: len(parseResult.Items),
			Items:      parseResult.Items,
			DurationMs: duration,
		}
	}

	// All retries failed
	duration := time.Since(start).Milliseconds()
	errorMsg := "unknown error"
	if lastErr != nil {
		errorMsg = lastErr.Error()
	}

	return FetchResult{
		Source:     feed.Name,
		Success:    false,
		Error:      fmt.Sprintf("failed after %d retries: %s", f.config.Settings.RetryMax, errorMsg),
		DurationMs: duration,
	}
}

// fetchURL performs the actual HTTP request
func (f *Fetcher) fetchURL(ctx context.Context, feed config.Feed) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feed.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range feed.Headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "feedpulse/1.0")
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// calculateBackoff calculates exponential backoff with jitter
func (f *Fetcher) calculateBackoff(attempt int) time.Duration {
	baseDelay := float64(f.config.Settings.RetryBaseDelayMs)
	delay := baseDelay * math.Pow(2, float64(attempt-1))
	
	// Add jitter (±25%)
	jitter := delay * 0.25 * (2*rand.Float64() - 1)
	totalDelay := delay + jitter
	
	return time.Duration(totalDelay) * time.Millisecond
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Status)
}

// isClientError checks if an error is a 4xx client error (don't retry these)
func isClientError(err error) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.StatusCode >= 400 && httpErr.StatusCode < 500
	}
	return false
}
