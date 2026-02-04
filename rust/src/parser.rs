use crate::models::{FeedItem, FeedType};
use anyhow::Result;
use chrono::{DateTime, Utc};
use serde_json::Value;
use sha2::{Digest, Sha256};

/// Parse and normalize a feed response into FeedItems
pub fn parse_feed(
    source_name: &str,
    feed_type: &FeedType,
    body: &str,
) -> Result<Vec<FeedItem>> {
    match feed_type {
        FeedType::Json => parse_json_feed(source_name, body),
        FeedType::Rss | FeedType::Atom => {
            // RSS/Atom not implemented for this experiment
            eprintln!("Warning: RSS/Atom parsing not implemented");
            Ok(vec![])
        }
    }
}

/// Parse JSON feed and normalize based on source structure
fn parse_json_feed(source_name: &str, body: &str) -> Result<Vec<FeedItem>> {
    let json: Value = serde_json::from_str(body)
        .map_err(|e| anyhow::anyhow!("malformed JSON: {}", e))?;
    
    // Detect feed type by structure
    if source_name.contains("HackerNews") || source_name.contains("HN") {
        parse_hackernews(&json, source_name)
    } else if source_name.contains("GitHub") {
        parse_github(&json, source_name)
    } else if source_name.contains("Reddit") {
        parse_reddit(&json, source_name)
    } else if source_name.contains("Lobsters") {
        parse_lobsters(&json, source_name)
    } else {
        // Generic JSON array parser
        parse_generic_json(&json, source_name)
    }
}

/// Parse HackerNews top stories (array of IDs)
fn parse_hackernews(json: &Value, source_name: &str) -> Result<Vec<FeedItem>> {
    let ids = json.as_array()
        .ok_or_else(|| anyhow::anyhow!("HackerNews: expected array of IDs"))?;
    
    let mut items = Vec::new();
    
    for (idx, id_value) in ids.iter().enumerate() {
        if let Some(id) = id_value.as_i64() {
            let title = format!("HN Story {}", id);
            let url = format!("https://news.ycombinator.com/item?id={}", id);
            
            let item_id = generate_id(source_name, &url);
            
            items.push(FeedItem {
                id: item_id,
                title,
                url,
                source: source_name.to_string(),
                timestamp: None,
                tags: vec![],
                raw_data: Some(id_value.to_string()),
            });
        } else {
            eprintln!(
                "Warning: {}: skipping item {} - invalid ID format",
                source_name, idx
            );
        }
    }
    
    Ok(items)
}

/// Parse GitHub search results
fn parse_github(json: &Value, source_name: &str) -> Result<Vec<FeedItem>> {
    let items_array = json
        .get("items")
        .and_then(|v| v.as_array())
        .ok_or_else(|| anyhow::anyhow!("GitHub: missing 'items' array"))?;
    
    let mut items = Vec::new();
    
    for (idx, item) in items_array.iter().enumerate() {
        match parse_github_item(item, source_name) {
            Ok(feed_item) => items.push(feed_item),
            Err(e) => {
                eprintln!(
                    "Warning: {}: skipping item {} - {}",
                    source_name, idx, e
                );
            }
        }
    }
    
    Ok(items)
}

fn parse_github_item(item: &Value, source_name: &str) -> Result<FeedItem> {
    let title = item
        .get("full_name")
        .or_else(|| item.get("name"))
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing 'full_name' or 'name'"))?
        .to_string();
    
    let url = item
        .get("html_url")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing 'html_url'"))?
        .to_string();
    
    let timestamp = item
        .get("updated_at")
        .or_else(|| item.get("created_at"))
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());
    
    let tags = item
        .get("topics")
        .and_then(|v| v.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect()
        })
        .unwrap_or_default();
    
    let item_id = generate_id(source_name, &url);
    
    Ok(FeedItem {
        id: item_id,
        title,
        url,
        source: source_name.to_string(),
        timestamp,
        tags,
        raw_data: Some(item.to_string()),
    })
}

/// Parse Reddit hot posts
fn parse_reddit(json: &Value, source_name: &str) -> Result<Vec<FeedItem>> {
    let children = json
        .get("data")
        .and_then(|v| v.get("children"))
        .and_then(|v| v.as_array())
        .ok_or_else(|| anyhow::anyhow!("Reddit: missing 'data.children' array"))?;
    
    let mut items = Vec::new();
    
    for (idx, child) in children.iter().enumerate() {
        let data = match child.get("data") {
            Some(d) => d,
            None => {
                eprintln!(
                    "Warning: {}: skipping item {} - missing 'data' field",
                    source_name, idx
                );
                continue;
            }
        };
        
        match parse_reddit_item(data, source_name) {
            Ok(feed_item) => items.push(feed_item),
            Err(e) => {
                eprintln!(
                    "Warning: {}: skipping item {} - {}",
                    source_name, idx, e
                );
            }
        }
    }
    
    Ok(items)
}

