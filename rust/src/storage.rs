use crate::fetcher::FetchResult;
use crate::models::{FeedItem, FetchLog};
use rusqlite::{params, Connection, Result as SqliteResult};
use std::path::Path;
use std::time::SystemTime;

pub struct Storage {
    conn: Connection,
}

impl Storage {
    pub fn new<P: AsRef<Path>>(path: P) -> Result<Self, String> {
        let conn = Connection::open(path)
            .map_err(|e| format!("Failed to open database: {}", e))?;

        let storage = Self { conn };
        storage.init_schema()?;
        Ok(storage)
    }

    fn init_schema(&self) -> Result<(), String> {
        self.conn.execute(
            "CREATE TABLE IF NOT EXISTS feed_items (
                id TEXT PRIMARY KEY,
                title TEXT NOT NULL,
                url TEXT NOT NULL,
                source TEXT NOT NULL,
                timestamp TEXT,
                tags TEXT,
                raw_data TEXT,
                created_at TEXT NOT NULL
            )",
            [],
        ).map_err(|e| format!("Failed to create feed_items table: {}", e))?;

        self.conn.execute(
            "CREATE TABLE IF NOT EXISTS fetch_log (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                source TEXT NOT NULL,
                fetched_at TEXT NOT NULL,
                status TEXT NOT NULL,
                items_count INTEGER DEFAULT 0,
                error_message TEXT,
                duration_ms INTEGER
            )",
            [],
        ).map_err(|e| format!("Failed to create fetch_log table: {}", e))?;

        self.conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_feed_items_source ON feed_items(source)",
            [],
        ).map_err(|e| format!("Failed to create index: {}", e))?;

        self.conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_feed_items_timestamp ON feed_items(timestamp)",
            [],
        ).map_err(|e| format!("Failed to create index: {}", e))?;

        self.conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_fetch_log_source ON fetch_log(source)",
            [],
        ).map_err(|e| format!("Failed to create index: {}", e))?;

        Ok(())
    }

    pub fn store_results(&self, results: &mut [FetchResult]) -> Result<(), String> {
        let tx = self.conn.unchecked_transaction()
            .map_err(|e| format!("Failed to start transaction: {}", e))?;

        let now = Self::current_timestamp();

        for result in results.iter_mut() {
            // Count existing items
            let mut new_count = 0;
            for item in &result.items {
                let exists: bool = tx.query_row(
                    "SELECT 1 FROM feed_items WHERE id = ?1",
                    params![&item.id],
                    |_| Ok(true),
                ).unwrap_or(false);

                if !exists {
                    new_count += 1;
                }
            }

            result.new_items = new_count;

            // Store items
            for item in &result.items {
                let tags_json = serde_json::to_string(&item.tags).unwrap_or_default();
                
                tx.execute(
                    "INSERT OR REPLACE INTO feed_items (id, title, url, source, timestamp, tags, raw_data, created_at)
                     VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)",
                    params![
                        &item.id,
                        &item.title,
                        &item.url,
                        &item.source,
                        &item.timestamp,
                        &tags_json,
                        &item.raw_data,
                        &now,
                    ],
                ).map_err(|e| format!("Failed to insert item: {}", e))?;
            }

            // Log fetch
            let status = if result.error.is_none() { "success" } else { "error" };
            tx.execute(
                "INSERT INTO fetch_log (source, fetched_at, status, items_count, error_message, duration_ms)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
                params![
                    &result.source,
                    &now,
                    status,
                    result.items.len() as i64,
                    &result.error,
                    result.duration_ms as i64,
                ],
            ).map_err(|e| format!("Failed to insert fetch log: {}", e))?;
        }

        tx.commit().map_err(|e| format!("Failed to commit transaction: {}", e))?;

        Ok(())
    }

    pub fn get_items(&self, source: Option<&str>, since: Option<&str>) -> Result<Vec<FeedItem>, String> {
        let mut query = "SELECT id, title, url, source, timestamp, tags, raw_data FROM feed_items WHERE 1=1".to_string();
        let mut params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

        if let Some(src) = source {
            query.push_str(" AND source = ?");
            params.push(Box::new(src.to_string()));
        }

        if let Some(_since_val) = since {
            // TODO: Parse duration and convert to timestamp
            query.push_str(" AND timestamp > ?");
            // For now, skip this filter
        }

        query.push_str(" ORDER BY timestamp DESC");

        let mut stmt = self.conn.prepare(&query)
            .map_err(|e| format!("Failed to prepare query: {}", e))?;

        let param_refs: Vec<&dyn rusqlite::ToSql> = params.iter().map(|p| p.as_ref()).collect();

        let rows = stmt.query_map(param_refs.as_slice(), |row| {
            let tags_json: String = row.get(5)?;
            let tags: Vec<String> = serde_json::from_str(&tags_json).unwrap_or_default();

            Ok(FeedItem {
                id: row.get(0)?,
                title: row.get(1)?,
                url: row.get(2)?,
                source: row.get(3)?,
                timestamp: row.get(4)?,
                tags,
                raw_data: row.get(6)?,
            })
        }).map_err(|e| format!("Failed to query items: {}", e))?;

        let mut items = Vec::new();
        for row in rows {
            items.push(row.map_err(|e| format!("Failed to read row: {}", e))?);
        }

        Ok(items)
    }

    pub fn get_source_stats(&self) -> Result<Vec<SourceStat>, String> {
        let mut stmt = self.conn.prepare(
            "SELECT 
                source,
                COUNT(*) as items,
                SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors,
                MAX(CASE WHEN status = 'success' THEN fetched_at END) as last_success
             FROM fetch_log
             GROUP BY source"
        ).map_err(|e| format!("Failed to prepare stats query: {}", e))?;

        let rows = stmt.query_map([], |row| {
            Ok(SourceStat {
                source: row.get(0)?,
                items: row.get(1)?,
                errors: row.get(2)?,
                last_success: row.get(3)?,
            })
        }).map_err(|e| format!("Failed to query stats: {}", e))?;

        let mut stats = Vec::new();
        for row in rows {
            stats.push(row.map_err(|e| format!("Failed to read stat row: {}", e))?);
        }

        Ok(stats)
    }

    fn current_timestamp() -> String {
        chrono::Utc::now().to_rfc3339()
    }

    /// Store a single item (useful for testing)
    pub fn store_item(&self, item: &FeedItem) -> Result<(), String> {
        let now = Self::current_timestamp();
        let tags_json = serde_json::to_string(&item.tags).unwrap_or_default();
        
        self.conn.execute(
            "INSERT OR REPLACE INTO feed_items (id, title, url, source, timestamp, tags, raw_data, created_at)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)",
            params![
                &item.id,
                &item.title,
                &item.url,
                &item.source,
                &item.timestamp,
                &tags_json,
                &item.raw_data,
                &now,
            ],
        ).map_err(|e| format!("Failed to insert item: {}", e))?;

        Ok(())
    }
}

#[derive(Debug)]
pub struct SourceStat {
    pub source: String,
    pub items: i64,
    pub errors: i64,
    pub last_success: Option<String>,
}
