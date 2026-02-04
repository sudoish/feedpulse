# Test Fixtures

Shared test data used to chaos-test all three implementations equally.

| File | Purpose |
|---|---|
| `valid-config.yaml` | Working config for normal runs |
| `invalid-config-missing-url.yaml` | Config with missing required field |
| `invalid-config-bad-yaml.yaml` | Config with wrong types and invalid values |
| `malformed-json-response.json` | Intentionally broken JSON (truncated, wrong types, nulls) |
| `empty-response.json` | Empty response body |
| `unicode-chaos.json` | Unicode edge cases (emoji, RTL, null bytes, whitespace) |
