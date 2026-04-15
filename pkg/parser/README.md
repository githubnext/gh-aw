# parser Package

> Markdown frontmatter parsing, import resolution, GitHub URL handling, and schema validation for agentic workflow files.

## Overview

The `parser` package is responsible for extracting and processing YAML frontmatter from agentic workflow `.md` files. Frontmatter defines the workflow's entire configuration — triggers, permissions, tools, safe outputs, engine settings, network restrictions, and runtime overrides. The markdown body that follows the frontmatter serves as the AI agent's prompt text.

Beyond basic frontmatter extraction, the package provides a rich import system that resolves `@import` directives (local files, GitHub URLs, fragments), an include expander for `@include` directives in the markdown body, a schedule parser that converts natural-language schedules into cron expressions, MCP server configuration extraction, and JSON schema–backed validation with actionable error messages.

The package is designed for use both in the main CLI binary and in WebAssembly contexts (see `*_wasm.go` files). Build constraints separate platform-specific implementations for remote fetching and filesystem access.

## Public API

### Types

| Type | Kind | Description |
|------|------|-------------|
| `FrontmatterResult` | struct | Result of extracting frontmatter from markdown content |
| `ImportCache` | struct | Thread-safe cache of resolved imports to avoid redundant fetches |
| `ImportDirectiveMatch` | struct | Parsed `@import` or `@include` directive line |
| `ImportError` | struct | Structured error for import resolution failures |
| `ImportCycleError` | struct | Structured error for circular import chains |
| `FormattedParserError` | struct | Pre-formatted parser error with display-ready message |
| `ImportsResult` | struct | Result of `ProcessImportsFromFrontmatterWithSource` |
| `ImportInputDefinition` | struct | Input definition from an imported workflow fragment |
| `ImportSpec` | struct | Resolved import specification (path, ref, optional flag) |
| `GitHubURLType` | string alias | Classifies a GitHub URL (`tree`, `blob`, `raw`, `run`, `pr`, etc.) |
| `GitHubURLComponents` | struct | Parsed components of a GitHub URL (owner, repo, ref, path, etc.) |
| `JSONPathLocation` | struct | Line/column location of a JSON path in YAML content |
| `JSONPathInfo` | struct | JSON path with human-readable description |
| `NestedSection` | struct | Locates nested YAML sections for error reporting |
| `PathSegment` | struct | A single segment in a resolved JSON path |
| `MCPServerConfig` | struct | Parsed MCP server configuration (type, command, URL, env, etc.) |
| `MCPServerInfo` | struct | Metadata about an MCP server entry |
| `ScheduleParser` | struct | Converts natural-language schedules to cron expressions |
| `DeprecatedField` | struct | A deprecated frontmatter field with migration guidance |
| `FileReader` | func type | `func(filePath string) ([]byte, error)` — abstraction for file reading |

### Functions

#### Frontmatter Extraction

| Function | Signature | Description |
|----------|-----------|-------------|
| `ExtractFrontmatterFromContent` | `func(content string) (*FrontmatterResult, error)` | Extracts YAML frontmatter between `---` delimiters from markdown |
| `ExtractFrontmatterFromBuiltinFile` | `func(path string, content []byte) (*FrontmatterResult, error)` | Extracts frontmatter from an embedded/built-in workflow file |
| `ExtractMarkdownContent` | `func(content string) (string, error)` | Returns the markdown body (everything after frontmatter) |
| `ExtractMarkdownSection` | `func(content, sectionName string) (string, error)` | Extracts a named `##` section from markdown |
| `ExtractWorkflowNameFromMarkdownBody` | `func(markdownBody, virtualPath string) (string, error)` | Derives the workflow name from the first `#` heading |
| `ExtractWorkflowNameFromContent` | `func(content, virtualPath string) (string, error)` | Combines frontmatter extraction and name derivation |

#### Import Processing

