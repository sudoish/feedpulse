use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::Path;
use url::Url;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(default)]
    pub settings: Settings,
    #[serde(default)]
    pub feeds: Vec<Feed>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Settings {
    #[serde(default = "default_max_concurrency")]
    pub max_concurrency: usize,
    #[serde(default = "default_timeout_secs")]
    pub default_timeout_secs: u64,
    #[serde(default = "default_retry_max")]
    pub retry_max: usize,
    #[serde(default = "default_retry_base_delay_ms")]
    pub retry_base_delay_ms: u64,
    #[serde(default = "default_database_path")]
    pub database_path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Feed {
    pub name: String,
    pub url: String,
    pub feed_type: String,
    #[serde(default = "default_refresh_interval")]
    pub refresh_interval_secs: u64,
    #[serde(default)]
    pub headers: HashMap<String, String>,
}

fn default_max_concurrency() -> usize { 5 }
fn default_timeout_secs() -> u64 { 10 }
fn default_retry_max() -> usize { 3 }
fn default_retry_base_delay_ms() -> u64 { 500 }
fn default_database_path() -> String { "feedpulse.db".to_string() }
fn default_refresh_interval() -> u64 { 300 }

impl Default for Settings {
    fn default() -> Self {
        Self {
            max_concurrency: default_max_concurrency(),
            default_timeout_secs: default_timeout_secs(),
            retry_max: default_retry_max(),
            retry_base_delay_ms: default_retry_base_delay_ms(),
            database_path: default_database_path(),
        }
    }
}

impl Config {
    pub fn load<P: AsRef<Path>>(path: P) -> Result<Self, String> {
        let content = fs::read_to_string(&path)
            .map_err(|e| format!("Failed to read config file: {}", e))?;
        
        let config: Config = serde_yaml::from_str(&content)
            .map_err(|e| format!("invalid config: {}", e))?;
        
        Ok(config)
    }

    pub fn validate(&self) -> Result<(), String> {
        // Validate settings
        if self.settings.max_concurrency < 1 || self.settings.max_concurrency > 50 {
            return Err(format!(
                "max_concurrency must be between 1-50, got {}",
                self.settings.max_concurrency
            ));
        }

        if self.settings.default_timeout_secs == 0 {
            return Err("default_timeout_secs must be positive".to_string());
        }

        // Validate feeds
        for feed in &self.feeds {
            // Name validation
            if feed.name.trim().is_empty() {
                return Err(format!("feed '{}': name cannot be empty", feed.name));
            }

            // URL validation
            if feed.url.trim().is_empty() {
                return Err(format!("feed '{}': missing field 'url'", feed.name));
            }

            Url::parse(&feed.url).map_err(|_| {
                format!("feed '{}': invalid URL '{}'", feed.name, feed.url)
            })?;

            // feed_type validation
            if !["json", "rss", "atom"].contains(&feed.feed_type.as_str()) {
                return Err(format!(
                    "feed '{}': feed_type must be one of: json, rss, atom (got '{}')",
                    feed.name, feed.feed_type
                ));
            }

            // refresh_interval validation
            if feed.refresh_interval_secs == 0 {
                return Err(format!(
                    "feed '{}': refresh_interval_secs must be positive",
                    feed.name
                ));
            }
        }

        Ok(())
    }
}
