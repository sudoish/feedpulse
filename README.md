# feedpulse

**An experiment:** The same CLI tool built 3 times — in Python, Go, and Rust — entirely by AI coding agents, to compare how each language's type system and compiler affect AI-generated code quality.

## What is this?

`feedpulse` is a concurrent feed aggregator CLI. It fetches data from multiple sources (HackerNews, GitHub, Reddit, Lobsters), normalizes it, stores it in SQLite, and generates reports.

The tool itself is straightforward. The interesting part is *how each language handles the AI's mistakes*.

## The Experiment

1. Give the exact same [SPEC.md](SPEC.md) to an AI coding agent 3 times
2. Each time, target a different language: Python, Go, Rust
3. Measure everything: prompts needed, bugs caught by compiler vs runtime, correctness, time
4. Compare findings in [RESULTS.md](RESULTS.md)

## Structure

```
feedpulse/
├── SPEC.md              # The shared specification (all 3 implementations use this)
├── python/              # Python implementation
├── go/                  # Go implementation
├── rust/                # Rust implementation
├── test-fixtures/       # Shared test data for chaos testing
└── RESULTS.md           # Final comparison and findings
```

## Why?

Everyone assumes Python is "the AI language" because LLMs have the most training data for it. But what happens when there's no compiler to catch mistakes? Does Rust's strict type system actually help AI write *correct* code? Does Go's simplicity make AI more productive?

This experiment finds out.

## Blog Post

Results and analysis will be published at [sudoish.com](https://sudoish.com).
