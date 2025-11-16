---
description: Conventional Commits Guidelines for GitHub Agentic Workflows
---

# Conventional Commits

This repository follows the [Conventional Commits](https://www.conventionalcommits.org/) specification for all commit messages.

## Commit Message Format

Each commit message must be structured as follows:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Type

The type must be one of the following:

- **feat**: A new feature for users or a significant improvement
- **fix**: A bug fix for users
- **docs**: Documentation only changes
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- **refactor**: A code change that neither fixes a bug nor adds a feature
- **perf**: A code change that improves performance
- **test**: Adding missing tests or correcting existing tests
- **build**: Changes that affect the build system or external dependencies
- **ci**: Changes to our CI configuration files and scripts (GitHub Actions workflows)
- **chore**: Other changes that don't modify src or test files
- **revert**: Reverts a previous commit

### Scope

The scope is optional and should be a noun describing a section of the codebase surrounded by parentheses:

- `feat(compiler)`: Add support for new frontmatter field
- `fix(cli)`: Correct log output formatting
- `docs(readme)`: Update installation instructions
- `refactor(workflow)`: Extract validation logic to separate file

Common scopes in this project:
- `cli` - Command-line interface code in `pkg/cli/`
- `workflow` - Workflow compilation and processing in `pkg/workflow/`
- `parser` - Markdown and frontmatter parsing in `pkg/parser/`
- `console` - Console output formatting
- `docs` - Documentation files
- `engine` - AI engine implementations (copilot, claude, codex)
- `mcp` - Model Context Protocol server integration
- `validation` - Validation logic
- `safe-outputs` - Safe output processing
- `github-api` - GitHub API integration

### Description

The description is a short summary of the code changes:

- Use the imperative, present tense: "change" not "changed" nor "changes"
- Don't capitalize the first letter
- No period (.) at the end
- Limit to 72 characters or less

### Body

The body is optional and should provide additional contextual information about the code changes:

- Use the imperative, present tense
- Explain the motivation for the change and contrast with previous behavior
- Wrap at 72 characters

### Footer

The footer is optional and should contain any information about breaking changes or reference issues:

- **BREAKING CHANGE**: A commit that has a footer `BREAKING CHANGE:`, or appends a `!` after the type/scope, introduces a breaking API change
- Reference GitHub issues: `Fixes #123`, `Closes #456`, `Refs #789`

## Examples

### Simple feature
```
feat(cli): add --verbose flag for detailed output
```

### Bug fix with scope and body
```
fix(compiler): prevent panic when frontmatter is empty

The compiler would panic when processing workflow files with empty
frontmatter. Add validation to check for empty frontmatter and return
a meaningful error message instead.

Fixes #1234
```

### Breaking change
```
feat(workflow)!: change default timeout from 10 to 5 minutes

BREAKING CHANGE: The default workflow timeout has been reduced from 10
minutes to 5 minutes to better align with typical workflow execution
times. Users who need longer timeouts should explicitly set the
timeout-minutes field in their workflow frontmatter.
```

### Documentation change
```
docs(setup): update CLI installation instructions

Add instructions for installing on Windows using winget package
manager. Include troubleshooting section for common authentication
issues.
```

### Refactoring
```
refactor(validation): extract expression safety validation to separate file

Move expression safety validation logic from validation.go to new
expression_safety.go file for better code organization. No functional
changes.
```

### Multiple changes
```
feat(mcp): add support for HTTP MCP servers with custom headers

- Add HTTP MCP server type configuration
- Support custom headers with secret interpolation
- Add validation for HTTP MCP server configuration
- Update documentation with HTTP MCP examples

Closes #2345
```

## Best Practices

1. **Keep commits focused**: Each commit should represent a single logical change
2. **Write clear descriptions**: Make it easy for others to understand what changed and why
3. **Use the right type**: Choose the type that best describes the nature of the change
4. **Reference issues**: Always reference the GitHub issue number when applicable
5. **Be specific with scope**: Use specific scopes to help locate changes in the codebase
6. **Explain breaking changes**: Clearly document any breaking changes in the footer

## Benefits

Following conventional commits provides several benefits:

- **Automated changelog generation**: Tools can automatically generate changelogs
- **Easier navigation**: Team members can quickly understand the nature of changes
- **Automated versioning**: Semantic version bumps can be determined automatically
- **Better collaboration**: Clear commit messages improve team communication
- **Simpler code review**: Reviewers can understand changes more quickly

## Tools

To help write conventional commits, you can use:

- **commitizen**: Interactive CLI for creating conventional commits
  ```bash
  npm install -g commitizen cz-conventional-changelog
  ```

- **commitlint**: Linter for commit messages
  ```bash
  npm install -g @commitlint/cli @commitlint/config-conventional
  ```

## Enforcement

GitHub Copilot will suggest commit messages following this format. When making commits, follow these guidelines to maintain consistency across the repository.

## References

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Angular Commit Message Guidelines](https://github.com/angular/angular/blob/main/CONTRIBUTING.md#commit)
- [Semantic Versioning](https://semver.org/)
