---
"gh-aw": minor
---

Add simonw/llm CLI integration with issue triage workflow

This adds support for using the simonw/llm CLI tool as a custom agentic engine in GitHub Agentic Workflows, with a complete issue triage workflow example. The integration includes:

- A reusable shared component (`.github/workflows/shared/simonw-llm.md`) that enables any workflow to use simonw/llm CLI as its execution engine
- Support for multiple LLM providers: OpenAI, Anthropic Claude, and GitHub Models (free tier)
- Automatic configuration and plugin management
- Safe-outputs integration for GitHub API operations
- An example workflow (`issue-triage-llm.md`) demonstrating automated issue triage
- Comprehensive documentation with setup instructions and examples
- Support for both automatic triggering (on issue opened) and manual workflow dispatch
