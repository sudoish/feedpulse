# Rust Rebuild Task — Complete Implementation

## Context
The current Rust implementation at `~/dev/feedpulse/rust/` is incomplete:
- ❌ **Zero test coverage** (Python has 576 LOC, Go has 847 LOC)
- ❌ Missing error scenario validation (16 scenarios in SPEC.md)
- ❌ RSS/Atom not implemented (marked as "not yet implemented")
- ❌ No integration tests or chaos testing fixtures

This rebuild aims to match the **comprehensiveness of the Python implementation** to enable a fair language comparison.

---

## Requirements

### 1. Comprehensive Test Suite
Match or exceed Python's test coverage (576 LOC across 3 test files):
- `tests/test_config.rs` — Config validation, all error scenarios
- `tests/test_parser.rs` — Feed parsing, normalization, edge cases
- `tests/test_error_scenarios.rs` — All 16 error scenarios from SPEC.md

**Minimum test counts:**
- Config validation: ≥10 tests
- Parser normalization: ≥15 tests  
- Error scenarios: 16 tests (one per scenario)
- Integration tests: ≥5 tests

### 2. Error Handling Matrix (16 Scenarios)
Implement **all** scenarios from SPEC.md section 6:

| Scenario | Expected Behavior |
|----------|-------------------|
| Config file missing | Print: "Error: config file not found: {path}" + exit 1 |
| Config file invalid YAML | Print: "Error: invalid config: {details}" + exit 1 |
| Config missing required field | Print: "Error: feed '{name}': missing field '{field}'" + exit 1 |
| Config invalid URL | Print: "Error: feed '{name}': invalid URL '{url}'" + exit 1 |
| DNS resolution failure | Retry, then log error, continue other feeds |
| HTTP timeout | Retry, then log error, continue other feeds |
| HTTP 429 (rate limit) | Retry with backoff, then log error, continue |
| HTTP 5xx | Retry with backoff, then log error, continue |
| HTTP 404 | No retry, log error, continue other feeds |
| Malformed JSON response | Log error, skip feed, continue others |
| JSON missing expected fields | Skip item, log warning, continue parsing |
| JSON wrong types | Attempt coercion, skip item if impossible |
| Database locked | Retry up to 3 times with 100ms delay |
| Database corrupted | Print error, suggest deleting DB, exit 1 |
| Ctrl+C during fetch | Cancel pending fetches, save completed results, exit |
| Disk full | Print error, exit 1 |
| No internet connection | All feeds fail gracefully, report shows all errors |

### 3. Feed Type Support
- ✅ JSON feeds (HackerNews, GitHub, Reddit, Lobsters)
- ✅ RSS parsing (use `rss` crate)
- ✅ Atom parsing (use `atom_syndication` crate)

If RSS/Atom are too complex for this iteration, **document the limitation clearly** in README.md and handle gracefully with proper error messages.

### 4. Code Organization
Refactor to proper Rust project structure:

```
rust/
├── src/
│   ├── lib.rs          # Library crate (public API)
│   ├── main.rs         # Binary crate (CLI entry)
│   ├── config.rs
│   ├── fetcher.rs
│   ├── parser.rs
│   ├── storage.rs
│   ├── models.rs
│   └── reporter.rs
├── tests/              # Integration tests
│   ├── test_config.rs
│   ├── test_parser.rs
│   └── test_error_scenarios.rs
├── Cargo.toml
└── README.md
```

### 5. Chaos Testing Integration
Use the shared `test-fixtures/` directory:
- `test-fixtures/malformed.json` — Invalid JSON
- `test-fixtures/missing-fields.json` — JSON with missing required fields
- `test-fixtures/wrong-types.json` — JSON with type mismatches
- `test-fixtures/invalid-config.yaml` — Bad YAML syntax
- `test-fixtures/missing-required.yaml` — Missing required fields

Write tests that load these fixtures and verify error handling.

---

## Success Criteria

