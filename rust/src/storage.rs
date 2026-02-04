use crate::models::{FeedItem, FeedResult, SourceReport};
use anyhow::{Context, Result};
use chrono::Utc;
use rusqlite::{params, Connection};
use std::path::Path;

/// Initialize database and create tables
pub fn init_database<P: AsRef<Path>>(db_path: P) -> Result<Connection> {
    let conn = Connection::open(db_path.as_ref())
        .with_context(|| format!("failed to open database: {}", db_path.as_ref().display()))?;
    
    // Create tables
    conn.execute(
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
    )
    .context("failed to create feed_items table")?;
    
    conn.execute(
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
    )
    .context("failed to create fetch_log table")?;
    
    // Create indexes
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_feed_items_source ON feed_items(source)",
        [],
    )
    .context("failed to create index on feed_items.source")?;
    
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_feed_items_timestamp ON feed_items(timestamp)",
        [],
    )
    .context("failed to create index on feed_items.timestamp")?;
    
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_fetch_log_source ON fetch_log(source)",
        [],
    )
    .context("failed to create index on fetch_log.source")?;
    
    Ok(conn)
}

/// Store fetch results in database (transactional)
pub fn store_results(conn: &Connection, results: &[FeedResult]) -> Result<(usize, usize)> {
    let tx = conn.unchecked_transaction()
        .context("failed to begin transaction")?;
    
    let now = Utc::now().to_rfc3339();
    let mut total_items = 0;
    let mut new_items = 0;
    
    for result in results {
        // Log the fetch
        tx.execute(
            "INSERT INTO fetch_log (source, fetched_at, status, items_count, error_message, duration_ms)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
            params![
                &result.source_name,
                &now,
                result.status.as_str(),
                result.items.len() as i64,
                &result.error_message,
                result.duration_ms as i64,
            ],
        )
        .with_context(|| format!("failed to log fetch for {}", result.source_name))?;
        
        // Store items (upsert)
        for item in &result.items {
            let tags_json = serde_json::to_string(&item.tags)
                .context("failed to serialize tags")?;
            
            // Check if item exists
            let exists: bool = tx
                .query_row(
                    "SELECT 1 FROM feed_items WHERE id = ?1",
                    params![&item.id],
                    |_| Ok(true),
                )
                .unwrap_or(false);
            
            if exists {
                // Update existing item
                tx.execute(
                    "UPDATE feed_items 
                     SET title = ?2, url = ?3, source = ?4, timestamp = ?5, tags = ?6, raw_data = ?7
                     WHERE id = ?1",
                    params![
                        &item.id,
                        &item.title,
                        &item.url,
                        &item.source,
                        &item.timestamp,
                        &tags_json,
                        &item.raw_data,
                    ],
                )
                .with_context(|| format!("failed to update item {}", item.id))?;
            } else {
                // Insert new item
                tx.execute(
                    "INSERT INTO feed_items (id, title, url, source, timestamp, tags, raw_data, created_at)
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
                )
                .with_context(|| format!("failed to insert item {}", item.id))?;
                
                new_items += 1;
            }
            
            total_items += 1;
        }
    }
    
    tx.commit().context("failed to commit transaction")?;
    
    Ok((total_items, new_items))
}

/// Generate a report of all sources
pub fn generate_report(conn: &Connection, source_filter: Option<&str>) -> Result<Vec<SourceReport>> {
    let mut reports = Vec::new();
    
    // Get list of sources
    let sources: Vec<String> = if let Some(filter) = source_filter {
        vec![filter.to_string()]
    } else {
        let mut stmt = conn
            .prepare("SELECT DISTINCT source FROM fetch_log ORDER BY source")
            .context("failed to prepare query for sources")?;
        
        let sources = stmt
            .query_map([], |row| row.get(0))
            .context("failed to query sources")?
            .collect::<Result<Vec<String>, _>>()
            .context("failed to collect sources")?;
        
        sources
    };
    
    for source in sources {
        // Count items
        let total_items: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM feed_items WHERE source = ?1",
                params![&source],
                |row| row.get(0),
            )
            .unwrap_or(0);
        
        // Count errors
        let error_count: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM fetch_log WHERE source = ?1 AND status = 'error'",
                params![&source],
                |row| row.get(0),
            )
            .unwrap_or(0);
        
        // Get last success
        let last_success: Option<String> = conn
            .query_row(
                "SELECT fetched_at FROM fetch_log 
                 WHERE source = ?1 AND status = 'success' 
                 ORDER BY fetched_at DESC LIMIT 1",
                params![&source],
                |row| row.get(0),
            )
            .ok();
        
        reports.push(SourceReport {
            source_name: source,
            total_items,
            error_count,
            last_success,
        });
    }
    
    Ok(reports)
}