fn parse_reddit_item(data: &Value, source_name: &str) -> Result<FeedItem> {
    let title = data
        .get("title")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing 'title'"))?
        .to_string();
    
    let url = data
        .get("url")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing 'url'"))?
        .to_string();
    
    let timestamp = data
        .get("created_utc")
        .and_then(|v| v.as_f64())
        .map(|ts| {
            let dt = DateTime::from_timestamp(ts as i64, 0)
                .unwrap_or_else(|| Utc::now());
            dt.to_rfc3339()
        });
    
    let tags = data
        .get("link_flair_text")
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty())
        .map(|s| vec![s.to_string()])
        .unwrap_or_default();
    
    let item_id = generate_id(source_name, &url);
    
    Ok(FeedItem {
        id: item_id,
        title,
        url,
        source: source_name.to_string(),
        timestamp,
        tags,
        raw_data: Some(data.to_string()),
    })
}

/// Parse Lobsters hot posts
fn parse_lobsters(json: &Value, source_name: &str) -> Result<Vec<FeedItem>> {
    let items_array = json
        .as_array()
        .ok_or_else(|| anyhow::anyhow!("Lobsters: expected array"))?;
    
    let mut items = Vec::new();
    
    for (idx, item) in items_array.iter().enumerate() {
        match parse_lobsters_item(item, source_name) {
            Ok(feed_item) => items.push(feed_item),
            Err(e) => {
                eprintln!(
                    "Warning: {}: skipping item {} - {}",
                    source_name, idx, e
                );
            }
        }
    }
    
    Ok(items)
}

fn parse_lobsters_item(item: &Value, source_name: &str) -> Result<FeedItem> {
    let title = item
        .get("title")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing 'title'"))?
        .to_string();
    
    let url = item
        .get("url")
        .or_else(|| item.get("comments_url"))
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow::anyhow!("missing 'url' or 'comments_url'"))?
        .to_string();
    
    let timestamp = item
        .get("created_at")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string());
    
    let tags = item
        .get("tags")
        .and_then(|v| v.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect()
        })
        .unwrap_or_default();
    
    let item_id = generate_id(source_name, &url);
    
    Ok(FeedItem {
        id: item_id,
        title,
        url,
        source: source_name.to_string(),
        timestamp,
        tags,
        raw_data: Some(item.to_string()),
    })
}

/// Generic JSON array parser (fallback)
fn parse_generic_json(json: &Value, source_name: &str) -> Result<Vec<FeedItem>> {
    let items_array = json
        .as_array()
        .ok_or_else(|| anyhow::anyhow!("expected JSON array"))?;
    
    let mut items = Vec::new();
    
    for (idx, item) in items_array.iter().enumerate() {
        // Try to extract title and url from common field names
        let title = item
            .get("title")
            .or_else(|| item.get("name"))
            .or_else(|| item.get("headline"))
            .and_then(|v| v.as_str())
            .unwrap_or("Untitled");
        
        let url = item
            .get("url")
            .or_else(|| item.get("link"))
            .or_else(|| item.get("href"))
            .and_then(|v| v.as_str());
        
        if let Some(url_str) = url {
            let item_id = generate_id(source_name, url_str);
            
            items.push(FeedItem {
                id: item_id,
                title: title.to_string(),
                url: url_str.to_string(),
                source: source_name.to_string(),
                timestamp: None,
                tags: vec![],
                raw_data: Some(item.to_string()),
            });
        } else {
            eprintln!(
                "Warning: {}: skipping item {} - no URL field found",
                source_name, idx
            );
        }
    }
    
    Ok(items)
}

/// Generate SHA256 ID from source name and URL
fn generate_id(source_name: &str, url: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(source_name.as_bytes());
    hasher.update(url.as_bytes());
    format!("{:x}", hasher.finalize())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_hackernews() {
        let json_str = r#"[123, 456, 789]"#;
        let json: Value = serde_json::from_str(json_str).unwrap();
        
        let items = parse_hackernews(&json, "HackerNews").unwrap();
        assert_eq!(items.len(), 3);
        assert_eq!(items[0].title, "HN Story 123");
        assert!(items[0].url.contains("123"));
    }

    #[test]
    fn test_parse_github_missing_title() {
        let json_str = r#"{"items": [{"html_url": "https://github.com/test"}]}"#;
        let json: Value = serde_json::from_str(json_str).unwrap();
        
        let items = parse_github(&json, "GitHub").unwrap();
        // Should skip item with missing title
        assert_eq!(items.len(), 0);
    }

    #[test]
    fn test_parse_reddit_item() {
        let json_str = r#"{
            "title": "Test Post",
            "url": "https://example.com",
            "created_utc": 1704067200.0,
            "link_flair_text": "Discussion"
        }"#;
        let json: Value = serde_json::from_str(json_str).unwrap();
        
        let item = parse_reddit_item(&json, "Reddit").unwrap();
        assert_eq!(item.title, "Test Post");
        assert_eq!(item.url, "https://example.com");
        assert_eq!(item.tags.len(), 1);
        assert_eq!(item.tags[0], "Discussion");
    }

    #[test]
    fn test_generate_id_deterministic() {
        let id1 = generate_id("source1", "https://example.com");
        let id2 = generate_id("source1", "https://example.com");
        assert_eq!(id1, id2);
        
        let id3 = generate_id("source2", "https://example.com");
        assert_ne!(id1, id3);
    }

    #[test]
    fn test_malformed_json() {
        let result = parse_json_feed("test", "not json");
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("malformed JSON"));
    }
}
