use crate::models::{Config, FeedConfig, FeedResult, FetchStatus};
use crate::parser;
use anyhow::Result;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Semaphore;
use tokio::time::sleep;

/// Fetch all feeds concurrently with retry logic
pub async fn fetch_all_feeds(config: &Config) -> Vec<FeedResult> {
    let semaphore = Arc::new(Semaphore::new(config.settings.max_concurrency));
    let mut tasks = Vec::new();
    
    for feed in &config.feeds {
        let feed = feed.clone();
        let settings = config.settings.clone();
        let sem = semaphore.clone();
        
        let task = tokio::spawn(async move {
            // Acquire semaphore permit
            let _permit = sem.acquire().await.unwrap();
            fetch_single_feed(&feed, &settings).await
        });
        
        tasks.push(task);
    }
    
    // Wait for all tasks to complete
    let mut results = Vec::new();
    for task in tasks {
        match task.await {
            Ok(result) => results.push(result),
            Err(e) => {
                eprintln!("Error: task panicked: {}", e);
            }
        }
    }
    
    results
}

/// Fetch a single feed with retry logic
async fn fetch_single_feed(
    feed: &FeedConfig,
    settings: &crate::models::Settings,
) -> FeedResult {
    let start_time = Instant::now();
    
    // Try to fetch with retries
    let fetch_result = fetch_with_retry(feed, settings).await;
    
    let duration_ms = start_time.elapsed().as_millis() as u64;
    
    match fetch_result {
        Ok(body) => {
            // Parse the response
            match parser::parse_feed(&feed.name, &feed.feed_type, &body) {
                Ok(items) => FeedResult {
                    source_name: feed.name.clone(),
                    status: FetchStatus::Success,
                    items,
                    duration_ms,
                    error_message: None,
                },
                Err(e) => {
                    eprintln!("Error: {}: parse error: {}", feed.name, e);
                    FeedResult {
                        source_name: feed.name.clone(),
                        status: FetchStatus::Error,
                        items: vec![],
                        duration_ms,
                        error_message: Some(format!("parse error: {}", e)),
                    }
                }
            }
        }
        Err(e) => FeedResult {
            source_name: feed.name.clone(),
            status: FetchStatus::Error,
            items: vec![],
            duration_ms,
            error_message: Some(e),
        },
    }
}

/// Fetch URL with exponential backoff retry logic
async fn fetch_with_retry(
    feed: &FeedConfig,
    settings: &crate::models::Settings,
) -> Result<String, String> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(settings.default_timeout_secs))
        .build()
        .map_err(|e| format!("failed to create HTTP client: {}", e))?;
    
    let mut last_error = String::new();
    
    for attempt in 0..=settings.retry_max {
        // Add jitter to retry delay
        if attempt > 0 {
            let base_delay = settings.retry_base_delay_ms;
            let delay = base_delay * 2_u64.pow(attempt - 1);
            let jitter_range = (delay / 10).max(1);
            let jitter = (rand::random::<u64>() % jitter_range) as i64;
            let actual_delay = (delay as i64 + jitter).max(0) as u64;
            
            sleep(Duration::from_millis(actual_delay)).await;
        }
        
        // Build request
        let mut request = client.get(&feed.url);
        
        // Add custom headers
        for (key, value) in &feed.headers {
            request = request.header(key, value);
        }
        
        // Execute request
        match request.send().await {
            Ok(response) => {
                let status = response.status();
                
                // Check HTTP status
                if status.is_success() {
                    // Success - read body
                    match response.text().await {
                        Ok(body) => return Ok(body),
                        Err(e) => {
                            last_error = format!("failed to read response body: {}", e);
                            continue;
                        }
                    }
                } else if status == reqwest::StatusCode::NOT_FOUND {
                    // 404 - don't retry
                    return Err(format!("HTTP 404"));
                } else if status.is_client_error() {
                    // 4xx - retry only for 429
                    if status == reqwest::StatusCode::TOO_MANY_REQUESTS {
                        last_error = format!("HTTP 429");
                        continue;
                    } else {
                        return Err(format!("HTTP {}", status.as_u16()));
                    }
                } else if status.is_server_error() {
                    // 5xx - retry
                    last_error = format!("HTTP {}", status.as_u16());
                    continue;
                } else {
                    // Unknown status
                    return Err(format!("HTTP {}", status.as_u16()));
                }
            }
            Err(e) => {
                // Network error - retry
                last_error = if e.is_timeout() {
                    "timeout".to_string()
                } else if e.is_connect() {
                    "connection failed".to_string()
                } else {
                    format!("network error: {}", e)
                };
                continue;
            }
        }
    }
    
    // All retries exhausted
    Err(format!("{} after {} retries", last_error, settings.retry_max))
}

// Simple random number generator for jitter
mod rand {
    use std::cell::Cell;
    use std::time::{SystemTime, UNIX_EPOCH};
    
    thread_local! {
        static SEED: Cell<u64> = Cell::new(
            SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_nanos() as u64
        );
    }
    
    pub fn random<T: From<u64>>() -> T {
        SEED.with(|seed| {
            let mut s = seed.get();
            s ^= s << 13;
            s ^= s >> 7;
            s ^= s << 17;
            seed.set(s);
            T::from(s)
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::{FeedType, Settings};
    use std::collections::HashMap;

    #[test]
    fn test_exponential_backoff_calculation() {
        // Test that delay increases exponentially
        let base = 100u64;
        let delay1 = base * 2_u64.pow(0);
        let delay2 = base * 2_u64.pow(1);
        let delay3 = base * 2_u64.pow(2);
        
        assert_eq!(delay1, 100);
        assert_eq!(delay2, 200);
        assert_eq!(delay3, 400);
    }

    #[tokio::test]
    async fn test_fetch_invalid_url() {
        let feed = FeedConfig {
            name: "Test".to_string(),
            url: "https://invalid.example.notarealurl".to_string(),
            feed_type: FeedType::Json,
            refresh_interval_secs: 300,
            headers: HashMap::new(),
        };
        
        let settings = Settings {
            retry_max: 1,
            retry_base_delay_ms: 10,
            ..Default::default()
        };
        
        let result = fetch_single_feed(&feed, &settings).await;
        assert_eq!(result.status, FetchStatus::Error);
        assert!(result.error_message.is_some());
    }
}
