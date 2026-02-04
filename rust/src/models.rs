use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Unified feed item schema after normalization
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeedItem {
    /// SHA256 hash of source_name + url
    pub id: String,
    /// Item title (required)
    pub title: String,
    /// Link to original item (required)
    pub url: String,
    /// Feed name from config (required)
    pub source: String,
    /// ISO 8601 datetime if available
    pub timestamp: Option<String>,
    /// Tags/categories
    #[serde(default)]
    pub tags: Vec<String>,
    /// Original JSON for debugging
    pub raw_data: Option<String>,
}

/// Result of fetching and parsing a single feed
#[derive(Debug, Clone)]
pub struct FeedResult {
    pub source_name: String,
    pub status: FetchStatus,
    pub items: Vec<FeedItem>,
    pub duration_ms: u64,
    pub error_message: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum FetchStatus {
    Success,
    Error,
}

impl FetchStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            FetchStatus::Success => "success",
            FetchStatus::Error => "error",
        }
    }
}

/// Feed type enumeration
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum FeedType {
    Json,
    Rss,
    Atom,
}

/// Configuration for a single feed source
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeedConfig {
    pub name: String,
    pub url: String,
    pub feed_type: FeedType,
    #[serde(default = "default_refresh_interval")]
    pub refresh_interval_secs: u64,
    #[serde(default)]
    pub headers: HashMap<String, String>,
}

fn default_refresh_interval() -> u64 {
    300
}

/// Global settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Settings {
    #[serde(default = "default_max_concurrency")]
    pub max_concurrency: usize,
    #[serde(default = "default_timeout")]
    pub default_timeout_secs: u64,
    #[serde(default = "default_retry_max")]
    pub retry_max: u32,
    #[serde(default = "default_retry_base_delay")]
    pub retry_base_delay_ms: u64,
    #[serde(default = "default_database_path")]
    pub database_path: String,
}

fn default_max_concurrency() -> usize {
    5
}

fn default_timeout() -> u64 {
    10
}

fn default_retry_max() -> u32 {
    3
}

fn default_retry_base_delay() -> u64 {
    500
}

fn default_database_path() -> String {
    "feedpulse.db".to_string()
}

impl Default for Settings {
    fn default() -> Self {
        Settings {
            max_concurrency: default_max_concurrency(),
            default_timeout_secs: default_timeout(),
            retry_max: default_retry_max(),
            retry_base_delay_ms: default_retry_base_delay(),
            database_path: default_database_path(),
        }
    }
}

/// Top-level configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(default)]
    pub settings: Settings,
    pub feeds: Vec<FeedConfig>,
}

/// Report entry for a single source
#[derive(Debug, Serialize)]
pub struct SourceReport {
    pub source_name: String,
    pub total_items: i64,
    pub error_count: i64,
    pub last_success: Option<String>,
}

impl SourceReport {
    pub fn error_rate(&self) -> f64 {
        let total_fetches = self.total_items + self.error_count;
        if total_fetches == 0 {
            0.0
        } else {
            (self.error_count as f64 / total_fetches as f64) * 100.0
        }
    }
}
