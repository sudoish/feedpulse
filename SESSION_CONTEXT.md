# feedpulse — Session Context

## Project Overview
An experiment comparing AI-generated code quality across Python, Go, and Rust.
Same spec, same AI agent, three languages. Measuring correctness, errors, and how well each language's type system catches AI mistakes.

## Repo Location
`/home/pacheco/dev/feedpulse/`

## Spec
Full spec at `SPEC.md` — concurrent feed aggregator CLI that fetches from HackerNews, GitHub, Reddit, Lobsters. Normalizes data, stores in SQLite, generates reports. 16 error handling scenarios defined.

## Agent Setup
- **Jarvis** (main, Opus) — orchestrator, delegates work
- **Jarvinho** (coding agent, Sonnet) — implements in isolation
- Config: `agents.list` in openclaw.json, `subagents.allowAgents: ["jarvinho"]` on main
- Jarvinho workspace: `~/.openclaw/workspace-jarvinho`

## Python Implementation — COMPLETE ✅
- **Location:** `/home/pacheco/dev/feedpulse/python/`
- **Agent time:** ~8 minutes
- **Lines of code:** 1,307 (7 modules)
- **Tests:** 43/43 passing
- **Error scenarios:** 16/16 handled
- **Runtime errors:** 1 (database corruption detection — fixed)
- **AI hallucinations:** 0
- **Compile/lint errors:** 0 (Python is interpreted)
- **Libraries:** aiohttp, Click, Rich, PyYAML, pytest
- **Live test:** 110 items fetched from 4 real feeds
- **Chaos test:** Passed (malformed JSON, bad configs, unicode)
- **BUILD_LOG.md** has full tracking details

## Rust Implementation — NEXT
- Goes in: `/home/pacheco/dev/feedpulse/rust/`
- Suggested libs: reqwest + tokio, serde + serde_yaml, rusqlite, clap, comfy-table
- Key hypothesis: Rust's compiler should catch more errors before runtime, but may require more iterations to satisfy the borrow checker

## Go Implementation — TODO
- Goes in: `/home/pacheco/dev/feedpulse/go/`

## Blog
Results will be published at sudoish.com. The angle: "I let AI write the same project in 3 languages and measured what broke."

## Results Template
`RESULTS.md` has the comparison table ready to fill in.
