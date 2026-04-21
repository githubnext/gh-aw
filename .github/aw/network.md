---
description: Network access configuration reference for gh-aw workflows — valid ecosystem identifiers, domain patterns, and common mistakes to avoid.
---

# Network Access Configuration

Use the `network` frontmatter field to control which domains an AI engine can reach during a workflow run. All traffic is enforced by the Agent Workflow Firewall (AWF).

## Quick Reference

```yaml
# Shorthand — use default infrastructure domains only
network: defaults

# Custom — allow defaults plus package registries for a Node.js project
network:
  allowed:
    - defaults
    - node

# Custom — allow specific external APIs
network:
  allowed:
    - defaults
    - api.example.com
    - "*.trusted-partner.com"

# No network access
network:
  allowed: []
```

## Valid Values for `network.allowed`

Each entry in `network.allowed` must be one of:

| Type | Examples | Notes |
|---|---|---|
| **Ecosystem identifier** | `defaults`, `node`, `python` | Expands to a curated list of domains for that runtime/tool |
| **Exact domain** | `api.example.com`, `registry.npmjs.org` | Must be a fully-qualified domain name (FQDN) |
| **Wildcard subdomain** | `*.example.com` | Matches `sub.example.com`, `deep.nested.example.com`, and `example.com` itself |

> ⚠️ **Bare shorthands like `npm`, `pypi`, or `localhost` are NOT valid** unless they are listed in the ecosystem identifiers table below. Using an unrecognised single-word entry causes a **compile-time error**. Use ecosystem identifiers (`node`, `python`) or explicit FQDNs (`registry.npmjs.org`, `pypi.org`) instead.

## Ecosystem Identifiers

These keywords expand to curated lists of domains maintained by gh-aw:

| Identifier | Runtime / Tool | Key Domains Enabled |
|---|---|---|
| `defaults` | Basic infrastructure | Certificate authorities, Ubuntu package verification, JSON schema |
| `github` | GitHub domains | `*.githubusercontent.com`, `codeload.github.com`, `docs.github.com` |
| `github-actions` | GitHub Actions artifacts | Azure Blob storage for action caches and artifacts |
| `node` | npm / yarn / pnpm | `registry.npmjs.org`, `npmjs.com`, `yarnpkg.com` |
| `python` | pip / PyPI / conda | `pypi.org`, `files.pythonhosted.org`, `pip.pypa.io` |
| `go` | Go modules | `proxy.golang.org`, `sum.golang.org`, `go.dev` |
| `dotnet` | NuGet / .NET | `api.nuget.org`, `nuget.org`, `dotnet.microsoft.com` |
| `java` | Maven / Gradle | `repo1.maven.org`, `plugins.gradle.org`, `jdk.java.net` |
| `ruby` | Bundler / RubyGems | `rubygems.org`, `api.rubygems.org` |
| `rust` | Cargo | `crates.io`, `index.crates.io`, `static.crates.io`, `sh.rustup.rs` |
| `swift` | Swift Package Manager | `swift.org`, `cocoapods.org` |
| `php` | Composer / Packagist | `packagist.org`, `repo.packagist.org`, `getcomposer.org` |
| `dart` | pub.dev | `pub.dev`, `pub.dartlang.org` |
| `haskell` | Hackage / GHCup | `*.hackage.haskell.org`, `get-ghcup.haskell.org` |
| `perl` | CPAN | `cpan.org`, `metacpan.org` |
| `containers` | Docker / GHCR | `ghcr.io`, `registry.hub.docker.com`, `*.docker.io` |
| `playwright` | Playwright browsers | `playwright.download.prss.microsoft.com`, `cdn.playwright.dev` |
| `linux-distros` | apt / yum / apk | `deb.debian.org`, `security.debian.org`, Ubuntu/Alpine mirrors |
| `terraform` | HashiCorp | `releases.hashicorp.com`, `registry.terraform.io` |
| `local` | Loopback addresses | `127.0.0.1`, `::1`, `localhost` |
| `bazel` | Bazel build | `releases.bazel.build`, `bcr.bazel.build` |
| `clojure` | Clojure / Clojars | `clojars.org`, `repo.clojars.org` |
| `deno` | Deno / JSR | `deno.land`, `jsr.io` |
| `elixir` | Hex.pm | `hex.pm`, `repo.hex.pm` |
| `fonts` | Google Fonts | `fonts.googleapis.com`, `fonts.gstatic.com` |
| `julia` | Julia packages | `pkg.julialang.org`, `julialang.org` |
| `kotlin` | Kotlin / JetBrains | `packages.jetbrains.team` |
| `lua` | LuaRocks | `luarocks.org` |
| `node-cdns` | JS CDNs | `cdn.jsdelivr.net`, `code.jquery.com`, `unpkg.com` |
| `ocaml` | OPAM | `opam.ocaml.org`, `ocaml.org` |
| `powershell` | PowerShell Gallery | `powershellgallery.com` |
| `r` | CRAN | `cran.r-project.org`, `cloud.r-project.org` |
| `scala` | sbt / Scala | `repo.scala-sbt.org`, `repo1.maven.org` |
| `zig` | Zig packages | `ziglang.org` |
| `dev-tools` | CI/CD tools | Renovate, Codecov, shields.io, and other dev tooling |
| `chrome` | Chrome / Chromium | `*.googleapis.com`, `*.gvt1.com` |

