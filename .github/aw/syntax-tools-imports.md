---
description: Cache, tool, import, and permission reference for GitHub Agentic Workflows frontmatter.
---

# Tools, Imports, and Permissions

### Cache Configuration

The `cache:` field supports the same syntax as the GitHub Actions `actions/cache` action:

**Single Cache:**

```yaml
cache:
  key: node-modules-${{ hashFiles('package-lock.json') }}
  path: node_modules
  restore-keys: |
    node-modules-
```

**Multiple Caches:**

```yaml
cache:
  - key: node-modules-${{ hashFiles('package-lock.json') }}
    path: node_modules
    restore-keys: |
      node-modules-
  - key: build-cache-${{ github.sha }}
    path:
      - dist
      - .cache
    restore-keys:
      - build-cache-
    fail-on-cache-miss: false
```

**Supported Cache Parameters:**

- `key:` - Cache key (required)
- `path:` - Files/directories to cache (required, string or array)
- `restore-keys:` - Fallback keys (string or array)
- `upload-chunk-size:` - Chunk size for large files (integer)
- `fail-on-cache-miss:` - Fail if cache not found (boolean)
- `lookup-only:` - Only check cache existence (boolean)

Cache steps are automatically added to the workflow job and the cache configuration is removed from the final `.lock.yml` file.


> **Memory configuration**: For detailed documentation on `cache-memory:`, `repo-memory:`, and `comment-memory:` configuration including advanced options and use cases, see [memory.md](memory.md).


## Tool Configuration

### General Tools

```yaml
tools:
  edit:           # File editing (required to write to files)
  web-fetch:       # Web content fetching
  web-search:      # Web searching
  bash:           # Shell commands
  - "gh label list:*"
  - "gh label view:*"
  - "git status"
```

### Custom MCP Tools

Stdio MCP servers must be Docker-based (use `container:` + `entrypoint:`). For Node/Python servers already installed on the runner, use HTTP transport instead:

```yaml
# Stdio (Docker-based)
mcp-servers:
  my-custom-tool:
    container: "ghcr.io/my-org/my-tool:latest"
    entrypoint: "my-tool"
    allowed:
      - custom_function_1
      - custom_function_2

# HTTP (for Node/Python servers running on the runner)
mcp-servers:
  my-node-tool:
    type: http
    url: "http://localhost:8765/mcp"
```

HTTP MCP servers are also supported with optional upstream authentication:

```yaml
mcp-servers:
  my-server:
    type: http
    url: "https://myserver.example.com/mcp"
    headers:
      Authorization: "Bearer ${{ secrets.API_KEY }}"    # Optional: custom headers
  my-oidc-server:
    type: http
    url: "https://myserver.example.com/mcp"
    auth:
      type: github-oidc                                  # GitHub Actions OIDC token authentication
      audience: "https://myserver.example.com"          # Optional: custom OIDC audience
```

`auth.type: github-oidc` uses GitHub Actions OIDC tokens for secure server-to-server authentication without static credentials. The `audience` field is optional and defaults to the server URL when omitted.

### Engine Network Permissions

Control network access for AI engines using the top-level `network:` field. If no `network:` permission is specified, it defaults to `network: defaults` which provides access to basic infrastructure only.

```yaml
engine:
  id: copilot

# Basic infrastructure only (default)
network: defaults

# Use ecosystem identifiers for common development tools
network:
  allowed:
    - defaults         # Basic infrastructure
    - python          # Python/PyPI ecosystem
    - node            # Node.js/NPM ecosystem
    - containers      # Container registries
    - "api.custom.com" # Custom domain
    - "https://secure.api.com" # Protocol-specific domain
  blocked:
    - "tracking.com"   # Block specific domains
    - "*.ads.com"      # Block domain patterns
    - ruby             # Block ecosystem identifiers
  firewall: true      # Enable AWF (Copilot engine only)

# Or allow specific domains only
network:
  allowed:
    - "api.github.com"
    - "*.trusted-domain.com"
    - "example.com"

# Or deny all network access
network: {}
```

