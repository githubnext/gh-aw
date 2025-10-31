# JavaScript Bundler for gh-aw

This directory contains a micro JavaScript bundler that enables modular development of `.cjs` files with local `require()` statements.

## Overview

The bundler allows developers to write modular JavaScript code by splitting functionality into separate files, then bundles them into single files that can be embedded into the Go binary.

## Features

- **Local require detection**: Automatically finds `require('./...')` and `require("./...")` statements
- **Content inlining**: Recursively inlines the content of required modules
- **Export removal**: Removes `module.exports` and `exports.*` statements from inlined code
- **Circular dependency prevention**: Tracks processed files to avoid infinite loops
- **Clear boundaries**: Adds comments showing where inlined code starts and ends

## Usage

### Building the Bundler

```bash
make bundle-js
```

This creates the `bundle-js` executable in the project root.

### Bundling a File

```bash
# Bundle to a specific output file
./bundle-js input.cjs output.cjs

# Bundle to stdout
./bundle-js input.cjs
```

### Example

**Input files:**

`lib/helper.cjs`:
```javascript
function validateInput(value) {
  if (!value) {
    return { valid: false, error: "Value is required" };
  }
  return { valid: true };
}

module.exports = { validateInput };
```

`main.cjs`:
```javascript
const { validateInput } = require('./lib/helper.cjs');

async function main() {
  const result = validateInput(userInput);
  console.log(result);
}

await main();
```

**Bundling:**

```bash
./bundle-js main.cjs main.bundled.cjs
```

**Output (`main.bundled.cjs`):**

```javascript
// === Inlined from ./lib/helper.cjs ===
function validateInput(value) {
  if (!value) {
    return { valid: false, error: "Value is required" };
  }
  return { valid: true };
}

// === End of ./lib/helper.cjs ===

async function main() {
  const result = validateInput(userInput);
  console.log(result);
}

await main();
```

## Shared Modules

The `pkg/workflow/js/lib/` directory contains shared modules that can be required by multiple scripts:

- `lib/sanitize.cjs`: Sanitization utilities for GitHub Actions output

## Development Workflow

1. **Write modular code** with local requires
2. **Test your modules** individually
3. **Bundle before embedding** into Go binary
4. **Commit bundled output** to repository

## Integration with Build Process

The bundler is designed to run as a pre-build step. Future enhancements may integrate it directly into the `make build` process to automatically bundle files before embedding.

## Testing

The bundler includes comprehensive tests in `pkg/workflow/bundler_test.go`:

```bash
go test -v ./pkg/workflow -run TestBundle
```

## Implementation Details

- **Language**: Go
- **Source**: `pkg/workflow/bundler.go`
- **CLI**: `cmd/bundle-js/main.go`
- **Tests**: `pkg/workflow/bundler_test.go`

## Limitations

- Only supports CommonJS-style `require()` (not ES6 `import`)
- Only bundles local files (paths starting with `./` or `../`)
- Does not bundle Node.js built-in modules or `node_modules` dependencies
- Does not handle dynamic requires (e.g., `require(variableName)`)

## Future Enhancements

- [ ] Automatic bundling as part of `make build`
- [ ] Source map generation
- [ ] Minification option
- [ ] Watch mode for development
- [ ] Support for `.js` files in addition to `.cjs`
