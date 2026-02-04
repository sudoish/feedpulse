# feedpulse - Python Implementation

A concurrent feed aggregator CLI that fetches, normalizes, and stores data from multiple feed sources.

## Features

- **Concurrent fetching** with configurable concurrency limits
- **Automatic retries** with exponential backoff for failed requests
- **Data normalization** for multiple feed types (HackerNews, GitHub, Reddit, Lobsters)
- **SQLite storage** with automatic deduplication
- **Rich CLI** with table-formatted reports
- **Comprehensive error handling** for 16+ failure scenarios

## Requirements

- Python 3.8 or higher
- pip (Python package manager)

## Installation

### Option 1: Install in development mode

```bash
cd python
pip install -e .
```

### Option 2: Install dependencies only

```bash
cd python
pip install -r requirements.txt
```

## Configuration

Create a `config.yaml` file with your feed sources:

```yaml
settings:
  max_concurrency: 5
  default_timeout_secs: 10
  retry_max: 3
  retry_base_delay_ms: 500
  database_path: "feedpulse.db"

feeds:
  - name: "HackerNews Top"
    url: "https://hacker-news.firebaseio.com/v0/topstories.json"
    feed_type: "json"
    refresh_interval_secs: 300
    headers: {}

  - name: "GitHub Trending"
    url: "https://api.github.com/search/repositories?q=stars:>1000&sort=stars"
    feed_type: "json"
    refresh_interval_secs: 600
    headers:
      Accept: "application/vnd.github.v3+json"

  - name: "Reddit Programming"
    url: "https://www.reddit.com/r/programming/hot.json"
    feed_type: "json"
    refresh_interval_secs: 300
    headers:
      User-Agent: "feedpulse/1.0"

  - name: "Lobsters"
    url: "https://lobste.rs/hottest.json"
    feed_type: "json"
    refresh_interval_secs: 300
    headers: {}
```

## Usage

### Fetch all feeds

```bash
feedpulse fetch --config config.yaml
```

Output:
```
Fetching 4 feeds (max concurrency: 5)...
  ✓ HackerNews Top          — 30 items (12 new) in 245ms
  ✓ GitHub Trending          — 30 items (5 new) in 512ms
  ✗ Reddit Programming       — error: HTTP 429 after 3 retries
  ✓ Lobsters                 — 25 items (25 new) in 189ms

Done: 3/4 succeeded, 85 items (42 new), 1 error
```

### Generate a report

```bash
# Table format (default)
feedpulse report

# JSON format
feedpulse report --format json

# CSV format
feedpulse report --format csv

# Filter by source
feedpulse report --source "HackerNews Top"
```

### List configured sources

```bash
feedpulse sources
```

### Get help

```bash
feedpulse --help
feedpulse fetch --help
```

## Running without installation

If you haven't installed the package, you can run it as a module:

```bash
cd python
python -m feedpulse fetch --config config.yaml
```

## Running Tests

```bash
cd python
pip install pytest
pytest -v
```

Run tests with coverage:

```bash
pip install pytest-cov
pytest --cov=feedpulse --cov-report=html
```

## Error Handling

The tool handles various error scenarios gracefully:

- **Config errors**: Clear error messages with field names
- **Network errors**: Automatic retry with exponential backoff
- **HTTP errors**: Retry for 5xx, skip retry for 404
- **Parse errors**: Skip malformed items, continue processing
- **Database errors**: Retry on lock, suggest fixes for corruption
- **Ctrl+C**: Cancel pending fetches, save completed results

## Database Schema

SQLite database with two tables:

- `feed_items`: Normalized feed items with deduplication by ID (SHA256 of source+URL)
- `fetch_log`: History of fetch attempts with status and error messages

## Architecture

```
feedpulse/
├── __init__.py       # Package version
├── __main__.py       # Module entry point
├── cli.py            # Click CLI commands
├── config.py         # YAML config parsing and validation
├── fetcher.py        # Async HTTP fetching with retries
├── parser.py         # Feed data normalization
├── storage.py        # SQLite database operations
└── models.py         # Data models and classes
```

## Development

### Code style

```bash
pip install black flake8
black feedpulse/
flake8 feedpulse/
```

### Type checking

```bash
pip install mypy
mypy feedpulse/
```

## License

This is an experimental project for comparing AI-generated code across languages.

## Notes

- Feed parsing auto-detects structure based on JSON shape
- RSS/Atom parsing not implemented (JSON only for this experiment)
- Database uses SQLite WAL mode for better concurrent write performance
- All timestamps are stored in UTC ISO 8601 format
