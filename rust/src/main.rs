mod cli;
mod config;
mod fetcher;
mod models;
mod parser;
mod storage;

use anyhow::Result;
use clap::Parser;
use cli::{Cli, Commands};
use comfy_table::{Cell, Color, Table};
use models::FetchStatus;
use std::process;

#[tokio::main]
async fn main() {
    if let Err(e) = run().await {
        eprintln!("Error: {}", e);
        process::exit(1);
    }
}

async fn run() -> Result<()> {
    let cli = Cli::parse();
    
    match cli.command {
        Commands::Fetch { config: config_path } => {
            handle_fetch(&config_path).await?;
        }
        Commands::Report {
            config: config_path,
            format,
            source,
            since,
        } => {
            handle_report(&config_path, &format, source.as_deref(), since.as_deref())?;
        }
        Commands::Sources { config: config_path } => {
            handle_sources(&config_path)?;
        }
    }
    
    Ok(())
}

async fn handle_fetch(config_path: &str) -> Result<()> {
    // Load configuration
    let config = config::load_config(config_path)?;
    
    eprintln!(
        "Fetching {} feeds (max concurrency: {})...",
        config.feeds.len(),
        config.settings.max_concurrency
    );
    
    // Fetch all feeds
    let results = fetcher::fetch_all_feeds(&config).await;
    
    // Print individual results
    let mut success_count = 0;
    
    for result in &results {
        match result.status {
            FetchStatus::Success => {
                eprintln!(
                    "  ✓ {:<25} — {} items in {}ms",
                    truncate(&result.source_name, 25),
                    result.items.len(),
                    result.duration_ms
                );
                success_count += 1;
            }
            FetchStatus::Error => {
                let error_msg = result
                    .error_message
                    .as_deref()
                    .unwrap_or("unknown error");
                eprintln!(
                    "  ✗ {:<25} — error: {}",
                    truncate(&result.source_name, 25),
                    error_msg
                );
            }
        }
    }
    
    // Store results in database
    let conn = storage::init_database(&config.settings.database_path)?;
    let (stored_total, new_items) = storage::store_results(&conn, &results)?;
    
    // Print summary
    eprintln!();
    eprintln!(
        "Done: {}/{} succeeded, {} items ({} new), {} errors",
        success_count,
        results.len(),
        stored_total,
        new_items,
        results.len() - success_count
    );
    
    Ok(())
}

fn handle_report(
    config_path: &str,
    format: &str,
    source_filter: Option<&str>,
    _since: Option<&str>,
) -> Result<()> {
    // Load configuration to get database path
    let config = config::load_config(config_path)?;
    
    // Open database
    let conn = storage::init_database(&config.settings.database_path)?;
    
    // Generate report
    let reports = storage::generate_report(&conn, source_filter)?;
    
    if reports.is_empty() {
        println!("No data available.");
        return Ok(());
    }
    
    match format {
        "json" => {
            let json = serde_json::to_string_pretty(&reports)
                .map_err(|e| anyhow::anyhow!("failed to serialize report: {}", e))?;
            println!("{}", json);
        }
        "csv" => {
            println!("Source,Items,Errors,Error Rate,Last Success");
            for report in &reports {
                println!(
                    "{},{},{},{:.1}%,{}",
                    report.source_name,
                    report.total_items,
                    report.error_count,
                    report.error_rate(),
                    report.last_success.as_deref().unwrap_or("never")
                );
            }
        }
        "table" | _ => {
            let mut table = Table::new();
            table.set_header(vec![
                Cell::new("Source").fg(Color::Cyan),
                Cell::new("Items").fg(Color::Cyan),
                Cell::new("Errors").fg(Color::Cyan),
                Cell::new("Error Rate").fg(Color::Cyan),
                Cell::new("Last Success").fg(Color::Cyan),
            ]);
            
            for report in &reports {
                table.add_row(vec![
                    Cell::new(&report.source_name),
                    Cell::new(report.total_items.to_string()),
                    Cell::new(report.error_count.to_string()),
                    Cell::new(format!("{:.1}%", report.error_rate())),
                    Cell::new(
                        report
                            .last_success
                            .as_ref()
                            .and_then(|s| s.split('T').next())
                            .unwrap_or("never"),
                    ),
                ]);
            }
            
            println!("{}", table);
            
            let total_items: i64 = reports.iter().map(|r| r.total_items).sum();
            println!(
                "\nTotal: {} items across {} sources",
                total_items,
                reports.len()
            );
        }
    }
    
    Ok(())
}

fn handle_sources(config_path: &str) -> Result<()> {
    // Load configuration
    let config = config::load_config(config_path)?;
    
    // Open database
    let conn = storage::init_database(&config.settings.database_path)?;
    
    // Generate report for all configured sources
    let reports = storage::generate_report(&conn, None)?;
    
    let mut table = Table::new();
    table.set_header(vec![
        Cell::new("Source").fg(Color::Cyan),
        Cell::new("URL").fg(Color::Cyan),
        Cell::new("Type").fg(Color::Cyan),
        Cell::new("Items").fg(Color::Cyan),
        Cell::new("Status").fg(Color::Cyan),
    ]);
    
    for feed in &config.feeds {
        let report = reports.iter().find(|r| r.source_name == feed.name);
        
        let (items_count, status) = if let Some(r) = report {
            let status = if r.last_success.is_some() {
                "✓ Active"
            } else {
                "✗ Errors"
            };
            (r.total_items.to_string(), status)
        } else {
            ("0".to_string(), "— Not fetched")
        };
        
        table.add_row(vec![
            Cell::new(&feed.name),
            Cell::new(truncate(&feed.url, 50)),
            Cell::new(format!("{:?}", feed.feed_type)),
            Cell::new(items_count),
            Cell::new(status),
        ]);
    }
    
    println!("{}", table);
    
    Ok(())
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len.saturating_sub(3)])
    }
}
