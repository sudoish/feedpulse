"""SQLite storage for feed items and fetch logs.

This module provides database operations for storing feed items and fetch logs.
Handles duplicate detection, transaction management, and query operations.
"""

import sys
import sqlite3
import json
from typing import Optional
from datetime import UTC, datetime
from pathlib import Path

from .models import FeedItem, FetchResult
from .exceptions import DatabaseLockedError, DatabaseCorruptedError, StorageError

# Compatibility alias for existing code
DatabaseError = StorageError


class FeedDatabase:
    """SQLite database for feed items and fetch logs"""
    
    SCHEMA = """
    CREATE TABLE IF NOT EXISTS feed_items (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        url TEXT NOT NULL,
        source TEXT NOT NULL,
        timestamp TEXT,
        tags TEXT,
        raw_data TEXT,
        created_at TEXT NOT NULL
    );
    
    CREATE TABLE IF NOT EXISTS fetch_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        source TEXT NOT NULL,
        fetched_at TEXT NOT NULL,
        status TEXT NOT NULL,
        items_count INTEGER DEFAULT 0,
        error_message TEXT,
        duration_ms INTEGER
    );
    
    CREATE INDEX IF NOT EXISTS idx_feed_items_source ON feed_items(source);
    CREATE INDEX IF NOT EXISTS idx_feed_items_timestamp ON feed_items(timestamp);
    CREATE INDEX IF NOT EXISTS idx_fetch_log_source ON fetch_log(source);
    """
    
    def __init__(self, db_path: str):
        self.db_path = db_path
        self._ensure_schema()
    
    def _get_connection(self, max_retries: int = 3) -> sqlite3.Connection:
        """Get database connection with retry on locked database"""
        for attempt in range(max_retries):
            try:
                conn = sqlite3.connect(self.db_path, timeout=5.0)
                conn.row_factory = sqlite3.Row
                return conn
            except sqlite3.OperationalError as e:
                if "locked" in str(e).lower() and attempt < max_retries - 1:
                    import time
                    time.sleep(0.1)  # 100ms delay
                    continue
                raise DatabaseError(f"Database locked after {max_retries} attempts")
            except Exception as e:
                raise DatabaseError(f"Failed to connect to database: {e}")
        
        raise DatabaseError("Failed to connect to database")
    
    def _ensure_schema(self):
        """Create tables and indexes if they don't exist"""
        try:
            conn = self._get_connection()
            try:
                conn.executescript(self.SCHEMA)
                conn.commit()
            finally:
                conn.close()
        except sqlite3.DatabaseError as e:
            error_str = str(e).lower()
            if any(word in error_str for word in ["malformed", "corrupt", "not a database"]):
                print(f"Error: database corrupted: {e}", file=sys.stderr)
                print(f"Suggestion: delete {self.db_path} and try again", file=sys.stderr)
                sys.exit(1)
            raise DatabaseError(f"Failed to create schema: {e}")
        except Exception as e:
            raise DatabaseError(f"Failed to initialize database: {e}")
    
    def store_results(self, results: list[FetchResult]) -> int:
        """
        Store fetch results (items and logs) in a single transaction.
        Returns total number of new items inserted.
        """
        if not results:
            return 0
        
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                total_new = 0
                
                for result in results:
                    # Log the fetch attempt
                    cursor.execute("""
                        INSERT INTO fetch_log 
                        (source, fetched_at, status, items_count, error_message, duration_ms)
                        VALUES (?, ?, ?, ?, ?, ?)
                    """, (
                        result.source,
                        datetime.now(UTC).isoformat(),
                        result.status,
                        len(result.items),
                        result.error_message,
                        result.duration_ms
                    ))
                    
                    # Store items (upsert - update if exists)
                    new_count = 0
                    for item in result.items:
                        # Check if item exists
                        cursor.execute("SELECT id FROM feed_items WHERE id = ?", (item.id,))
                        exists = cursor.fetchone() is not None
                        
                        if not exists:
                            new_count += 1
                        
                        # Upsert
                        cursor.execute("""
                            INSERT OR REPLACE INTO feed_items
                            (id, title, url, source, timestamp, tags, raw_data, created_at)
                            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                        """, (
                            item.id,
                            item.title,
                            item.url,
                            item.source,
                            item.timestamp,
                            json.dumps(item.tags) if item.tags else None,
                            item.raw_data,
                            item.created_at or datetime.now(UTC).isoformat()
                        ))
                    
                    result.items_new = new_count
                    total_new += new_count
                
                conn.commit()
                return total_new
            
            finally:
                conn.close()
        
        except sqlite3.OperationalError as e:
            if "locked" in str(e).lower():
                raise DatabaseError("Database locked - could not complete transaction")
            elif "disk" in str(e).lower() or "full" in str(e).lower():
                print("Error: disk full - cannot write to database", file=sys.stderr)
                sys.exit(1)
            raise DatabaseError(f"Database error: {e}")
        
        except sqlite3.DatabaseError as e:
            error_str = str(e).lower()
            if any(word in error_str for word in ["malformed", "corrupt", "not a database"]):
                print(f"Error: database corrupted: {e}", file=sys.stderr)
                print(f"Suggestion: delete {self.db_path} and try again", file=sys.stderr)
                sys.exit(1)
            raise DatabaseError(f"Database error: {e}")
        
        except Exception as e:
            raise DatabaseError(f"Failed to store results: {e}")
    
    def get_report_data(self, source_filter: Optional[str] = None):
        """Get summary report data"""
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                
                # Get items count per source
                cursor.execute("""
                    SELECT source, COUNT(*) as count
                    FROM feed_items
                    GROUP BY source
                """)
                items_by_source = {row['source']: row['count'] for row in cursor.fetchall()}
                
                # Get error counts and last success per source
                cursor.execute("""
                    SELECT 
                        source,
                        SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as error_count,
                        MAX(CASE WHEN status = 'success' THEN fetched_at ELSE NULL END) as last_success
                    FROM fetch_log
                    GROUP BY source
                """)
                
                report = []
                for row in cursor.fetchall():
                    source = row['source']
                    items_count = items_by_source.get(source, 0)
                    error_count = row['error_count'] or 0
                    last_success = row['last_success']
                    
                    # Calculate total fetches
                    cursor.execute(
                        "SELECT COUNT(*) as total FROM fetch_log WHERE source = ?",
                        (source,)
                    )
                    total_fetches = cursor.fetchone()['total']
                    
                    error_rate = (error_count / total_fetches * 100) if total_fetches > 0 else 0
                    
                    if source_filter and source != source_filter:
                        continue
                    
                    report.append({
                        'source': source,
                        'items': items_count,
                        'errors': error_count,
                        'error_rate': error_rate,
                        'last_success': last_success or 'never'
                    })
                
                return report
            
            finally:
                conn.close()
        
        except Exception as e:
            raise DatabaseError(f"Failed to get report data: {e}")
    
    def get_sources_status(self):
        """Get status of all sources"""
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                
                # Get last fetch status for each source
                cursor.execute("""
                    SELECT source, status, fetched_at, error_message
                    FROM fetch_log
                    WHERE id IN (
                        SELECT MAX(id) FROM fetch_log GROUP BY source
                    )
                    ORDER BY source
                """)
                
                sources = []
                for row in cursor.fetchall():
                    sources.append({
                        'source': row['source'],
                        'status': row['status'],
                        'last_fetch': row['fetched_at'],
                        'error': row['error_message']
                    })
                
                return sources
            
            finally:
                conn.close()
        
        except Exception as e:
            raise DatabaseError(f"Failed to get sources status: {e}")
    
    # Helper methods for testing and direct API access
    
    def insert_items(self, items: list[FeedItem]) -> int:
        """Insert feed items directly into database.
        
        This is a helper method primarily for testing. For production use,
        prefer store_results() which handles both items and logging.
        
        Args:
            items: List of FeedItem objects to insert
            
        Returns:
            Number of new items inserted (excluding duplicates)
        """
        if not items:
            return 0
        
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                new_count = 0
                
                for item in items:
                    # Check if item exists
                    cursor.execute("SELECT id FROM feed_items WHERE id = ?", (item.id,))
                    exists = cursor.fetchone() is not None
                    
                    if not exists:
                        new_count += 1
                        cursor.execute("""
                            INSERT INTO feed_items
                            (id, title, url, source, timestamp, tags, raw_data, created_at)
                            VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                        """, (
                            item.id,
                            item.title,
                            item.url,
                            item.source,
                            item.timestamp,
                            json.dumps(item.tags) if item.tags else None,
                            item.raw_data,
                            item.created_at or datetime.now(UTC).isoformat()
                        ))
                
                conn.commit()
                return new_count
            
            finally:
                conn.close()
        
        except Exception as e:
            raise DatabaseError(f"Failed to insert items: {e}")
    
    def get_items_by_source(self, source: str, limit: Optional[int] = None) -> list[FeedItem]:
        """Get all items from a specific source.
        
        Args:
            source: Source feed name
            limit: Maximum number of items to return
            
        Returns:
            List of FeedItem objects
        """
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                
                query = """
                    SELECT id, title, url, source, timestamp, tags, raw_data, created_at
                    FROM feed_items
                    WHERE source = ?
                    ORDER BY timestamp DESC
                """
                
                if limit:
                    query += f" LIMIT {limit}"
                
                cursor.execute(query, (source,))
                
                items = []
                for row in cursor.fetchall():
                    tags = json.loads(row['tags']) if row['tags'] else []
                    item = FeedItem(
                        id=row['id'],
                        title=row['title'],
                        url=row['url'],
                        source=row['source'],
                        timestamp=row['timestamp'],
                        tags=tags,
                        raw_data=row['raw_data'],
                        created_at=row['created_at']
                    )
                    items.append(item)
                
                return items
            
            finally:
                conn.close()
        
        except Exception as e:
            raise DatabaseError(f"Failed to get items by source: {e}")
    
    def get_all_items(self, limit: Optional[int] = None) -> list[FeedItem]:
        """Get all items from database.
        
        Args:
            limit: Maximum number of items to return
            
        Returns:
            List of FeedItem objects
        """
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                
                query = """
                    SELECT id, title, url, source, timestamp, tags, raw_data, created_at
                    FROM feed_items
                    ORDER BY timestamp DESC
                """
                
                if limit:
                    query += f" LIMIT {limit}"
                
                cursor.execute(query)
                
                items = []
                for row in cursor.fetchall():
                    tags = json.loads(row['tags']) if row['tags'] else []
                    item = FeedItem(
                        id=row['id'],
                        title=row['title'],
                        url=row['url'],
                        source=row['source'],
                        timestamp=row['timestamp'],
                        tags=tags,
                        raw_data=row['raw_data'],
                        created_at=row['created_at']
                    )
                    items.append(item)
                
                return items
            
            finally:
                conn.close()
        
        except Exception as e:
            raise DatabaseError(f"Failed to get all items: {e}")
    
    def log_fetch(self, result: FetchResult) -> None:
        """Log a fetch result to the database.
        
        Args:
            result: FetchResult object to log
        """
        try:
            conn = self._get_connection()
            try:
                cursor = conn.cursor()
                
                cursor.execute("""
                    INSERT INTO fetch_log 
                    (source, fetched_at, status, items_count, error_message, duration_ms)
                    VALUES (?, ?, ?, ?, ?, ?)
                """, (
                    result.source,
                    datetime.now(UTC).isoformat(),
                    result.status,
                    result.items_new,
                    result.error_message,
                    result.duration_ms
                ))
                
                conn.commit()
            
            finally:
                conn.close()
        
        except Exception as e:
            raise DatabaseError(f"Failed to log fetch: {e}")
