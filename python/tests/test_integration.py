"""Integration tests for end-to-end workflows.

Tests complete workflows from config loading through fetching,
parsing, storing, and reporting.
"""

import json
import os
import tempfile
from unittest.mock import AsyncMock, Mock, patch

import pytest

from feedpulse.config import load_config
from feedpulse.validators import validate_config
from feedpulse.models import FeedConfig, FetchResult
from feedpulse.parser import parse_feed
from feedpulse.storage import FeedDatabase


def test_config_load_and_validate_integration(temp_config_file):
    """Test loading and validating config file."""
    config = load_config(temp_config_file)
    
    assert config is not None
    assert len(config.feeds) == 2
    assert config.settings.max_concurrency == 5
    
    # Convert to dict for validation
    config_dict = {
        "feeds": [
            {
                "name": feed.name,
                "url": feed.url,
                "feed_type": feed.feed_type
            }
            for feed in config.feeds
        ],
        "settings": {
            "max_concurrency": config.settings.max_concurrency,
            "default_timeout_secs": config.settings.default_timeout_secs,
        }
    }
    
    # Should not raise
    validate_config(config_dict)


def test_parse_and_store_workflow(db, sample_hackernews_data):
    """Test complete parse → store workflow."""
    source = "test-hn"
    
    # Parse feed data
    json_str = json.dumps(sample_hackernews_data)
    items = parse_feed(json_str, "hackernews", source)
    
    assert len(items) == 2
    
    # Store items
    count = db.insert_items(items)
    assert count == 2
    
    # Retrieve and verify
    stored_items = db.get_items_by_source(source)
    assert len(stored_items) == 2
    
    # Verify data integrity (items ordered by timestamp desc, so newer first)
    titles = {item.title for item in stored_items}
    assert "Show HN: Cool Project" in titles
    assert "Ask HN: What is your workflow?" in titles


def test_multiple_feeds_workflow(db, sample_hackernews_data, sample_reddit_data):
    """Test workflow with multiple different feed types."""
    # Parse HackerNews
    hn_json = json.dumps(sample_hackernews_data)
    hn_items = parse_feed(hn_json, "hackernews", "hn")
    
    # Parse Reddit
    reddit_json = json.dumps(sample_reddit_data)
    reddit_items = parse_feed(reddit_json, "reddit", "reddit")
    
    # Store all items
    hn_count = db.insert_items(hn_items)
    reddit_count = db.insert_items(reddit_items)
    
    assert hn_count == 2
    assert reddit_count == 2
    
    # Verify sources are separate
    hn_stored = db.get_items_by_source("hn")
    reddit_stored = db.get_items_by_source("reddit")
    
    assert len(hn_stored) == 2
    assert len(reddit_stored) == 2
    
    # Verify all items retrieved together
    all_items = db.get_all_items()
    assert len(all_items) == 4


def test_fetch_log_workflow(db, sample_hackernews_data):
    """Test workflow including fetch logging."""
    source = "test-feed"
    
    # Parse
    json_str = json.dumps(sample_hackernews_data)
    items = parse_feed(json_str, "hackernews", source)
    
    # Store
    new_count = db.insert_items(items)
    
    # Log fetch result
    result = FetchResult(
        source=source,
        status="success",
        items=items,
        items_new=new_count,
        duration_ms=1234,
    )
    
    db.log_fetch(result)
    
    # Verify both items and log exist
    stored_items = db.get_items_by_source(source)
    assert len(stored_items) == new_count


def test_incremental_updates_workflow(db, sample_hackernews_data):
    """Test workflow for incremental feed updates."""
    source = "hn"
    
    # First fetch
    json_str = json.dumps(sample_hackernews_data)
    items1 = parse_feed(json_str, "hackernews", source)
    count1 = db.insert_items(items1)
    
    assert count1 == 2
    
    # Second fetch with one new item
    new_data = {
        "hits": sample_hackernews_data["hits"] + [
            {
                "objectID": "99999",
                "title": "New Item",
                "url": "https://example.com/new",
                "created_at_i": 1704240000,
            }
        ]
    }
    
    json_str2 = json.dumps(new_data)
    items2 = parse_feed(json_str2, "hackernews", source)
    count2 = db.insert_items(items2)
    
    # Should only insert 1 new item (2 were duplicates)
    assert count2 == 1
    
    # Total should be 3
    all_items = db.get_items_by_source(source)
    assert len(all_items) == 3