| Function | Signature | Description |
|----------|-----------|-------------|
| `ProcessImportsFromFrontmatterWithSource` | `func(frontmatter map[string]any, baseDir string, cache *ImportCache, ...) (*ImportsResult, error)` | Resolves all `@import` directives in frontmatter, merging imported configs |
| `ParseImportDirective` | `func(line string) *ImportDirectiveMatch` | Parses a single `@import` or `@include` line |
| `NewImportCache` | `func(repoRoot string) *ImportCache` | Creates a new import cache rooted at the repository |
| `ExpandIncludesWithManifest` | `func(content, baseDir string, extractTools bool) (string, []string, error)` | Expands `@include` directives in markdown body and returns included file paths |
| `ExpandIncludesForEngines` | `func(content, baseDir string) ([]string, error)` | Returns engine names referenced via `@include` |
| `ExpandIncludesForSafeOutputs` | `func(content, baseDir string) ([]string, error)` | Returns safe output types referenced via `@include` |

#### GitHub URL Parsing

| Function | Signature | Description |
|----------|-----------|-------------|
| `ParseGitHubURL` | `func(urlStr string) (*GitHubURLComponents, error)` | Parses any GitHub URL into structured components |
| `ParseRunURLExtended` | `func(input string) (*GitHubURLComponents, error)` | Parses a workflow run URL (extended formats) |
| `ParsePRURL` | `func(prURL string) (owner, repo string, prNumber int, err error)` | Parses a pull request URL |
| `ParseRepoFileURL` | `func(fileURL string) (owner, repo, ref, filePath string, err error)` | Parses a repository file URL |
| `IsValidGitHubIdentifier` | `func(s string) bool` | Validates a GitHub username/org/repo name |
| `GetGitHubHost` | `func() string` | Returns the GitHub host (supports GHES via `GH_HOST`) |
| `GetGitHubHostForRepo` | `func(owner, repo string) string` | Returns the GitHub host for a specific repo |
| `GetGitHubToken` | `func() (string, error)` | Returns the GitHub auth token from the environment |

#### Remote Fetching

| Function | Signature | Description |
|----------|-----------|-------------|
| `ResolveIncludePath` | `func(filePath, baseDir string, cache *ImportCache) (string, error)` | Resolves a relative or GitHub URL path to an absolute path or fetches remotely |
| `DownloadFileFromGitHub` | `func(owner, repo, path, ref string) ([]byte, error)` | Downloads a file from GitHub via the API |
| `DownloadFileFromGitHubForHost` | `func(owner, repo, path, ref, host string) ([]byte, error)` | Downloads a file from a specific GitHub host |
| `ResolveRefToSHAForHost` | `func(owner, repo, ref, host string) (string, error)` | Resolves a branch/tag ref to a commit SHA |
| `ListWorkflowFiles` | `func(owner, repo, ref, workflowPath string) ([]string, error)` | Lists workflow files in a remote repository |

#### MCP Configuration

| Function | Signature | Description |
|----------|-----------|-------------|
| `ExtractMCPConfigurations` | `func(frontmatter map[string]any, serverFilter string) ([]MCPServerConfig, error)` | Extracts all MCP server configurations from frontmatter |
| `ParseMCPConfig` | `func(toolName string, mcpSection any, toolConfig map[string]any) (MCPServerConfig, error)` | Parses a single MCP server entry |
| `IsMCPType` | `func(typeStr string) bool` | Validates an MCP transport type string |

#### Schedule Parsing

| Function | Signature | Description |
|----------|-----------|-------------|
| `ParseSchedule` | `func(input string) (cron, original string, err error)` | Parses natural-language or cron schedule to a cron expression |
| `ScatterSchedule` | `func(fuzzyCron, workflowIdentifier string) (string, error)` | Deterministically scatters a daily/hourly cron to reduce thundering herd |
| `IsDailyCron` | `func(cron string) bool` | Detects whether a cron expression runs daily |
| `IsHourlyCron` | `func(cron string) bool` | Detects whether a cron expression runs hourly |
| `IsWeeklyCron` | `func(cron string) bool` | Detects whether a cron expression runs weekly |
| `IsFuzzyCron` | `func(cron string) bool` | Detects whether a cron is a fuzzy wildcard |
| `IsCronExpression` | `func(input string) bool` | Detects whether a string is already a cron expression |

