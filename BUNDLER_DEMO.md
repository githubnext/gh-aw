# JavaScript Bundler Demonstration

This document demonstrates the JavaScript bundler's ability to eliminate code duplication by using modular imports.

## Problem: Code Duplication

Before the bundler, the `sanitizeContent` function was duplicated across multiple files:
- `pkg/workflow/js/collect_ndjson_output.cjs` (765 lines)
- `pkg/workflow/js/compute_text.cjs` (271 lines) 
- `pkg/workflow/js/sanitize_output.cjs` (225 lines)

Each file contained nearly identical implementations of sanitization logic (150+ lines of duplicated code).

## Solution: Modular Development with Bundler

### Step 1: Extract Shared Code

Create `pkg/workflow/js/lib/sanitize.cjs`:
```javascript
// @ts-check
/**
 * Shared sanitization utilities for GitHub Actions output
 */
function sanitizeContent(content, maxLength) {
  // ... implementation ...
}

module.exports = { sanitizeContent };
```

### Step 2: Use Require in Source Files

Modify source files to use the shared module:

**Before:**
```javascript
// compute_text.cjs (271 lines with duplicated sanitizeContent)
function sanitizeContent(content) {
  // 150+ lines of duplicated code
}

async function main() {
  const sanitized = sanitizeContent(text);
  // ...
}
```

**After (Source):**
```javascript
// compute_text.src.cjs (120 lines - much cleaner!)
const { sanitizeContent } = require('./lib/sanitize.cjs');

async function main() {
  const sanitized = sanitizeContent(text);
  // ...
}
```

### Step 3: Bundle Before Build

```bash
# Bundle the source file
./bundle-js pkg/workflow/js/compute_text.src.cjs pkg/workflow/js/compute_text.cjs

# The bundled file can now be embedded via go:embed
```

**Bundled Output:**
```javascript
// compute_text.cjs (bundled)
// === Inlined from ./lib/sanitize.cjs ===
function sanitizeContent(content, maxLength) {
  // ... complete implementation ...
}
// === End of ./lib/sanitize.cjs ===

async function main() {
  const sanitized = sanitizeContent(text);
  // ...
}
```

## Benefits

1. **DRY Principle**: Write the sanitization logic once, use it everywhere
2. **Maintainability**: Bug fixes only need to happen in one place
3. **Modularity**: Code is organized into logical, reusable modules
4. **Single File Output**: The bundler produces single-file outputs perfect for embedding
5. **No Runtime Dependencies**: Bundled code has no require() calls at runtime

## Example Usage

### Original File Sizes (with duplication)
- `collect_ndjson_output.cjs`: 765 lines
- `compute_text.cjs`: 271 lines
- `sanitize_output.cjs`: 225 lines
- **Total**: 1,261 lines

### After Refactoring (with bundler)
- `lib/sanitize.cjs`: 155 lines (shared)
- `collect_ndjson_output.src.cjs`: 610 lines (without duplicated sanitize)
- `compute_text.src.cjs`: 116 lines (without duplicated sanitize)
- `sanitize_output.src.cjs`: 70 lines (without duplicated sanitize)
- **Source Total**: 951 lines (310 lines saved!)
- **Bundled files**: Same size as before (but generated from clean source)

## Build Integration

### Current Workflow
1. Developer writes `.src.cjs` files with `require()` statements
2. Run `./bundle-js file.src.cjs file.cjs` to create bundled version
3. Bundled `.cjs` files are embedded via `go:embed`
4. Commit both source and bundled files to git

### Future Enhancement
Integrate bundling into `make build`:
```makefile
build: bundle-all-js
	go build -o gh-aw ./cmd/gh-aw

bundle-all-js:
	@for src in pkg/workflow/js/*.src.cjs; do \
		./bundle-js $$src $${src%.src.cjs}.cjs; \
	done
```

## Real-World Example

See `/tmp/bundler-demo/example_script.cjs` for a complete working example showing:
- Local require of shared module
- Bundling process
- Output verification
- Module.exports removal

## Conclusion

The JavaScript bundler enables:
- ✅ Modular code organization
- ✅ Elimination of code duplication  
- ✅ Single-file outputs for embedding
- ✅ Improved maintainability
- ✅ Better developer experience

All while maintaining the same embedded file structure required by the Go build process.
