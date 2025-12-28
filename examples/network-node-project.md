---
on:
  workflow_dispatch:
    inputs:
      package_name:
        description: 'npm package to analyze'
        required: true
        default: 'express'

permissions:
  contents: read

engine: copilot

network:
  allowed:
    - defaults    # Basic infrastructure (certificates, Ubuntu)
    - node        # npm, yarn, pnpm, registry.npmjs.org
    - github      # GitHub API and resources

tools:
  bash:

safe-outputs:
  create-issue:
    title-prefix: "[node-analysis] "
    labels: [automation, analysis]
---

Analyze the npm package "{{ inputs.package_name }}" by:

1. Using npm to fetch package information from the npm registry
2. Checking the package's dependencies and peer dependencies
3. Reviewing package metadata (version, maintainers, license)
4. Identifying any security vulnerabilities using npm audit
5. Creating a summary issue with your findings

The network configuration allows access to:
- **defaults**: Basic infrastructure for certificates and system packages
- **node**: Full Node.js ecosystem (registry.npmjs.org, yarnpkg.com, nodejs.org)
- **github**: GitHub API for creating the issue

This demonstrates proper network configuration for Node.js development workflows.
