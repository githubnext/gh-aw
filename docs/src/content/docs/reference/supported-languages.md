---
title: Supported Languages & Ecosystems
description: Language ecosystem identifiers for configuring network access in agentic workflows
sidebar:
  order: 1350
---

Agentic workflows run inside an Ubuntu/Linux sandbox. Each programming language has a corresponding **ecosystem identifier** that grants the workflow access to that language's package registry and toolchain domains. Set these identifiers in the [`network.allowed`](/gh-aw/reference/network/) field of your workflow frontmatter.

## Language Ecosystem Identifiers

| Language | Ecosystem Identifier | Package Manager |
|----------|---------------------|-----------------|
| Python | `python` | pip, conda |
| JavaScript / TypeScript | `node` | npm, yarn, pnpm |
| Java | `java` | Maven, Gradle |
| Go | `go` | Go modules |
| Rust | `rust` | Cargo |
| C# / .NET | `dotnet` | NuGet |
| Ruby | `ruby` | Bundler, RubyGems |
| PHP | `php` | Composer |
| Swift | `swift` | SwiftPM (Linux only) |
| Kotlin | `kotlin` + `java` | Gradle |
| Dart | `dart` | pub |
| C / C++ | `defaults` | System toolchain (gcc, cmake) |

> [!NOTE]
> Swift projects that depend on Apple-only frameworks (UIKit, AppKit, SwiftUI on macOS) are not supported — the sandbox runs Ubuntu Linux.

## Infrastructure Ecosystems

These identifiers are not language-specific but pair with any language workflow:

| Identifier | Use for |
|------------|---------|
| `defaults` | Basic infrastructure: certificates, JSON schema, Ubuntu mirrors. This is the default when `network:` is not specified, and is recommended as the starting baseline for most workflows. |
| `github` | GitHub domains (`github.com`, `raw.githubusercontent.com`, etc.) |
| `containers` | Docker Hub, GitHub Container Registry, Quay, GCR |
| `linux-distros` | Debian, Ubuntu, Alpine package repositories (`apt`, `apk`) |

## Configuration Examples

### Single language

```aw wrap
---
network:
  allowed:
    - defaults
    - python
---
```

### JVM family (Java + Kotlin)

```aw wrap
---
network:
  allowed:
    - defaults
    - java
    - kotlin
---
```

### Multi-language

```aw wrap
---
network:
  allowed:
    - defaults
    - node
    - python
    - containers
    - github
---
```

## Less Common Languages

Additional language ecosystems are available for less common languages including Elixir, Haskell, Julia, Perl, OCaml, Deno, and Terraform. See the [Ecosystem Identifiers table](/gh-aw/reference/network/#ecosystem-identifiers) in the Network Permissions reference for the most up-to-date list of supported identifiers.

## Related Documentation

- [Network Permissions](/gh-aw/reference/network/) — Network configuration reference and ecosystem identifiers table
- [Network Configuration Guide](/gh-aw/guides/network-configuration/) — Practical patterns and troubleshooting
- [Sandbox](/gh-aw/reference/sandbox/) — Sandbox environment details
