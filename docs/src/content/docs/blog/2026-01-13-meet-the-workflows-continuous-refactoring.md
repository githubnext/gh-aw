---
title: "Meet the Workflows: Continuous Refactoring"
description: "Agents that identify structural improvements and systematically refactor code"
authors:
  - dsyme
  - pelikhan
  - mnkiefer
date: 2026-01-13T02:15:00
sidebar:
  label: "Continuous Refactoring"
prev:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-continuous-simplicity/
  label: "Continuous Simplicity Workflows"
next:
  link: /gh-aw/blog/2026-01-13-meet-the-workflows-continuous-style/
  label: "Continuous Style Workflows"
---

<img src="/gh-aw/peli.png" alt="Peli de Halleux" width="200" style="float: right; margin: 0 0 20px 20px; border-radius: 8px;" />

Welcome back to [Peli's Agent Factory](/gh-aw/blog/2026-01-12-welcome-to-pelis-agent-factory/)!

In our [previous post](/gh-aw/blog/2026-01-13-meet-the-workflows-continuous-simplicity/), we met a core concept of using automated agents in software repositories: agents that detect complexity and propose simpler solutions. These workflows trail behind development, cleaning up unnecessary complexity and duplicate code.

Now let's explore similar agents that take a deeper structural view. As explained in the [Continuous Simplicity post](/gh-aw/blog/2026-01-13-meet-the-workflows-continuous-simplicity/), autonomous cleanup agents work tirelessly in the background. Here we see how that principle extends to *structural refactoring* - systematically improving code organization, identifying misplaced functions, and enforcing consistent patterns across the entire codebase.

## Continuous Refactoring Workflows

These agents analyze code structure and suggest systematic improvements:

- **[Semantic Function Refactor](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/semantic-function-refactor.md?plain=1)** - Spots refactoring opportunities we might have missed  
- **[Go Pattern Detector](https://github.com/githubnext/gh-aw/tree/bb7946527af340043f1ebb31fc21bd491dd0f42d/.github/workflows/go-pattern-detector.md?plain=1)** - Detects common Go patterns and anti-patterns for consistency  

### Semantic Function Refactor: The Structure Analyzer

The **Semantic Function Refactor** workflow embodies one of the most powerful capabilities of AI agents: holding an entire codebase in context. It analyzes all Go source files in the `pkg/` directory to identify functions that might be in the wrong place.

As codebases evolve, functions sometimes end up in files where they don't quite belong. A utility function gets added to a specific feature file, or several files end up with similar responsibilities. Humans struggle to notice these organizational issues because we work on one file at a time, can't hold all function names and purposes in memory, and focus on making code work rather than on where it lives.

The workflow performs comprehensive discovery by

1. collecting all function names from non-test Go files, then
2. grouping functions semantically by name and purpose.

It then identifies functions that don't fit their current file's theme as outliers, uses Serena-powered semantic code analysis to detect potential duplicates, and creates single consolidated refactoring issues.

The workflow follows a "one file per feature" principle: files should be named after their primary purpose, and functions within each file should align with that purpose.

The workflow discovers patterns like helper functions scattered across feature files that should really be in `utils/`, similar validation logic that could be consolidated, functions with names suggesting they belong in different packages, and test utilities that somehow ended up in production code files.

The workflow first closes existing open issues with the `[refactor]` prefix before creating new ones. This prevents issue accumulation and ensures recommendations stay current. It creates at most 1 new issue per run, limiting noise while maintaining continuous analysis.

### Go Pattern Detector: The Consistency Enforcer

The **Go Pattern Detector** uses ast-grep to scan for specific code patterns and anti-patterns. Unlike semantic analysis that understands meaning, this workflow uses abstract syntax tree (AST) pattern matching to find exact structural patterns.

Currently the workflow detects use of `json:"-"` tags in Go structs - a pattern that can indicate fields that should be private but aren't, serialization logic that could be cleaner, or potential API design issues.

The workflow runs in two phases. First, AST scanning runs on a standard GitHub Actions runner:

```bash
# Install ast-grep
cargo install ast-grep --locked

# Scan for patterns
sg --pattern 'json:"-"' --lang go .
```

If patterns are found, it triggers the second phase where the coding agent analyzes the detected patterns, reviews context around each match, determines if patterns are problematic, and creates issues with specific recommendations.

This architecture is brilliant for efficiency. Fast AST scanning uses minimal resources, expensive AI analysis only runs when needed, false positives don't consume AI budget, and the approach scales to frequent checks without cost concerns.

The workflow is designed to be extended with additional pattern checks - common anti-patterns like ignored errors or global state, project-specific conventions, performance anti-patterns, and security-sensitive patterns.

### Duplicate Code Detector: The Bridge

As we saw in the [Continuous Simplicity post](/gh-aw/blog/2026-01-13-meet-the-workflows-continuous-simplicity/), the **Duplicate Code Detector** uses semantic analysis to find duplicated patterns. This serves dual purposes: identifying obvious duplication that can be immediately consolidated, and spotting structural patterns suggesting deeper architectural improvements.

When duplicate patterns span multiple packages or involve different abstractions, the solution isn't simple extraction - it requires thoughtful refactoring to find the right abstraction level and ownership.

## The Power of Continuous Refactoring

These workflows demonstrate how AI agents can maintain institutional knowledge about code organization. Semantic Function Refactor ensures functions live in logical files, Go Pattern Detector enforces consistent idioms, and Duplicate Code Detector identifies both simple and complex duplication. The agents identify patterns across the entire codebase while humans make architectural decisions about abstractions. Issues provide specific file references and code examples, and changes go through normal review processes.

The benefits compound over time: better organization makes code easier to find, consistent patterns reduce cognitive load, reduced duplication improves maintainability, and clean structure attracts further cleanliness.

The workflows never stop learning. As the codebase evolves, they continuously analyze new code, identifying patterns that emerged since the last run. They're particularly valuable in AI-assisted development, where code gets written quickly and organizational concerns can take a backseat to functionality.

## Next Up: Continuous Style

Beyond structure and organization, there's another dimension of code quality: presentation and style. How do we maintain beautiful, consistent console output and formatting?

Continue reading: [Continuous Style Workflows →](/gh-aw/blog/2026-01-13-meet-the-workflows-continuous-style/)

## Learn More

- **[GitHub Agentic Workflows](https://githubnext.github.io/gh-aw/)** - The technology behind the workflows
- **[Quick Start](https://githubnext.github.io/gh-aw/setup/quick-start/)** - How to write and compile workflows

---

*This is part 3 of a 19-part series exploring the workflows in Peli's Agent Factory.*
