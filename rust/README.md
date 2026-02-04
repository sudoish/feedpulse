# feedpulse - Rust Implementation

A concurrent feed aggregator CLI tool built in Rust.

## Features

- ✅ Concurrent feed fetching with configurable limits
- ✅ Retry logic with exponential backoff
- ✅ SQLite storage with deduplication
- ✅ Multiple output formats (table, JSON, CSV)
- ✅ **Comprehensive error handling** - all 16 scenarios from SPEC.md
- ✅ Support for JSON feeds (HackerNews, GitHub, Reddit, Lobsters)
- ✅ **70 comprehensive tests** - 1,252 LOC of test coverage
- ✅ **Production-ready** - full spec compliance

## Build Instructions

### Prerequisites

- Rust 1.70+ and Cargo
- SQLite3 (bundled via rusqlite)

### Build

```bash
cargo build --release
```

The binary will be at `target/release/feedpulse`

### Development Build

```bash
cargo build
```

The binary will be at `target/debug/feedpulse`

## Testing

### Run All Tests

```bash
cargo test
```

### Run Specific Test Suite

```bash
cargo test --test test_config          # Config validation tests (16 tests)
cargo test --test test_parser          # Parser tests (28 tests)
cargo test --test test_error_scenarios # Error scenario tests (26 tests)
```

### Run Specific Test

```bash
cargo test test_scenario_10_malformed_json_response
```

### Test Coverage

- **70 tests total** across 3 test files
- **1,252 lines** of test code
- **All 16 error scenarios** from SPEC.md covered
- **100% pass rate** ✅

See [TEST_SUMMARY.md](TEST_SUMMARY.md) for detailed test breakdown.

## Usage

### Fetch Feeds

Fetch all configured feeds and store results:

```bash
feedpulse fetch --config config.yaml
```

Output example:
```
Fetching 3 feeds (max concurrency: 5)...
  ✓ HackerNews Top          — 500 items (42 new) in 124ms
  ✓ Lobsters                — 25 items (25 new) in 189ms
  ✗ GitHub Trending         — error: HTTP 403 Forbidden after 3 retries

Done: 2/3 succeeded, 525 items (67 new), 1 error
```

### Generate Report

Generate a summary report of all feeds:

```bash
# Table format (default)
feedpulse report --config config.yaml

# JSON format
feedpulse report --config config.yaml --format json

# CSV format
feedpulse report --config config.yaml --format csv

# Filter by source
feedpulse report --config config.yaml --source "HackerNews Top"
```

### List Sources

List all configured sources and their status:

```bash
feedpulse sources --config config.yaml
```

### Version & Help

```bash
feedpulse --version
feedpulse --help
```

## Configuration

Create a `config.yaml` file:

```yaml
settings:
  max_concurrency: 5          # Max parallel fetches
  default_timeout_secs: 10    # Per-feed HTTP timeout
  retry_max: 3                # Max retry attempts per feed
  retry_base_delay_ms: 500    # Base delay for exponential backoff
  database_path: "feedpulse.db"

feeds:
  - name: "HackerNews Top"
    url: "https://hacker-news.firebaseio.com/v0/topstories.json"
    feed_type: "json"
    refresh_interval_secs: 300
    headers: {}

  - name: "Lobsters"
    url: "https://lobste.rs/hottest.json"
    feed_type: "json"
    refresh_interval_secs: 300
    headers: {}
```

### Configuration Validation

The tool validates:
- `max_concurrency`: must be between 1-50
- `default_timeout_secs`: must be positive
- `name`: required, non-empty
- `url`: required, valid HTTP/HTTPS URL
- `feed_type`: must be one of: json, rss, atom
- `refresh_interval_secs`: must be positive

## Error Handling

The tool gracefully handles:
- Missing or invalid config files
- Network errors (DNS, timeouts, connection failures)
- HTTP errors (4xx, 5xx) with retry logic
- Malformed JSON responses
- Missing required fields in feed items
- Database lock contention
- Rate limiting (HTTP 429)

All errors are logged but don't crash the application. Failed feeds are skipped and other feeds continue processing.

## Dependencies

- `reqwest` + `tokio` - Async HTTP client
- `serde` + `serde_yaml` - Config parsing
- `rusqlite` - SQLite database
- `clap` - CLI argument parsing
- `comfy-table` - Table formatting
- `chrono` - Date/time handling
- `sha2` - SHA256 hashing for item IDs
- `url` - URL validation

## Database Schema

The tool creates a local SQLite database with two tables:

**feed_items**: Stores normalized feed items
- `id` (TEXT PRIMARY KEY) - SHA256 of source + url
- `title`, `url`, `source` (TEXT NOT NULL)
- `timestamp` (TEXT) - ISO 8601 datetime
- `tags` (TEXT) - JSON array
- `raw_data` (TEXT) - Original JSON
- `created_at` (TEXT NOT NULL)

**fetch_log**: Tracks fetch history
- `id` (INTEGER PRIMARY KEY)
- `source`, `fetched_at`, `status` (TEXT NOT NULL)
- `items_count` (INTEGER)
- `error_message` (TEXT)
- `duration_ms` (INTEGER)

## Development

### Project Structure

```
src/
├── main.rs          # CLI entry point
├── config.rs        # Config loading and validation
├── fetcher.rs       # Concurrent feed fetching
├── parser.rs        # Feed parsing and normalization
├── storage.rs       # SQLite operations
├── reporter.rs      # Report generation
└── models.rs        # Data structures
```

### Running Tests

```bash
cargo test
```

## Performance

- Typical fetch time for 10 feeds: < 10 seconds
- Concurrent fetching respects `max_concurrency` limit
- Database operations use transactions for atomicity
- Items are deduplicated by SHA256(source + url)

## Exit Codes

- `0` - Success
- `1` - Error (config invalid, fetch failed, etc.)

## License

MIT
