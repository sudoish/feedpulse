"""Tests for configuration validation"""

import pytest
import sys
import tempfile
from pathlib import Path

from feedpulse.config import validate_url, validate_feed, validate_settings, ConfigError


def test_validate_url():
    """Test URL validation"""
    assert validate_url("https://example.com") == True
    assert validate_url("http://example.com") == True
    assert validate_url("https://example.com/path?query=1") == True
    
    assert validate_url("ftp://example.com") == False
    assert validate_url("not-a-url") == False
    assert validate_url("") == False


def test_validate_feed_success():
    """Test successful feed validation"""
    feed_data = {
        'name': 'Test Feed',
        'url': 'https://example.com/feed',
        'feed_type': 'json',
        'refresh_interval_secs': 600,
        'headers': {'User-Agent': 'test'}
    }
    
    feed = validate_feed(feed_data, 1)
    assert feed.name == 'Test Feed'
    assert feed.url == 'https://example.com/feed'
    assert feed.feed_type == 'json'
    assert feed.refresh_interval_secs == 600
    assert feed.headers == {'User-Agent': 'test'}


def test_validate_feed_missing_name():
    """Test feed validation with missing name"""
    feed_data = {
        'url': 'https://example.com',
        'feed_type': 'json'
    }
    
    with pytest.raises(ConfigError, match="missing field 'name'"):
        validate_feed(feed_data, 1)


def test_validate_feed_empty_name():
    """Test feed validation with empty name"""
    feed_data = {
        'name': '',
        'url': 'https://example.com',
        'feed_type': 'json'
    }
    
    with pytest.raises(ConfigError, match="non-empty string"):
        validate_feed(feed_data, 1)


def test_validate_feed_missing_url():
    """Test feed validation with missing URL"""
    feed_data = {
        'name': 'Test',
        'feed_type': 'json'
    }
    
    with pytest.raises(ConfigError, match="missing field 'url'"):
        validate_feed(feed_data, 1)


def test_validate_feed_invalid_url():
    """Test feed validation with invalid URL"""
    feed_data = {
        'name': 'Test',
        'url': 'not-a-url',
        'feed_type': 'json'
    }
    
    with pytest.raises(ConfigError, match="invalid URL"):
        validate_feed(feed_data, 1)


def test_validate_feed_missing_feed_type():
    """Test feed validation with missing feed_type"""
    feed_data = {
        'name': 'Test',
        'url': 'https://example.com'
    }
    
    with pytest.raises(ConfigError, match="missing field 'feed_type'"):
        validate_feed(feed_data, 1)


def test_validate_feed_invalid_feed_type():
    """Test feed validation with invalid feed_type"""
    feed_data = {
        'name': 'Test',
        'url': 'https://example.com',
        'feed_type': 'xml'
    }
    
    with pytest.raises(ConfigError, match="must be one of: json, rss, atom"):
        validate_feed(feed_data, 1)


def test_validate_feed_invalid_refresh_interval():
    """Test feed validation with invalid refresh interval"""
    feed_data = {
        'name': 'Test',
        'url': 'https://example.com',
        'feed_type': 'json',
        'refresh_interval_secs': -1
    }
    
    with pytest.raises(ConfigError, match="positive integer"):
        validate_feed(feed_data, 1)


def test_validate_settings_defaults():
    """Test settings validation with defaults"""
    settings = validate_settings({})
    assert settings.max_concurrency == 5
    assert settings.default_timeout_secs == 10
    assert settings.retry_max == 3
    assert settings.retry_base_delay_ms == 500
    assert settings.database_path == "feedpulse.db"


def test_validate_settings_custom():
    """Test settings validation with custom values"""
    settings_data = {
        'max_concurrency': 10,
        'default_timeout_secs': 30,
        'retry_max': 5,
        'retry_base_delay_ms': 1000,
        'database_path': 'custom.db'
    }
    
    settings = validate_settings(settings_data)
    assert settings.max_concurrency == 10
    assert settings.default_timeout_secs == 30
    assert settings.retry_max == 5
    assert settings.retry_base_delay_ms == 1000
    assert settings.database_path == 'custom.db'


def test_validate_settings_invalid_max_concurrency():
    """Test settings validation with invalid max_concurrency"""
    with pytest.raises(ConfigError, match="between 1 and 50"):
        validate_settings({'max_concurrency': 0})
    
    with pytest.raises(ConfigError, match="between 1 and 50"):
        validate_settings({'max_concurrency': 51})


def test_validate_settings_invalid_timeout():
    """Test settings validation with invalid timeout"""
    with pytest.raises(ConfigError, match="positive integer"):
        validate_settings({'default_timeout_secs': -1})
