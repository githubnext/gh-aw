---
"githubnext/gh-aw": minor
---

Add automatic runtime setup detection and insertion for workflow steps

This implements automatic insertion of runtime setup steps (actions/setup-node, actions/setup-python, actions/setup-go, etc.) based on detected commands in custom steps and MCP configurations. The system now automatically detects when runtimes are needed, selects appropriate versions, and inserts setup steps in the correct order before custom steps.

Key features:
- Automatic detection of runtime requirements from commands (npm, python, uv, go, ruby, dotnet, java, elixir, haskell, etc.)
- Smart skipping when users already have setup actions
- Runtime struct architecture for centralized configuration
- Support for 9 different programming languages/tools
- Alphabetically sorted runtime setup steps for consistency
