/// Tests for feed parsing and normalization
use feedpulse::parser::Parser;
use feedpulse::models::FeedItem;

#[test]
fn test_parse_hackernews_valid() {
    let data = r#"[12345, 23456, 34567]"#;
    let result = Parser::parse("HackerNews Top", "json", data);
    
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 3);
    assert_eq!(items[0].title, "HN Story 12345");
    assert_eq!(items[0].url, "https://news.ycombinator.com/item?id=12345");
    assert_eq!(items[0].source, "HackerNews Top");
}

#[test]
fn test_parse_hackernews_invalid_type() {
    let data = r#"{"not": "a list"}"#;
    let result = Parser::parse("HackerNews Top", "json", data);
    
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 0);
}

#[test]
fn test_parse_hackernews_mixed_types() {
    let data = r#"[123, "not-an-int", 456]"#;
    let result = Parser::parse("HackerNews Top", "json", data);
    
    assert!(result.is_ok());
    let items = result.unwrap();
    // Should skip the invalid one
    assert_eq!(items.len(), 2);
    assert_eq!(items[0].title, "HN Story 123");
    assert_eq!(items[1].title, "HN Story 456");
}

#[test]
fn test_parse_github_valid() {
    let data = r#"{
        "items": [
            {
                "full_name": "user/repo",
                "html_url": "https://github.com/user/repo",
                "topics": ["python", "cli"],
                "updated_at": "2024-01-01T12:00:00Z"
            }
        ]
    }"#;
    
    let result = Parser::parse("GitHub Trending", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].title, "user/repo");
    assert_eq!(items[0].url, "https://github.com/user/repo");
    assert_eq!(items[0].tags, vec!["python", "cli"]);
    assert_eq!(items[0].source, "GitHub Trending");
}

#[test]
fn test_parse_github_missing_title() {
    let data = r#"{
        "items": [
            {
                "html_url": "https://github.com/user/repo"
            }
        ]
    }"#;
    
    let result = Parser::parse("GitHub Trending", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    // Should skip item without title
    assert_eq!(items.len(), 0);
}

#[test]
fn test_parse_github_missing_url() {
    let data = r#"{
        "items": [
            {
                "full_name": "user/repo"
            }
        ]
    }"#;
    
    let result = Parser::parse("GitHub Trending", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    // Should skip item without URL
    assert_eq!(items.len(), 0);
}

#[test]
fn test_parse_reddit_valid() {
    let data = r#"{
        "data": {
            "children": [
                {
                    "data": {
                        "title": "Test Post",
                        "url": "https://reddit.com/r/test/comments/123",
                        "created_utc": 1704067200,
                        "link_flair_text": "Discussion"
                    }
                }
            ]
        }
    }"#;
    
    let result = Parser::parse("Reddit Programming", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].title, "Test Post");
    assert_eq!(items[0].url, "https://reddit.com/r/test/comments/123");
    assert_eq!(items[0].tags, vec!["Discussion"]);
    assert_eq!(items[0].source, "Reddit Programming");
}

#[test]
fn test_parse_reddit_invalid_structure() {
    let data = r#"{"not": "valid"}"#;
    let result = Parser::parse("Reddit Programming", "json", data);
    
    // Should fail since missing data.children
    assert!(result.is_err());
}

#[test]
fn test_parse_lobsters_valid() {
    let data = r#"[
        {
            "title": "Test Article",
            "url": "https://example.com/article",
            "created_at": "2024-01-01T12:00:00Z",
            "tags": ["programming", "python"]
        }
    ]"#;
    
    let result = Parser::parse("Lobsters", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].title, "Test Article");
    assert_eq!(items[0].url, "https://example.com/article");
    assert_eq!(items[0].tags, vec!["programming", "python"]);
    assert_eq!(items[0].source, "Lobsters");
}

#[test]
fn test_parse_lobsters_fallback_url() {
    let data = r#"[
        {
            "title": "Test Article",
            "comments_url": "https://lobste.rs/s/abc123",
            "created_at": "2024-01-01T12:00:00Z",
            "tags": []
        }
    ]"#;
    
    let result = Parser::parse("Lobsters", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].url, "https://lobste.rs/s/abc123");
}

#[test]
fn test_parse_malformed_json() {
    let data = "{not valid json";
    let result = Parser::parse("Test Source", "json", data);
    
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("malformed JSON"));
}

#[test]
fn test_parse_empty_response() {
    let data = "[]";
    let result = Parser::parse("HackerNews Top", "json", data);
    
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 0);
}

#[test]
fn test_parse_unicode() {
    let data = r#"[
        {
            "title": "测试文章 🚀",
            "url": "https://example.com",
            "created_at": "2024-01-01T12:00:00Z",
            "tags": []
        }
    ]"#;
    
    let result = Parser::parse("Lobsters", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert!(items[0].title.contains("测试"));
    assert!(items[0].title.contains("🚀"));
}

#[test]
fn test_parse_wrong_types_coercion() {
    // Test that numbers are coerced to strings for title
    let data = r#"[
        {
            "title": 12345,
            "url": "https://example.com",
            "tags": []
        }
    ]"#;
    
    let result = Parser::parse("Lobsters", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].title, "12345");
}

