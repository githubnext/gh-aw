# pkg/workflow/js and pkg/workflow/sh - Build Artifacts

**⚠️ DO NOT EDIT FILES IN THESE DIRECTORIES**

These directories contain build artifacts that are automatically generated during the build process.

## Source of Truth

The authoritative source files are located in:
- `actions/setup/js/` - JavaScript source files and tests
- `actions/setup/sh/` - Shell script source files

## Build Process

During `make build`, the `sync-scripts` target automatically copies files from `actions/setup/` to these directories:
- `actions/setup/js/` → `pkg/workflow/js/`
- `actions/setup/sh/` → `pkg/workflow/sh/`

The Go code then uses `//go:embed` directives to embed these files into the binary.

## Why This Structure?

Go's `//go:embed` directive cannot access files outside the package directory (cannot use `../`). To work around this constraint while maintaining a single source of truth:
1. Source files live in `actions/setup/` (committed to git)
2. Build artifacts are synced to `pkg/workflow/` (not tracked in git)
3. Go embeds from `pkg/workflow/` during compilation

## Making Changes

To modify JavaScript or shell scripts:
1. Edit files in `actions/setup/js/` or `actions/setup/sh/`
2. Run tests: `cd actions/setup && npm test`
3. Run build: `make build` (this will sync your changes)

## Git Tracking

- `pkg/workflow/js/` and `pkg/workflow/sh/` are in `.gitignore`
- These directories are generated during build and should not be committed
- The source files in `actions/setup/` are tracked and committed
