# Go Rebuild Task — Enhanced Implementation

## Context
The current Go implementation is solid (21/25 score) but only received a basic task spec. Both Rust and Python received detailed rebuild specifications and dramatically improved:
- **Rust:** 0 tests → 70 tests, 1,252 LOC
- **Python:** 43 tests → 138 tests, 2,406 LOC

For a fair comparison, Go deserves the same treatment.

**Current Go Stats:**
- Source LOC: 1,353
- Test LOC: 847
- Tests: 32
- Score: 21/25
- Status: Production-ready (but incomplete)

**Goal:** Apply comprehensive requirements to see Go's full potential with AI assistance.

---

## Requirements

### 1. Enhanced Test Suite (Target: ≥1,500 LOC)
Current: 847 LOC across 3 test files (32 tests)  
Target: ≥1,500 LOC across 6+ test files (≥70 tests)

**New test files to add:**
- `internal/*/integration_test.go` — End-to-end workflow tests
- `internal/*/edge_cases_test.go` — Boundary conditions
- Expand existing `*_test.go` files

**Test categories to expand:**

1. **Config validation** (current: ~8 tests → target: 15+)
   - Invalid URLs (malformed, unsupported schemes: ftp://, file://)
   - Timeout/concurrency edge cases (0, negative, >1000)
   - Empty config, missing sections
   - Unicode in config values (测试, 🚀)
   - Very long field values (10,000+ chars)
   - YAML edge cases (anchors, aliases, multiline)

2. **Parser edge cases** (current: ~10 tests → target: 25+)
   - Extremely nested JSON (10+ levels deep)
   - Very large responses (1MB+ JSON)
   - Empty arrays, null values at every field
   - Timestamps in every format (Unix, ISO, RFC, custom strings)
   - Special characters (<>&"', emojis, RTL text: العربية)
   - Partial data scenarios (some fields present, others missing)
   - Content-Type edge cases (charset=utf-8, application/json; charset=iso-8859-1)

3. **Error scenarios** (current: ~12 tests → target: 20+)
   - All 16 from SPEC.md
   - Additional scenarios:
     - Context cancellation (graceful shutdown)
     - DNS lookup failures
     - Connection refused
     - SSL/TLS certificate errors
     - HTTP redirect loops (301/302/308)
     - Slow response (data trickling in)
     - Invalid Content-Type headers
     - Gzip/deflate decompression errors
     - IPv6 vs IPv4 handling
     - HTTP/2 vs HTTP/1.1 compatibility

4. **Storage tests** (current: ~5 tests → target: 20+)
   - Duplicate detection accuracy (by ID and content hash)
   - Transaction rollback on error
   - Database file corruption recovery
   - Concurrent read/write operations
   - Query performance (1000+ items)
   - Schema validation and migrations
   - Foreign key constraints
   - NULL handling
   - Empty database queries
   - Large batch inserts (1000+ items)
   - Item ordering (by timestamp, source)
   - Database path validation
   - Special characters in data
   - Unicode data (emoji, CJK, RTL)

5. **Integration tests** (new: target 15+ tests)
   - Full fetch → parse → store → report workflow
   - Multi-feed concurrent execution
   - Resume after partial failure
   - Config reload without restart
   - Performance tests (10+ feeds, 1000+ items)
   - Memory usage under load
   - Goroutine leak detection
   - Race condition testing (`go test -race`)

### 2. Code Quality Improvements

**Add custom error types:**
```go
// internal/errors/errors.go (NEW)
package errors

import "fmt"

// Domain-specific errors
type ConfigError struct { /* ... */ }
type NetworkError struct { /* ... */ }
type ParseError struct { /* ... */ }
type StorageError struct { /* ... */ }
type ValidationError struct { /* ... */ }
```

**Add validation package:**
```go
// internal/validator/validator.go (NEW)
package validator

func ValidateURL(url string) error { /* ... */ }
func ValidateTimeout(timeout int) error { /* ... */ }
func ValidateConcurrency(max int) error { /* ... */ }
func ValidateFeedConfig(feed *config.Feed) error { /* ... */ }
```

**Improve error messages:**
- Current: "invalid config"
- Enhanced: "invalid config at feeds[0].url: must be HTTP or HTTPS, got: ftp://example.com"

**Add structured logging:**
- Use `log/slog` (Go 1.21+) for structured logging
- Add log levels (DEBUG, INFO, WARN, ERROR)
- Context-aware logging

**Performance optimizations:**
- Batch database inserts (current: one-by-one)
- Connection pooling for HTTP client
- Database connection pooling
- Goroutine pool for concurrent fetches (limit active goroutines)

### 3. Enhanced Features

**Better CLI:**
- Add `--dry-run` mode (show what would be fetched)
- Add `--verbose` levels (0-3 for different detail)
- Add `--format json|table|csv` for report output
- Add progress bars (using a library like `github.com/schollz/progressbar`)
- Add `--check` flag to validate config without fetching

**Observability:**
- Add structured logging (slog)
- Add metrics collection (fetch times, error rates)
- Add `--stats` flag to show historical performance
- Add request ID tracing

**RSS/Atom Support:**
- Implement RSS parsing (currently marked "not implemented")
- Implement Atom parsing
- Use standard library `encoding/xml` or third-party library

### 4. Documentation

**Comprehensive README.md:**
- Installation (go install, from source)
- Quick start (5-minute setup)
- Configuration reference (every field explained with examples)
- Usage examples (common workflows with code snippets)
- Troubleshooting section (common errors and fixes)
- Architecture overview (packages, data flow diagram)
- Performance characteristics (benchmarks)

**Add CONTRIBUTING.md:**
- Development setup
- Running tests (`go test -v -race ./...`)
- Code style guide (gofmt, golangci-lint)
- PR process
- Commit message format
- Issue guidelines

**Add package documentation:**
```go
// Package parser provides feed parsing and normalization.
//
// It supports multiple feed formats including JSON, RSS, and Atom.
// Each parser is responsible for converting source-specific formats
// into the common FeedItem model.
//
// Example:
//   parser := NewParser()
//   result := parser.Parse("HackerNews", "json", rawData)
//   if len(result.Errors) > 0 {
//       // handle errors
//   }
package parser
```

### 5. Project Structure Improvements

**Add these files/packages:**
```
go/
├── cmd/
│   └── feedpulse/
│       └── main.go
├── internal/
│   ├── cli/
│   │   ├── commands.go
│   │   └── commands_test.go
│   ├── config/
│   │   ├── config.go
│   │   ├── config_test.go
│   │   └── validator.go           # NEW: Config validation
│   ├── errors/                     # NEW: Custom error types
│   │   ├── errors.go
│   │   └── errors_test.go
│   ├── fetcher/
│   │   ├── fetcher.go
│   │   ├── fetcher_test.go
│   │   └── integration_test.go    # NEW
│   ├── parser/
│   │   ├── parser.go
│   │   ├── parser_test.go
│   │   ├── edge_cases_test.go     # NEW
│   │   └── rss.go                 # NEW: RSS/Atom parsing
│   ├── storage/
│   │   ├── storage.go
│   │   ├── storage_test.go
│   │   └── integration_test.go    # NEW
│   ├── models/                     # NEW: Shared models
│   │   └── models.go
│   └── testutil/                   # NEW: Test helpers
│       └── testutil.go
├── docs/                           # NEW
│   └── architecture.md
├── go.mod
├── go.sum
├── Makefile                        # NEW: Build automation
├── README.md
└── CONTRIBUTING.md                 # NEW
```

---

## Success Criteria

✅ Test LOC ≥1,500 (target: exceed Python's 2,406)  
✅ Tests ≥70 (target: match Python's 138)  
✅ All tests passing: `go test -v ./...`  
✅ Race detector clean: `go test -race ./...`  
✅ Test coverage ≥80%: `go test -coverprofile=coverage.out ./...`  
✅ All 16 error scenarios validated  
✅ Enhanced documentation (README, CONTRIBUTING, package docs)  
✅ RSS/Atom parsing implemented OR clearly documented as deferred

---

## Specific Tasks

### Phase 1: Test Expansion (High Priority)
1. Create `internal/testutil/testutil.go` with shared test helpers
2. Create `*_integration_test.go` files with 15+ end-to-end tests
3. Create `*_edge_cases_test.go` files with 30+ boundary tests
4. Expand `storage_test.go` from 5 to 20+ tests
5. Expand `parser_test.go` from 10 to 25+ tests
6. Expand `config_test.go` from 8 to 15+ tests
7. Add table-driven tests for all parsers

### Phase 2: Error Handling (High Priority)
1. Create `internal/errors/errors.go` with domain-specific errors
2. Update all packages to use custom errors
3. Add error wrapping with context (`fmt.Errorf("%w", err)`)
4. Improve error messages (include suggestions)

### Phase 3: Documentation (Medium Priority)
1. Write comprehensive README.md (500+ lines)
2. Add package-level documentation to all packages
3. Create CONTRIBUTING.md (300+ lines)
4. Add code examples to complex functions
5. Create `docs/architecture.md`

### Phase 4: Code Quality (Medium Priority)
1. Create `internal/config/validator.go` for validation
2. Add structured logging with `log/slog`
3. Refactor duplicated code
4. Add performance profiling and optimize hot paths

### Phase 5: Enhanced Features (Lower Priority)
1. Implement RSS/Atom parsing
2. Add `--dry-run` mode
3. Add `--verbose` levels
4. Add `--format` options (json, csv)
5. Add progress bars

---

## Testing Checklist

Run these to validate completion:

```bash
# All tests pass
go test -v ./...

# Race detector
go test -race ./...

# Test coverage ≥80%
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Linting passes
golangci-lint run

# Build succeeds
go build -o feedpulse cmd/feedpulse/main.go

# Binary runs
./feedpulse fetch --config config.yaml
```

---

## Comparison Targets

We're competing against:
- **Python (post-rebuild):** 2,406 LOC tests, 138 tests, 25/25 score
- **Rust (post-rebuild):** 1,252 LOC tests, 70 tests, 22/25 score

**Go's advantages:**
- Excellent concurrency (goroutines, channels)
- Fast compilation and execution
- Strong standard library
- Built-in race detector
- Simple, explicit code

**Target Go score after rebuild:** 24/25 (match Python's quality)

---

## Expected Outcomes

### Quantitative:
- Test LOC: 847 → **1,800+** (exceed Rust, approach Python)
- Tests: 32 → **80+** (exceed Rust and Python)
- Test coverage: ~70% → **85%+**
- Test files: 3 → **8+**

### Qualitative:
- **Best concurrency story** (goroutines showcase)
- **Fastest execution** (compiled binary, no runtime)
- **Cleanest error handling** (explicit, no exceptions)
- **Best tooling integration** (go test, go tool cover, race detector)

---

## Go-Specific Testing Patterns

### 1. Table-Driven Tests
```go
func TestParseTimestamp(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    string
        wantErr bool
    }{
        {"unix timestamp", 1609459200, "2021-01-01T00:00:00Z", false},
        {"ISO string", "2021-01-01T12:00:00Z", "2021-01-01T12:00:00Z", false},
        {"nil", nil, "", false},
        {"invalid", "not a timestamp", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseTimestamp(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseTimestamp() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Test Helpers
```go
// testutil/testutil.go
func NewTestDB(t *testing.T) *storage.Storage {
    t.Helper()
    tmpFile, err := os.CreateTemp("", "test-*.db")
    if err != nil {
        t.Fatalf("failed to create temp db: %v", err)
    }
    t.Cleanup(func() { os.Remove(tmpFile.Name()) })
    
    db, err := storage.NewStorage(tmpFile.Name())
    if err != nil {
        t.Fatalf("failed to init db: %v", err)
    }
    return db
}
```

### 3. Integration Tests with Context
```go
func TestIntegrationFetchAndStore(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Test implementation...
}
```

### 4. Race Detection Tests
```go
func TestConcurrentFetches(t *testing.T) {
    // This test specifically checks for race conditions
    // Run with: go test -race
    
    fetcher := NewFetcher(/* ... */)
    
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            fetcher.Fetch(/* ... */)
        }()
    }
    wg.Wait()
}
```

---

## Notes

- **Maintain Go idioms** — explicit error handling, no exceptions
- **Use standard library first** — avoid third-party deps when possible
- **Test with race detector** — `go test -race` must pass
- **Idiomatic naming** — `NewParser()`, `ParseFeed()`, `feedItem`
- **Documentation comments** — start with package/function name

---

## Reference Files

- **Current implementation:** `~/dev/feedpulse/go/`
- **Spec:** `~/dev/feedpulse/SPEC.md`
- **Python rebuild task:** `~/dev/feedpulse/PYTHON_REBUILD_TASK.md` (for comparison)
- **Rust rebuild task:** `~/dev/feedpulse/RUST_REBUILD_TASK.md` (for comparison)
- **Test fixtures:** `~/dev/feedpulse/test-fixtures/`

---

**Estimated Time:** 2-3 hours (AI-assisted)

**Success Metric:** Go achieves 24/25 score with:
- Most tests (80+)
- Excellent test coverage (85%+)
- Best concurrency showcase
- Comprehensive documentation
- Production-ready

Let's show what Go can do with proper specifications! 🐹