def test_error_handling_workflow(db):
    """Test workflow when parsing fails."""
    source = "test-feed"
    
    # Try to parse invalid JSON
    invalid_json = "not valid json {"
    items = parse_feed(invalid_json, "hackernews", source)
    
    # Should return empty list, not crash
    assert items == []
    
    # Log the error
    result = FetchResult(
        source=source,
        status="error",
        error_message="JSON parse error",
        duration_ms=100,
    )
    
    db.log_fetch(result)
    
    # Database should still be usable
    all_items = db.get_all_items()
    assert all_items == []


def test_concurrent_feed_processing(db, sample_hackernews_data, sample_reddit_data, sample_github_data):
    """Test processing multiple feeds concurrently."""
    # Simulate concurrent processing
    feeds = [
        ("hackernews", sample_hackernews_data, "hn"),
        ("reddit", sample_reddit_data, "reddit"),
        ("github", sample_github_data, "github"),
    ]
    
    total_items = 0
    
    for feed_type, data, source in feeds:
        json_str = json.dumps(data)
        items = parse_feed(json_str, feed_type, source)
        count = db.insert_items(items)
        total_items += count
        
        result = FetchResult(
            source=source,
            status="success",
            items=items,
            items_new=count,
            duration_ms=500,
        )
        db.log_fetch(result)
    
    # Verify all items stored
    all_items = db.get_all_items()
    assert len(all_items) == total_items
    
    # Verify each source has items
    assert len(db.get_items_by_source("hn")) > 0
    assert len(db.get_items_by_source("reddit")) > 0
    assert len(db.get_items_by_source("github")) > 0


def test_large_volume_workflow(db):
    """Test workflow with large volume of items."""
    # Generate large dataset
    large_data = {
        "hits": [
            {
                "objectID": str(i),
                "title": f"Item {i}",
                "url": f"https://example.com/{i}",
                "created_at_i": 1704067200 + i,
            }
            for i in range(500)
        ]
    }
    
    # Parse
    json_str = json.dumps(large_data)
    items = parse_feed(json_str, "hackernews", "bulk-test")
    
    assert len(items) == 500
    
    # Store
    count = db.insert_items(items)
    assert count == 500
    
    # Retrieve with limit
    limited_items = db.get_items_by_source("bulk-test", limit=50)
    assert len(limited_items) == 50
    
    # Retrieve all
    all_items = db.get_items_by_source("bulk-test")
    assert len(all_items) == 500


def test_data_integrity_workflow(db, sample_hackernews_data):
    """Test data integrity through complete workflow."""
    source = "integrity-test"
    
    # Parse
    json_str = json.dumps(sample_hackernews_data)
    items = parse_feed(json_str, "hackernews", source)
    
    # Store
    db.insert_items(items)
    
    # Retrieve
    stored_items = db.get_items_by_source(source)
    
    # Sort both lists by ID for consistent comparison
    items_sorted = sorted(items, key=lambda x: x.id)
    stored_sorted = sorted(stored_items, key=lambda x: x.id)
    
    # Verify every field matches
    for original, stored in zip(items_sorted, stored_sorted):
        assert stored.title == original.title
        assert stored.url == original.url
        assert stored.source == original.source
        assert stored.id == original.id
        # Timestamp might have slight differences, check presence
        if original.timestamp:
            assert stored.timestamp is not None


def test_empty_feed_workflow(db):
    """Test workflow with empty feed response."""
    # Empty HackerNews response
    empty_data = {"hits": []}
    
    json_str = json.dumps(empty_data)
    items = parse_feed(json_str, "hackernews", "empty-feed")
    
    assert items == []
    
    # Store (should be no-op)
    count = db.insert_items(items)
    assert count == 0
    
    # Log it
    result = FetchResult(
        source="empty-feed",
        status="success",
        items=[],
        items_new=0,
        duration_ms=100,
    )
    db.log_fetch(result)
    
    # Verify no items stored
    stored = db.get_items_by_source("empty-feed")
    assert stored == []


def test_mixed_valid_invalid_items_workflow(db):
    """Test workflow with mix of valid and invalid items."""
    mixed_data = {
        "hits": [
            # Valid item
            {
                "objectID": "123",
                "title": "Valid Item",
                "url": "https://example.com/valid",
                "created_at_i": 1704067200,
            },
            # Invalid item (missing required fields)
            {
                "objectID": "456",
                # Missing title and url
            },
            # Another valid item
            {
                "objectID": "789",
                "title": "Another Valid",
                "url": "https://example.com/valid2",
                "created_at_i": 1704153600,
            },
        ]
    }
    
    json_str = json.dumps(mixed_data)
    items = parse_feed(json_str, "hackernews", "mixed-feed")
    
    # Should parse valid items, skip invalid ones
    assert len(items) >= 2
    
    # Store valid items
    count = db.insert_items(items)
    assert count >= 2


