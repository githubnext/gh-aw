# stringutil Package

The `stringutil` package provides utility functions for working with strings. It is organized into focused sub-files covering ANSI stripping, identifier normalization, sanitization, URL utilities, and PAT (Personal Access Token) validation.

## Overview

| Sub-file | Functions |
|----------|-----------|
| `stringutil.go` | General string helpers |
| `ansi.go` | ANSI escape-code stripping |
| `identifiers.go` | Workflow name and path normalization |
| `sanitize.go` | Security-sensitive string sanitization |
| `urls.go` | URL normalization and domain extraction |
| `pat_validation.go` | GitHub PAT classification and validation |

## General Utilities (`stringutil.go`)

### `Truncate(s string, maxLen int) string`

Truncates `s` to at most `maxLen` characters, appending `"..."` when truncation occurs. For `maxLen ≤ 3` the string is truncated without ellipsis.

```go
stringutil.Truncate("hello world", 8) // "hello..."
stringutil.Truncate("hi", 8)          // "hi"
```

### `NormalizeWhitespace(content string) string`

Collapses multiple consecutive whitespace characters (spaces, tabs, newlines) into a single space and trims leading/trailing whitespace.

### `ParseVersionValue(version any) string`

Converts a `any`-typed version value (typically from YAML parsing, which may produce `int`, `float64`, or `string`) into a string. Returns an empty string for nil.

```go
stringutil.ParseVersionValue("20")    // "20"
stringutil.ParseVersionValue(20)      // "20"
stringutil.ParseVersionValue(20.0)    // "20"
```

### `IsPositiveInteger(s string) bool`

Returns `true` if `s` is a non-empty string containing only digit characters (`0–9`).

## ANSI Escape Code Stripping (`ansi.go`)

### `StripANSI(s string) string`

Removes all ANSI/VT100 escape sequences from `s`. Handles CSI sequences (e.g. `\x1b[31m` for colors) and other ESC-prefixed sequences. This function is used before writing text into YAML files to prevent invisible characters from corrupting workflow output.

```go
colored := "\x1b[32mSuccess\x1b[0m"
plain := stringutil.StripANSI(colored) // "Success"
```

## Identifier Normalization (`identifiers.go`)

### `NormalizeWorkflowName(name string) string`

Removes `.md` and `.lock.yml` extensions from workflow names, returning the bare workflow identifier.

```go
stringutil.NormalizeWorkflowName("weekly-research.md")       // "weekly-research"
stringutil.NormalizeWorkflowName("weekly-research.lock.yml") // "weekly-research"
stringutil.NormalizeWorkflowName("weekly-research")          // "weekly-research"
```

### `NormalizeSafeOutputIdentifier(identifier string) string`

Converts dashes to underscores in safe-output identifiers, normalizing the user-facing `dash-separated` format to the internal `underscore_separated` format.

```go
stringutil.NormalizeSafeOutputIdentifier("create-issue") // "create_issue"
```

### `MarkdownToLockFile(mdPath string) string`

Converts a workflow markdown path (`.md`) to its compiled lock file path (`.lock.yml`). Returns the path unchanged if it already ends with `.lock.yml`.

```go
stringutil.MarkdownToLockFile(".github/workflows/test.md")
// → ".github/workflows/test.lock.yml"
```

### `LockFileToMarkdown(lockPath string) string`

Converts a compiled lock file path (`.lock.yml`) back to its markdown source path (`.md`). Returns the path unchanged if it already ends with `.md`.

```go
stringutil.LockFileToMarkdown(".github/workflows/test.lock.yml")
// → ".github/workflows/test.md"
```

## Sanitization (`sanitize.go`)

These functions remove sensitive information to prevent accidental leakage in logs or error messages.

### `SanitizeErrorMessage(message string) string`

Redacts potential secret key names from error messages. Matches uppercase `SNAKE_CASE` identifiers (e.g. `MY_SECRET_KEY`, `API_TOKEN`) and PascalCase identifiers ending with security-related suffixes (e.g. `GitHubToken`, `ApiKey`). Common GitHub Actions workflow keywords (`GITHUB`, `RUNNER`, `WORKFLOW`, etc.) are excluded from redaction.

```go
stringutil.SanitizeErrorMessage("Error: MY_SECRET_TOKEN is invalid")
// → "Error: [REDACTED] is invalid"
```

### `SanitizeParameterName(name string) string`

Sanitizes a parameter name for use as a GitHub Actions output or environment variable name. Replaces non-alphanumeric characters with underscores.

### `SanitizePythonVariableName(name string) string`

Sanitizes a string for use as a Python variable name. Similar to `SanitizeParameterName` but follows Python identifier rules.

### `SanitizeToolID(toolID string) string`

Sanitizes a tool identifier for safe use in generated code. Replaces characters that are not valid in identifiers with underscores.

### `SanitizeForFilename(slug string) string`

Converts a string into a filesystem-safe filename by lowercasing and replacing non-alphanumeric characters with hyphens.

## URL Utilities (`urls.go`)

### `NormalizeGitHubHostURL(rawHostURL string) string`

Normalizes a GitHub host URL by ensuring it has an `https://` scheme and no trailing slash. Accepts bare hostnames, URLs with or without a scheme, and URLs with trailing slashes.

```go
stringutil.NormalizeGitHubHostURL("github.example.com")        // "https://github.example.com"
stringutil.NormalizeGitHubHostURL("https://github.com/")       // "https://github.com"
```

### `ExtractDomainFromURL(urlStr string) string`

Extracts the hostname (without port) from a URL string. Falls back to simple string parsing when `url.Parse` cannot handle the input.

```go
stringutil.ExtractDomainFromURL("https://api.github.com/repos") // "api.github.com"
```

## PAT Validation (`pat_validation.go`)

### `PATType`

A string type representing the category of a GitHub Personal Access Token.

| Constant | Value | Prefix |
|----------|-------|--------|
| `PATTypeFineGrained` | `"fine-grained"` | `github_pat_` |
| `PATTypeClassic` | `"classic"` | `ghp_` |
| `PATTypeOAuth` | `"oauth"` | `gho_` |
| `PATTypeUnknown` | `"unknown"` | (other) |

Methods: `String() string`, `IsFineGrained() bool`, `IsValid() bool`

### `ClassifyPAT(token string) PATType`

Determines the token type from its prefix.

### `ValidateCopilotPAT(token string) error`

Returns `nil` if the token is a fine-grained PAT; returns an actionable error message with a link to create the correct token type otherwise.

```go
if err := stringutil.ValidateCopilotPAT(token); err != nil {
    fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
}
```

### `GetPATTypeDescription(token string) string`

Returns a human-readable description of the token type (e.g. `"fine-grained personal access token"`).

## Design Notes

- All debug output uses namespace-prefixed loggers (`stringutil:identifiers`, `stringutil:sanitize`, `stringutil:urls`, `stringutil:pat_validation`) and is only emitted when `DEBUG=stringutil:*`.
- `SanitizeErrorMessage` is intentionally conservative: it excludes common GitHub Actions keywords to avoid over-redacting legitimate error messages.
- `StripANSI` handles both CSI sequences (`ESC[`) and other ESC-prefixed sequences to cover the full range of ANSI escape codes found in terminal output.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
