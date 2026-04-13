# fileutil Package

The `fileutil` package provides utility functions for safe file path validation and common file operations.

## Overview

This package focuses on security-conscious file handling: path validation, boundary enforcement, and straightforward file/directory operations. It also provides a cross-platform tar extraction helper.

## Functions

### Path Validation

#### `ValidateAbsolutePath(path string) (string, error)`

Validates that a file path is absolute and safe to use. The function:
1. Rejects empty paths.
2. Cleans the path with `filepath.Clean` to normalize `.` and `..` components.
3. Verifies the cleaned path is absolute.

Returns the cleaned absolute path on success, or an error otherwise. Use this before any file operation to defend against relative path traversal.

```go
import "github.com/github/gh-aw/pkg/fileutil"

cleanPath, err := fileutil.ValidateAbsolutePath(userInput)
if err != nil {
    return fmt.Errorf("invalid path: %w", err)
}
content, err := os.ReadFile(cleanPath)
```

#### `MustBeWithin(base, candidate string) error`

Checks that `candidate` is located within the `base` directory tree. Both paths are resolved through `filepath.EvalSymlinks` (falling back to `filepath.Abs` for paths that do not yet exist on disk) before comparison, preventing both `..` traversal and symlink escapes.

```go
if err := fileutil.MustBeWithin("/workspace", outputPath); err != nil {
    return fmt.Errorf("output path escapes workspace: %w", err)
}
```

### File and Directory Checks

#### `FileExists(path string) bool`

Returns `true` if `path` exists and is a regular file (not a directory).

#### `DirExists(path string) bool`

Returns `true` if `path` exists and is a directory.

#### `IsDirEmpty(path string) bool`

Returns `true` if the directory at `path` contains no entries. Returns `true` if the directory cannot be read.

### File Operations

#### `CopyFile(src, dst string) error`

Copies the file at `src` to `dst` using buffered I/O. Creates `dst` if it does not exist; truncates it if it does. Calls `Sync` on the destination before closing.

```go
if err := fileutil.CopyFile("source.txt", "destination.txt"); err != nil {
    return fmt.Errorf("copy failed: %w", err)
}
```

### Archive Operations

#### `ExtractFileFromTar(data []byte, path string) ([]byte, error)`

Extracts a single file by `path` from a tar archive stored in `data`. Uses Go's `archive/tar` for cross-platform compatibility.

Security guarantees:
- `path` must be a local, relative path (no `..` components or absolute paths).
- Individual tar entries with unsafe names are skipped, not extracted.

```go
tarData, _ := io.ReadAll(response.Body)
content, err := fileutil.ExtractFileFromTar(tarData, "bin/gh")
if err != nil {
    return fmt.Errorf("binary not found in release archive: %w", err)
}
```

## Design Notes

- All debug output uses `logger.New("fileutil:fileutil")` and `logger.New("fileutil:tar")` and is only emitted when `DEBUG=fileutil:*`.
- `MustBeWithin` resolves symlinks before comparison, providing defence-in-depth against symlink attacks in addition to the `..` checking that `ValidateAbsolutePath` provides.
- `ExtractFileFromTar` rejects path-traversal payloads in both the caller-supplied path and in tar entry names using `filepath.IsLocal`.