#### Schema Validation

| Function | Signature | Description |
|----------|-----------|-------------|
| `ValidateMainWorkflowFrontmatterWithSchemaAndLocation` | `func(frontmatter map[string]any, filePath string) error` | JSON-schema validates frontmatter and returns located errors |
| `GetCompiledRepoConfigSchema` | `func() (*jsonschema.Schema, error)` | Returns the compiled JSON schema for repo config |
| `GetSafeOutputTypeKeys` | `func() ([]string, error)` | Returns valid safe-output type keys from the schema |
| `GetMainWorkflowDeprecatedFields` | `func() ([]DeprecatedField, error)` | Returns deprecated frontmatter fields with migration notes |
| `FindDeprecatedFieldsInFrontmatter` | `func(map[string]any, []DeprecatedField) []DeprecatedField` | Finds deprecated fields present in a parsed frontmatter map |
| `FindClosestMatches` | `func(target string, candidates []string, maxResults int) []string` | Finds the closest string matches (for typo suggestions) |
| `LevenshteinDistance` | `func(a, b string) int` | Computes edit distance between two strings |

#### Frontmatter Hashing

| Function | Signature | Description |
|----------|-----------|-------------|
| `ComputeFrontmatterHashFromFile` | `func(filePath string, cache *ImportCache) (string, error)` | Computes a stable hash of a workflow's frontmatter (including imports) |
| `ComputeFrontmatterHashFromFileWithParsedFrontmatter` | `func(filePath string, parsedFrontmatter map[string]any, ...) (string, error)` | Computes hash from already-parsed frontmatter |
| `ComputeFrontmatterHashFromFileWithReader` | `func(filePath string, cache *ImportCache, fileReader FileReader) (string, error)` | Computes hash with a custom file reader |

#### Error Formatting

| Function | Signature | Description |
|----------|-----------|-------------|
| `FormatImportCycleError` | `func(*ImportCycleError) error` | Formats a cycle error with the import chain |
| `FormatImportError` | `func(*ImportError, yamlContent string) error` | Formats an import error with YAML context |
| `NewFormattedParserError` | `func(formatted string) *FormattedParserError` | Creates a pre-formatted parser error |

#### JSON Path Location

| Function | Signature | Description |
|----------|-----------|-------------|
| `ExtractJSONPathFromValidationError` | `func(err error) []JSONPathInfo` | Extracts JSON path info from a schema validation error |
| `LocateJSONPathInYAML` | `func(yamlContent, jsonPath string) JSONPathLocation` | Maps a JSON path to a line number in YAML text |
| `LocateJSONPathInYAMLWithAdditionalProperties` | `func(yamlContent, jsonPath, errorMessage string) JSONPathLocation` | Maps path with additional-property context |

#### Trigger Helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsLabelOnlyEvent` | `func(eventValue any) bool` | Detects whether a trigger only activates on label events |
| `IsNonConflictingCommandEvent` | `func(eventValue any) bool` | Detects whether a trigger is a non-conflicting slash command |

#### Virtual Filesystem and Workflow Update Helpers

| Function | Signature | Description |
|----------|-----------|-------------|
| `RegisterBuiltinVirtualFile` | `func(path string, content []byte)` | Registers embedded virtual file content under an `@builtin:` path |
| `BuiltinVirtualFileExists` | `func(path string) bool` | Returns whether a built-in virtual file path has been registered |
| `GetBuiltinFrontmatterCache` | `func(path string) (*FrontmatterResult, bool)` | Gets cached frontmatter parse results for built-in virtual files |
| `SetBuiltinFrontmatterCache` | `func(path string, result *FrontmatterResult) *FrontmatterResult` | Stores a frontmatter parse result in the built-in cache |
| `ReadFile` | `func(path string) ([]byte, error)` | Reads file content through parser virtual/builtin-aware file resolution |
| `MergeTools` | `func(base, additional map[string]any) (map[string]any, error)` | Merges two tool configuration maps with MCP-aware conflict handling |
| `UpdateWorkflowFrontmatter` | `func(workflowPath string, updateFunc func(frontmatter map[string]any) error, verbose bool) error` | Reads, updates, and rewrites workflow frontmatter with a callback |
| `EnsureToolsSection` | `func(frontmatter map[string]any) map[string]any` | Ensures `tools` exists and is a map in frontmatter |
| `QuoteCronExpressions` | `func(yamlContent string) string` | Ensures schedule cron values in YAML are quoted |