/// Get items for report with optional filters
pub fn get_items(
    conn: &Connection,
    source_filter: Option<&str>,
    since: Option<&str>,
    limit: Option<usize>,
) -> Result<Vec<FeedItem>> {
    let mut query = "SELECT id, title, url, source, timestamp, tags, raw_data, created_at 
                     FROM feed_items WHERE 1=1".to_string();
    
    let mut params_vec: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();
    
    if let Some(source) = source_filter {
        query.push_str(" AND source = ?");
        params_vec.push(Box::new(source.to_string()));
    }
    
    if let Some(since_val) = since {
        query.push_str(" AND created_at >= ?");
        params_vec.push(Box::new(since_val.to_string()));
    }
    
    query.push_str(" ORDER BY created_at DESC");
    
    if let Some(limit_val) = limit {
        query.push_str(&format!(" LIMIT {}", limit_val));
    }
    
    let mut stmt = conn.prepare(&query).context("failed to prepare query")?;
    
    let params_refs: Vec<&dyn rusqlite::ToSql> = params_vec.iter().map(|p| p.as_ref()).collect();
    
    let items = stmt
        .query_map(&params_refs[..], |row| {
            let tags_str: String = row.get(5)?;
            let tags: Vec<String> = serde_json::from_str(&tags_str).unwrap_or_default();
            
            Ok(FeedItem {
                id: row.get(0)?,
                title: row.get(1)?,
                url: row.get(2)?,
                source: row.get(3)?,
                timestamp: row.get(4)?,
                tags,
                raw_data: row.get(6)?,
            })
        })
        .context("failed to query items")?
        .collect::<Result<Vec<_>, _>>()
        .context("failed to collect items")?;
    
    Ok(items)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::FetchStatus;

    #[test]
    fn test_init_database() {
        let temp = tempfile::NamedTempFile::new().unwrap();
        let conn = init_database(temp.path()).unwrap();
        
        // Verify tables exist
        let table_count: i64 = conn
            .query_row(
                "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('feed_items', 'fetch_log')",
                [],
                |row| row.get(0),
            )
            .unwrap();
        
        assert_eq!(table_count, 2);
    }

    #[test]
    fn test_store_and_retrieve() {
        let temp = tempfile::NamedTempFile::new().unwrap();
        let conn = init_database(temp.path()).unwrap();
        
        let item = FeedItem {
            id: "test123".to_string(),
            title: "Test Item".to_string(),
            url: "https://example.com".to_string(),
            source: "Test Source".to_string(),
            timestamp: None,
            tags: vec!["tag1".to_string()],
            raw_data: None,
        };
        
        let result = FeedResult {
            source_name: "Test Source".to_string(),
            status: FetchStatus::Success,
            items: vec![item.clone()],
            duration_ms: 100,
            error_message: None,
        };
        
        let (total, new) = store_results(&conn, &[result]).unwrap();
        assert_eq!(total, 1);
        assert_eq!(new, 1);
        
        // Store again - should update
        let result2 = FeedResult {
            source_name: "Test Source".to_string(),
            status: FetchStatus::Success,
            items: vec![item],
            duration_ms: 100,
            error_message: None,
        };
        
        let (total2, new2) = store_results(&conn, &[result2]).unwrap();
        assert_eq!(total2, 1);
        assert_eq!(new2, 0); // No new items, just update
    }

    #[test]
    fn test_generate_report() {
        let temp = tempfile::NamedTempFile::new().unwrap();
        let conn = init_database(temp.path()).unwrap();
        
        let result = FeedResult {
            source_name: "Test Source".to_string(),
            status: FetchStatus::Success,
            items: vec![],
            duration_ms: 100,
            error_message: None,
        };
        
        store_results(&conn, &[result]).unwrap();
        
        let reports = generate_report(&conn, None).unwrap();
        assert_eq!(reports.len(), 1);
        assert_eq!(reports[0].source_name, "Test Source");
    }
}
