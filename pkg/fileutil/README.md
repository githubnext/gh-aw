# fileutil Package

The `fileutil` package provides utility functions for safe file path validation and common file operations.

## Overview

This package focuses on security-conscious file handling: path validation, boundary enforcement, and straightforward file/directory operations. It also provides a cross-platform tar extraction helper.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `ValidateAbsolutePath` | `func(path string) (string, error)` | Validates that a file path is absolute and safe; rejects empty paths, cleans with `filepath.Clean`, and verifies the result is absolute |
| `ValidatePathWithinBase` | `func(base, candidate string) error` | Checks that `candidate` is located within the `base` directory tree; resolves symlinks before comparison to prevent traversal and symlink escapes |
| `FileExists` | `func(path string) bool` | Returns `true` if `path` exists and is a regular file (not a directory) |
| `DirExists` | `func(path string) bool` | Returns `true` if `path` exists and is a directory |
| `IsDirEmpty` | `func(path string) bool` | Returns `true` if the directory at `path` contains no entries; also returns `true` if the directory cannot be read |
| `CopyFile` | `func(src, dst string) error` | Copies the file at `src` to `dst` using buffered I/O; calls `Sync` on the destination before closing |
| `EnsureParentDir` | `func(path string, perm os.FileMode) error` | Ensures the parent directory of `path` exists, creating it recursively with the given permissions; returns an error for empty paths |
| `ExtractFileFromTar` | `func(data []byte, path string) ([]byte, error)` | Extracts a single file by `path` from a tar archive; rejects unsafe entry names (absolute or `..`-containing paths) |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/fileutil"

// Validate and clean a user-supplied path
cleanPath, err := fileutil.ValidateAbsolutePath(userInput)
if err != nil {
    return fmt.Errorf("invalid path: %w", err)
}

// Ensure output path stays within workspace
if err := fileutil.ValidatePathWithinBase("/workspace", outputPath); err != nil {
    return fmt.Errorf("output path escapes workspace: %w", err)
}

// Copy a file
if err := fileutil.CopyFile("source.txt", "destination.txt"); err != nil {
    return fmt.Errorf("copy failed: %w", err)
}
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — debug logging

## Design Notes

- All debug output uses `logger.New("fileutil:fileutil")` and `logger.New("fileutil:tar")` and is only emitted when `DEBUG=fileutil:*`.
- `ValidatePathWithinBase` resolves symlinks before comparison, providing defence-in-depth against symlink attacks in addition to the `..` checking that `ValidateAbsolutePath` provides.
- `ExtractFileFromTar` rejects path-traversal payloads in both the caller-supplied path and in tar entry names using `filepath.IsLocal`.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
