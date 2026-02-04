"""Data models for feedpulse"""

from dataclasses import dataclass, field
from typing import Optional, List, Dict
from datetime import datetime
import hashlib
import json


@dataclass
class FeedConfig:
    """Configuration for a single feed"""
    name: str
    url: str
    feed_type: str
    refresh_interval_secs: int = 300
    headers: Dict[str, str] = field(default_factory=dict)


@dataclass
class Settings:
    """Global settings"""
    max_concurrency: int = 5
    default_timeout_secs: int = 10
    retry_max: int = 3
    retry_base_delay_ms: int = 500
    database_path: str = "feedpulse.db"


@dataclass
class Config:
    """Complete configuration"""
    settings: Settings
    feeds: List[FeedConfig]


@dataclass
class FeedItem:
    """Normalized feed item"""
    id: str
    title: str
    url: str
    source: str
    timestamp: Optional[str] = None
    tags: List[str] = field(default_factory=list)
    raw_data: Optional[str] = None
    created_at: Optional[str] = None

    @staticmethod
    def generate_id(source: str, url: str) -> str:
        """Generate SHA256 ID from source name and URL"""
        combined = f"{source}:{url}"
        return hashlib.sha256(combined.encode('utf-8')).hexdigest()

    @classmethod
    def create(cls, title: str, url: str, source: str, 
               timestamp: Optional[str] = None,
               tags: Optional[List[str]] = None,
               raw_data: Optional[dict] = None) -> 'FeedItem':
        """Create a FeedItem with auto-generated ID"""
        item_id = cls.generate_id(source, url)
        return cls(
            id=item_id,
            title=title,
            url=url,
            source=source,
            timestamp=timestamp,
            tags=tags or [],
            raw_data=json.dumps(raw_data) if raw_data else None,
            created_at=datetime.utcnow().isoformat()
        )


@dataclass
class FetchResult:
    """Result of fetching a single feed"""
    source: str
    status: str  # 'success' or 'error'
    items: List[FeedItem] = field(default_factory=list)
    error_message: Optional[str] = None
    duration_ms: int = 0
    items_new: int = 0
