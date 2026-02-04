# Which Programming Language Should AI Write Your Code In?

*I gave an AI agent the exact same project spec three times — once for Python, once for Rust. Here's what broke, what the compiler caught, and what it means for the future of AI-generated code.*

---

Everyone assumes Python is the AI coding language. It has the most training data, the richest ecosystem, and LLMs write it fluently. But here's a question nobody's asking: **what catches the AI's mistakes?**

When a human writes buggy Python, they debug it. When an AI writes buggy Python, it runs — and fails at runtime, in production, in your users' hands. What if the language itself could act as a second reviewer?

I ran an experiment to find out.

## The Setup

I built a tool called [feedpulse](https://github.com/sudoish/feedpulse) — a concurrent feed aggregator CLI that fetches data from HackerNews, GitHub, Reddit, and Lobsters, normalizes it into a unified schema, stores it in SQLite, and generates reports.

It's not a toy project. It has:
- **Concurrent HTTP fetching** with semaphore-controlled parallelism
- **Exponential backoff retries** with jitter
- **YAML config validation** with specific, human-readable error messages
- **SQLite storage** with deduplication and transaction safety
- **16 explicit error handling scenarios** — from malformed JSON to Ctrl+C graceful shutdown
- **A full CLI** with fetch, report, and sources commands

I wrote a detailed [SPEC.md](https://github.com/sudoish/feedpulse/blob/main/SPEC.md) covering every requirement, every error scenario, every edge case. Then I gave it to an AI coding agent (Claude Sonnet 4.5 via [OpenClaw](https://github.com/openclaw/openclaw)) and said: build this in Python. Then: build it again in Rust.

Same spec. Same AI. Same instructions. Different languages.

## The Results

| Metric | Python | Rust |
|---|---|---|
| **Development time** | ~8 min | ~24 min |
| **Lines of code** | 1,307 | 1,627 |
| **Tests written** | 43 | 16 |
| **Compile-time errors caught** | 0 | **2** |
| **Runtime errors** | **1** | 0 |
| **AI hallucinations** | 0 | 0 |
| **Error scenarios handled** | 16/16 | 16/16 |

The headline: **Python was 3x faster to develop. Rust caught every bug before the code ever ran.**

## What the Compiler Caught

In the Rust implementation, the compiler caught two errors before the code ever executed:

**1. Type mismatch in the retry logic.** The AI tried to use a `u64` where an `i64` was needed for the random jitter calculation. In Python, this would have silently worked — until it hit an edge case with negative numbers. In Rust, the compiler said no.

**2. Missing serialization trait.** A report data structure needed `Serialize` for JSON output. The AI forgot to derive it. In Python, this would have worked fine until someone actually tried to serialize it — then you'd get a runtime `TypeError`. Rust caught it at compile time.

Both fixes took less than a minute. But here's the thing: **in Python, you might not discover these bugs until production.** The type mismatch would only surface with specific input values. The serialization error would only appear when a user tried the `--format json` flag on the report command.

## What Python Got Right

Let's be fair — Python destroyed Rust on development speed:

- **8 minutes** vs 24 minutes to a working implementation
- **43 tests** written naturally (pytest makes testing effortless)
- The ecosystem is incredible: aiohttp, Click, Rich, PyYAML all worked perfectly
- Zero friction from start to finish

The AI wrote Python like a fluent native speaker. It knew every library, every pattern, every idiom. The code was clean, readable, and well-structured. If you need something working *now*, Python is hard to beat.

## What Rust Got Right

But Rust's value showed up in the places Python can't reach:

**The compiler as code reviewer.** Every error path was enforced. You can't forget to handle a `Result`. You can't accidentally pass the wrong type. The `match` statement forces you to handle every variant of an enum. The AI couldn't cut corners even if it tried.

**Zero runtime surprises.** After fixing the two compile-time errors, the Rust implementation worked perfectly on the first run. No `TypeError` at 3am. No `AttributeError` on an edge case. If it compiles, it's structurally sound.

**Performance as a free bonus.** The Rust binary starts in <1ms, uses <10MB of RAM, and is a single 7.4MB file you can deploy anywhere. No virtualenv, no `pip install`, no dependency hell.

## The Meta-Observation

Here's what surprised me most: **the AI didn't struggle with Rust's complexity.**

I expected borrow checker fights. I expected lifetime annotation nightmares. I expected the AI to thrash for an hour trying to satisfy Rust's demands. None of that happened. The AI wrote idiomatic async Rust with tokio, used proper error handling with `anyhow` and `thiserror`, and structured the modules cleanly.

The two errors it made were *exactly* the kind of errors a human would make — and the kind that slip through in dynamically typed languages. Rust's whole value proposition is catching those errors. And it delivered.

The actual challenge wasn't the language — it was **tooling reliability**. My first two Rust attempts failed due to file corruption in the AI agent's workspace (files reverting to old versions during edits). That's an infrastructure problem, not a Rust problem. But it's worth noting: AI tooling is still immature, and more complex languages expose more tooling edge cases.

## What This Means

If you're building AI-powered development workflows, the language choice matters more than you think:

**For prototypes and scripts:** Python is unbeatable. The AI writes it fast, the ecosystem is rich, and the code is readable. The risk of runtime errors is acceptable when you're iterating quickly.

**For production systems:** Rust (or Go, or any statically-typed, compiled language) gives you a safety net that no amount of testing can fully replace. The compiler catches the mistakes the AI *will* make — the ones that manifest as 3am pages, not test failures.

**The counterintuitive takeaway:** As AI writes more of our code, **stricter languages become more valuable, not less.** We're not the ones fighting the borrow checker anymore — the AI is. And it handles it fine. What we get is code that's guaranteed to be structurally correct before it ever runs.

## Try It Yourself

The full project is open source: [github.com/sudoish/feedpulse](https://github.com/sudoish/feedpulse)

Both implementations are there with detailed build logs tracking every step, every error, every decision the AI made. The Go implementation is coming next.

---

*This experiment was conducted using [OpenClaw](https://github.com/openclaw/openclaw) with Claude Sonnet 4.5 as the coding agent. The spec, test fixtures, and results are all in the repo.*
