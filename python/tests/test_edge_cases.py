"""Edge case and boundary condition tests.

Tests unusual inputs, extreme values, and corner cases that might
not be covered by normal functional tests.
"""

import json
import pytest

# validate_config is in validators module
from feedpulse.exceptions import ValidationError, URLValidationError
from feedpulse.models import FeedItem
from feedpulse.parser import (
    coerce_to_string,
    parse_feed,
    parse_github,
    parse_hackernews,
    parse_lobsters,
    parse_reddit,
    parse_timestamp,
)
from feedpulse.utils import (
    generate_id,
    safe_dict_get,
    truncate_string,
)
from feedpulse.validators import (
    validate_concurrency,
    validate_config,
    validate_feed_config,
    validate_timeout,
    validate_url,
)


# ============================================================================
# URL Validation Edge Cases
# ============================================================================


def test_url_validation_empty():
    """Test URL validation with empty string."""
    with pytest.raises(URLValidationError):
        validate_url("")


def test_url_validation_none_type():
    """Test URL validation with wrong type."""
    with pytest.raises(URLValidationError):
        validate_url(123)  # type: ignore


def test_url_validation_missing_scheme():
    """Test URL validation without scheme."""
    with pytest.raises(URLValidationError):
        validate_url("example.com")


def test_url_validation_unsupported_scheme():
    """Test URL validation with unsupported schemes."""
    with pytest.raises(URLValidationError):
        validate_url("ftp://example.com")
    
    with pytest.raises(URLValidationError):
        validate_url("file:///etc/passwd")


def test_url_validation_missing_hostname():
    """Test URL validation without hostname."""
    with pytest.raises(URLValidationError):
        validate_url("https://")


def test_url_validation_spaces():
    """Test URL validation with spaces."""
    with pytest.raises(URLValidationError):
        validate_url("https://example.com/path with spaces")


def test_url_validation_double_dots():
    """Test URL validation with double dots."""
    with pytest.raises(URLValidationError):
        validate_url("https://example..com")


def test_url_validation_valid_complex():
    """Test URL validation with valid complex URLs."""
    # Should not raise
    validate_url("https://example.com:8080/path?query=value&other=123#fragment")
    validate_url("http://sub.domain.example.com/very/long/path")
    validate_url("https://example.com/unicode/测试/path")


# ============================================================================
# Timeout & Concurrency Validation Edge Cases
# ============================================================================


def test_timeout_validation_zero():
    """Test timeout validation with zero."""
    with pytest.raises(ValidationError):
        validate_timeout(0)


def test_timeout_validation_negative():
    """Test timeout validation with negative value."""
    with pytest.raises(ValidationError):
        validate_timeout(-10)


def test_timeout_validation_too_large():
    """Test timeout validation with excessive value."""
    with pytest.raises(ValidationError):
        validate_timeout(999999)


def test_timeout_validation_wrong_type():
    """Test timeout validation with wrong type."""
    with pytest.raises(ValidationError):
        validate_timeout("10")  # type: ignore
    
    with pytest.raises(ValidationError):
        validate_timeout(10.5)  # type: ignore


def test_concurrency_validation_zero():
    """Test concurrency validation with zero."""
    with pytest.raises(ValidationError):
        validate_concurrency(0)


def test_concurrency_validation_negative():
    """Test concurrency validation with negative value."""
    with pytest.raises(ValidationError):
        validate_concurrency(-5)


def test_concurrency_validation_too_large():
    """Test concurrency validation with excessive value."""
    with pytest.raises(ValidationError):
        validate_concurrency(1000)


def test_concurrency_validation_wrong_type():
    """Test concurrency validation with wrong type."""
    with pytest.raises(ValidationError):
        validate_concurrency("5")  # type: ignore


# ============================================================================
# Timestamp Parsing Edge Cases
# ============================================================================


def test_parse_timestamp_zero():
    """Test parsing timestamp zero (Unix epoch)."""
    result = parse_timestamp(0)
    assert result is not None
    assert "1970-01-01" in result


