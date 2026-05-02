# AW Harness

> ⚠️ **EXPERIMENTAL** — This package is experimental and subject to breaking changes without notice. Do not use in production workflows.

The AW Harness (`aw_harness.cjs`) is a Node.js execution engine for the `engine: aw` mode of GitHub Agentic Workflows (gh-aw). It runs a single Pi `AgentSession` with budget management, steering, and observability.

## Architecture

The harness is a Pi SDK application: it creates a single `AgentSession` and runs a compiled prompt through it, with gh-aw Pi extensions providing budget gating, steering, and observability. See [`specs/aw-harness.md`](../specs/aw-harness.md) for the full specification.

## Status

| Feature | Status |
|---------|--------|
| Project scaffolding | ✅ |
| Provider setup extension | ✅ |
| Cost tracker extension | ✅ |
| Steering extension | ✅ |
| Repair extension | ✅ |
| Observability extension | ✅ |
| Loader (config.json + prompt.txt) | ✅ |
| User extension loading | ✅ |
| Context assembly | ✅ |
| Entry point | ✅ |
| gh-aw compiler integration | 🔲 |

## Development

```bash
# Install dependencies
make deps-aw-harness

# Build (bundles to actions/setup/js/aw_harness.cjs)
make aw-harness

# Run tests
cd aw-harness && npm test

# Typecheck only
cd aw-harness && npm run typecheck

# Lint
make lint-aw-harness
```

## Project Structure

```
aw-harness/
├── package.json           # Pinned dependencies
├── tsconfig.json          # TypeScript config (ES2024, noEmit)
├── vitest.config.ts       # Vitest test runner config
├── build.ts               # esbuild → dist/aw_harness.cjs
├── src/
│   ├── index.ts           # Entry point (EXPERIMENTAL)
│   ├── types.ts           # Shared config types
│   ├── loader.ts          # config.json + prompt.txt reader
│   ├── context.ts         # Prompt assembly with imports
│   ├── user-extensions.ts # Load user-declared extensions
│   └── extensions/
│       ├── provider-setup.ts   # Register LLM providers from env vars
│       ├── cost-tracker.ts     # Budget gates via turn_end events
│       ├── steering.ts         # Time/budget pressure messages
│       ├── repair.ts           # Broken session recovery
│       └── observability.ts    # JSONL events + OTel + step summary
├── test/
│   ├── loader.test.ts
│   ├── context.test.ts
│   ├── user-extensions.test.ts
│   └── extensions/
│       ├── provider-setup.test.ts
│       ├── cost-tracker.test.ts
│       ├── steering.test.ts
│       ├── repair.test.ts
│       └── observability.test.ts
└── dist/
    └── aw_harness.cjs     # Copied to actions/setup/js/aw_harness.cjs
```

## Invocation

The gh-aw compiler pre-processes the workflow markdown and provides the harness with:
- `config.json` — parsed harness configuration
- `prompt.txt` — extracted prompt body

```
node aw_harness.cjs --config <config-path> --prompt <prompt-path>
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Prompt completed successfully |
| `1` | Session failed or budget exceeded |
| `2` | Invocation error (missing arguments, unreadable files) |
