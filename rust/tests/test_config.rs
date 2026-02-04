/// Tests for configuration validation
use feedpulse::config::Config;
use std::fs;
use tempfile::NamedTempFile;

#[test]
fn test_config_missing_file() {
    let result = Config::load("/nonexistent/config.yaml");
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("Failed to read config file"), "Expected 'Failed to read config file', got: {}", err);
}

#[test]
fn test_config_invalid_yaml() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, "invalid: yaml: content: [").unwrap();
    
    let result = Config::load(temp_file.path());
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("invalid config"), "Expected 'invalid config', got: {}", err);
}

#[test]
fn test_config_missing_required_field() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
feeds:
  - name: "Test"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path());
    assert!(config.is_err() || config.unwrap().validate().is_err());
}

#[test]
fn test_config_invalid_url() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
feeds:
  - name: "Test"
    url: "not-a-valid-url"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("invalid URL"), "Expected 'invalid URL', got: {}", err);
}

#[test]
fn test_validate_feed_empty_name() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
feeds:
  - name: ""
    url: "https://example.com"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("name cannot be empty"), "Expected 'name cannot be empty', got: {}", err);
}

#[test]
fn test_validate_feed_missing_feed_type() {
    let yaml = r#"
feeds:
  - name: "Test"
    url: "https://example.com"
"#;
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, yaml).unwrap();
    
    // Should fail on deserialization since feed_type is required
    let result = Config::load(temp_file.path());
    assert!(result.is_err());
}

#[test]
fn test_validate_feed_invalid_feed_type() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: xml
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("must be one of: json, rss, atom"), "Expected feed_type error, got: {}", err);
}

#[test]
fn test_validate_settings_defaults() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    assert_eq!(config.settings.max_concurrency, 5);
    assert_eq!(config.settings.default_timeout_secs, 10);
    assert_eq!(config.settings.retry_max, 3);
    assert_eq!(config.settings.retry_base_delay_ms, 500);
    assert_eq!(config.settings.database_path, "feedpulse.db");
}

#[test]
fn test_validate_settings_custom() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
settings:
  max_concurrency: 10
  default_timeout_secs: 30
  retry_max: 5
  retry_base_delay_ms: 1000
  database_path: "custom.db"
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    assert_eq!(config.settings.max_concurrency, 10);
    assert_eq!(config.settings.default_timeout_secs, 30);
    assert_eq!(config.settings.retry_max, 5);
    assert_eq!(config.settings.retry_base_delay_ms, 1000);
    assert_eq!(config.settings.database_path, "custom.db");
}

#[test]
fn test_validate_settings_invalid_max_concurrency_zero() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
settings:
  max_concurrency: 0
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("between 1-50"), "Expected max_concurrency error, got: {}", err);
}

#[test]
fn test_validate_settings_invalid_max_concurrency_high() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
settings:
  max_concurrency: 51
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("between 1-50"), "Expected max_concurrency error, got: {}", err);
}

#[test]
fn test_validate_settings_invalid_timeout() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
settings:
  default_timeout_secs: 0
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: json
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("must be positive"), "Expected timeout error, got: {}", err);
}

#[test]
fn test_validate_feed_invalid_refresh_interval() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
feeds:
  - name: "Test"
    url: "https://example.com"
    feed_type: json
    refresh_interval_secs: 0
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    let result = config.validate();
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("must be positive"), "Expected refresh_interval error, got: {}", err);
}

#[test]
fn test_validate_url_formats() {
    // Valid URLs
    let valid_urls = vec![
        "https://example.com",
        "http://example.com",
        "https://example.com/path?query=1",
        "https://example.com:8080/path",
    ];
    
    for url in valid_urls {
        let yaml = format!(r#"
feeds:
  - name: "Test"
    url: "{}"
    feed_type: json
"#, url);
        let mut temp_file = NamedTempFile::new().unwrap();
        fs::write(&temp_file, yaml).unwrap();
        
        let config = Config::load(temp_file.path()).unwrap();
        assert!(config.validate().is_ok(), "Should accept valid URL: {}", url);
    }
}

#[test]
fn test_config_full_valid() {
    let mut temp_file = NamedTempFile::new().unwrap();
    fs::write(&temp_file, r#"
settings:
  max_concurrency: 5
  default_timeout_secs: 10
  retry_max: 3
  retry_base_delay_ms: 500
  database_path: "feedpulse.db"

feeds:
  - name: "Test Feed 1"
    url: "https://example.com/feed1"
    feed_type: json
    refresh_interval_secs: 300
    headers:
      User-Agent: "test"
  
  - name: "Test Feed 2"
    url: "https://example.com/feed2"
    feed_type: rss
"#).unwrap();
    
    let config = Config::load(temp_file.path()).unwrap();
    assert!(config.validate().is_ok());
    assert_eq!(config.feeds.len(), 2);
    assert_eq!(config.feeds[0].name, "Test Feed 1");
    assert_eq!(config.feeds[1].name, "Test Feed 2");
}

#[test]
fn test_load_from_test_fixtures() {
    // Test loading the valid config from test-fixtures
    let valid_config_path = "../test-fixtures/valid-config.yaml";
    if std::path::Path::new(valid_config_path).exists() {
        let config = Config::load(valid_config_path);
        assert!(config.is_ok(), "Should load valid config from test-fixtures");
    }
    
    // Test loading invalid configs
    let invalid_yaml_path = "../test-fixtures/invalid-config-bad-yaml.yaml";
    if std::path::Path::new(invalid_yaml_path).exists() {
        let config = Config::load(invalid_yaml_path);
        assert!(config.is_err(), "Should fail on bad YAML");
    }
    
    let missing_url_path = "../test-fixtures/invalid-config-missing-url.yaml";
    if std::path::Path::new(missing_url_path).exists() {
        let result = Config::load(missing_url_path);
        if let Ok(config) = result {
            assert!(config.validate().is_err(), "Should fail validation on missing URL");
        }
    }
}
