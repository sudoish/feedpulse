"""Tests for feed parsing and normalization"""

import pytest
import json

from feedpulse.parser import (
    parse_hackernews, parse_github, parse_reddit, parse_lobsters,
    parse_feed, coerce_to_string, parse_timestamp
)


def test_coerce_to_string():
    """Test string coercion"""
    assert coerce_to_string("hello") == "hello"
    assert coerce_to_string(123) == "123"
    assert coerce_to_string(45.67) == "45.67"
    assert coerce_to_string(True) == "True"


def test_parse_timestamp_unix():
    """Test parsing unix timestamp"""
    ts = parse_timestamp(1704067200)
    assert ts.startswith("2024")


def test_parse_timestamp_iso():
    """Test parsing ISO 8601 timestamp"""
    ts = parse_timestamp("2024-01-01T12:00:00Z")
    assert "2024-01-01" in ts


def test_parse_timestamp_invalid():
    """Test parsing invalid timestamp"""
    assert parse_timestamp("not-a-timestamp") is None
    assert parse_timestamp(None) is None
    assert parse_timestamp("") is None
    assert parse_timestamp([]) is None


def test_parse_hackernews_valid():
    """Test parsing HackerNews response"""
    data = [12345, 23456, 34567]
    items = parse_hackernews(data, "HN Test")
    
    assert len(items) == 3
    assert items[0].title == "HN Story 12345"
    assert items[0].url == "https://news.ycombinator.com/item?id=12345"
    assert items[0].source == "HN Test"


def test_parse_hackernews_invalid_type():
    """Test parsing HackerNews with wrong type"""
    data = {"not": "a list"}
    items = parse_hackernews(data, "HN Test")
    assert len(items) == 0


def test_parse_hackernews_mixed_types():
    """Test parsing HackerNews with mixed types"""
    data = [123, "not-an-int", 456]
    items = parse_hackernews(data, "HN Test")
    # Should skip the invalid one
    assert len(items) == 2


def test_parse_github_valid():
    """Test parsing GitHub response"""
    data = {
        "items": [
            {
                "full_name": "user/repo",
                "html_url": "https://github.com/user/repo",
                "topics": ["python", "cli"],
                "updated_at": "2024-01-01T12:00:00Z"
            }
        ]
    }
    
    items = parse_github(data, "GitHub Test")
    assert len(items) == 1
    assert items[0].title == "user/repo"
    assert items[0].url == "https://github.com/user/repo"
    assert items[0].tags == ["python", "cli"]
    assert items[0].source == "GitHub Test"


def test_parse_github_missing_title():
    """Test parsing GitHub with missing title"""
    data = {
        "items": [
            {
                "html_url": "https://github.com/user/repo"
            }
        ]
    }
    
    items = parse_github(data, "GitHub Test")
    # Should skip item without title
    assert len(items) == 0


def test_parse_github_missing_url():
    """Test parsing GitHub with missing URL"""
    data = {
        "items": [
            {
                "full_name": "user/repo"
            }
        ]
    }
    
    items = parse_github(data, "GitHub Test")
    # Should skip item without URL
    assert len(items) == 0


def test_parse_reddit_valid():
    """Test parsing Reddit response"""
    data = {
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
    }
    
    items = parse_reddit(data, "Reddit Test")
    assert len(items) == 1
    assert items[0].title == "Test Post"
    assert items[0].url == "https://reddit.com/r/test/comments/123"
    assert items[0].tags == ["Discussion"]
    assert items[0].source == "Reddit Test"


def test_parse_reddit_invalid_structure():
    """Test parsing Reddit with invalid structure"""
    data = {"not": "valid"}
    items = parse_reddit(data, "Reddit Test")
    assert len(items) == 0


def test_parse_lobsters_valid():
    """Test parsing Lobsters response"""
    data = [
        {
            "title": "Test Article",
            "url": "https://example.com/article",
            "created_at": "2024-01-01T12:00:00Z",
            "tags": ["programming", "python"]
        }
    ]
    
    items = parse_lobsters(data, "Lobsters Test")
    assert len(items) == 1
    assert items[0].title == "Test Article"
    assert items[0].url == "https://example.com/article"
    assert items[0].tags == ["programming", "python"]
    assert items[0].source == "Lobsters Test"


def test_parse_lobsters_fallback_url():
    """Test parsing Lobsters with fallback to comments_url"""
    data = [
        {
            "title": "Test Article",
            "comments_url": "https://lobste.rs/s/abc123",
            "created_at": "2024-01-01T12:00:00Z",
            "tags": []
        }
    ]
    
    items = parse_lobsters(data, "Lobsters Test")
    assert len(items) == 1
    assert items[0].url == "https://lobste.rs/s/abc123"


def test_parse_feed_malformed_json():
    """Test parsing malformed JSON"""
    malformed = "{not valid json"
    items = parse_feed(malformed, "json", "Test Source")
    assert len(items) == 0


def test_parse_feed_empty_response():
    """Test parsing empty response"""
    items = parse_feed("[]", "json", "Test Source")
    assert len(items) == 0


def test_parse_feed_unicode():
    """Test parsing Unicode content"""
    data = json.dumps([
        {
            "title": "测试文章 🚀",
            "url": "https://example.com",
            "created_at": "2024-01-01T12:00:00Z",
            "tags": []
        }
    ])
    
    items = parse_feed(data, "json", "Test Source")
    assert len(items) == 1
    assert "测试" in items[0].title
    assert "🚀" in items[0].title
