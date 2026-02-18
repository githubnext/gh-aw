# Contributing to GitHub Agentic Workflows

Thank you for your interest in contributing to GitHub Agentic Workflows! We welcome contributions from the community and are excited to work with you.

**⚠️ IMPORTANT: This project requires agentic development using GitHub Copilot Agent. No local development environment is needed or expected.**

**🚫 Traditional Pull Requests Are Not Enabled**: You cannot create pull requests directly. Instead, you create detailed agentic plans in issues, and GitHub Copilot Agent will create and implement the PR for you after maintainer approval.

## 🤖 Agentic Development Workflow

GitHub Agentic Workflows is developed **exclusively through GitHub Copilot Agent**. This means:

- ✅ **All development happens through Copilot Agent** - agent creates and manages pull requests
- ✅ **No local setup required** - agents handle building, testing, and validation
- ✅ **Automated quality assurance** - CI runs all checks on agent-created PRs
- ❌ **Traditional pull requests are not enabled** - contributors craft agentic plans instead
- ❌ **Local development is not supported** - all work is done through the agent

### Why Agentic Development?

This project practices what it preaches: agentic workflows are used to build agentic workflows. Benefits include:

- **Consistency**: All changes go through the same automated quality gates
- **Accessibility**: No need to set up local development environments
- **Best practices**: Agents follow established patterns and guidelines automatically
- **Dogfooding**: We use our own tools to build our tools

## 🚀 Quick Start for Contributors

⚠️ **IMPORTANT: Traditional pull requests are not enabled for this repository.** Instead of creating a PR yourself, you craft a complete agentic plan that GitHub Copilot Agent will execute.

### Step 1: Analyze with an Agent (for bug reports)

**Before filing a contribution request**, use an agent to:

- Scan the source code to identify root causes (for bugs)
- Analyze execution patterns and trace the issue
- Research similar issues and solutions
- Propose specific fixes with code examples
- Create a comprehensive plan for the changes needed

### Step 2: Open an Issue with Your Agentic Plan

**Create an issue** with your detailed agentic plan:

- Describe what you want to contribute
- Include your agent's analysis and findings (for bugs)
- Explain the use case and expected behavior
- Provide a **complete, step-by-step plan** for the agent to follow
- Include specific implementation details and examples
- Tag with appropriate labels (see [Label Guidelines](scratchpad/labels.md))