def test_parse_timestamp_negative():
    """Test parsing negative timestamp (before epoch)."""
    result = parse_timestamp(-86400)
    assert result is not None
    assert "1969-12-31" in result


def test_parse_timestamp_far_future():
    """Test parsing timestamp far in the future."""
    # Year 2100
    result = parse_timestamp(4102444800)
    assert result is not None
    assert "2100" in result


def test_parse_timestamp_float():
    """Test parsing float timestamp with milliseconds."""
    result = parse_timestamp(1704067200.123)
    assert result is not None
    assert "2024" in result


def test_parse_timestamp_various_formats():
    """Test parsing various ISO 8601 formats."""
    formats = [
        "2024-01-01T12:00:00Z",
        "2024-01-01T12:00:00+00:00",
        "2024-01-01T12:00:00.123Z",
        "2024-01-01T12:00:00.123456+00:00",
    ]
    
    for fmt in formats:
        result = parse_timestamp(fmt)
        assert result is not None
        assert "2024-01-01" in result


def test_parse_timestamp_edge_values():
    """Test parsing edge case timestamp values."""
    assert parse_timestamp("") is None
    assert parse_timestamp("invalid") is None
    assert parse_timestamp(float('inf')) is None
    assert parse_timestamp(float('nan')) is None


# ============================================================================
# String Coercion Edge Cases
# ============================================================================


def test_coerce_to_string_none():
    """Test coercing None to string."""
    assert coerce_to_string(None) == ""


def test_coerce_to_string_empty_list():
    """Test coercing empty list."""
    assert coerce_to_string([]) == "[]"


def test_coerce_to_string_nested_structure():
    """Test coercing deeply nested structure."""
    nested = {"a": {"b": {"c": {"d": [1, 2, 3]}}}}
    result = coerce_to_string(nested)
    assert "a" in result
    assert "b" in result


def test_coerce_to_string_with_max_length():
    """Test coercing with max length truncation."""
    long_text = "A" * 1000
    result = coerce_to_string(long_text, max_length=100)
    assert len(result) == 100


def test_coerce_to_string_special_types():
    """Test coercing special Python types."""
    assert coerce_to_string(True) == "True"
    assert coerce_to_string(False) == "False"
    assert coerce_to_string(b"bytes") == "b'bytes'"


# ============================================================================
# Feed Parsing Edge Cases
# ============================================================================


def test_parse_feed_extremely_nested_json():
    """Test parsing deeply nested JSON structure."""
    # Create deeply nested structure
    data = {"level1": {"level2": {"level3": {"level4": {"level5": {
        "hits": [{"objectID": "123", "title": "Deep", "url": "https://example.com"}]
    }}}}}}
    
    json_str = json.dumps(data)
    # This should handle gracefully without stack overflow
    items = parse_feed(json_str, "hackernews", "test")
    # Might be empty due to unexpected structure, but shouldn't crash
    assert isinstance(items, list)


def test_parse_feed_very_large_json():
    """Test parsing very large JSON response."""
    # Create large response with many items
    large_data = {
        "hits": [
            {
                "objectID": str(i),
                "title": f"Item {i}",
                "url": f"https://example.com/{i}",
                "created_at_i": 1704067200 + i,
            }
            for i in range(1000)
        ]
    }
    
    json_str = json.dumps(large_data)
    items = parse_feed(json_str, "hackernews", "test")
    
    assert len(items) == 1000


def test_parse_feed_null_values_everywhere():
    """Test parsing feed with null values in all fields."""
    data = {
        "hits": [
            {
                "objectID": None,
                "title": None,
                "url": None,
                "created_at_i": None,
            }
        ]
    }
    
    json_str = json.dumps(data)
    items = parse_feed(json_str, "hackernews", "test")
    
    # Should handle gracefully, might skip items with null required fields
    assert isinstance(items, list)


