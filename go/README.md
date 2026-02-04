# feedpulse - Go Implementation

A concurrent feed aggregator CLI tool that fetches, normalizes, and stores data from multiple JSON feeds.

## Features

- Concurrent fetching with configurable limits (semaphore pattern)
- Exponential backoff retry logic with jitter
- Support for 4 feed types: HackerNews, GitHub, Reddit, Lobsters
- SQLite storage with deduplication
- Multiple output formats: table, JSON, CSV
- Comprehensive error handling

## Requirements

- Go 1.20 or later
- CGO enabled (for SQLite)
- Linux/macOS (Windows should work but untested)

## Building

```bash
go build -o feedpulse ./cmd/feedpulse
```

## Usage

### Fetch feeds

```bash
./feedpulse fetch --config config.yaml
```

### Generate report

```bash
# Table format (default)
./feedpulse report --config config.yaml

# JSON format
./feedpulse report --config config.yaml --format json

# CSV format
./feedpulse report --config config.yaml --format csv

# Filter by source
./feedpulse report --config config.yaml --source "HackerNews Top"
```

### List sources

```bash
./feedpulse sources --config config.yaml
```

## Configuration

See `../test-fixtures/valid-config.yaml` for an example configuration file.

Required fields:
- `settings.max_concurrency`: Max parallel fetches (1-50)
- `settings.default_timeout_secs`: HTTP timeout per feed
- `settings.retry_max`: Number of retry attempts
- `settings.retry_base_delay_ms`: Base delay for exponential backoff
- `settings.database_path`: Path to SQLite database file
- `feeds[]`: Array of feed configurations

Each feed requires:
- `name`: Unique feed name
- `url`: Valid HTTP/HTTPS URL
- `feed_type`: One of: json, rss, atom (only json implemented)
- `headers`: Optional HTTP headers

## Architecture

- `cmd/feedpulse/main.go`: Entry point
- `internal/config/`: Configuration parsing and validation
- `internal/fetcher/`: Concurrent HTTP fetching with retries
- `internal/parser/`: Feed normalization (4 feed types)
- `internal/storage/`: SQLite operations
- `internal/cli/`: Cobra command definitions

## Error Handling

The tool gracefully handles:
- Missing or invalid config files
- Network failures and timeouts
- HTTP errors (4xx, 5xx)
- Malformed JSON
- Missing required fields
- Database errors
- Ctrl+C during fetch (saves completed results)

## Dependencies

- `github.com/mattn/go-sqlite3`: SQLite driver
- `github.com/spf13/cobra`: CLI framework
- `github.com/olekukonko/tablewriter`: Table formatting
- `gopkg.in/yaml.v3`: YAML parsing

## Testing

```bash
go test ./...
```

## Performance

Fetching 4 feeds typically completes in under 2 seconds with max_concurrency=5.

## License

MIT
