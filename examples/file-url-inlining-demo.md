---
description: Example workflow demonstrating file/URL inlining syntax
on: workflow_dispatch
engine: copilot
permissions: read
---

# File/URL Inlining Demo

This workflow demonstrates the new inline syntax for including file and URL content.

## 1. Full File Inlining

The full content of the LICENSE file:

```
@LICENSE
```

## 2. Line Range Inlining

Here are lines 1-5 from the README.md file:

```
@README.md:1-5
```

## 3. Code Snippet from Source

Let's look at the main function (lines 10-30):

```go
@cmd/gh-aw/main.go:10-30
```

## 4. Remote Content (Commented Out)

<!-- 
Note: URL inlining would work like this, but commented out to avoid network requests:

@https://raw.githubusercontent.com/githubnext/gh-aw/main/README.md
-->

## 5. Contact Information

For questions, email: support@example.com (this email address is NOT processed as a file reference)

## Task

Analyze the included content and provide a summary of what you see.
