"""Shared pytest fixtures for FeedPulse tests.

This module provides reusable fixtures for database setup, mock data,
temporary files, and common test utilities.
"""

import json
import os
import sqlite3
import tempfile
from pathlib import Path
from typing import Any

import pytest

from feedpulse.config import load_config
from feedpulse.models import FeedConfig, FeedItem, Settings
from feedpulse.storage import FeedDatabase


@pytest.fixture
def temp_db_path() -> str:
    """Create a temporary database file path.
    
    Yields:
        Path to temporary database file
    """
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
        db_path = f.name
    
    yield db_path
    
    # Cleanup
    try:
        os.unlink(db_path)
    except FileNotFoundError:
        pass


@pytest.fixture
def db(temp_db_path: str) -> FeedDatabase:
    """Create a FeedDatabase instance with temporary file.
    
    Args:
        temp_db_path: Temporary database path from fixture
        
    Returns:
        FeedDatabase instance
    """
    return FeedDatabase(temp_db_path)


@pytest.fixture
def sample_feed_items() -> list[FeedItem]:
    """Create sample feed items for testing.
    
    Returns:
        List of 3 sample FeedItem objects
    """
    return [
        FeedItem.create(
            title="Test Item 1",
            url="https://example.com/1",
            source="test-feed",
            timestamp="2024-01-01T12:00:00+00:00",
            tags=["test", "python"],
        ),
        FeedItem.create(
            title="Test Item 2",
            url="https://example.com/2",
            source="test-feed",
            timestamp="2024-01-02T12:00:00+00:00",
            tags=["test"],
        ),
        FeedItem.create(
            title="Test Item 3",
            url="https://example.com/3",
            source="other-feed",
            timestamp="2024-01-03T12:00:00+00:00",
        ),
    ]


@pytest.fixture
def sample_hackernews_data() -> dict[str, Any]:
    """Sample HackerNews API response data.
    
    Returns:
        Dict simulating HackerNews API response
    """
    return {
        "hits": [
            {
                "objectID": "12345",
                "title": "Show HN: Cool Project",
                "url": "https://example.com/cool",
                "created_at_i": 1704067200,
                "author": "testuser",
            },
            {
                "objectID": "23456",
                "title": "Ask HN: What is your workflow?",
                "url": "https://news.ycombinator.com/item?id=23456",
                "created_at_i": 1704153600,
                "author": "curious",
            },
        ]
    }


@pytest.fixture
def sample_reddit_data() -> dict[str, Any]:
    """Sample Reddit API response data.
    
    Returns:
        Dict simulating Reddit API response
    """
    return {
        "data": {
            "children": [
                {
                    "data": {
                        "title": "TIL something interesting",
                        "url": "https://example.com/til",
                        "created_utc": 1704067200,
                        "author": "redditor1",
                        "link_flair_text": "Science",
                    }
                },
                {
                    "data": {
                        "title": "Discussion: Python vs Rust",
                        "url": "https://reddit.com/r/programming/abc",
                        "created_utc": 1704153600,
                        "author": "redditor2",
                    }
                },
            ]
        }
    }


@pytest.fixture
def sample_github_data() -> dict[str, Any]:
    """Sample GitHub API response data.
    
    Returns:
        Dict simulating GitHub API response
    """
    return {
        "items": [
            {
                "full_name": "user/awesome-repo",
                "html_url": "https://github.com/user/awesome-repo",
                "topics": ["python", "asyncio", "cli"],
                "updated_at": "2024-01-01T12:00:00Z",
                "description": "An awesome repository",
            },
            {
                "full_name": "org/cool-project",
                "html_url": "https://github.com/org/cool-project",
                "topics": ["rust", "performance"],
                "updated_at": "2024-01-02T12:00:00Z",
            },
        ]
    }


@pytest.fixture
def sample_lobsters_data() -> list[dict[str, Any]]:
    """Sample Lobsters API response data.
    
    Returns:
        List simulating Lobsters API response
    """
    return [
        {
            "title": "Interesting Article",
            "url": "https://example.com/article",
            "short_id": "abc123",
            "created_at": "2024-01-01T12:00:00.000-06:00",
            "tags": ["programming", "tutorial"],
        },
        {
            "title": "Discussion about compilers",
            "short_id": "def456",
            "created_at": "2024-01-02T12:00:00.000-06:00",
            "tags": ["compilers"],
        },
    ]


@pytest.fixture
def temp_config_file() -> str:
    """Create a temporary config file.
    
    Yields:
        Path to temporary config file
    """
    config_data = """
feeds:
  - name: test-hn
    url: https://hn.algolia.com/api/v1/search_by_date?tags=story
    feed_type: hackernews

  - name: test-reddit
    url: https://www.reddit.com/r/programming.json
    feed_type: reddit

settings:
  max_concurrency: 5
  default_timeout_secs: 10
  database_path: test.db
"""
    
    with tempfile.NamedTemporaryFile(
        mode='w', suffix='.yaml', delete=False
    ) as f:
        f.write(config_data)
        config_path = f.name
    
    yield config_path
    
    # Cleanup
    try:
        os.unlink(config_path)
    except FileNotFoundError:
        pass


@pytest.fixture
def corrupted_db_path() -> str:
    """Create a corrupted database file for testing error handling.
    
    Yields:
        Path to corrupted database file
    """
    with tempfile.NamedTemporaryFile(
        mode='wb', suffix=".db", delete=False
    ) as f:
        # Write random bytes to simulate corruption
        f.write(b"This is not a valid SQLite database file!")
        db_path = f.name
    
    yield db_path
    
    # Cleanup
    try:
        os.unlink(db_path)
    except FileNotFoundError:
        pass


@pytest.fixture
def mock_feed_config() -> FeedConfig:
    """Create a mock feed configuration.
    
    Returns:
        Sample FeedConfig object
    """
    return FeedConfig(
        name="test-feed",
        url="https://example.com/api/feed",
        feed_type="hackernews",
        refresh_interval_secs=300,
        headers={"User-Agent": "FeedPulse/Test"},
    )


@pytest.fixture
def mock_settings() -> Settings:
    """Create mock application settings.
    
    Returns:
        Sample Settings object
    """
    return Settings(
        max_concurrency=5,
        default_timeout_secs=10,
        retry_max=3,
        retry_base_delay_ms=500,
        database_path="test.db",
    )