✅ All 16 error scenarios handled correctly  
✅ Test suite ≥500 LOC (match Python's comprehensiveness)  
✅ All tests pass: `cargo test`  
✅ Builds cleanly: `cargo build --release`  
✅ CLI works: `./target/release/feedpulse fetch --config config.yaml`  
✅ README.md updated with:
   - Build instructions
   - Test instructions
   - Known limitations (if RSS/Atom deferred)

---

## Implementation Notes

### Recommended Crates
```toml
[dependencies]
clap = { version = "4", features = ["derive"] }
tokio = { version = "1", features = ["full"] }
reqwest = { version = "0.11", features = ["json"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
serde_yaml = "0.9"
rusqlite = { version = "0.30", features = ["bundled"] }
anyhow = "1"           # Error handling
thiserror = "1"        # Custom errors
log = "0.4"
env_logger = "0.10"
comfy-table = "7"      # Report formatting

# Optional for RSS/Atom
rss = "2"
atom_syndication = "0.12"

[dev-dependencies]
tempfile = "3"         # For test databases
mockito = "1"          # HTTP mocking (optional)
```

### Error Handling Pattern
Use `anyhow::Result` for application errors and `thiserror` for domain errors:

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ConfigError {
    #[error("config file not found: {0}")]
    NotFound(String),
    
    #[error("invalid config: {0}")]
    InvalidYaml(String),
    
    #[error("feed '{feed}': missing field '{field}'")]
    MissingField { feed: String, field: String },
    
    #[error("feed '{feed}': invalid URL '{url}'")]
    InvalidUrl { feed: String, url: String },
}
```

### Test Structure Example
```rust
#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    
    #[test]
    fn test_config_missing_file() {
        let result = load_config("nonexistent.yaml");
        assert!(result.is_err());
        let err = result.unwrap_err();
        assert!(err.to_string().contains("config file not found"));
    }
    
    #[test]
    fn test_malformed_json() {
        let fixture = fs::read_to_string("../test-fixtures/malformed.json").unwrap();
        let result = parse_json("test", &fixture);
        assert!(result.is_err());
        assert!(result.unwrap_err().contains("malformed JSON"));
    }
    
    // ... 40+ more tests
}
```

---

## Execution Plan

1. **Backup current code** — already done (rust-backup-1770231104/)
2. **Refactor project structure** — add lib.rs, move tests/ out
3. **Implement error types** — thiserror enums for all scenarios
4. **Write failing tests first** — TDD approach for 16 scenarios
5. **Fix fetcher retry logic** — exponential backoff, Ctrl+C handling
6. **Fix parser edge cases** — coercion, missing fields, type mismatches
7. **Add integration tests** — use test-fixtures/
8. **Update README.md** — build/test instructions, limitations
9. **Run full test suite** — verify all pass
10. **Compare metrics** — LOC, test count, error coverage

---

## Reference Files

- **Spec:** `~/dev/feedpulse/SPEC.md` (full requirements)
- **Python impl:** `~/dev/feedpulse/python/` (gold standard)
- **Go impl:** `~/dev/feedpulse/go/` (alternative reference)
- **Test fixtures:** `~/dev/feedpulse/test-fixtures/`
- **Current Rust:** `~/dev/feedpulse/rust/` (needs rebuild)

---

## Expected Timeline

- **Total time:** ~2-4 hours (AI-assisted)
- **Test writing:** 40-50% of time
- **Error handling:** 30-40% of time
- **Feature completion:** 10-20% of time

---

## Deliverables

1. ✅ Working Rust implementation matching Python's comprehensiveness
2. ✅ Test suite ≥500 LOC with all tests passing
3. ✅ Updated EVALUATION.md with new Rust scores
4. ✅ Blog-ready comparison data (Python vs Go vs Rust — fair fight)

---

**Start command:** `cargo test` (should see ~40-50 tests)  
**Success metric:** All tests pass, EVALUATION.md shows Rust ≥20/25

Good luck! 🦀
