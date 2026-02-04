use clap::{Parser, Subcommand};
use std::path::PathBuf;
use std::process;

mod config;
mod fetcher;
mod parser;
mod storage;
mod reporter;
mod models;

use config::Config;
use fetcher::Fetcher;
use storage::Storage;
use reporter::Reporter;

#[derive(Parser)]
#[command(name = "feedpulse")]
#[command(version = "1.0.0")]
#[command(about = "Concurrent Feed Aggregator CLI", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Fetch all feeds and store results
    Fetch {
        #[arg(long, default_value = "config.yaml")]
        config: PathBuf,
    },
    /// Generate summary report
    Report {
        #[arg(long, default_value = "config.yaml")]
        config: PathBuf,
        #[arg(long, default_value = "table")]
        format: String,
        #[arg(long)]
        source: Option<String>,
        #[arg(long)]
        since: Option<String>,
    },
    /// List configured sources and their status
    Sources {
        #[arg(long, default_value = "config.yaml")]
        config: PathBuf,
    },
}

#[tokio::main]
async fn main() {
    let cli = Cli::parse();

    match cli.command {
        Commands::Fetch { config } => {
            if let Err(e) = run_fetch(config).await {
                eprintln!("Error: {}", e);
                process::exit(1);
            }
        }
        Commands::Report { config, format, source, since } => {
            if let Err(e) = run_report(config, format, source, since).await {
                eprintln!("Error: {}", e);
                process::exit(1);
            }
        }
        Commands::Sources { config } => {
            if let Err(e) = run_sources(config).await {
                eprintln!("Error: {}", e);
                process::exit(1);
            }
        }
    }
}

async fn run_fetch(config_path: PathBuf) -> Result<(), String> {
    // Load config
    let config = Config::load(&config_path)?;
    
    config.validate()?;

    // Initialize storage
    let storage = Storage::new(&config.settings.database_path)
        .map_err(|e| format!("Failed to initialize database: {}", e))?;

    // Fetch feeds
    let fetcher = Fetcher::new(config.clone());
    let mut results = fetcher.fetch_all().await;

    // Store results (updates new_items count)
    storage.store_results(&mut results)
        .map_err(|e| format!("Failed to store results: {}", e))?;

    // Print individual results
    for result in &results {
        fetcher::print_result(result);
    }

    // Print summary
    print_fetch_summary(&results);

    Ok(())
}

async fn run_report(
    config_path: PathBuf,
    format: String,
    source: Option<String>,
    since: Option<String>,
) -> Result<(), String> {
    let config = Config::load(&config_path)?;

    let storage = Storage::new(&config.settings.database_path)
        .map_err(|e| format!("Failed to initialize database: {}", e))?;

    let reporter = Reporter::new(storage);
    reporter.generate_report(&format, source.as_deref(), since.as_deref())
        .map_err(|e| format!("Failed to generate report: {}", e))?;

    Ok(())
}

async fn run_sources(config_path: PathBuf) -> Result<(), String> {
    let config = Config::load(&config_path)?;

    config.validate()?;

    let storage = Storage::new(&config.settings.database_path)
        .map_err(|e| format!("Failed to initialize database: {}", e))?;

    let reporter = Reporter::new(storage);
    reporter.list_sources(&config)
        .map_err(|e| format!("Failed to list sources: {}", e))?;

    Ok(())
}

fn print_fetch_summary(results: &[fetcher::FetchResult]) {
    let total = results.len();
    let succeeded = results.iter().filter(|r| r.error.is_none()).count();
    let total_items: usize = results.iter().map(|r| r.items.len()).sum();
    let new_items: usize = results.iter().map(|r| r.new_items).sum();
    let errors = total - succeeded;

    println!("\nDone: {}/{} succeeded, {} items ({} new), {} error{}",
        succeeded, total, total_items, new_items, errors, if errors != 1 { "s" } else { "" });
}
