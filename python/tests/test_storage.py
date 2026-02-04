"""Comprehensive tests for storage module.

Tests database operations including:
- Schema creation and validation
- Item insertion and retrieval
- Duplicate detection
- Transaction handling
- Error scenarios
- Concurrent operations
- Query performance
"""

import os
import sqlite3
import tempfile
import time
from concurrent.futures import ThreadPoolExecutor

import pytest

from feedpulse.models import FeedItem, FetchResult
from feedpulse.storage import FeedDatabase, DatabaseError


def test_database_initialization(temp_db_path):
    """Test database file is created with correct schema."""
    db = FeedDatabase(temp_db_path)
    
    assert os.path.exists(temp_db_path)
    
    # Verify tables exist
    conn = sqlite3.connect(temp_db_path)
    cursor = conn.cursor()
    
    cursor.execute(
        "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
    )
    tables = [row[0] for row in cursor.fetchall()]
    
    assert "feed_items" in tables
    assert "fetch_log" in tables
    
    conn.close()


def test_insert_items(db, sample_feed_items):
    """Test inserting feed items into database."""
    result = db.insert_items(sample_feed_items)
    
    assert result == 3
    
    # Verify items are in database
    items = db.get_items_by_source("test-feed")
    assert len(items) == 2
    
    items = db.get_items_by_source("other-feed")
    assert len(items) == 1


def test_duplicate_detection(db, sample_feed_items):
    """Test that duplicate items are not inserted."""
    # Insert items first time
    count1 = db.insert_items(sample_feed_items)
    assert count1 == 3
    
    # Try inserting same items again
    count2 = db.insert_items(sample_feed_items)
    assert count2 == 0
    
    # Verify total count is still 3
    all_items = db.get_all_items()
    assert len(all_items) == 3


def test_duplicate_detection_by_id(db):
    """Test duplicate detection using item ID."""
    item1 = FeedItem.create(
        title="Test",
        url="https://example.com/test",
        source="test",
    )
    
    # Create item with same source and URL (should have same ID)
    item2 = FeedItem.create(
        title="Test Modified",  # Different title
        url="https://example.com/test",
        source="test",
    )
    
    assert item1.id == item2.id
    
    # Insert both - should only insert one
    db.insert_items([item1])
    count = db.insert_items([item2])
    
    assert count == 0


def test_get_items_by_source(db, sample_feed_items):
    """Test retrieving items filtered by source."""
    db.insert_items(sample_feed_items)
    
    test_feed_items = db.get_items_by_source("test-feed")
    assert len(test_feed_items) == 2
    
    for item in test_feed_items:
        assert item.source == "test-feed"
    
    other_feed_items = db.get_items_by_source("other-feed")
    assert len(other_feed_items) == 1
    assert other_feed_items[0].source == "other-feed"


def test_get_items_by_source_with_limit(db, sample_feed_items):
    """Test retrieving items with limit parameter."""
    db.insert_items(sample_feed_items)
    
    items = db.get_items_by_source("test-feed", limit=1)
    assert len(items) == 1


def test_get_all_items(db, sample_feed_items):
    """Test retrieving all items from database."""
    db.insert_items(sample_feed_items)
    
    all_items = db.get_all_items()
    assert len(all_items) == 3


def test_get_all_items_with_limit(db, sample_feed_items):
    """Test retrieving all items with limit."""
    db.insert_items(sample_feed_items)
    
    items = db.get_all_items(limit=2)
    assert len(items) == 2


def test_log_fetch(db):
    """Test logging fetch results."""
    fetch_result = FetchResult(
        source="test-feed",
        status="success",
        items=[],
        items_new=5,
        duration_ms=1234,
    )
    
    db.log_fetch(fetch_result)
    
    # Verify log entry exists
    conn = sqlite3.connect(db.db_path)
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM fetch_log WHERE source = ?", ("test-feed",))
    row = cursor.fetchone()
    
    assert row is not None
    assert row[1] == "test-feed"  # source
    assert row[3] == "success"  # status
    assert row[4] == 5  # items_count
    conn.close()


def test_log_fetch_with_error(db):
    """Test logging fetch results with error."""
    fetch_result = FetchResult(
        source="test-feed",
        status="error",
        error_message="Network timeout",
        duration_ms=5000,
    )
    
    db.log_fetch(fetch_result)
    
    # Verify error is logged
    conn = sqlite3.connect(db.db_path)
    cursor = conn.cursor()
    cursor.execute("SELECT error_message FROM fetch_log WHERE source = ?", ("test-feed",))
    row = cursor.fetchone()
    
    assert row[0] == "Network timeout"
    conn.close()


def test_item_with_tags(db):
    """Test storing and retrieving items with tags."""
    item = FeedItem.create(
        title="Tagged Item",
        url="https://example.com/tagged",
        source="test",
        tags=["python", "asyncio", "cli"],
    )
    
    db.insert_items([item])
    
    items = db.get_items_by_source("test")
    assert len(items) == 1
    assert items[0].tags == ["python", "asyncio", "cli"]


def test_item_with_raw_data(db):
    """Test storing and retrieving items with raw JSON data."""
    raw_data = {
        "custom_field": "value",
        "nested": {"data": [1, 2, 3]},
    }
    
    item = FeedItem.create(
        title="Item with raw data",
        url="https://example.com/raw",
        source="test",
        raw_data=raw_data,
    )
    
    db.insert_items([item])
    
    items = db.get_items_by_source("test")
    assert len(items) == 1
    assert items[0].raw_data is not None


