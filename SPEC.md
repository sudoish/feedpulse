# feedpulse — Concurrent Feed Aggregator CLI

## Overview

A CLI tool that fetches multiple data feeds (RSS/JSON APIs), validates and normalizes the data, stores results in a local SQLite database, and generates summary reports.

This project exists as an experiment: the same spec will be implemented three times — in **Python**, **Go**, and **Rust** — each built entirely by an AI coding agent. The goal is to compare how each language's type system, compiler, and runtime behavior affect AI-generated code quality, correctness, and safety.

---

## Project Structure

Each implementation lives in its own directory:

```
feedpulse/
├── SPEC.md              # This file (shared spec)
├── python/              # Python implementation
├── go/                  # Go implementation
├── rust/                # Rust implementation
├── test-fixtures/       # Shared test data (bad JSON, malformed configs, etc.)
└── RESULTS.md           # Final comparison and findings
```

---

## Functional Requirements

### 1. Configuration

The tool reads a YAML configuration file (`config.yaml`) specifying feed sources and global settings.

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

**Validation rules (must be enforced):**
- `name`: required, non-empty string
- `url`: required, must be a valid HTTP/HTTPS URL
- `feed_type`: required, must be one of: `json`, `rss`, `atom`
- `refresh_interval_secs`: optional, positive integer, default 300
- `default_timeout_secs`: optional, positive integer, default 10
- `max_concurrency`: optional, positive integer between 1-50, default 5
- `database_path`: optional, default `"feedpulse.db"`

**On validation failure:** Print a clear, specific error message indicating which field failed and why. Exit with code 1. Do NOT print a stack trace.

---

### 2. Concurrent Fetching

Fetch all configured feeds concurrently, respecting the `max_concurrency` limit (semaphore pattern).

**For each feed:**
1. Make an HTTP GET request with configured headers and timeout
2. On failure (network error, timeout, HTTP 4xx/5xx): retry with exponential backoff
   - Delay formula: `retry_base_delay_ms * 2^attempt` (with jitter)
   - After `retry_max` attempts, mark feed as failed and continue to next
3. On success: pass raw response body to the parser

**Concurrency requirements:**
- Use the language's idiomatic concurrency model (asyncio, goroutines, tokio)
- A failing feed must NOT block or crash other feeds
- All fetches must respect the timeout — no hanging forever

---

### 3. Data Normalization

Each feed returns a different JSON structure. The parser must extract items and normalize them into a unified schema.

**Unified `FeedItem` schema:**

| Field       | Type              | Required | Notes                                    |
|-------------|-------------------|----------|------------------------------------------|
| `id`        | string            | yes      | Generated: SHA256 of `source_name + url` |
| `title`     | string            | yes      | Extracted from feed item                 |
| `url`       | string            | yes      | Link to the original item                |
| `source`    | string            | yes      | Feed name from config                    |
| `timestamp` | ISO 8601 datetime | no       | Parse from feed if available             |
| `tags`      | list of strings   | no       | Extract if available, default empty      |
| `raw_data`  | string (JSON)     | no       | Original item JSON for debugging         |

**Parser requirements per feed type:**

- **HackerNews:** Top stories returns an array of IDs. For this experiment, just store the IDs as items with title = "HN Story {id}" and url = "https://news.ycombinator.com/item?id={id}". (Avoids N+1 API calls.)
- **GitHub:** Extract from `items[]`: `full_name` → title, `html_url` → url, `topics` → tags, `updated_at` → timestamp
- **Reddit:** Extract from `data.children[].data`: `title` → title, `url` → url, `created_utc` → timestamp (convert from unix), `link_flair_text` → tags (as single-item list if present)
- **Lobsters:** Extract from root array: `title` → title, `url` → url (fall back to `comments_url`), `created_at` → timestamp, `tags` → tags

**Error handling during parsing:**
- Missing required field (title or url) → skip item, log warning with item index and source
- Wrong type (e.g., title is a number) → attempt string coercion, skip if impossible
- Malformed JSON → skip entire feed, log error, do NOT crash
- Unexpected extra fields → ignore them

---

### 4. Local Storage (SQLite)

Store normalized items in a local SQLite database.

**Schema:**

```sql
CREATE TABLE IF NOT EXISTS feed_items (
    id TEXT PRIMARY KEY,           -- SHA256 hash
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    source TEXT NOT NULL,
    timestamp TEXT,                -- ISO 8601 or NULL
    tags TEXT,                     -- JSON array as string
    raw_data TEXT,
    created_at TEXT NOT NULL       -- When feedpulse stored this item
);

CREATE TABLE IF NOT EXISTS fetch_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    fetched_at TEXT NOT NULL,      -- ISO 8601
    status TEXT NOT NULL,          -- 'success' | 'error'
    items_count INTEGER DEFAULT 0,
    error_message TEXT,
    duration_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_feed_items_source ON feed_items(source);
CREATE INDEX IF NOT EXISTS idx_feed_items_timestamp ON feed_items(timestamp);
CREATE INDEX IF NOT EXISTS idx_fetch_log_source ON fetch_log(source);
```

**Requirements:**
- Create tables on first run (auto-migration)
- Deduplicate items by `id` (upsert — update if exists)
- All database writes happen in a single transaction per fetch cycle
- Handle database lock errors gracefully (retry or queue)

---

### 5. CLI Interface

The tool must support the following commands:

```
feedpulse fetch [--config path]       # Fetch all feeds and store results
feedpulse report [--config path]      # Generate summary report
         --format json|table|csv      # Output format (default: table)
         --source <name>              # Filter by source name
         --since <duration>           # Filter items newer than (e.g., "24h", "7d")
feedpulse sources [--config path]     # List configured sources and their status
feedpulse --version                   # Print version
feedpulse --help                      # Print help
```

**`fetch` output:**
```
Fetching 4 feeds (max concurrency: 5)...
  ✓ HackerNews Top          — 30 items (12 new) in 245ms
  ✓ GitHub Trending          — 30 items (5 new) in 512ms
  ✗ Reddit Programming       — error: HTTP 429 after 3 retries
  ✓ Lobsters                 — 25 items (25 new) in 189ms

Done: 3/4 succeeded, 85 items (42 new), 1 error
```

**`report` output (table format):**
```
╭──────────────────────┬───────┬──────────┬────────────┬──────────────────╮
│ Source               │ Items │ Errors   │ Error Rate │ Last Success     │
├──────────────────────┼───────┼──────────┼────────────┼──────────────────┤
│ HackerNews Top       │   142 │        0 │       0.0% │ 2025-01-15 14:30 │
│ GitHub Trending      │    89 │        1 │       2.3% │ 2025-01-15 14:30 │
│ Reddit Programming   │     0 │        5 │     100.0% │ never            │
│ Lobsters             │   201 │        0 │       0.0% │ 2025-01-15 14:30 │
╰──────────────────────┴───────┴──────────┴────────────┴──────────────────╯

Total: 432 items across 4 sources
```

---

### 6. Error Handling Matrix

Every implementation must handle these scenarios correctly. This is the core of the experiment.

| Scenario                          | Expected Behavior                                    |
|-----------------------------------|------------------------------------------------------|
| Config file missing               | Print: "Error: config file not found: {path}" + exit 1 |
| Config file invalid YAML          | Print: "Error: invalid config: {details}" + exit 1   |
| Config missing required field     | Print: "Error: feed '{name}': missing field '{field}'" + exit 1 |
| Config invalid URL                | Print: "Error: feed '{name}': invalid URL '{url}'" + exit 1 |
| DNS resolution failure            | Retry, then log error, continue other feeds          |
| HTTP timeout                      | Retry, then log error, continue other feeds          |
| HTTP 429 (rate limit)             | Retry with backoff, then log error, continue         |
| HTTP 5xx                          | Retry with backoff, then log error, continue         |
| HTTP 404                          | No retry, log error, continue other feeds            |
| Malformed JSON response           | Log error, skip feed, continue others                |
| JSON missing expected fields      | Skip item, log warning, continue parsing             |
| JSON wrong types                  | Attempt coercion, skip item if impossible            |
| Database locked                   | Retry up to 3 times with 100ms delay                 |
| Database corrupted                | Print error, suggest deleting DB, exit 1             |
| Ctrl+C during fetch               | Cancel pending fetches, save completed results, exit |
| Disk full                         | Print error, exit 1                                  |
| No internet connection            | All feeds fail gracefully, report shows all errors   |

---

## Non-Functional Requirements

- **No external services** beyond the configured feed URLs (no API keys, no cloud dependencies)
- **Deterministic output** for the same input (sorted, consistent formatting)
- **Reasonable performance** — fetching 10 feeds should complete in < 30 seconds
- **Clean exit codes** — 0 for success, 1 for errors
- **Logging** — use stderr for logs/warnings, stdout for output
- **No panics/unhandled exceptions** — every error path must be covered

---

## Dependencies (Suggested)

Each implementation should use well-known, idiomatic libraries:

### Python
- `aiohttp` — async HTTP
- `pyyaml` — config parsing
- `sqlite3` — database (stdlib)
- `click` or `argparse` — CLI
- `rich` — table formatting (optional)

### Go
- `net/http` — HTTP (stdlib)
- `gopkg.in/yaml.v3` — config parsing
- `github.com/mattn/go-sqlite3` — database
- `github.com/spf13/cobra` — CLI
- `github.com/olekukonez/tablewriter` — table formatting

### Rust
- `reqwest` + `tokio` — async HTTP
- `serde` + `serde_yaml` — config parsing
- `rusqlite` — database
- `clap` — CLI
- `comfy-table` — table formatting

---

## AI Agent Instructions

When building this project, you are given this spec and nothing else. Implement the full tool according to this specification.

**Rules:**
1. Follow the spec exactly — do not skip error handling
2. Use idiomatic patterns for your language
3. Write unit tests for: config validation, data normalization, error scenarios
4. Include a `README.md` with build and run instructions
5. The final binary/script must be runnable with: `feedpulse fetch --config config.yaml`

**Do NOT:**
- Ask clarifying questions — the spec is intentionally complete
- Simplify the error handling — that's the whole point
- Use AI-specific libraries or frameworks
- Skip any of the 6 functional requirements

---

## Measurement Protocol

After each implementation is complete, the following will be recorded:

1. **Prompt count** — How many prompts were needed to reach a working build
2. **Compile/lint errors** — Errors caught before runtime (Go, Rust only; Python linting)
3. **Runtime errors** — Errors discovered only during execution
4. **Test results** — How many tests pass on first try
5. **Chaos test** — Feed it the shared `test-fixtures/` with intentionally broken data
6. **Lines of code** — `wc -l` on source files (excluding tests)
7. **Graceful degradation score** — Run with no internet, bad config, corrupt DB — count correct behaviors out of the 16 scenarios above
8. **Wall clock time** — Total time from first prompt to passing all tests
9. **AI hallucination count** — Times the AI used a non-existent API or wrong function signature

Results will be documented in `RESULTS.md`.
