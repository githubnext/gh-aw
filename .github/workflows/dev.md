---
on: 
  workflow_dispatch:
name: Dev
description: Build and test this project
timeout-minutes: 15
strict: false
sandbox: false
engine: copilot

permissions:
  contents: read
---

# Build and Test Project

Build and test the gh-aw project to ensure code quality.

**Requirements:**
1. Run `make build` to build the binary (this handles Go module downloads automatically)
2. Run `make test` to run the test suite
3. Report any failures with details about what went wrong
4. If all steps pass, confirm the build and tests completed successfully
