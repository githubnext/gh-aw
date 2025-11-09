---
name: PR Status Check
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  contents: read
  statuses: write
safe-outputs:
  create-commit-status:
    context: "ai-review"
---

# AI Code Review Status

Analyze the pull request changes and create a commit status based on the code quality.

Evaluate the following:
1. Code follows best practices
2. No obvious bugs or security issues
3. Tests are included if needed
4. Documentation is updated if needed

After your analysis, use the `create-commit-status` tool to create a commit status with:
- state: "success" if the code looks good, "pending" if needs minor fixes, "failure" if has serious issues
- description: A brief summary of your findings (max 140 characters)
- target_url: (optional) Link to detailed analysis if you create an issue or discussion

Example status messages:
- "✓ Code looks good - well tested and documented"
- "⚠ Consider adding tests for new functionality"
- "✗ Found potential security issue in authentication code"
