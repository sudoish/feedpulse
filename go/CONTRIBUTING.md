# Contributing to FeedPulse Go

Thank you for your interest in contributing to FeedPulse! This document provides guidelines and instructions for contributing to the Go implementation.

## Table of Contents

- [Development Setup](#development-setup)
- [Code Style](#code-style)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Commit Guidelines](#commit-guidelines)
- [Issue Guidelines](#issue-guidelines)
- [Architecture Guidelines](#architecture-guidelines)

## Development Setup

### Prerequisites

- Go 1.21+ (for `log/slog` and other modern features)
- SQLite 3
- Git
- Make (optional, for build automation)

### Initial Setup

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/feedpulse.git
cd feedpulse/go

# Install dependencies
go mod download

# Verify setup
go test ./...
```

### Environment Setup

```bash
# Optional: Set up pre-commit hooks
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
go fmt ./...
go vet ./...
go test -race -short ./...
EOF
chmod +x .git/hooks/pre-commit
```

## Code Style

### Go Conventions

FeedPulse follows standard Go conventions:

1. **Use `gofmt`**: All code must be formatted with `gofmt`
   ```bash
   go fmt ./...
   ```

2. **Use `golangci-lint`**: Install and run linter
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   golangci-lint run
   ```

3. **Naming Conventions**:
   - Exported functions: `NewParser()`, `ParseFeed()`
   - Unexported functions: `parseJSON()`, `validateURL()`
   - Interfaces: `Parser`, `Storage` (no "I" prefix)
   - Constants: `MaxConcurrency`, `DefaultTimeout`

4. **Error Handling**:
   ```go
   // ✅ Good: Explicit error handling
   if err != nil {
       return fmt.Errorf("failed to parse feed: %w", err)
   }

   // ❌ Bad: Ignoring errors
   parseResult, _ := parser.Parse(data)
   ```

5. **Package Documentation**:
   ```go
   // Package parser provides feed parsing and normalization.
   //
   // It supports multiple feed formats including JSON, RSS, and Atom.
   // Each parser is responsible for converting source-specific formats
   // into the common FeedItem model.
   package parser
   ```

### Go-Specific Patterns

#### Error Wrapping

```go
// Use custom error types from internal/errors
if err := validateURL(url); err != nil {
    return errors.NewValidationError("url", url, "format", err.Error())
}
```

#### Struct Initialization

```go
// ✅ Good: Explicit field names
item := storage.FeedItem{
    ID:        id,
    Title:     title,
    URL:       url,
    Source:    source,
    CreatedAt: time.Now(),
}

// ❌ Bad: Positional arguments (fragile)
item := storage.FeedItem{id, title, url, source, nil, nil, nil, time.Now()}
```

#### Nil Checks

```go
// ✅ Good: Check for nil before dereferencing
if item.Timestamp != nil {
    timestamp := *item.Timestamp
    // use timestamp
}

// ❌ Bad: Potential panic
timestamp := *item.Timestamp
```

## Testing

### Test Structure

FeedPulse uses Go's standard testing framework with table-driven tests:

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

### Test Helpers

Use `t.Helper()` for test helper functions:

```go
func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# Specific package
go test ./internal/parser

# Specific test
go test -run TestParseTimestamp ./internal/parser

# Short mode (skip integration tests)
go test -short ./...

# Race detector
go test -race ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage Requirements

- **New code**: ≥80% coverage
- **Bug fixes**: Add regression test
- **Critical paths**: ≥90% coverage (parsing, storage)

### Integration Tests

Mark integration tests appropriately:

```go
func TestIntegration_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    
    // Test implementation...
}
```

### Race Detection

All PRs must pass race detector:

```bash
go test -race ./...
```

## Pull Request Process

### Before Submitting

1. **Run Tests**
   ```bash
   go test ./...
   go test -race ./...
   ```

2. **Check Formatting**
   ```bash
   go fmt ./...
   ```

3. **Run Linter**
   ```bash
   golangci-lint run
   ```

4. **Update Documentation** (if applicable)
   - Update README.md for new features
   - Add package documentation
   - Update SPEC.md if behavior changes

### PR Title Format

```
type(scope): brief description

Examples:
feat(parser): add RSS parsing support
fix(storage): prevent race condition in SaveItems
docs(readme): update installation instructions
test(parser): add edge case tests for unicode
refactor(config): extract validation logic
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `test`: Adding or updating tests
- `refactor`: Code restructuring without behavior change
- `perf`: Performance improvement
- `chore`: Maintenance tasks

### PR Description Template

```markdown
## Description
Brief description of changes

## Motivation
Why is this change needed?

## Changes
- Change 1
- Change 2

## Testing
How was this tested?

## Screenshots (if applicable)
Add screenshots for UI changes

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] `go fmt` run
- [ ] `go vet` passes
- [ ] `go test -race` passes
- [ ] Backwards compatible (or breaking change documented)
```

### Review Process

1. **Automated Checks**: CI must pass (tests, linting, race detector)
2. **Code Review**: At least one approval required
3. **Testing**: Manual testing for significant changes
4. **Documentation**: Ensure docs are updated
5. **Merge**: Squash and merge preferred

## Commit Guidelines

### Commit Message Format

```
type(scope): subject

body (optional)

footer (optional)
```

Example:

```
feat(parser): add Lobsters feed support

Implement parser for Lobsters JSON format with support for:
- Comment URLs as fallback
- Tag extraction
- Timestamp parsing

Closes #42
```

### Commit Best Practices

- **Atomic commits**: One logical change per commit
- **Descriptive messages**: Explain "why", not just "what"
- **Reference issues**: Use "Closes #123" or "Fixes #456"
- **Sign commits**: `git commit -s` (if required)

## Issue Guidelines

### Bug Reports

```markdown
**Describe the bug**
Clear description of the issue

**To Reproduce**
Steps to reproduce:
1. Create config with...
2. Run `feedpulse fetch...`
3. Observe error...

**Expected behavior**
What should happen

**Actual behavior**
What actually happens

**Environment**
- Go version: 1.21
- OS: Linux/Mac/Windows
- FeedPulse version: v1.0.0

**Logs**
```
Paste relevant logs here
```

**Additional context**
Any other relevant information
```

### Feature Requests

```markdown
**Problem Statement**
What problem does this solve?

**Proposed Solution**
How should this work?

**Alternatives Considered**
What other approaches did you consider?

**Additional Context**
Any mockups, examples, or references
```

### Questions

Use GitHub Discussions for questions, not issues.

## Architecture Guidelines

### Package Organization

```
internal/
├── cli/       # Command-line interface (isolated)
├── config/    # Configuration (no business logic)
├── errors/    # Error types (shared)
├── fetcher/   # HTTP operations (no parsing)
├── parser/    # Parsing only (no fetching/storage)
├── storage/   # Database operations (no business logic)
└── testutil/  # Shared test helpers
```

### Dependency Rules

- **No circular dependencies**: Use interfaces for decoupling
- **Config → Parser → Storage**: Unidirectional flow
- **Errors**: Can be imported by all packages
- **Testutil**: Test packages only

### Interface Design

```go
// ✅ Good: Small, focused interfaces
type Parser interface {
    Parse(source, feedType string, data []byte) ParseResult
}

// ❌ Bad: Large, god interfaces
type FeedProcessor interface {
    Fetch() error
    Parse() error
    Store() error
    Report() error
}
```

### Error Handling

Use custom error types from `internal/errors`:

```go
// Network error
if err != nil {
    return errors.NewNetworkError(url, "fetch", "connection timeout", err)
}

// Parse error
if err := json.Unmarshal(data, &result); err != nil {
    return errors.NewParseError(source, feedType, "malformed JSON", err)
}

// Validation error
if len(name) == 0 {
    return errors.NewValidationError("name", name, "required", "feed name cannot be empty")
}
```

### Concurrency

- **Use goroutines** for I/O-bound operations (fetching)
- **Avoid goroutines** for CPU-bound operations (parsing)
- **Protect shared state** with mutexes or channels
- **Test with `-race`** detector always

```go
// ✅ Good: Bounded concurrency
sem := make(chan struct{}, maxConcurrency)
for _, feed := range feeds {
    sem <- struct{}{}  // Acquire
    go func(f Feed) {
        defer func() { <-sem }()  // Release
        fetchFeed(f)
    }(feed)
}
```

### Database Operations

- **Use transactions** for multi-statement operations
- **Prepare statements** for repeated queries
- **Close resources** with `defer`
- **Handle SQLite locks** gracefully

```go
// ✅ Good: Transaction with defer
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback()  // Rollback if Commit not called

// ... operations ...

return tx.Commit()
```

## Performance Considerations

### Optimization Guidelines

1. **Profile first**: Use `pprof` to identify bottlenecks
   ```bash
   go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
   go tool pprof cpu.prof
   ```

2. **Benchmark changes**: Add benchmarks for performance-critical code
   ```go
   func BenchmarkParse(b *testing.B) {
       data := loadTestData()
       parser := NewParser()
       
       b.ResetTimer()
       for i := 0; i < b.N; i++ {
           parser.Parse("GitHub", "json", data)
       }
   }
   ```

3. **Avoid premature optimization**: Optimize only when measurements show need

### Common Optimizations

- **Batch database operations**: Use transactions for multiple inserts
- **Reuse HTTP clients**: Don't create new clients per request
- **Buffer pooling**: Use `sync.Pool` for frequently allocated objects
- **Minimize allocations**: Reuse slices and structs where safe

## Documentation

### Package Documentation

Every package needs a package comment:

```go
// Package parser provides feed parsing and normalization.
//
// It supports multiple feed formats including JSON, RSS, and Atom.
// Each parser is responsible for converting source-specific formats
// into the common FeedItem model.
//
// Example usage:
//   parser := NewParser()
//   result := parser.Parse("GitHub", "json", rawData)
//   if len(result.Errors) > 0 {
//       // handle errors
//   }
package parser
```

### Function Documentation

Document exported functions with usage examples:

```go
// NewStorage creates a new storage instance and initializes the database schema.
//
// The database is created if it doesn't exist. WAL mode is enabled automatically
// for better concurrent read performance.
//
// Example:
//   storage, err := NewStorage("feedpulse.db")
//   if err != nil {
//       log.Fatal(err)
//   }
//   defer storage.Close()
func NewStorage(dbPath string) (*Storage, error) {
    // implementation
}
```

## Getting Help

- **Questions**: [GitHub Discussions]
- **Bugs**: [GitHub Issues]
- **Security**: security@example.com (private)
- **Chat**: [Discord/Slack]

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see LICENSE file).

Thank you for contributing to FeedPulse! 🎉
