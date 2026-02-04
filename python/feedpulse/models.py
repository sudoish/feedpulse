"""Data models for feedpulse.

This module defines the core data structures used throughout the application:
- FeedConfig: Configuration for individual feeds
- Settings: Global application settings
- Config: Complete configuration (settings + feeds)
- FeedItem: Normalized feed item representation
- FetchResult: Result of a feed fetch operation
"""

from dataclasses import dataclass, field
from typing import Optional
from datetime import UTC, datetime
import hashlib
import json


@dataclass
class FeedConfig:
    """Configuration for a single feed.
    
    Attributes:
        name: Display name for the feed
        url: Feed URL to fetch from
        feed_type: Type of feed (hackernews, reddit, lobsters, github)
        refresh_interval_secs: How often to refresh (default: 300)
        headers: Optional HTTP headers for requests
    """
    name: str
    url: str
    feed_type: str
    refresh_interval_secs: int = 300
    headers: dict[str, str] = field(default_factory=dict)


@dataclass
class Settings:
    """Global application settings.
    
    Attributes:
        max_concurrency: Maximum concurrent feed fetches (default: 5)
        default_timeout_secs: Default HTTP timeout (default: 10)
        retry_max: Maximum retry attempts (default: 3)
        retry_base_delay_ms: Base delay for exponential backoff (default: 500)
        database_path: Path to SQLite database (default: "feedpulse.db")
    """
    max_concurrency: int = 5
    default_timeout_secs: int = 10
    retry_max: int = 3
    retry_base_delay_ms: int = 500
    database_path: str = "feedpulse.db"


@dataclass
class Config:
    """Complete application configuration.
    
    Attributes:
        settings: Global settings
        feeds: List of feed configurations
    """
    settings: Settings
    feeds: list[FeedConfig]


@dataclass
class FeedItem:
    """Normalized feed item representation.
    
    Attributes:
        id: Unique identifier (SHA256 hash)
        title: Item title
        url: Item URL
        source: Source feed name
        timestamp: Publication timestamp (ISO 8601)
        tags: List of tags/categories
        raw_data: Original JSON data (serialized)
        created_at: When item was first seen (ISO 8601)
    """
    id: str
    title: str
    url: str
    source: str
    timestamp: Optional[str] = None
    tags: list[str] = field(default_factory=list)
    raw_data: Optional[str] = None
    created_at: Optional[str] = None

    @staticmethod
    def generate_id(source: str, url: str) -> str:
        """Generate SHA256 ID from source name and URL.
        
        Args:
            source: Feed source name
            url: Item URL
            
        Returns:
            64-character hex string (SHA256 hash)
        """
        combined = f"{source}:{url}"
        return hashlib.sha256(combined.encode('utf-8')).hexdigest()

    @classmethod
    def create(
        cls,
        title: str,
        url: str,
        source: str,
        timestamp: Optional[str] = None,
        tags: Optional[list[str]] = None,
        raw_data: Optional[dict] = None,
    ) -> "FeedItem":
        """Create a FeedItem with auto-generated ID.
        
        Args:
            title: Item title
            url: Item URL
            source: Feed source name
            timestamp: Publication timestamp
            tags: Optional tags list
            raw_data: Optional raw JSON data
            
        Returns:
            New FeedItem instance
        """
        item_id = cls.generate_id(source, url)
        return cls(
            id=item_id,
            title=title,
            url=url,
            source=source,
            timestamp=timestamp,
            tags=tags or [],
            raw_data=json.dumps(raw_data) if raw_data else None,
            created_at=datetime.now(UTC).isoformat(),
        )


@dataclass
class FetchResult:
    """Result of fetching a single feed.
    
    Attributes:
        source: Feed source name
        status: Fetch status ('success' or 'error')
        items: List of fetched items
        error_message: Error message if status is 'error'
        duration_ms: Time taken to fetch (milliseconds)
        items_new: Count of new items (not duplicates)
    """
    source: str
    status: str  # 'success' or 'error'
    items: list[FeedItem] = field(default_factory=list)
    error_message: Optional[str] = None
    duration_ms: int = 0
    items_new: int = 0
