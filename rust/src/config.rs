use crate::models::{Config, FeedConfig};
use anyhow::{Context, Result};
use std::fs;
use std::path::Path;
use url::Url;

/// Load and validate configuration from a YAML file
pub fn load_config<P: AsRef<Path>>(path: P) -> Result<Config> {
    let path = path.as_ref();
    
    // Check if file exists
    if !path.exists() {
        anyhow::bail!("config file not found: {}", path.display());
    }
    
    // Read file contents
    let contents = fs::read_to_string(path)
        .with_context(|| format!("failed to read config file: {}", path.display()))?;
    
    // Parse YAML
    let config: Config = serde_yaml::from_str(&contents)
        .with_context(|| format!("invalid config: failed to parse YAML"))?;
    
    // Validate configuration
    validate_config(&config)?;
    
    Ok(config)
}

/// Validate the entire configuration
fn validate_config(config: &Config) -> Result<()> {
    // Validate settings
    validate_settings(config)?;
    
    // Validate each feed
    if config.feeds.is_empty() {
        anyhow::bail!("config must contain at least one feed");
    }
    
    for feed in &config.feeds {
        validate_feed(feed)?;
    }
    
    Ok(())
}

/// Validate global settings
fn validate_settings(config: &Config) -> Result<()> {
    let settings = &config.settings;
    
    // max_concurrency must be between 1 and 50
    if settings.max_concurrency < 1 || settings.max_concurrency > 50 {
        anyhow::bail!(
            "settings: max_concurrency must be between 1 and 50, got {}",
            settings.max_concurrency
        );
    }
    
    // default_timeout_secs must be positive
    if settings.default_timeout_secs == 0 {
        anyhow::bail!("settings: default_timeout_secs must be positive");
    }
    
    // retry_base_delay_ms must be positive
    if settings.retry_base_delay_ms == 0 {
        anyhow::bail!("settings: retry_base_delay_ms must be positive");
    }
    
    // database_path must not be empty
    if settings.database_path.is_empty() {
        anyhow::bail!("settings: database_path cannot be empty");
    }
    
    Ok(())
}

/// Validate a single feed configuration
fn validate_feed(feed: &FeedConfig) -> Result<()> {
    // Name must not be empty
    if feed.name.trim().is_empty() {
        anyhow::bail!("feed: name cannot be empty");
    }
    
    // URL must be valid and HTTP/HTTPS
    if feed.url.trim().is_empty() {
        anyhow::bail!("feed '{}': missing field 'url'", feed.name);
    }
    
    let parsed_url = Url::parse(&feed.url)
        .with_context(|| format!("feed '{}': invalid URL '{}'", feed.name, feed.url))?;
    
    let scheme = parsed_url.scheme();
    if scheme != "http" && scheme != "https" {
        anyhow::bail!(
            "feed '{}': URL must use http or https scheme, got '{}'",
            feed.name,
            scheme
        );
    }
    
    // refresh_interval_secs must be positive
    if feed.refresh_interval_secs == 0 {
        anyhow::bail!(
            "feed '{}': refresh_interval_secs must be positive",
            feed.name
        );
    }
    
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::{FeedType, Settings};
    use std::collections::HashMap;

    #[test]
    fn test_validate_empty_name() {
        let feed = FeedConfig {
            name: "".to_string(),
            url: "https://example.com".to_string(),
            feed_type: FeedType::Json,
            refresh_interval_secs: 300,
            headers: HashMap::new(),
        };
        
        assert!(validate_feed(&feed).is_err());
    }

    #[test]
    fn test_validate_missing_url() {
        let feed = FeedConfig {
            name: "Test Feed".to_string(),
            url: "".to_string(),
            feed_type: FeedType::Json,
            refresh_interval_secs: 300,
            headers: HashMap::new(),
        };
        
        let result = validate_feed(&feed);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("missing field 'url'"));
    }

    #[test]
    fn test_validate_invalid_url() {
        let feed = FeedConfig {
            name: "Test Feed".to_string(),
            url: "not-a-url".to_string(),
            feed_type: FeedType::Json,
            refresh_interval_secs: 300,
            headers: HashMap::new(),
        };
        
        let result = validate_feed(&feed);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("invalid URL"));
    }

    #[test]
    fn test_validate_non_http_url() {
        let feed = FeedConfig {
            name: "Test Feed".to_string(),
            url: "ftp://example.com".to_string(),
            feed_type: FeedType::Json,
            refresh_interval_secs: 300,
            headers: HashMap::new(),
        };
        
        let result = validate_feed(&feed);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("http or https"));
    }

    #[test]
    fn test_validate_valid_feed() {
        let feed = FeedConfig {
            name: "Test Feed".to_string(),
            url: "https://example.com".to_string(),
            feed_type: FeedType::Json,
            refresh_interval_secs: 300,
            headers: HashMap::new(),
        };
        
        assert!(validate_feed(&feed).is_ok());
    }

    #[test]
    fn test_validate_max_concurrency_bounds() {
        let mut config = Config {
            settings: Settings {
                max_concurrency: 0,
                ..Default::default()
            },
            feeds: vec![FeedConfig {
                name: "Test".to_string(),
                url: "https://example.com".to_string(),
                feed_type: FeedType::Json,
                refresh_interval_secs: 300,
                headers: HashMap::new(),
            }],
        };
        
        assert!(validate_settings(&config).is_err());
        
        config.settings.max_concurrency = 51;
        assert!(validate_settings(&config).is_err());
        
        config.settings.max_concurrency = 5;
        assert!(validate_settings(&config).is_ok());
    }
}
