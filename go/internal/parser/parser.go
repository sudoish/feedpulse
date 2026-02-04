package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"feedpulse/internal/storage"
)

// ParseResult represents the result of parsing a feed
type ParseResult struct {
	Items  []storage.FeedItem
	Errors []string
}

// Parser handles feed parsing and normalization
type Parser struct{}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses raw feed data and returns normalized items
func (p *Parser) Parse(source string, feedType string, data []byte) ParseResult {
	var result ParseResult

	switch feedType {
	case "json":
		result = p.parseJSON(source, data)
	case "rss":
		result.Errors = append(result.Errors, "RSS parsing not implemented in this version")
	case "atom":
		result.Errors = append(result.Errors, "Atom parsing not implemented in this version")
	default:
		result.Errors = append(result.Errors, fmt.Sprintf("unknown feed type: %s", feedType))
	}

	return result
}

// parseJSON parses JSON feeds (HackerNews, GitHub, Reddit, Lobsters)
func (p *Parser) parseJSON(source string, data []byte) ParseResult {
	var result ParseResult

	// Try to parse as generic JSON first
	var rawJSON interface{}
	if err := json.Unmarshal(data, &rawJSON); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("malformed JSON: %v", err))
		return result
	}

	// Detect feed structure and parse accordingly
	switch v := rawJSON.(type) {
	case []interface{}:
		// Could be HackerNews (array of IDs) or Lobsters (array of objects)
		if len(v) > 0 {
			if _, ok := v[0].(float64); ok {
				// HackerNews: array of numeric IDs
				result = p.parseHackerNews(source, v)
			} else if _, ok := v[0].(map[string]interface{}); ok {
				// Lobsters: array of story objects
				result = p.parseLobsters(source, v)
			}
		}
	case map[string]interface{}:
		// Could be GitHub or Reddit (both have nested structure)
		if items, ok := v["items"].([]interface{}); ok {
			// GitHub: has "items" array
			result = p.parseGitHub(source, items)
		} else if data, ok := v["data"].(map[string]interface{}); ok {
			// Reddit: has "data" object
			if children, ok := data["children"].([]interface{}); ok {
				result = p.parseReddit(source, children)
			}
		}
	}

	if len(result.Items) == 0 && len(result.Errors) == 0 {
		result.Errors = append(result.Errors, "unrecognized feed structure")
	}

	return result
}

// parseHackerNews parses HackerNews top stories (array of IDs)
func (p *Parser) parseHackerNews(source string, items []interface{}) ParseResult {
	var result ParseResult

	for i, item := range items {
		id, ok := item.(float64)
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: expected numeric ID, got %T", i, item))
			continue
		}

		idStr := strconv.Itoa(int(id))
		title := fmt.Sprintf("HN Story %s", idStr)
		url := fmt.Sprintf("https://news.ycombinator.com/item?id=%s", idStr)

		feedItem := storage.FeedItem{
			ID:        p.generateID(source, url),
			Title:     title,
			URL:       url,
			Source:    source,
			CreatedAt: time.Now(),
		}

		result.Items = append(result.Items, feedItem)
	}

	return result
}

// parseGitHub parses GitHub API response
func (p *Parser) parseGitHub(source string, items []interface{}) ParseResult {
	var result ParseResult

	for i, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: expected object, got %T", i, item))
			continue
		}

		// Extract required fields
		title, titleOk := p.getString(obj, "full_name")
		url, urlOk := p.getString(obj, "html_url")

		if !titleOk || !urlOk {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: missing required field (full_name or html_url)", i))
			continue
		}

		feedItem := storage.FeedItem{
			ID:        p.generateID(source, url),
			Title:     title,
			URL:       url,
			Source:    source,
			CreatedAt: time.Now(),
		}

		// Optional: timestamp
		if timestamp, ok := p.getString(obj, "updated_at"); ok {
			feedItem.Timestamp = &timestamp
		}

		// Optional: tags (topics)
		if topics, ok := obj["topics"].([]interface{}); ok {
			for _, topic := range topics {
				if topicStr, ok := topic.(string); ok {
					feedItem.Tags = append(feedItem.Tags, topicStr)
				}
			}
		}

		result.Items = append(result.Items, feedItem)
	}

	return result
}

// parseReddit parses Reddit API response
func (p *Parser) parseReddit(source string, children []interface{}) ParseResult {
	var result ParseResult

	for i, child := range children {
		childObj, ok := child.(map[string]interface{})
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: expected object, got %T", i, child))
			continue
		}

		data, ok := childObj["data"].(map[string]interface{})
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: missing data object", i))
			continue
		}

		// Extract required fields
		title, titleOk := p.getString(data, "title")
		url, urlOk := p.getString(data, "url")

		if !titleOk || !urlOk {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: missing required field (title or url)", i))
			continue
		}

		feedItem := storage.FeedItem{
			ID:        p.generateID(source, url),
			Title:     title,
			URL:       url,
			Source:    source,
			CreatedAt: time.Now(),
		}

		// Optional: timestamp (created_utc is Unix timestamp)
		if createdUtc, ok := data["created_utc"].(float64); ok {
			timestamp := time.Unix(int64(createdUtc), 0).Format(time.RFC3339)
			feedItem.Timestamp = &timestamp
		}

		// Optional: tags (link_flair_text)
		if flair, ok := p.getString(data, "link_flair_text"); ok && flair != "" {
			feedItem.Tags = []string{flair}
		}

		result.Items = append(result.Items, feedItem)
	}

	return result
}

// parseLobsters parses Lobsters API response
func (p *Parser) parseLobsters(source string, items []interface{}) ParseResult {
	var result ParseResult

	for i, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: expected object, got %T", i, item))
			continue
		}

		// Extract required fields
		title, titleOk := p.getString(obj, "title")
		url, urlOk := p.getString(obj, "url")

		// Fall back to comments_url if url is not present
		if !urlOk {
			url, urlOk = p.getString(obj, "comments_url")
		}

		if !titleOk || !urlOk {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: missing required field (title or url)", i))
			continue
		}

		feedItem := storage.FeedItem{
			ID:        p.generateID(source, url),
			Title:     title,
			URL:       url,
			Source:    source,
			CreatedAt: time.Now(),
		}

		// Optional: timestamp
		if timestamp, ok := p.getString(obj, "created_at"); ok {
			feedItem.Timestamp = &timestamp
		}

		// Optional: tags
		if tags, ok := obj["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					feedItem.Tags = append(feedItem.Tags, tagStr)
				}
			}
		}

		result.Items = append(result.Items, feedItem)
	}

	return result
}

// getString attempts to extract a string value from a map
// It handles type coercion for common cases
func (p *Parser) getString(obj map[string]interface{}, key string) (string, bool) {
	val, ok := obj[key]
	if !ok {
		return "", false
	}

	switch v := val.(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return strconv.Itoa(v), true
	case bool:
		return strconv.FormatBool(v), true
	default:
		// Try to JSON encode it as a fallback
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes), true
		}
		log.Printf("warning: cannot convert field %s to string (type: %T)", key, v)
		return "", false
	}
}

// generateID creates a deterministic ID from source name and URL
func (p *Parser) generateID(source, url string) string {
	hash := sha256.Sum256([]byte(source + url))
	return hex.EncodeToString(hash[:])
}