**Important Notes:**

- Network permissions apply to AI engines' WebFetch and WebSearch tools
- Uses top-level `network:` field (not nested under engine permissions)
- `defaults` now includes only basic infrastructure (certificates, JSON schema, Ubuntu, etc.)
- Use ecosystem identifiers (`python`, `node`, `java`, etc.) for language-specific tools
- When custom permissions are specified with `allowed:` list, deny-by-default policy is enforced
- Supports exact domain matches and wildcard patterns (where `*` matches any characters, including nested subdomains)
- **Protocol-specific filtering**: Prefix domains with `http://` or `https://` for protocol restrictions
- **Domain blocklist**: Use `blocked:` field to explicitly deny domains or ecosystem identifiers
- **Firewall support**: Copilot engine supports AWF (Agent Workflow Firewall) for domain-based access control
- Claude engine uses hooks for enforcement; Codex support planned

**Permission Modes:**

1. **Basic infrastructure**: `network: defaults` or no `network:` field (certificates, JSON schema, Ubuntu only)
2. **Ecosystem access**: `network: { allowed: [defaults, python, node, ...] }` (development tool ecosystems)
3. **No network access**: `network: {}` (deny all)
4. **Specific domains**: `network: { allowed: ["api.example.com", ...] }` (granular access control)
5. **Block specific domains**: `network: { blocked: ["tracking.com", "*.ads.com", ...] }` (deny-list)

**Available Ecosystem Identifiers:**

Each ecosystem identifier enables network access to the domains required by that language's package manager and toolchain. When writing workflows that involve package management, builds, or tests, **always include the ecosystem identifier matching the repository's primary language** in addition to `defaults`.

