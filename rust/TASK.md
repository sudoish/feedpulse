# Task: Implement feedpulse CLI in Rust

## Spec Location
Read the full specification at: `/home/pacheco/dev/feedpulse/SPEC.md`

## Test Fixtures
Available at: `/home/pacheco/dev/feedpulse/test-fixtures/`

## Requirements
1. Read SPEC.md completely
2. Implement all 6 functional requirements
3. Handle all 16 error scenarios
4. Write unit tests for each module
5. Use these dependencies:
   - reqwest + tokio for async HTTP
   - serde + serde_yaml for config parsing
   - rusqlite for SQLite
   - clap for CLI
   - comfy-table or similar for table formatting

## Success Criteria
- All tests pass
- Can fetch from real feeds (../test-fixtures/valid-config.yaml)
- Handles all error scenarios gracefully
- Binary compiles without warnings

## Track These Metrics
- Compile errors caught
- Runtime errors encountered
- Lines of code
- Number of test cases
- AI hallucinations (wrong APIs, made-up functions)

START TIME: $(date '+%H:%M:%S')
