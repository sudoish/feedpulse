use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "feedpulse")]
#[command(version = "1.0.0")]
#[command(about = "Concurrent feed aggregator CLI", long_about = None)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Fetch all feeds and store results
    Fetch {
        /// Path to config file
        #[arg(long, default_value = "config.yaml")]
        config: String,
    },
    
    /// Generate summary report
    Report {
        /// Path to config file
        #[arg(long, default_value = "config.yaml")]
        config: String,
        
        /// Output format
        #[arg(long, default_value = "table", value_parser = ["json", "table", "csv"])]
        format: String,
        
        /// Filter by source name
        #[arg(long)]
        source: Option<String>,
        
        /// Filter items newer than (e.g., "24h", "7d")
        #[arg(long)]
        since: Option<String>,
    },
    
    /// List configured sources and their status
    Sources {
        /// Path to config file
        #[arg(long, default_value = "config.yaml")]
        config: String,
    },
}