| Identifier | Runtimes / Languages | Package Manager / Domains |
|---|---|---|
| `defaults` | All (always include) | Certificates, JSON schema, Ubuntu mirrors, Microsoft sources |
| `github` | Any | GitHub domains (`github.com`, `*.githubusercontent.com`, `codeload.github.com`, etc.) |
| `local` | Any | Loopback (`localhost`, `127.0.0.1`, `::1`) |
| `dev-tools` | Any | CI/CD services (Codecov, Shields.io, Snyk, Renovate, CircleCI, etc.) |
| `default-safe-outputs` | Any | Compound: `defaults` + `dev-tools` + `github` + `local` — recommended for `safe-outputs.allowed-domains` |
| `containers` | Docker, OCI | Docker Hub, GHCR, Quay, GCR, MCR (`registry.hub.docker.com`, `ghcr.io`, etc.) |
| `linux-distros` | Any | apt, yum/dnf (Debian, Alpine, Fedora, `deb.debian.org`, `dl-cdn.alpinelinux.org`, etc.) |
| `playwright` | Any | Playwright browser automation (`cdn.playwright.dev`, `playwright.download.prss.microsoft.com`) |
| `chrome` | Any | Headless Chrome/Puppeteer (`*.google.com`, `*.googleapis.com`, `*.gvt1.com`) |
| `fonts` | Any | Google Fonts (`fonts.googleapis.com`, `fonts.gstatic.com`) |
| `terraform` | Terraform, OpenTofu | HashiCorp registry (`registry.terraform.io`, `releases.hashicorp.com`) |
| `bazel` | Any | Bazel build system (`releases.bazel.build`, `bcr.bazel.build`) |
| `clojure` | Clojure | Clojars (`clojars.org`) |
| `dart` | Dart, Flutter | pub.dev (`pub.dev`, `storage.googleapis.com`) |
| `deno` | JavaScript, TypeScript | Deno runtime (`deno.land`, `jsr.io`, `googleapis.deno.dev`) |
| `dotnet` | C#, F#, VB.NET | NuGet (`nuget.org`, `api.nuget.org`, `dotnetcli.blob.core.windows.net`, etc.) |
| `elixir` | Elixir | Hex (`hex.pm`, `cdn.hex.pm`) |
| `go` | Go | Go modules (`proxy.golang.org`, `sum.golang.org`, `pkg.go.dev`) |
| `haskell` | Haskell | Hackage, GHCup (`hackage.haskell.org`, `get-ghcup.haskell.org`, `downloads.haskell.org`) |
| `java` | Java, Groovy | Maven, Gradle (`repo1.maven.org`, `plugins.gradle.org`, `api.adoptium.net`, etc.) |
| `julia` | Julia | Julia packages (`pkg.julialang.org`, `storage.julialang.net`) |
| `kotlin` | Kotlin | JetBrains packages (`download.jetbrains.com`, `packages.jetbrains.team`) |
| `latex` | LaTeX, TeX | CTAN, TUG, MiKTeX (`ctan.org`, `tug.org`, `packages.miktex.org`) — **note**: TeX Live `tlmgr` uses redirected CTAN mirrors not reachable through the firewall; prefer `apt-get install texlive-full` (via `defaults`) or MiKTeX |
| `lean` | Lean | Lean packages (`lean-lang.org`, `reservoir.lean-lang.org`) |
| `lua` | Lua | LuaRocks (`luarocks.org`) |
| `node` | Node.js, JavaScript, TypeScript | npm, yarn, pnpm, Bun (`registry.npmjs.org`, `yarnpkg.com`, `bun.sh`, etc.) |
| `node-cdns` | JavaScript, TypeScript | Node.js CDN assets (jsDelivr, jQuery CDN — `cdn.jsdelivr.net`, `code.jquery.com`) |
| `ocaml` | OCaml | opam (`opam.ocaml.org`) |
| `perl` | Perl | CPAN (`cpan.org`, `metacpan.org`) |
| `php` | PHP | Composer, Packagist (`packagist.org`, `getcomposer.org`) |
| `powershell` | PowerShell | PowerShell Gallery (`powershellgallery.com`) |
| `python` | Python | pip, conda, PyPI (`pypi.org`, `files.pythonhosted.org`, `conda.anaconda.org`, etc.) |
| `python-native` | Python (native extensions) | PyPI + crates.io — for pyo3/maturin builds |
| `r` | R | CRAN (`cran.r-project.org`, `cloud.r-project.org`) |
| `ruby` | Ruby | RubyGems, Bundler (`rubygems.org`, `api.rubygems.org`, etc.) |
| `rust` | Rust | Cargo, crates.io (`crates.io`, `static.crates.io`, `sh.rustup.rs`) |
| `scala` | Scala | sbt, JitPack (`repo.scala-sbt.org`, `jitpack.io`) |
| `swift` | Swift | Swift Package Manager, CocoaPods (`swift.org`, `download.swift.org`, `cocoapods.org`) |
| `zig` | Zig | Zig packages (`ziglang.org`) |

**Network Inference Rule:** When a workflow will run `dotnet restore`, `pip install`, `npm install`, `go mod download`, `mvn install`, etc., infer the ecosystem from the repository language and include it. Examples:

- `.NET` repository (`.csproj`, `.fsproj`, `*.sln`, `*.slnx`) → `network: { allowed: [defaults, dotnet] }`
- Python repository (`requirements.txt`, `pyproject.toml`) → `network: { allowed: [defaults, python] }`
- Node.js repository (`package.json`) → `network: { allowed: [defaults, node] }`
- Go repository (`go.mod`) → `network: { allowed: [defaults, go] }`
- Java repository (`pom.xml`, `build.gradle`) → `network: { allowed: [defaults, java] }`

## Imports Field

Import shared components using the `imports:` field in frontmatter:

```yaml
---
on: issues
engine: copilot
imports:
  - copilot-setup-steps.yml    # Import setup steps from copilot-setup-steps.yml
  - shared/security-notice.md
  - shared/tool-setup.md
  - shared/mcp/tavily.md
---
```

**Object form with inputs** — Use `path:`/`uses:` + `with:`/`inputs:` to pass values to shared workflows that define an `import-schema:`. Optional `checkout:` and `env:` fields customize the import:

