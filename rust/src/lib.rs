// Library crate for feedpulse - exposes public API for testing and reuse

pub mod config;
pub mod fetcher;
pub mod models;
pub mod parser;
pub mod reporter;
pub mod storage;

// Re-export commonly used types
pub use config::{Config, Feed, Settings};
pub use models::FeedItem;
pub use parser::Parser;
pub use storage::Storage;