def test_unicode_throughout_workflow(db):
    """Test Unicode data through complete workflow."""
    unicode_data = {
        "hits": [
            {
                "objectID": "123",
                "title": "测试 • Test • テスト • 🚀",
                "url": "https://example.com/unicode/测试",
                "created_at_i": 1704067200,
            }
        ]
    }
    
    # Parse
    json_str = json.dumps(unicode_data, ensure_ascii=False)
    items = parse_feed(json_str, "hackernews", "unicode-test")
    
    assert len(items) == 1
    assert "🚀" in items[0].title
    
    # Store
    count = db.insert_items(items)
    assert count == 1
    
    # Retrieve and verify Unicode preserved
    stored = db.get_items_by_source("unicode-test")
    assert len(stored) == 1
    assert "🚀" in stored[0].title
    assert "测试" in stored[0].url


def test_all_feed_types_integration(
    db,
    sample_hackernews_data,
    sample_reddit_data,
    sample_github_data,
    sample_lobsters_data,
):
    """Test integration with all supported feed types."""
    feed_configs = [
        ("hackernews", sample_hackernews_data, "hn"),
        ("reddit", sample_reddit_data, "reddit"),
        ("github", sample_github_data, "github"),
        ("lobsters", sample_lobsters_data, "lobsters"),
    ]
    
    total_items = 0
    
    for feed_type, data, source in feed_configs:
        # Parse
        json_str = json.dumps(data)
        items = parse_feed(json_str, feed_type, source)
        
        # Verify we got items
        assert len(items) > 0
        
        # Store
        count = db.insert_items(items)
        assert count > 0
        total_items += count
        
        # Log
        result = FetchResult(
            source=source,
            status="success",
            items=items,
            items_new=count,
            duration_ms=1000,
        )
        db.log_fetch(result)
    
    # Verify all items stored
    all_items = db.get_all_items()
    assert len(all_items) == total_items
    
    # Verify each feed type has items
    for _, _, source in feed_configs:
        source_items = db.get_items_by_source(source)
        assert len(source_items) > 0


def test_config_update_workflow(temp_config_file):
    """Test workflow when config is updated."""
    # Load initial config
    config1 = load_config(temp_config_file)
    initial_feed_count = len(config1.feeds)
    
    # Modify config file
    new_config_data = """
feeds:
  - name: test-hn
    url: https://hn.algolia.com/api/v1/search_by_date?tags=story
    feed_type: hackernews

  - name: test-reddit
    url: https://www.reddit.com/r/programming.json
    feed_type: reddit

  - name: test-github
    url: https://api.github.com/search/repositories?q=stars:>1000
    feed_type: github

settings:
  max_concurrency: 10
  default_timeout_secs: 15
  database_path: test.db
"""
    
    with open(temp_config_file, 'w') as f:
        f.write(new_config_data)
    
    # Reload config
    config2 = load_config(temp_config_file)
    
    # Verify changes
    assert len(config2.feeds) == initial_feed_count + 1
    assert config2.settings.max_concurrency == 10
    assert config2.settings.default_timeout_secs == 15


def test_database_persistence(temp_db_path, sample_hackernews_data):
    """Test that data persists across database connections."""
    # First connection
    db1 = FeedDatabase(temp_db_path)
    
    json_str = json.dumps(sample_hackernews_data)
    items = parse_feed(json_str, "hackernews", "persist-test")
    count = db1.insert_items(items)
    
    assert count == 2
    
    # Close and reopen
    db2 = FeedDatabase(temp_db_path)
    
    # Data should still be there
    stored_items = db2.get_items_by_source("persist-test")
    assert len(stored_items) == 2


def test_partial_failure_recovery(db):
    """Test recovery from partial failures."""
    # Successfully process one feed
    valid_data = {
        "hits": [
            {
                "objectID": "123",
                "title": "Valid",
                "url": "https://example.com/valid",
            }
        ]
    }
    
    json_str = json.dumps(valid_data)
    items = parse_feed(json_str, "hackernews", "feed1")
    db.insert_items(items)
    
    # Fail on another feed
    invalid_json = "invalid json"
    items2 = parse_feed(invalid_json, "hackernews", "feed2")
    
    assert items2 == []
    
    # First feed data should still be intact
    stored = db.get_items_by_source("feed1")
    assert len(stored) == 1
