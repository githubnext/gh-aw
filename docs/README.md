# gh-aw Documentation Workspace

This directory contains the Astro/Starlight site for GitHub Agentic Workflows docs.

## Operator Quick Start

Run all commands from the repository root:

```bash
make deps-docs
make dev-docs
```

Then open `http://localhost:4321`.

## Build and Preview

```bash
make build-docs
make preview-docs
```

`build-docs` writes the static site to `docs/dist`.

## Where To Edit Content

- Main docs pages: `docs/src/content/docs/`
- Blog pages: `docs/src/content/docs/blog/`
- Public static assets: `docs/public/`
- Starlight config: `docs/astro.config.mjs`

## Suggested Validation Before PR

From repository root:

```bash
make lint
make test
make build-docs
```

For a docs-only change, this keeps checks aligned with repo standards while confirming docs compile.

## Troubleshooting

- Reinstall docs dependencies: `make deps-docs`
- Remove docs build artifacts: `make clean-docs`
- If Astro dev fails after dependency updates, rerun `make deps-docs` and then `make dev-docs`
