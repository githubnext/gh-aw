---
title: Network Permissions
description: Control network access for AI engines using ecosystem identifiers and domain allowlists
sidebar:
  order: 1300
---

Control network access for AI engines using the top-level `network` field to specify which domains and services your agentic workflows can access during execution. Supported by all four engines (Copilot, Claude, Codex, Gemini) via the AWF firewall.

If no `network:` permission is specified, it defaults to `network: defaults`, which allows basic infrastructure domains (certificates, JSON schema, Ubuntu, common package mirrors, Microsoft sources).

> [!TIP]
> New to network configuration? See the [Network Configuration Guide](/gh-aw/guides/network-configuration/) for practical examples and troubleshooting tips.

## Configuration

Network permissions follow the principle of least privilege:

- **Default** (`network: defaults`): Basic infrastructure only
- **Selective** (`network: { allowed: [...] }`): Only listed domains/ecosystems
- **No access** (`network: {}`): All network blocked

Listed domains automatically match all subdomains; wildcard patterns (`*.example.com`) are also supported — see [Wildcard Domain Patterns](#wildcard-domain-patterns).

```yaml wrap
network:
  allowed:
    - defaults              # Basic infrastructure
    - python               # Python/PyPI ecosystem
    - node                 # Node.js/NPM ecosystem
    - "api.example.com"    # Custom domain
```

## Blocking Domains

Use the `blocked` field to exclude specific domains or ecosystems from the allowed set. Blocked entries take precedence over allowed ones and include all subdomains — useful for privacy (block trackers), security (block known-bad domains), or compliance:

```yaml wrap
network:
  allowed:
    - defaults
    - github
    - node
  blocked:
    - python               # Block Python ecosystem
    - "cdn.example.com"    # Block specific CDN
```

## Protocol-Specific Domain Filtering

Restrict domains to a specific protocol (HTTP or HTTPS only) for legacy systems, strict HTTPS enforcement, or gradual migration. Currently supported by the Copilot and Claude engines with AWF firewall enabled; domains without a protocol prefix allow both HTTP and HTTPS.

```yaml wrap
engine: copilot
network:
  allowed:
    - "https://secure.api.example.com"   # HTTPS-only access
    - "http://legacy.example.com"        # HTTP-only access  
    - "example.org"                      # Both protocols (default)
    - "https://*.api.example.com"        # HTTPS wildcard
```

**Compiled to AWF:**

```bash
--allow-domains ...,example.org,http://legacy.example.com,https://secure.api.example.com,...
```

## Content Sanitization

The `network:` configuration also controls which domains are allowed in sanitized content. URLs from domains not in the allowed list are replaced with `(redacted)` to prevent potential data exfiltration through untrusted links.

> [!TIP]
> If you see `(redacted)` in workflow outputs, add the domain to your `network.allowed` list. This applies the same domain allowlist to both network egress (when firewall is enabled) and content sanitization.

GitHub domains (`github.com`, `githubusercontent.com`, etc.) are always allowed by default.

## Ecosystem Identifiers

Mix ecosystem identifiers with specific domains for fine-grained control:

| Identifier | Includes |
|------------|----------|
| `defaults` | Basic infrastructure (certificates, JSON schema, Ubuntu, package mirrors) |
| `github` | GitHub domains (`github.com`, `docs.github.com`, `github.blog`, `*.githubusercontent.com`, and related) |
| `local` | Loopback addresses (`localhost`, `127.0.0.1`, `::1`) |
| `dev-tools` | Popular CI/CD and developer tool services (Codecov, Shields.io, Snyk, Renovate, CircleCI, etc.) |
| `default-safe-outputs` | Compound: `defaults` + `dev-tools` + `github` + `local` — recommended baseline for `safe-outputs.allowed-domains` |
| `containers` | Docker Hub, GitHub Container Registry, Quay, GCR, MCR |
| `linux-distros` | Debian, Ubuntu, Alpine, Fedora, and other Linux package repositories |
| `playwright` | Playwright browser testing (see [Playwright Reference](/gh-aw/reference/playwright/)) |
| `chrome` | Headless Chrome/Puppeteer (`*.google.com`, `*.googleapis.com`, `*.gvt1.com`) |
| `fonts` | Google Fonts (`fonts.googleapis.com`, `fonts.gstatic.com`) |
| `terraform` | HashiCorp registry, apt/yum releases |
| `bazel` | Bazel build system (`releases.bazel.build`, `bcr.bazel.build`) |
| `clojure` | Clojure packages (`clojars.org`) |
| `dart` | Dart/Flutter packages (`pub.dev`, `storage.googleapis.com`) |
| `deno` | Deno runtime (`deno.land`, `jsr.io`, `googleapis.deno.dev`) |
| `dotnet` | NuGet packages and .NET SDK |
| `elixir` | Elixir packages (`hex.pm`) |
| `go` | Go modules (`proxy.golang.org`, `sum.golang.org`) |
| `haskell` | Haskell packages (`hackage.haskell.org`, GHCup) |
| `java` | Maven Central, Gradle, Adoptium |
| `julia` | Julia packages (`pkg.julialang.org`, `storage.julialang.net`) |
| `kotlin` | Kotlin/JetBrains packages (`download.jetbrains.com`) |
| `latex` | CTAN, TUG, MiKTeX packages — **note**: TeX Live's `tlmgr` uses redirected CTAN mirrors not reachable through the firewall; prefer `apt-get install texlive-full` (covered by `defaults`) or MiKTeX (`packages.miktex.org`) |
| `lean` | Lean packages (`lean-lang.org`, `reservoir.lean-lang.org`) |
| `lua` | Lua packages (`luarocks.org`) |
| `node` | npm, yarn, pnpm, Node.js, Bun |
| `node-cdns` | Node.js CDN assets (jsDelivr, jQuery CDN) |
| `ocaml` | OCaml packages (`opam.ocaml.org`) |
| `perl` | Perl/CPAN packages (`cpan.org`, `metacpan.org`) |
| `php` | PHP Composer packages (`packagist.org`, `getcomposer.org`) |
| `powershell` | PowerShell Gallery (`powershellgallery.com`) |
| `python` | PyPI, conda, pythonhosted.org |
| `python-native` | PyPI + crates.io — for Python packages with native extensions (pyo3/maturin) |
| `r` | R/CRAN packages (`cran.r-project.org`) |
| `ruby` | RubyGems, Bundler |
| `rust` | Rust crates (`crates.io`, `static.rust-lang.org`, `sh.rustup.rs`) |
| `scala` | Scala packages (`repo.scala-sbt.org`, `jitpack.io`) |
| `swift` | Swift packages (`swift.org`, `cocoapods.org`) |
| `zig` | Zig packages (`ziglang.org`) |

### Ecosystem Identifier Validation

Single-word entries in `network.allowed` that match the ecosystem identifier pattern (`[a-z][a-z0-9-]*`) are validated against the known ecosystem list at compile time. An unrecognized identifier produces a compilation error with the full list of valid options:

```yaml wrap
# ❌ Compilation error: 'rustxxxx' is not a valid ecosystem identifier
network:
  allowed:
    - defaults
    - rustxxxx

# ✅ Use the correct identifier
network:
  allowed:
    - defaults
    - rust

# ✅ Dotted domain names are validated as domains, not ecosystem identifiers
network:
  allowed:
    - defaults
    - crates.io
```

## Strict Mode Validation

When [strict mode](/gh-aw/reference/frontmatter/#strict-mode-strict) is enabled (default), strict mode warns when you list individual ecosystem member domains (e.g., `pypi.org`, `npmjs.org`) and recommends the ecosystem identifier instead (e.g., `python`, `node`). Custom domains outside known ecosystems are allowed without warnings.

````yaml wrap
# ⚠ Warns: recommend 'python' instead of 'pypi.org'
strict: true
network:
  allowed:
    - defaults
    - "pypi.org"
    - "npmjs.org"

# ✅ No warnings
strict: true
network:
  allowed:
    - defaults
    - python
    - node
    - "api.example.com"  # Custom domain
````

The emitted message takes the form:

````text
recommend using ecosystem identifiers instead of individual domain names for better maintainability: 'pypi.org' → 'python', 'npmjs.org' → 'node'
````

## Implementation

All engines (Copilot, Claude, Codex, Gemini) enforce network permissions through AWF (Agent Workflow Firewall) — a wrapper sourced from [github.com/github/gh-aw-firewall](https://github.com/github/gh-aw-firewall) that enforces domain-based access controls via `--allow-domains`. AWF automatically includes all subdomains (e.g., `github.com` allows `api.github.com`), supports wildcard patterns, and logs all network activity for audit.

```yaml wrap
engine: copilot          # or claude, codex, gemini
network:
  allowed:
    - defaults             # Basic infrastructure
    - python              # Python ecosystem
    - "api.example.com"   # Custom domain
```

Each engine has a built-in default domain list for its CLI authentication. See [`domains.go`](https://github.com/github/gh-aw/blob/main/pkg/workflow/domains.go) for the full lists.

### Firewall Log Level

Control the verbosity of AWF firewall logs using `log-level`. Options: `debug` (verbose), `info` (default), `warn`, `error`.

```yaml wrap
network:
  firewall:
    log-level: info
  allowed:
    - defaults
    - python
```

### SSL Bump for HTTPS Inspection

Enable SSL bump to filter HTTPS traffic by URL path patterns instead of just domain names — useful when you need to allow specific API endpoints while blocking others on the same domain. Use `allow-urls` to specify the permitted HTTPS patterns.

```yaml wrap
network:
  firewall:
    ssl-bump: true
    allow-urls:
      - "https://github.com/githubnext/*"
      - "https://api.github.com/repos/*/issues"
  allowed:
    - defaults
```

**Security**: SSL bump intercepts and decrypts HTTPS as a man-in-the-middle — only enable when URL-level filtering is necessary, and craft `allow-urls` patterns carefully to avoid breaking legitimate connections. Requires AWF v0.9.0+ and does not apply to Sandbox Runtime (SRT). See [Sandbox Configuration](/gh-aw/reference/sandbox/) for full AWF options.

### Effective Token Steering

The AWF API proxy automatically injects budget-warning system messages as the run approaches its effective-token budget. Warnings fire at 80%, 90%, 95%, and 99% of the configured `max-effective-tokens`, giving the agent a chance to wrap up work before the budget is exhausted. Token steering requires AWF `v0.25.44` or later; for older pinned versions the setting is silently dropped at compile time.

To disable token steering (and budget enforcement entirely), set `max-effective-tokens` to a negative value:

```yaml wrap
max-effective-tokens: -1
```

See [Max Effective Tokens](/gh-aw/reference/glossary/#max-effective-tokens-max-effective-tokens) for budget configuration.

### Disabling the Firewall

The firewall is enabled by default via `sandbox.agent: awf`. When disabled, network permissions still apply for content sanitization but the agent can make unrestricted network requests. Only disable during development or when AWF is incompatible with your workflow; keep it enabled in production.

## Caller-Extensible Allowlist (`network.allowed-input`)

Reusable workflows compiled to `.lock.yml` bake their `network.allowed` into the lock file, so consumers normally can't extend it without forking. Set `network.allowed-input: true` to expose a `workflow_call` input named `network_allowed` that lets callers add domains or ecosystems at runtime.

The compiled `network.allowed` remains the baseline; the caller's value is unioned in before AWF starts, with ecosystem shorthands expanded to their concrete domain sets and the result deduplicated.

```yaml wrap
# source workflow (compiled to a reusable .lock.yml)
on:
  workflow_call:
network:
  allowed:
    - defaults
  allowed-input: true
```

```yaml wrap
# consumer workflow
jobs:
  run:
    uses: owner/repo/.github/workflows/worker.lock.yml@v1
    with:
      network_allowed: rust,github.com
```

The `network_allowed` input is a string accepting comma-separated ecosystem identifiers and/or domains. The behavior of the source workflow is unchanged when `allowed-input` is omitted or `false`.

## Wildcard Domain Patterns

Wildcard patterns (`*.example.com`) match the base domain and all subdomains at any depth, making subdomain intent explicit. Only a single leading wildcard is allowed (`*.*.example.com` is invalid), and it must be followed by a dot and domain. Both `example.com` and `*.example.com` match all subdomains.

```yaml wrap
network:
  allowed:
    - defaults
    - "*.cdn.example.com"     # Matches img.cdn.example.com, static.cdn.example.com
    - "*.storage.example.com" # Matches files.storage.example.com
```

## Troubleshooting

If you encounter network access blocked errors, verify that required domains or ecosystems are in the `allowed` list. Start with `network: defaults` and add specific requirements incrementally. Network access violations are logged in workflow execution logs.

Use `gh aw logs --run-id <run-id>` to view firewall activity and identify blocked domains. For detailed diagnostics, use `gh aw audit <run-id>` — the **Firewall Analysis** section lists every domain request with its allow/deny status, request volume, and policy attribution. Pass two run IDs to compare firewall behavior between runs:

```bash
gh aw audit 12345678              # Single run
gh aw audit 12345678 12345679     # Compare two runs
```

See the [Network Configuration Guide](/gh-aw/guides/network-configuration/#troubleshooting-firewall-blocking) and [Audit Commands](/gh-aw/reference/audit/) for more.

## Related Documentation

- [Network Configuration Guide](/gh-aw/guides/network-configuration/) - Practical examples and common patterns
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter configuration guide
- [Tools](/gh-aw/reference/tools/) - Tool-specific network access configuration
- [Playwright](/gh-aw/reference/playwright/) - Browser automation and network requirements
- [Audit Commands](/gh-aw/reference/audit/) - Firewall analysis and cross-run diff for understanding domain allow/block behavior
- [Security Guide](/gh-aw/introduction/architecture/) - Comprehensive security guidance