### Constants / Variables

| Name | Type | Description |
|------|------|-------------|
| `ValidMCPTypes` | `[]string` | Valid MCP transport types: `"stdio"`, `"http"`, `"local"` |
| `IncludeDirectivePattern` | `*regexp.Regexp` | Matches `@import`, `@include`, and `{{#import ...}}` directives |
| `LegacyIncludeDirectivePattern` | `*regexp.Regexp` | Matches legacy `@import`/`@include` forms |
| `DefaultFileReader` | `FileReader` | Default file reader using `os.ReadFile` |
| `RepoConfigSchema` | `string` | Embedded JSON schema for repo-level configuration |

## Usage Examples

### Parse frontmatter from a workflow file

```go
content, _ := os.ReadFile("my-workflow.md")
result, err := parser.ExtractFrontmatterFromContent(string(content))
if err != nil {
    log.Fatal(err)
}
fmt.Println("Triggers:", result.Frontmatter["on"])
fmt.Println("Prompt:", result.MarkdownBody)
```

### Resolve imports

```go
cache := parser.NewImportCache("/path/to/repo")
imports, err := parser.ProcessImportsFromFrontmatterWithSource(
    result.Frontmatter,
    filepath.Dir("my-workflow.md"),
    cache,
    "my-workflow.md",
    result.FrontmatterYAML,
)
```

### Parse a schedule

```go
cron, original, err := parser.ParseSchedule("every day at 9am")
// cron = "0 9 * * *"
```

### Validate frontmatter

```go
err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(
    frontmatter, "my-workflow.md",
)
```

### Extract MCP server configurations

```go
servers, err := parser.ExtractMCPConfigurations(frontmatter, "")
for _, s := range servers {
    fmt.Printf("%s: type=%s\n", s.Name, s.Type)
}
```

## Architecture

The parsing pipeline for a workflow file proceeds as:

1. **Read** the raw markdown file content.
2. **Extract** the YAML frontmatter block between `---` delimiters (`ExtractFrontmatterFromContent`).
3. **Process imports**: resolve all `@import` directives recursively, merge imported YAML configurations, and deduplicate (`ProcessImportsFromFrontmatterWithSource`).
4. **Validate** the merged frontmatter against the JSON schema (`ValidateMainWorkflowFrontmatterWithSchemaAndLocation`).
5. **Expand includes** in the markdown body (`ExpandIncludesWithManifest`).
6. **Pass** the merged frontmatter and markdown body to `pkg/workflow` for compilation.

Import caching is crucial for performance and cycle detection. The `ImportCache` tracks visited paths within a single compilation run to prevent infinite recursion.

## Dependencies

**Internal**:
- `pkg/types` — `BaseMCPServerConfig`
- `pkg/logger` — debug logging

**External**:
- `github.com/santhosh-tekuri/jsonschema/v5` — JSON schema validation
- `go.yaml.in/yaml/v3` — YAML parsing
- `goccy/go-yaml` — YAML 1.1/1.2 compatible parsing (for GitHub Actions compatibility)

## Thread Safety

`ImportCache` is designed for use within a single goroutine per compilation run. Its internal map is not concurrency-safe. For concurrent compilations, create a separate `ImportCache` per compilation.

The `DefaultFileReader` variable is safe to read but MUST NOT be mutated after package initialization. Tests may replace it with a custom `FileReader` to inject virtual filesystem content.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
