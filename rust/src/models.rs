use serde::{Deserialize, Serialize};
use sha2::{Sha256, Digest};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeedItem {
    pub id: String,
    pub title: String,
    pub url: String,
    pub source: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub timestamp: Option<String>,
    #[serde(default)]
    pub tags: Vec<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub raw_data: Option<String>,
}

impl FeedItem {
    pub fn new(title: String, url: String, source: String) -> Self {
        let id = Self::generate_id(&source, &url);
        Self {
            id,
            title,
            url,
            source,
            timestamp: None,
            tags: Vec::new(),
            raw_data: None,
        }
    }

    pub fn generate_id(source: &str, url: &str) -> String {
        let mut hasher = Sha256::new();
        hasher.update(source.as_bytes());
        hasher.update(url.as_bytes());
        format!("{:x}", hasher.finalize())
    }

    pub fn with_timestamp(mut self, timestamp: Option<String>) -> Self {
        self.timestamp = timestamp;
        self
    }

    pub fn with_tags(mut self, tags: Vec<String>) -> Self {
        self.tags = tags;
        self
    }

    pub fn with_raw_data(mut self, raw_data: String) -> Self {
        self.raw_data = Some(raw_data);
        self
    }
}

#[derive(Debug, Clone)]
pub struct FetchLog {
    pub source: String,
    pub fetched_at: String,
    pub status: String,
    pub items_count: usize,
    pub error_message: Option<String>,
    pub duration_ms: u64,
}