```yaml
imports:
  - path: shared/tool-setup.md
    with:
      environment: staging
      max-issues: 3
    env:
      MY_VAR: "value"         # Optional: pass env vars into the imported workflow
    checkout: main            # Optional: ref to check out when this import is processed
  - uses: shared/security-notice.md  # 'uses' is an alias for 'path'
```

- `env:` - Environment variables passed into the imported workflow context (object). Use when a shared workflow relies on environment variables that must be supplied by the importing workflow.
- `checkout:` - Ref (branch, tag, or SHA) to check out when processing this import (string). Overrides the default checkout for this specific import entry.

Inside the imported workflow, access values via `${{ github.aw.import-inputs.<name> }}`.

### Import File Structure

Import files are in `.github/workflows/shared/` and can contain:

- Tool configurations
- Safe-outputs configurations
- Text content
- Mixed frontmatter + content

The following frontmatter fields in imported files are merged into the importing workflow:

- `tools:` - Merged with the importing workflow's tools
- `safe-outputs:` - Merged with safe-output configuration
- `env:` - Environment variables merged; conflicts between two imports defining the same key are compilation errors (remove the duplicate or move it to the main workflow to override)
- `checkout:` - Checkout configurations appended (main workflow's checkouts take precedence)
- `github-app:` - Top-level GitHub App credentials (first-wins across imports)
- `on.github-app:` - Activation GitHub App credentials (first-wins across imports)
- `steps:`, `pre-steps:`, `pre-agent-steps:`, `post-steps:` - Steps appended in import order
- `runtimes:`, `network:`, `permissions:`, `services:`, `cache:`, `features:`, `mcp-servers:`

Example import file:

```markdown
---
tools:
  github:
    allowed: [get_repository, list_commits]
safe-outputs:
  create-issue:
    labels: [automation]
env:
  MY_VAR: "shared-value"
checkout:
  fetch-depth: 0
---

Additional instructions for the coding agent.
```

### Special Import: copilot-setup-steps.yml

The `copilot-setup-steps.yml` file receives special handling when imported. Instead of importing the entire job structure, **only the steps** from the `copilot-setup-steps` job are extracted and inserted **at the start** of your workflow's agent job.

**Key behaviors:**

- Only the steps array is imported (job metadata like `runs-on`, `permissions` is ignored)
- Imported steps are placed **at the start** of the agent job (before all other steps)
- Other imported steps are placed after copilot-setup-steps but before main frontmatter steps
- Main frontmatter steps come last
- Final order: **copilot-setup-steps → other imported steps → main frontmatter steps**
- Supports both `.yml` and `.yaml` extensions
- Enables clean reuse of common setup configurations across workflows

**Example:**

```yaml
---
on: issue_comment
engine: copilot
imports:
  - copilot-setup-steps.yml
  - shared/common-tools.md
steps:
  - name: Custom environment setup
    run: echo "Main frontmatter step runs last"
---
```

In the compiled workflow, the order is: copilot-setup-steps → imported steps from shared/common-tools.md → main frontmatter steps.

## Permission Patterns

**IMPORTANT**: Agentic workflows should NOT include write permissions (`issues: write`, `pull-requests: write`, `contents: write`). The safe-outputs system provides these capabilities through separate, secured jobs with appropriate permissions. NO write permissions should be granted to the main AI processing job, it will only cause a later compilation error.

### Read-Only Pattern

```yaml
permissions:
  contents: read
  metadata: read
```

### Output Processing Pattern (Recommended)

```yaml
permissions:
  contents: read      # Main job minimal permissions
  actions: read

safe-outputs:
  create-issue:       # Automatic issue creation
  add-comment:  # Automatic comment creation
  create-pull-request: # Automatic PR creation
```

**Key Benefits of Safe-Outputs:**

- **Security**: Main job runs with minimal permissions
- **Separation of Concerns**: Write operations are handled by dedicated jobs
- **Permission Management**: Safe-outputs jobs automatically receive required permissions
- **Audit Trail**: Clear separation between AI processing and GitHub API interactions
