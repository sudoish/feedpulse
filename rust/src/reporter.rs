use crate::config::Config;
use crate::storage::Storage;
use comfy_table::{Table, Cell, Color, Attribute};

pub struct Reporter {
    storage: Storage,
}

impl Reporter {
    pub fn new(storage: Storage) -> Self {
        Self { storage }
    }

    pub fn generate_report(
        &self,
        format: &str,
        source: Option<&str>,
        since: Option<&str>,
    ) -> Result<(), String> {
        let stats = self.storage.get_source_stats()?;

        match format {
            "table" => self.print_table_report(&stats, source),
            "json" => self.print_json_report(&stats, source),
            "csv" => self.print_csv_report(&stats, source),
            _ => return Err(format!("Unknown format: {}", format)),
        }

        Ok(())
    }

    fn print_table_report(&self, stats: &[crate::storage::SourceStat], filter_source: Option<&str>) {
        let mut table = Table::new();
        table.set_header(vec![
            Cell::new("Source").add_attribute(Attribute::Bold),
            Cell::new("Items").add_attribute(Attribute::Bold),
            Cell::new("Errors").add_attribute(Attribute::Bold),
            Cell::new("Error Rate").add_attribute(Attribute::Bold),
            Cell::new("Last Success").add_attribute(Attribute::Bold),
        ]);

        let filtered_stats: Vec<_> = if let Some(src) = filter_source {
            stats.iter().filter(|s| s.source == src).collect()
        } else {
            stats.iter().collect()
        };

        let mut total_items = 0;
        let mut total_errors = 0;

        for stat in &filtered_stats {
            let total = stat.items + stat.errors;
            let error_rate = if total > 0 {
                (stat.errors as f64 / total as f64) * 100.0
            } else {
                0.0
            };

            let last_success = stat.last_success.as_deref().unwrap_or("never");

            table.add_row(vec![
                Cell::new(&stat.source),
                Cell::new(stat.items.to_string()),
                Cell::new(stat.errors.to_string()),
                Cell::new(format!("{:.1}%", error_rate)),
                Cell::new(last_success),
            ]);

            total_items += stat.items;
            total_errors += stat.errors;
        }

        println!("{}", table);
        println!("\nTotal: {} items across {} sources", total_items, filtered_stats.len());
    }

    fn print_json_report(&self, stats: &[crate::storage::SourceStat], filter_source: Option<&str>) {
        let filtered_stats: Vec<_> = if let Some(src) = filter_source {
            stats.iter().filter(|s| s.source == src).collect()
        } else {
            stats.iter().collect()
        };

        let json = serde_json::json!({
            "sources": filtered_stats.iter().map(|stat| {
                let total = stat.items + stat.errors;
                let error_rate = if total > 0 {
                    (stat.errors as f64 / total as f64) * 100.0
                } else {
                    0.0
                };

                serde_json::json!({
                    "source": stat.source,
                    "items": stat.items,
                    "errors": stat.errors,
                    "error_rate": format!("{:.1}%", error_rate),
                    "last_success": stat.last_success,
                })
            }).collect::<Vec<_>>(),
        });

        println!("{}", serde_json::to_string_pretty(&json).unwrap());
    }

    fn print_csv_report(&self, stats: &[crate::storage::SourceStat], filter_source: Option<&str>) {
        let filtered_stats: Vec<_> = if let Some(src) = filter_source {
            stats.iter().filter(|s| s.source == src).collect()
        } else {
            stats.iter().collect()
        };

        println!("Source,Items,Errors,Error Rate,Last Success");

        for stat in filtered_stats {
            let total = stat.items + stat.errors;
            let error_rate = if total > 0 {
                (stat.errors as f64 / total as f64) * 100.0
            } else {
                0.0
            };

            let last_success = stat.last_success.as_deref().unwrap_or("never");

            println!(
                "{},{},{},{:.1}%,{}",
                stat.source, stat.items, stat.errors, error_rate, last_success
            );
        }
    }

    pub fn list_sources(&self, config: &Config) -> Result<(), String> {
        let stats = self.storage.get_source_stats()?;
        let stats_map: std::collections::HashMap<_, _> = stats.iter()
            .map(|s| (s.source.as_str(), s))
            .collect();

        let mut table = Table::new();
        table.set_header(vec![
            Cell::new("Source").add_attribute(Attribute::Bold),
            Cell::new("URL").add_attribute(Attribute::Bold),
            Cell::new("Type").add_attribute(Attribute::Bold),
            Cell::new("Status").add_attribute(Attribute::Bold),
        ]);

        for feed in &config.feeds {
            let status = if let Some(stat) = stats_map.get(feed.name.as_str()) {
                if stat.last_success.is_some() {
                    "✓ Active"
                } else {
                    "✗ Failing"
                }
            } else {
                "○ Never fetched"
            };

            table.add_row(vec![
                Cell::new(&feed.name),
                Cell::new(&feed.url),
                Cell::new(&feed.feed_type),
                Cell::new(status),
            ]);
        }

        println!("{}", table);

        Ok(())
    }
}