def test_empty_database_queries(db):
    """Test queries on empty database."""
    items = db.get_all_items()
    assert items == []
    
    items = db.get_items_by_source("nonexistent")
    assert items == []


def test_concurrent_inserts(db):
    """Test concurrent insert operations."""
    def insert_items(source_num):
        items = [
            FeedItem.create(
                title=f"Item {i} from {source_num}",
                url=f"https://example.com/{source_num}/{i}",
                source=f"source-{source_num}",
            )
            for i in range(10)
        ]
        return db.insert_items(items)
    
    # Insert from multiple threads
    with ThreadPoolExecutor(max_workers=5) as executor:
        results = list(executor.map(insert_items, range(5)))
    
    # All inserts should succeed
    assert all(r == 10 for r in results)
    
    # Verify total count
    all_items = db.get_all_items()
    assert len(all_items) == 50


def test_large_batch_insert(db):
    """Test inserting large batch of items."""
    items = [
        FeedItem.create(
            title=f"Item {i}",
            url=f"https://example.com/item/{i}",
            source="bulk-test",
        )
        for i in range(1000)
    ]
    
    start = time.time()
    count = db.insert_items(items)
    duration = time.time() - start
    
    assert count == 1000
    assert duration < 5.0  # Should complete in under 5 seconds
    
    # Verify count
    retrieved = db.get_items_by_source("bulk-test")
    assert len(retrieved) == 1000


def test_item_ordering(db):
    """Test that items are ordered by timestamp descending."""
    items = [
        FeedItem.create(
            title=f"Item {i}",
            url=f"https://example.com/{i}",
            source="test",
            timestamp=f"2024-01-{i:02d}T12:00:00+00:00",
        )
        for i in range(1, 6)
    ]
    
    db.insert_items(items)
    
    retrieved = db.get_items_by_source("test")
    
    # Should be in descending order by timestamp
    for i in range(len(retrieved) - 1):
        assert retrieved[i].timestamp >= retrieved[i + 1].timestamp


def test_database_path_validation(temp_db_path):
    """Test database handles various path scenarios."""
    # Valid path
    db1 = FeedDatabase(temp_db_path)
    assert db1.db_path == temp_db_path
    
    # Another valid path
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
        another_path = f.name
    
    try:
        db2 = FeedDatabase(another_path)
        assert os.path.exists(another_path)
    finally:
        # Cleanup
        if os.path.exists(another_path):
            os.unlink(another_path)


def test_schema_idempotency(temp_db_path):
    """Test that schema creation is idempotent."""
    # Create database twice
    db1 = FeedDatabase(temp_db_path)
    db2 = FeedDatabase(temp_db_path)
    
    # Should not raise errors
    item = FeedItem.create(
        title="Test",
        url="https://example.com",
        source="test",
    )
    
    db1.insert_items([item])
    items = db2.get_all_items()
    
    assert len(items) == 1


def test_transaction_rollback(db):
    """Test that failed transactions are rolled back."""
    item = FeedItem.create(
        title="Test",
        url="https://example.com",
        source="test",
    )
    
    db.insert_items([item])
    
    # Try to insert item with invalid data by accessing DB directly
    # This should fail and not affect existing data
    try:
        conn = sqlite3.connect(db.db_path)
        cursor = conn.cursor()
        cursor.execute(
            "INSERT INTO feed_items (id, title, url, source, created_at) "
            "VALUES (?, ?, ?, ?, ?)",
            ("invalid", None, "url", "test", "2024-01-01"),  # title is NOT NULL
        )
        conn.commit()
        conn.close()
    except sqlite3.IntegrityError:
        pass
    
    # Verify original item still exists
    items = db.get_all_items()
    assert len(items) == 1


def test_null_timestamp_handling(db):
    """Test items with null timestamps are handled correctly."""
    item = FeedItem.create(
        title="No timestamp",
        url="https://example.com/no-ts",
        source="test",
        timestamp=None,
    )
    
    db.insert_items([item])
    
    items = db.get_items_by_source("test")
    assert len(items) == 1
    assert items[0].timestamp is None


def test_special_characters_in_data(db):
    """Test handling of special characters in item data."""
    item = FeedItem.create(
        title="Test with 'quotes' and \"double quotes\" and <tags>",
        url="https://example.com/special?param=value&other=123",
        source="test-source",
        tags=["tag-with-dash", "tag_with_underscore", "tag.with.dots"],
    )
    
    db.insert_items([item])
    
    items = db.get_items_by_source("test-source")
    assert len(items) == 1
    assert "quotes" in items[0].title
    assert items[0].url == "https://example.com/special?param=value&other=123"


def test_unicode_data(db):
    """Test handling of Unicode characters in item data."""
    item = FeedItem.create(
        title="测试 • Test • テスト • 🚀",
        url="https://example.com/unicode",
        source="test",
        tags=["日本語", "中文", "emoji🎉"],
    )
    
    db.insert_items([item])
    
    items = db.get_items_by_source("test")
    assert len(items) == 1
    assert "🚀" in items[0].title
    assert "emoji🎉" in items[0].tags


def test_very_long_content(db):
    """Test handling of very long content."""
    long_title = "A" * 10000
    long_url = f"https://example.com/{'x' * 5000}"
    
    item = FeedItem.create(
        title=long_title,
        url=long_url,
        source="test",
    )
    
    db.insert_items([item])
    
    items = db.get_items_by_source("test")
    assert len(items) == 1
    assert len(items[0].title) == 10000
    assert len(items[0].url) > 5000