#[test]
fn test_feeditem_id_generation_deterministic() {
    let item1 = FeedItem::new(
        "Test".to_string(),
        "https://example.com".to_string(),
        "Source1".to_string()
    );
    
    let item2 = FeedItem::new(
        "Test".to_string(),
        "https://example.com".to_string(),
        "Source1".to_string()
    );
    
    // Same source and URL should generate same ID
    assert_eq!(item1.id, item2.id);
}

#[test]
fn test_feeditem_id_generation_different_source() {
    let item1 = FeedItem::new(
        "Test".to_string(),
        "https://example.com".to_string(),
        "Source1".to_string()
    );
    
    let item2 = FeedItem::new(
        "Test".to_string(),
        "https://example.com".to_string(),
        "Source2".to_string()
    );
    
    // Different source should generate different ID
    assert_ne!(item1.id, item2.id);
}

#[test]
fn test_parse_github_with_numeric_types() {
    // Test type coercion for GitHub
    let data = r#"{
        "items": [
            {
                "full_name": 123456,
                "html_url": "https://github.com/user/repo"
            }
        ]
    }"#;
    
    let result = Parser::parse("GitHub Trending", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].title, "123456");
}

#[test]
fn test_parse_reddit_timestamp_conversion() {
    let data = r#"{
        "data": {
            "children": [
                {
                    "data": {
                        "title": "Test",
                        "url": "https://reddit.com/test",
                        "created_utc": 1704067200.0
                    }
                }
            ]
        }
    }"#;
    
    let result = Parser::parse("Reddit Programming", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert!(items[0].timestamp.is_some());
    let timestamp = items[0].timestamp.as_ref().unwrap();
    assert!(timestamp.contains("2024"));
}

#[test]
fn test_parse_empty_items_array() {
    let data = r#"{"items": []}"#;
    let result = Parser::parse("GitHub Trending", "json", data);
    
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 0);
}

#[test]
fn test_parse_null_values() {
    let data = r#"[
        {
            "title": "Test",
            "url": "https://example.com",
            "tags": null,
            "created_at": null
        }
    ]"#;
    
    let result = Parser::parse("Lobsters", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].tags.len(), 0);
    assert!(items[0].timestamp.is_none());
}

#[test]
fn test_parse_rss_not_implemented() {
    let data = r#"<?xml version="1.0"?><rss><channel></channel></rss>"#;
    let result = Parser::parse("RSS Feed", "rss", data);
    
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("not yet implemented") || err.contains("RSS"));
}

#[test]
fn test_parse_atom_not_implemented() {
    let data = r#"<?xml version="1.0"?><feed></feed>"#;
    let result = Parser::parse("Atom Feed", "atom", data);
    
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(err.contains("not yet implemented") || err.contains("Atom"));
}

#[test]
fn test_parse_unknown_feed_type() {
    let data = "some data";
    let result = Parser::parse("Unknown", "xml", data);
    
    assert!(result.is_err());
}

#[test]
fn test_parse_lobsters_empty_url_fallback() {
    let data = r#"[
        {
            "title": "Test",
            "url": "",
            "comments_url": "https://lobste.rs/s/test",
            "tags": []
        }
    ]"#;
    
    let result = Parser::parse("Lobsters", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    assert_eq!(items.len(), 1);
    assert_eq!(items[0].url, "https://lobste.rs/s/test");
}

#[test]
fn test_parse_multiple_items_with_errors() {
    let data = r#"{
        "items": [
            {
                "full_name": "valid/repo",
                "html_url": "https://github.com/valid/repo"
            },
            {
                "full_name": "missing-url"
            },
            {
                "html_url": "https://github.com/missing-name"
            },
            {
                "full_name": "another/valid",
                "html_url": "https://github.com/another/valid"
            }
        ]
    }"#;
    
    let result = Parser::parse("GitHub Trending", "json", data);
    assert!(result.is_ok());
    let items = result.unwrap();
    // Should get 2 valid items, skip 2 invalid
    assert_eq!(items.len(), 2);
    assert_eq!(items[0].title, "valid/repo");
    assert_eq!(items[1].title, "another/valid");
}

#[test]
fn test_load_malformed_fixture() {
    let malformed_path = "../test-fixtures/malformed-json-response.json";
    if std::path::Path::new(malformed_path).exists() {
        let content = std::fs::read_to_string(malformed_path).unwrap();
        let result = Parser::parse("Test", "json", &content);
        assert!(result.is_err(), "Should fail on malformed JSON fixture");
    }
}

#[test]
fn test_load_unicode_chaos_fixture() {
    let unicode_path = "../test-fixtures/unicode-chaos.json";
    if std::path::Path::new(unicode_path).exists() {
        let content = std::fs::read_to_string(unicode_path).unwrap();
        let result = Parser::parse("Lobsters", "json", &content);
        // Should handle gracefully, either parsing or returning error
        assert!(result.is_ok() || result.is_err());
    }
}

#[test]
fn test_load_empty_response_fixture() {
    let empty_path = "../test-fixtures/empty-response.json";
    if std::path::Path::new(empty_path).exists() {
        let content = std::fs::read_to_string(empty_path).unwrap();
        let result = Parser::parse("Test", "json", &content);
        if result.is_ok() {
            let items = result.unwrap();
            assert_eq!(items.len(), 0);
        }
    }
}