## Invalid Shorthands

These values look like ecosystem identifiers but are **not recognised** — using them in `network.allowed` causes a **compile-time error**:

| Invalid value | What you probably meant | Correct value |
|---|---|---|
| `npm` | npm registry | `node` |
| `pypi` | Python Package Index | `python` |
| `pip` | pip package manager | `python` |
| `cargo` | Rust crate registry | `rust` |
| `gem` or `gems` | RubyGems | `ruby` |
| `nuget` | NuGet package registry | `dotnet` |
| `maven` | Maven Central | `java` |
| `gradle` | Gradle plugins | `java` |
| `composer` | PHP Composer | `php` |
| `docker` | Docker Hub / GHCR | `containers` |
| `localhost` | Loopback interface | `local` |

## Domain Pattern Rules

- **Wildcard `*` requires a dot prefix**: `*.example.com` is valid; bare `*` is blocked (and rejected outright in strict mode).
- **Protocol prefix is not supported**: `https://api.example.com` is not a valid entry — omit the scheme and write `api.example.com`.
- **Subdomains must be explicit**: `github.com` does not cover `api.github.com`; use `*.github.com` or add both entries.

## Inferring the Right Ecosystem From Repository Files

When a workflow builds, tests, or installs packages, always add the matching ecosystem alongside `defaults`:

| File indicators | Ecosystem to add | Enables |
|---|---|---|
| `package.json`, `yarn.lock`, `pnpm-lock.yaml`, `.nvmrc` | `node` | `registry.npmjs.org` |
| `requirements.txt`, `pyproject.toml`, `uv.lock`, `Pipfile` | `python` | `pypi.org`, `files.pythonhosted.org` |
| `go.mod`, `go.sum` | `go` | `proxy.golang.org`, `sum.golang.org` |
| `*.csproj`, `*.sln`, `*.slnx` | `dotnet` | `api.nuget.org` |
| `pom.xml`, `build.gradle` | `java` | `repo1.maven.org` |
| `Gemfile`, `*.gemspec` | `ruby` | `rubygems.org` |
| `Cargo.toml` | `rust` | `crates.io` |
| `Package.swift` | `swift` | `swift.org` |
| `composer.json` | `php` | `packagist.org` |
| `pubspec.yaml` | `dart` | `pub.dev` |

> ⚠️ **`network: defaults` alone is never sufficient for code workflows** — `defaults` covers basic infrastructure (certificate authorities, Ubuntu verification) but cannot reach package registries. Always add the language ecosystem identifier.

## Common Patterns

### Workflow that reads GitHub data only

```yaml
network:
  allowed:
    - defaults
    - github
```

### Node.js CI workflow

```yaml
network:
  allowed:
    - defaults
    - node
```

### Multi-language project

```yaml
network:
  allowed:
    - defaults
    - node
    - python
```

### Calling an external API

```yaml
network:
  allowed:
    - defaults
    - api.myservice.com
    - "*.myservice.com"
```

### No outbound network access

```yaml
network:
  allowed: []
```
