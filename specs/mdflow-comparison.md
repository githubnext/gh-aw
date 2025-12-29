# mdflow Syntax Comparison

**Document Type**: Analysis  
**Date**: 2025-12-29  
**Status**: Informational

## Purpose

This document provides a brief comparison between [mdflow](https://github.com/johnlindquist/mdflow) and GitHub Agentic Workflows syntax and design philosophy. For a comprehensive analysis, see [docs/MDFLOW_SYNTAX_COMPARISON.md](../docs/MDFLOW_SYNTAX_COMPARISON.md).

## Quick Comparison

### Design Philosophy

**mdflow**: Personal productivity tool for local AI task automation
- Unix philosophy (stdin/stdout, composition)
- Convention over configuration
- Direct CLI invocation
- Flexible, minimal validation

**GitHub Agentic Workflows**: Repository automation platform for teams
- Security-first design
- Explicit configuration
- Event-driven execution
- Structured, validated workflows

### Syntax Approach

| Aspect | mdflow | gh-aw |
|--------|--------|-------|
| **Filename** | Encodes command (`task.claude.md`) | Descriptive name (`issue-responder.md`) |
| **Engine** | Inferred from filename | Declared in frontmatter (`engine: copilot`) |
| **Frontmatter** | Maps to CLI flags | GitHub Actions + AI config |
| **Templates** | LiquidJS (`{{ _var }}`) | GitHub expressions (`${{ var }}`) |
| **Imports** | Rich (`@path`, globs, symbols, URLs) | Simple (file list in `imports:`) |
| **Execution** | CLI command (`mdflow file.md`) | GitHub webhook trigger |
| **Output** | Terminal stdout | GitHub API (safe-outputs) |
| **Security** | User permissions | Sandboxed, allowlisted |

## Key Insights

### What mdflow Does Well

1. **Simplicity**: Minimal configuration for quick tasks
2. **Composability**: Pipe and chain like Unix tools
3. **Flexibility**: Use any CLI AI tool
4. **Interactive Mode**: Work with AI conversationally
5. **Rich Imports**: Glob patterns, symbol extraction, command inlines

### What gh-aw Does Well

1. **Security**: Read-only default, explicit permissions, validation
2. **GitHub Integration**: Native Issues, PRs, Actions support
3. **Team Workflows**: Shared, auditable automation
4. **Compile-Time Validation**: Catch errors before execution
5. **Structured Outputs**: Safe-outputs with sanitization

## Potential Improvements for gh-aw

Based on mdflow's design, these features could enhance gh-aw:

1. **Enhanced Import System**:
   - Glob pattern support: `imports: ["shared/**/*.md"]`
   - Symbol extraction: Import specific functions/types from code
   - Command inlines: `` !`git log -5 --oneline` ``

2. **Template Improvements**:
   - Conditional includes: `{% if condition %}...{% endif %}`
   - Loop support: `{% for item in items %}...{% endfor %}`
   - Better string manipulation helpers

3. **Context Dashboard**:
   - Pre-compilation summary of what will be included
   - Token estimates for imports
   - Visual tree of file inclusions

4. **Dry-Run Enhancement**:
   - Show fully expanded prompt with all imports
   - Validation without compilation
   - Interactive edit before compiling

5. **Workflow Libraries**:
   - Shared workflow patterns across repositories
   - Version pinning for imported workflows
   - Workflow marketplace integration

## Design Differences and Trade-offs

### mdflow's Trade-offs

**Strengths**:
- Fast to write and execute
- Minimal boilerplate
- Great for personal use

**Weaknesses**:
- No compile-time validation
- Security relies on user caution
- Not designed for CI/CD
- Limited structured output

### gh-aw's Trade-offs

**Strengths**:
- Production-ready security
- Team collaboration features
- Comprehensive validation
- GitHub-native integration

**Weaknesses**:
- More verbose configuration
- Steeper learning curve
- Cannot run interactively
- Requires GitHub Actions

## Use Case Alignment

### Use mdflow When:
- Doing ad-hoc local tasks
- Quick code reviews or explanations
- Personal productivity automation
- Experimenting with prompts
- Need interactive AI sessions

### Use gh-aw When:
- Automating repository workflows
- Managing issues and PRs at scale
- Running scheduled reports
- Enforcing team policies
- Need audit trails and security

## Recommendations

1. **Keep Different Focus**: Both tools serve different needs - don't try to make gh-aw into mdflow or vice versa

2. **Selective Adoption**: Consider adopting mdflow's import patterns and template features where they enhance security and usability

3. **Document Differences**: Help users understand which tool fits their use case

4. **Cross-Learning**: Study mdflow's UX for inspiration on reducing configuration burden

5. **Maintain Security**: Any features adopted from mdflow must maintain gh-aw's security guarantees

## See Also

- [Full Comparison Document](../docs/MDFLOW_SYNTAX_COMPARISON.md) - Comprehensive analysis with examples
- [mdflow Repository](https://github.com/johnlindquist/mdflow) - Original project
- [Workflow Structure](../docs/src/content/docs/reference/workflow-structure.md) - gh-aw syntax reference
- [Frontmatter Reference](../docs/src/content/docs/reference/frontmatter.md) - gh-aw configuration options

---

**Last Updated**: 2025-12-29  
**Next Review**: When considering syntax changes or import system improvements
