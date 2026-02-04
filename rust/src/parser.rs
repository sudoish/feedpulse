use crate::models::FeedItem;
use serde_json::Value;

pub struct Parser;

impl Parser {
    pub fn parse(source: &str, feed_type: &str, body: &str) -> Result<Vec<FeedItem>, String> {
        match feed_type {
            "json" => Self::parse_json(source, body),
            "rss" | "atom" => Err("RSS/Atom parsing not yet implemented".to_string()),
            _ => Err(format!("Unknown feed type: {}", feed_type)),
        }
    }

    fn parse_json(source: &str, body: &str) -> Result<Vec<FeedItem>, String> {
        let json: Value = serde_json::from_str(body)
            .map_err(|e| format!("malformed JSON: {}", e))?;

        // Detect feed type based on source name or structure
        if source.contains("HackerNews") || source.contains("Hacker News") {
            Self::parse_hackernews(source, &json)
        } else if source.contains("GitHub") {
            Self::parse_github(source, &json)
        } else if source.contains("Reddit") {
            Self::parse_reddit(source, &json)
        } else if source.contains("Lobsters") {
            Self::parse_lobsters(source, &json)
        } else {
            // Generic JSON parsing - try to extract items
            Ok(Vec::new())
        }
    }

    fn parse_hackernews(source: &str, json: &Value) -> Result<Vec<FeedItem>, String> {
        let mut items = Vec::new();

        if let Some(ids) = json.as_array() {
            for id_value in ids {
                if let Some(id) = id_value.as_u64() {
                    let title = format!("HN Story {}", id);
                    let url = format!("https://news.ycombinator.com/item?id={}", id);
                    let item = FeedItem::new(title, url, source.to_string());
                    items.push(item);
                }
            }
        }

        Ok(items)
    }

    fn parse_github(source: &str, json: &Value) -> Result<Vec<FeedItem>, String> {
        let mut items = Vec::new();

        if let Some(item_array) = json.get("items").and_then(|v| v.as_array()) {
            for (idx, item_value) in item_array.iter().enumerate() {
                match Self::extract_github_item(source, item_value) {
                    Ok(Some(item)) => items.push(item),
                    Ok(None) => {
                        eprintln!("Warning: {} item {}: missing required field", source, idx);
                    }
                    Err(e) => {
                        eprintln!("Warning: {} item {}: {}", source, idx, e);
                    }
                }
            }
        }

        Ok(items)
    }

    fn extract_github_item(source: &str, item: &Value) -> Result<Option<FeedItem>, String> {
        let title = match item.get("full_name") {
            Some(Value::String(s)) => s.clone(),
            Some(other) => other.to_string(),
            None => return Ok(None),
        };

        let url = match item.get("html_url") {
            Some(Value::String(s)) => s.clone(),
            Some(other) => other.to_string(),
            None => return Ok(None),
        };

        let timestamp = item.get("updated_at")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        let tags = if let Some(Value::Array(topics)) = item.get("topics") {
            topics.iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect()
        } else {
            Vec::new()
        };

        let raw_data = serde_json::to_string(item).ok();

        Ok(Some(
            FeedItem::new(title, url, source.to_string())
                .with_timestamp(timestamp)
                .with_tags(tags)
                .with_raw_data(raw_data.unwrap_or_default())
        ))
    }

    fn parse_reddit(source: &str, json: &Value) -> Result<Vec<FeedItem>, String> {
        let mut items = Vec::new();

        let children = json
            .get("data")
            .and_then(|d| d.get("children"))
            .and_then(|c| c.as_array())
            .ok_or("Reddit feed missing data.children")?;

        for (idx, child) in children.iter().enumerate() {
            if let Some(data) = child.get("data") {
                match Self::extract_reddit_item(source, data) {
                    Ok(Some(item)) => items.push(item),
                    Ok(None) => {
                        eprintln!("Warning: {} item {}: missing required field", source, idx);
                    }
                    Err(e) => {
                        eprintln!("Warning: {} item {}: {}", source, idx, e);
                    }
                }
            }
        }

        Ok(items)
    }

    fn extract_reddit_item(source: &str, data: &Value) -> Result<Option<FeedItem>, String> {
        let title = match data.get("title") {
            Some(Value::String(s)) => s.clone(),
            Some(other) => other.to_string(),
            None => return Ok(None),
        };

        let url = match data.get("url") {
            Some(Value::String(s)) => s.clone(),
            Some(other) => other.to_string(),
            None => return Ok(None),
        };

        let timestamp = data.get("created_utc")
            .and_then(|v| v.as_f64())
            .map(|ts| {
                let dt = chrono::DateTime::from_timestamp(ts as i64, 0)
                    .unwrap_or_default();
                dt.to_rfc3339()
            });

        let tags = if let Some(Value::String(flair)) = data.get("link_flair_text") {
            vec![flair.clone()]
        } else {
            Vec::new()
        };

        let raw_data = serde_json::to_string(data).ok();

        Ok(Some(
            FeedItem::new(title, url, source.to_string())
                .with_timestamp(timestamp)
                .with_tags(tags)
                .with_raw_data(raw_data.unwrap_or_default())
        ))
    }

    fn parse_lobsters(source: &str, json: &Value) -> Result<Vec<FeedItem>, String> {
        let mut items = Vec::new();

        let item_array = json.as_array()
            .ok_or("Lobsters feed is not an array")?;

        for (idx, item_value) in item_array.iter().enumerate() {
            match Self::extract_lobsters_item(source, item_value) {
                Ok(Some(item)) => items.push(item),
                Ok(None) => {
                    eprintln!("Warning: {} item {}: missing required field", source, idx);
                }
                Err(e) => {
                    eprintln!("Warning: {} item {}: {}", source, idx, e);
                }
            }
        }

        Ok(items)
    }

    fn extract_lobsters_item(source: &str, item: &Value) -> Result<Option<FeedItem>, String> {
        let title = match item.get("title") {
            Some(Value::String(s)) => s.clone(),
            Some(other) => other.to_string(),
            None => return Ok(None),
        };

        let url = match item.get("url") {
            Some(Value::String(s)) if !s.is_empty() => s.clone(),
            _ => {
                // Fall back to comments_url
                match item.get("comments_url") {
                    Some(Value::String(s)) => s.clone(),
                    Some(other) => other.to_string(),
                    None => return Ok(None),
                }
            }
        };

        let timestamp = item.get("created_at")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string());

        let tags = if let Some(Value::Array(tag_arr)) = item.get("tags") {
            tag_arr.iter()
                .filter_map(|v| v.as_str().map(|s| s.to_string()))
                .collect()
        } else {
            Vec::new()
        };

        let raw_data = serde_json::to_string(item).ok();

        Ok(Some(
            FeedItem::new(title, url, source.to_string())
                .with_timestamp(timestamp)
                .with_tags(tags)
                .with_raw_data(raw_data.unwrap_or_default())
        ))
    }
}