See [Reporting Issues and Feature Requests](#reporting-issues-and-feature-requests) for complete guidelines.

**Example agentic plan in an issue:**

```markdown
## Add support for custom MCP server timeout configuration

### Analysis
The current MCP server configuration lacks a timeout field, which can cause workflows to hang indefinitely when servers don't respond.

### Implementation Plan
Please implement the following changes:

1. **Update Schema** (`pkg/parser/schemas/frontmatter.json`):
   - Add a `timeout` field to MCP server configuration schema
   - Type: integer
   - Range: 5-300 seconds
   - Default: 30 seconds

2. **Update Validation** (`pkg/workflow/mcp_validation.go`):
   - Add validation for timeout values between 5-300 seconds
   - Use error message format: "[what's wrong]. [what's expected]. [example]"
   - Example error: "timeout value 400 exceeds maximum. Expected value between 5-300 seconds. Example: timeout: 60"

3. **Add Tests** (`pkg/workflow/mcp_validation_test.go`):
   - Test valid timeout values (5, 30, 300)
   - Test invalid timeout values (0, 4, 301, 1000)
   - Test missing timeout (should use default)

4. **Update Documentation** (`docs/src/content/docs/reference/frontmatter.md`):
   - Add timeout field to MCP server configuration examples
   - Explain timeout behavior and defaults
   - Show example with custom timeout value

5. **Follow Guidelines**:
   - Use console formatting from `pkg/console` for CLI output
   - Follow error message style guide
   - Run `make agent-finish` before completing
```

### Step 3: Copilot Agent Will Be Assigned

Once your issue is reviewed and approved by a maintainer:

1. **Issue is assigned to Copilot Agent**: A maintainer will assign the issue to GitHub Copilot Agent
2. **Agent creates a PR**: The agent will create a pull request implementing your plan
3. **Agent executes your plan**: The agent will follow your step-by-step instructions
4. **Agent handles validation**: The agent will read guidelines, make changes, run tests
5. **Review and iterate**: The agent will respond to feedback and update the PR

### Step 4: Agent Handles Everything

The GitHub Copilot Agent will:

- Read your agentic plan from the issue
- Read relevant documentation and specifications
- Make code changes following established patterns
- Run `make agent-finish` to validate changes
- Format code, run linters, execute tests
- Recompile workflows to ensure compatibility
- Respond to review feedback and make adjustments

### No Local Setup Needed

You don't need to install Go, Node.js, or any dependencies. The agent runs in GitHub's infrastructure with all tools pre-configured. **You don't create a PR yourself** - instead, you create a detailed plan that the agent will execute.

## 📝 How to Contribute via GitHub Copilot Agent

All contributions are made through **agentic plans in GitHub issues**, which GitHub Copilot Agent then implements in pull requests. The agent has access to comprehensive documentation and follows established patterns automatically.

### How the Process Works

1. **You create an issue** with a detailed agentic plan describing what needs to be done
2. **Maintainer assigns the issue** to GitHub Copilot Agent after review
3. **Agent creates a PR** and implements your plan
4. **Agent follows your instructions** and handles all the technical details
5. **Maintainers review** and provide feedback
6. **Agent iterates** based on review comments until approved
7. **PR is merged** when all checks pass and reviews are satisfied

**You do not create pull requests yourself.** Instead, you craft comprehensive plans that the agent executes.

### What the Agent Handles

The GitHub Copilot Agent automatically:

- **Reads specifications** from `scratchpad/`, `skills/`, and `.github/instructions/`
- **Follows code organization patterns** (see [scratchpad/code-organization.md](scratchpad/code-organization.md))
- **Implements validation** following the architecture in [scratchpad/validation-architecture.md](scratchpad/validation-architecture.md)
- **Uses console formatting** from `pkg/console` for CLI output
- **Writes error messages** following the [Error Message Style Guide](.github/instructions/error-messages.instructions.md)
- **Runs all quality checks**: `make agent-finish` (build, test, recompile, format, lint)
- **Updates documentation** for new features
- **Creates tests** for new functionality

### Reporting Issues and Feature Requests

Before filing an issue, **use an agent to perform thorough analysis and research**. This accelerates implementation and helps maintainers focus on high-quality contributions.

#### 🤖 Use Agents for Bug Analysis

**Bug reports submitted with minimal analysis or research are likely to be ignored.**

Use an agent to analyze the source code, identify root causes, propose fixes, and research similar issues before filing a bug report.

#### 🐛 Debugging Workflow Failures

For workflow failures, use this prompt with your agent:

```markdown
Please debug this workflow failure:
https://github.com/owner/repo/actions/runs/RUN_ID

Load [https://github.com/github/gh-aw/.github/agents/agentic-workflows.agent.md and](https://github.com/github/gh-aw/blob/main/.github/agents/agentic-workflows.agent.md) investigate:
- Why the workflow failed
- What tools were missing
- How to fix the configuration

Generate an investigation report and a plan to address the issue for an agent.
```

The agent will use `gh aw` or the mcp server to analyze the failure. See [`.github/aw/debug-agentic-workflow.md`](.github/aw/debug-agentic-workflow.md) for details.

#### 📝 Issue Guidelines

When filing issues with agentic plans:

- **Bugs**: Include thorough agent analysis, root cause, proposed fix, and detailed implementation plan
- **Features**: Explain the use case, provide examples, suggest implementation approach with step-by-step instructions
- **Workflow failures**: Debug with agents first, then report with analysis and remediation plan
- **Implementation details**: Be specific about file changes, function names, validation rules, test cases
- **Complete plans**: The more detailed your plan, the better the agent can execute it
- Follow [Label Guidelines](scratchpad/labels.md)
- The agent will read the issue and implement your plan in a PR

**Quality of the agentic plan directly impacts implementation success.** Provide comprehensive, step-by-step instructions with specific details.

### Code Quality Standards

GitHub Copilot Agent automatically enforces:

#### Error Messages

All validation errors follow the template: **[what's wrong]. [what's expected]. [example]**

```go
// Agent produces error messages like this:
return fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d, +1w, +1mo. Example: +3d", deltaStr)
```

The agent runs `make lint-errors` to verify error message quality.

#### Console Output

The agent uses styled console functions from `pkg/console`:

```go
import "github.com/github/gh-aw/pkg/console"

fmt.Println(console.FormatSuccessMessage("Operation completed"))
fmt.Println(console.FormatInfoMessage("Processing workflow..."))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

#### File Organization

The agent follows these principles:

- **Prefer many small files** over large monolithic files
- **Group by functionality**, not by type
- **Use descriptive names** that clearly indicate purpose
- **Follow established patterns** from the codebase

**Key Patterns the Agent Uses**:

1. **Create Functions Pattern** - One file per GitHub entity creation
   - Examples: `create_issue.go`, `create_pull_request.go`, `create_discussion.go`

2. **Engine Separation Pattern** - Each engine has its own file
   - Examples: `copilot_engine.go`, `claude_engine.go`, `codex_engine.go`
   - Shared helpers in `engine_helpers.go`

3. **Focused Utilities Pattern** - Self-contained feature files
   - Examples: `expressions.go`, `strings.go`, `artifacts.go`

See [Code Organization Patterns](scratchpad/code-organization.md) for details.

#### Validation Patterns

The agent places validation logic appropriately:

**Centralized validation** (`pkg/workflow/validation.go`):

- Cross-cutting concerns
- Core workflow integrity
- GitHub Actions compatibility

**Domain-specific validation** (dedicated files):

- `strict_mode_validation.go` - Security enforcement
- `pip_validation.go` - Python packages
- `npm_validation.go` - NPM packages
- `docker_validation.go` - Docker images
- `expression_safety.go` - Expression security

See [Validation Architecture](scratchpad/validation-architecture.md) for the complete decision tree.

#### File Path Security

All file operations must validate paths to prevent path traversal attacks:

**Use `fileutil.ValidateAbsolutePath` before file operations:**

```go
import "github.com/github/gh-aw/pkg/fileutil"

// Validate path before reading/writing files
cleanPath, err := fileutil.ValidateAbsolutePath(userInputPath)
if err != nil {
    return fmt.Errorf("invalid path: %w", err)
}
content, err := os.ReadFile(cleanPath)
```

**Security checks performed:**
- Normalizes path using `filepath.Clean` (removes `.` and `..` components)
- Verifies path is absolute (blocks relative path traversal)
- Returns descriptive errors for invalid paths

**When to use:**
- Before `os.ReadFile`, `os.WriteFile`, `os.Stat`, `os.Open`
- Before `os.MkdirAll` or other directory operations
- After constructing paths with `filepath.Join`
- When processing user-provided file paths

This provides defense-in-depth against path traversal vulnerabilities (e.g., `../../../etc/passwd`).

#### CLI Breaking Changes

The agent evaluates whether changes are breaking:

- **Breaking**: Removing/renaming commands or flags, changing JSON output structure, altering defaults
- **Non-breaking**: Adding new commands/flags, adding output fields, bug fixes

For breaking changes, the agent:

- Uses `major` changeset type
- Provides migration guidance
- Documents in CHANGELOG.md

See [Breaking CLI Rules](scratchpad/breaking-cli-rules.md) for details.

## 🔄 Pull Request Process via GitHub Copilot Agent

All pull requests are created and managed by GitHub Copilot Agent based on agentic plans in issues:

1. **Create an issue with your agentic plan:**
   - Open an issue describing what needs to be done in detail
   - Provide a complete, step-by-step implementation plan
   - Include clear context, examples, and specific technical details
   - Tag appropriately using [Label Guidelines](scratchpad/labels.md)

2. **Maintainer reviews and assigns:**
   - Maintainer reviews your agentic plan
   - If approved, maintainer assigns the issue to GitHub Copilot Agent
   - Agent automatically creates a PR to implement your plan

3. **Agent creates and implements the PR:**
   - Agent reads your plan from the issue
   - Agent reads specifications and guidelines
   - Agent makes changes following established patterns
   - Agent runs `make agent-finish` automatically

4. **Automated quality checks:**
   - CI runs on agent-created PRs
   - All checks must pass (build, test, lint, recompile)
   - Agent responds to CI failures and fixes them

5. **Review and iterate:**
   - Maintainers review the PR
   - Provide feedback as comments
   - Agent responds to feedback and makes adjustments
   - Once approved, PR is merged

**Remember: You don't create the PR yourself.** You create an issue with a detailed plan, and after approval, the agent creates the PR and implements your plan.

### What Gets Validated

Every agent-created PR automatically runs:

- `make build` - Ensures Go code compiles
- `make test` - Runs all unit and integration tests
- `make lint` - Checks code quality and style
- `make recompile` - Recompiles all workflows to ensure compatibility
- `make fmt` - Formats Go code
- `make lint-errors` - Validates error message quality

## 🏗️ Project Structure (For Agent Reference)

The agent understands this structure:

```text
/
├── cmd/gh-aw/           # Main CLI application
├── pkg/                 # Core Go packages
│   ├── cli/             # CLI command implementations
│   ├── console/         # Console formatting utilities
│   ├── parser/          # Markdown frontmatter parsing
│   └── workflow/        # Workflow compilation and processing
├── scratchpad/               # Technical specifications the agent reads
├── skills/              # Specialized knowledge for agents
├── .github/             # Instructions and sample workflows
│   ├── instructions/    # Agent instructions
│   └── workflows/       # Sample workflows and CI
└── Makefile             # Build automation (agent uses this)
```

## 📋 Dependency License Policy

This project uses an MIT license and only accepts dependencies with compatible licenses.

### Allowed Licenses

The following open-source licenses are compatible with our MIT license:

- **MIT** - Most permissive, allows reuse with minimal restrictions
- **Apache-2.0** - Permissive license with patent grant
- **BSD-2-Clause, BSD-3-Clause** - Simple permissive licenses
- **ISC** - Simplified permissive license similar to MIT

### Disallowed Licenses

The following licenses are **not allowed** as they conflict with our MIT license or impose unacceptable restrictions:

- **GPL, LGPL, AGPL** - Copyleft licenses that would force us to release under GPL
- **SSPL** - Server Side Public License with restrictive requirements
- **Proprietary/Commercial** - Closed-source licenses requiring payment or special terms

### Before Adding a Dependency

GitHub Copilot Agent automatically checks licenses when adding dependencies. However, if you're evaluating a dependency:

1. **Check its license**: Run `make license-check` after adding the dependency
2. **Review the report**: Run `make license-report` to generate a CSV of all licenses
3. **If unsure**: Ask in your PR - maintainers will help evaluate edge cases

### License Checking

The project includes automated license compliance checking:

- **CI Workflow**: `.github/workflows/license-check.yml` runs on every PR that changes `go.mod`
- **Local Check**: Run `make license-check` to verify all dependencies (installs `go-licenses` on-demand)
- **License Report**: Run `make license-report` to see detailed license information

All dependencies are automatically scanned using Google's `go-licenses` tool in CI, which classifies licenses by type and identifies potential compliance issues. Note that `go-licenses` is not actively maintained, so we install it on-demand rather than as a regular build dependency.

## 🤖 Automated Dependency Updates (Dependabot)

This project uses GitHub Dependabot to automatically keep dependencies up-to-date with weekly security patches and version updates.

### What Dependabot Monitors

Dependabot is configured in `.github/dependabot.yml` to monitor:

1. **Go modules** (`/go.mod`) - Weekly updates for Go dependencies
2. **npm packages** - Weekly updates for:
   - Documentation site (`/docs/package.json`)
   - GitHub Actions setup scripts (`/actions/setup/js/package.json`)
   - Workflow dependencies (`/.github/workflows/package.json`)
3. **Python packages** (`/.github/workflows/requirements.txt`) - Weekly updates for workflow scripts

### Expected Behavior

- **Schedule**: Dependabot checks for updates **every Monday** (weekly interval)
- **Pull Requests**: Creates automated PRs from `dependabot[bot]` for:
  - Security vulnerabilities (immediate)
  - Version updates (weekly batch)
- **Limit**: Maximum of 10 open PRs per ecosystem to prevent overwhelming maintainers

### What to Expect from Dependabot PRs

Dependabot PRs will:
- Have clear titles like "Bump lodash from 4.17.20 to 4.17.21 in /docs"
- Include changelog links and release notes
- Show compatibility score based on semantic versioning
- Automatically rebase when the base branch changes

### Troubleshooting Dependabot

If Dependabot stops creating PRs:

1. **Check repository settings**: Go to Settings → Security → Dependabot
   - Ensure "Dependabot alerts" is enabled
   - Ensure "Dependabot security updates" is enabled
   - Ensure "Dependabot version updates" is enabled

2. **Verify configuration**: Check `.github/dependabot.yml` syntax
   - Directory paths must match locations of dependency files
   - Ecosystem names must be exact: `gomod`, `npm`, `pip`

3. **Check for rate limits**: Dependabot may be rate-limited if there are too many updates

4. **Manual trigger**: You can manually trigger Dependabot from repository Settings → Security → Dependabot

### Handling Dependabot PRs

When reviewing Dependabot PRs:

1. **Review the changes**: Check the changelog and compatibility score
2. **Let CI run**: Wait for all GitHub Actions checks to pass
3. **Test if needed**: For major version updates, test locally or let the agent verify
4. **Merge quickly**: Security updates should be merged as soon as CI passes
5. **Batch updates**: For minor version updates, you can merge multiple PRs at once

### Security Patches

Dependabot prioritizes security patches:
- Security vulnerabilities are updated **immediately** (not weekly)
- PRs are tagged with severity level (critical, high, medium, low)
- Security PRs should be reviewed and merged within 24-48 hours

## 🧪 Testing

For comprehensive testing guidelines including assert vs require usage, table-driven test patterns, and best practices, see **[scratchpad/testing.md](scratchpad/testing.md)**.

Quick reference:
- `make test-unit` - Fast unit tests (~25s)
- `make test` - Full test suite (~30s)
- `make agent-finish` - Complete validation before committing

## 🚫 Spam Prevention

**Be nice, don't spam.** The project maintainers reserve the right to clean up spam, unsolicited promotions, or off-topic content as needed to keep discussions focused and valuable for all contributors.

This includes but is not limited to:
- Repeated identical or similar comments across multiple issues or pull requests
- Unsolicited promotional content or advertisements
- Off-topic comments that don't contribute to the discussion
- Automated bot comments without prior approval

## 🤝 Community

- Join the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)
- Participate in discussions on GitHub issues
- Collaborate through GitHub Copilot Agent PRs

## 📜 Code of Conduct

This project follows the GitHub Community Guidelines. Please be respectful and inclusive in all interactions.

## ❓ Getting Help

- **For bugs or features**: Open a GitHub issue with a detailed agentic plan
- **For questions**: Ask in issues, discussions, or Discord
- **For examples**: Look at existing issues and agent-created PRs
- **Remember**: You don't create PRs - you create issues with plans that agents implement

## 🎯 Why No Traditional Pull Requests?

This project is built using agentic workflows to demonstrate their capabilities:

- **Dogfooding**: We use our own tools to build our tools
- **Accessibility**: No need for complex local setup or Git workflows
- **Consistency**: All changes go through the same automated process
- **Best practices**: Agents follow guidelines automatically
- **Focus on outcomes**: Describe what you want, not how to build it
- **Quality plans**: Forces contributors to think through the entire implementation before starting

**Traditional PRs would bypass the agentic workflow**, defeating the purpose of this project. By crafting detailed agentic plans in issues, you're participating in the future of software development.

The [Development Guide](DEVGUIDE.md) exists as reference for the agent, not for local setup.

Thank you for contributing to GitHub Agentic Workflows! 🤖🎉
