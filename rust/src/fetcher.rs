use crate::config::{Config, Feed};
use crate::models::FeedItem;
use crate::parser::Parser;
use reqwest::Client;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Semaphore;
use tokio::time::sleep;

#[derive(Debug, Clone)]
pub struct FetchResult {
    pub source: String,
    pub items: Vec<FeedItem>,
    pub new_items: usize,
    pub duration_ms: u64,
    pub error: Option<String>,
}

pub struct Fetcher {
    config: Config,
    client: Client,
}

impl Fetcher {
    pub fn new(config: Config) -> Self {
        let client = Client::builder()
            .timeout(Duration::from_secs(config.settings.default_timeout_secs))
            .build()
            .expect("Failed to build HTTP client");

        Self { config, client }
    }

    pub async fn fetch_all(&self) -> Vec<FetchResult> {
        let feeds = self.config.feeds.clone();
        let max_concurrency = self.config.settings.max_concurrency;

        println!(
            "Fetching {} feeds (max concurrency: {})...",
            feeds.len(),
            max_concurrency
        );

        let semaphore = Arc::new(Semaphore::new(max_concurrency));
        let mut tasks = Vec::new();

        for feed in feeds {
            let sem = semaphore.clone();
            let client = self.client.clone();
            let retry_max = self.config.settings.retry_max;
            let retry_base_delay = self.config.settings.retry_base_delay_ms;

            let task = tokio::spawn(async move {
                let _permit = sem.acquire().await.unwrap();
                Self::fetch_feed(client, feed, retry_max, retry_base_delay).await
            });

            tasks.push(task);
        }

        let mut results = Vec::new();
        for task in tasks {
            if let Ok(result) = task.await {
                results.push(result);
            }
        }

        results
    }

    async fn fetch_feed(
        client: Client,
        feed: Feed,
        retry_max: usize,
        retry_base_delay: u64,
    ) -> FetchResult {
        let start = Instant::now();
        let source = feed.name.clone();

        for attempt in 0..=retry_max {
            match Self::try_fetch(&client, &feed).await {
                Ok(items) => {
                    let duration_ms = start.elapsed().as_millis() as u64;
                    return FetchResult {
                        source,
                        items,
                        new_items: 0, // Will be updated by storage
                        duration_ms,
                        error: None,
                    };
                }
                Err(e) => {
                    if attempt < retry_max {
                        // Check if we should retry based on error type
                        let should_retry = match &e {
                            FetchError::Http(status) => {
                                // Don't retry 4xx errors except 429
                                if status.as_u16() == 429 {
                                    true
                                } else if status.is_client_error() {
                                    false
                                } else {
                                    true // Retry 5xx and other errors
                                }
                            }
                            _ => true, // Retry network errors, timeouts, etc.
                        };

                        if should_retry {
                            let delay = retry_base_delay * 2_u64.pow(attempt as u32);
                            sleep(Duration::from_millis(delay)).await;
                            continue;
                        }
                    }

                    // All retries exhausted or non-retryable error
                    let duration_ms = start.elapsed().as_millis() as u64;
                    return FetchResult {
                        source,
                        items: Vec::new(),
                        new_items: 0,
                        duration_ms,
                        error: Some(format!("{} after {} retries", e, retry_max)),
                    };
                }
            }
        }

        unreachable!()
    }

    async fn try_fetch(client: &Client, feed: &Feed) -> Result<Vec<FeedItem>, FetchError> {
        let mut request = client.get(&feed.url);

        for (key, value) in &feed.headers {
            request = request.header(key, value);
        }

        let response = request.send().await.map_err(|e| {
            if e.is_timeout() {
                FetchError::Timeout
            } else if e.is_connect() {
                FetchError::Connect
            } else {
                FetchError::Network(e.to_string())
            }
        })?;

        let status = response.status();
        if !status.is_success() {
            return Err(FetchError::Http(status));
        }

        let body = response.text().await.map_err(|e| FetchError::Network(e.to_string()))?;

        // Parse feed
        Parser::parse(&feed.name, &feed.feed_type, &body)
            .map_err(|e| FetchError::Parse(e))
    }

}

pub fn print_result(result: &FetchResult) {
    if let Some(error) = &result.error {
        eprintln!("  ✗ {:<25} — error: {}", result.source, error);
    } else {
        println!(
            "  ✓ {:<25} — {} items ({} new) in {}ms",
            result.source,
            result.items.len(),
            result.new_items,
            result.duration_ms
        );
    }
}

#[derive(Debug)]
enum FetchError {
    Timeout,
    Connect,
    Network(String),
    Http(reqwest::StatusCode),
    Parse(String),
}

impl std::fmt::Display for FetchError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            FetchError::Timeout => write!(f, "HTTP timeout"),
            FetchError::Connect => write!(f, "DNS resolution failure"),
            FetchError::Network(msg) => write!(f, "network error: {}", msg),
            FetchError::Http(status) => write!(f, "HTTP {}", status),
            FetchError::Parse(msg) => write!(f, "parse error: {}", msg),
        }
    }
}