def test_parse_feed_mixed_types_in_array():
    """Test parsing when array contains mixed types."""
    data = {
        "hits": [
            {"objectID": "123", "title": "Valid", "url": "https://example.com"},
            "not a dict",
            123,
            None,
            {"objectID": "456", "title": "Also Valid", "url": "https://example.com/2"},
        ]
    }
    
    json_str = json.dumps(data)
    items = parse_feed(json_str, "hackernews", "test")
    
    # Should skip invalid items but parse valid ones
    assert len(items) >= 2


def test_parse_reddit_missing_data_key():
    """Test parsing Reddit response missing 'data' key."""
    invalid_data = {"children": []}
    items = parse_reddit(invalid_data, "test")
    assert items == []


def test_parse_reddit_missing_children_key():
    """Test parsing Reddit response missing 'children' key."""
    invalid_data = {"data": {}}
    items = parse_reddit(invalid_data, "test")
    assert items == []


def test_parse_github_missing_items_key():
    """Test parsing GitHub response missing 'items' key."""
    invalid_data = {"total_count": 0}
    items = parse_github(invalid_data, "test")
    assert items == []


def test_parse_lobsters_not_a_list():
    """Test parsing Lobsters response that's not a list."""
    invalid_data = {"stories": []}
    items = parse_lobsters(invalid_data, "test")
    assert items == []


def test_parse_hackernews_not_a_dict():
    """Test parsing HackerNews response that's neither dict nor list."""
    invalid_data = "not a dict or list"
    items = parse_hackernews(invalid_data, "test")
    assert items == []


# ============================================================================
# FeedItem Edge Cases
# ============================================================================


def test_feed_item_id_generation_consistency():
    """Test that same inputs always generate same ID."""
    id1 = generate_id("title", "https://example.com")
    id2 = generate_id("title", "https://example.com")
    
    assert id1 == id2
    assert len(id1) == 64  # SHA256 hex


def test_feed_item_id_generation_uniqueness():
    """Test that different inputs generate different IDs."""
    id1 = generate_id("title1", "https://example.com")
    id2 = generate_id("title2", "https://example.com")
    id3 = generate_id("title1", "https://different.com")
    
    assert id1 != id2
    assert id1 != id3
    assert id2 != id3


def test_feed_item_with_empty_strings():
    """Test creating FeedItem with empty strings."""
    item = FeedItem.create(
        title="",
        url="",
        source="",
    )
    
    assert item.title == ""
    assert item.url == ""
    assert item.source == ""


def test_feed_item_with_very_long_values():
    """Test creating FeedItem with extremely long values."""
    long_title = "A" * 100000
    long_url = f"https://example.com/{'x' * 50000}"
    
    item = FeedItem.create(
        title=long_title,
        url=long_url,
        source="test",
    )
    
    assert len(item.title) == 100000
    assert len(item.url) > 50000


def test_feed_item_with_unicode():
    """Test creating FeedItem with Unicode characters."""
    item = FeedItem.create(
        title="测试 • Test • テスト • 🚀 • مرحبا",
        url="https://example.com/unicode/测试",
        source="test",
        tags=["日本語", "中文", "العربية", "emoji🎉"],
    )
    
    assert "🚀" in item.title
    assert "测试" in item.url
    assert "emoji🎉" in item.tags


def test_feed_item_with_special_characters():
    """Test creating FeedItem with special characters."""
    item = FeedItem.create(
        title='Test with "quotes" and \'apostrophes\' and <html>',
        url="https://example.com/path?query=value&other=123#fragment",
        source="test-source",
    )
    
    assert '"quotes"' in item.title
    assert "<html>" in item.title


# ============================================================================
# Utility Function Edge Cases
# ============================================================================


def test_safe_dict_get_with_non_dict():
    """Test safe_dict_get with non-dictionary value."""
    assert safe_dict_get("not a dict", "key", default="missing") == "missing"
    assert safe_dict_get(123, "key", default="missing") == "missing"
    assert safe_dict_get(None, "key", default="missing") == "missing"


def test_safe_dict_get_deep_path():
    """Test safe_dict_get with very deep path."""
    data = {"a": {"b": {"c": {"d": {"e": {"f": "found"}}}}}}
    
    assert safe_dict_get(data, "a", "b", "c", "d", "e", "f") == "found"
    assert safe_dict_get(data, "a", "b", "x", "d", "e", "f", default="missing") == "missing"


def test_safe_dict_get_with_none_values():
    """Test safe_dict_get when path contains None values."""
    data = {"a": None, "b": {"c": None}}
    
    assert safe_dict_get(data, "a", "nested", default="missing") == "missing"
    assert safe_dict_get(data, "b", "c", "nested", default="missing") == "missing"


def test_truncate_string_edge_cases():
    """Test truncate_string with edge cases."""
    # String shorter than max_length
    assert truncate_string("short", 100) == "short"
    
    # String exactly max_length
    assert truncate_string("exact", 5) == "exact"
    
    # Empty string
    assert truncate_string("", 10) == ""
    
    # max_length shorter than suffix
    result = truncate_string("long text here", 2, suffix="...")
    assert len(result) == 2


# ============================================================================
# Config Validation Edge Cases
# ============================================================================


def test_validate_config_empty_feeds():
    """Test config validation with empty feeds list."""
    config = {"feeds": []}
    
    with pytest.raises(ValidationError):
        validate_config(config)


def test_validate_config_missing_feeds():
    """Test config validation without feeds section."""
    config = {"settings": {}}
    
    with pytest.raises(ValidationError):
        validate_config(config)


def test_validate_config_feeds_not_list():
    """Test config validation when feeds is not a list."""
    config = {"feeds": "not a list"}
    
    with pytest.raises(ValidationError):
        validate_config(config)


def test_validate_feed_config_missing_url():
    """Test feed config validation without URL."""
    feed = {"feed_type": "hackernews"}
    
    with pytest.raises(ValidationError):
        validate_feed_config(feed)


def test_validate_feed_config_missing_type():
    """Test feed config validation without feed_type."""
    feed = {"url": "https://example.com"}
    
    with pytest.raises(ValidationError):
        validate_feed_config(feed)


def test_validate_feed_config_invalid_type():
    """Test feed config validation with invalid feed_type."""
    feed = {
        "url": "https://example.com",
        "feed_type": "invalid-type"
    }
    
    with pytest.raises(ValidationError):
        validate_feed_config(feed)


# ============================================================================
# JSON Parsing Edge Cases
# ============================================================================


def test_parse_feed_json_with_comments():
    """Test parsing JSON with comments (invalid JSON)."""
    json_with_comments = """
    {
        // This is a comment
        "hits": []
    }
    """
    
    items = parse_feed(json_with_comments, "hackernews", "test")
    # Should handle gracefully, return empty list on parse error
    assert items == []


def test_parse_feed_json_with_trailing_commas():
    """Test parsing JSON with trailing commas (invalid JSON)."""
    json_with_trailing = """
    {
        "hits": [
            {"objectID": "123", "title": "Test", "url": "https://example.com"},
        ]
    }
    """
    
    items = parse_feed(json_with_trailing, "hackernews", "test")
    # Should handle gracefully
    assert items == []


def test_parse_feed_incomplete_json():
    """Test parsing incomplete/truncated JSON."""
    incomplete_json = '{"hits": [{"objectID": "123", "title": '
    
    items = parse_feed(incomplete_json, "hackernews", "test")
    assert items == []


def test_parse_feed_json_with_control_characters():
    """Test parsing JSON with control characters."""
    # JSON with embedded newlines and tabs
    json_str = json.dumps({
        "hits": [{
            "objectID": "123",
            "title": "Test\nwith\nnewlines\tand\ttabs",
            "url": "https://example.com"
        }]
    })
    
    items = parse_feed(json_str, "hackernews", "test")
    assert len(items) >= 1
    if items:
        assert "\n" in items[0].title or "\\n" in items[0].title
