---
on:
  workflow_dispatch:
    inputs:
      project_url:
        description: 'GitHub repository URL to analyze'
        required: true

permissions:
  contents: read

engine: copilot

network:
  allowed:
    - defaults      # Basic infrastructure
    - python        # PyPI and Python packages
    - node          # npm and Node.js packages
    - go            # Go modules
    - containers    # Docker registries
    - github        # GitHub API

tools:
  bash:

safe-outputs:
  create-issue:
    title-prefix: "[multi-lang-analysis] "
    labels: [automation, analysis]
---

Analyze the project at "{{ inputs.project_url }}" to identify:

1. Which programming languages are used
2. What package managers are present (requirements.txt, package.json, go.mod, Dockerfile)
3. Dependencies from each ecosystem
4. Potential security vulnerabilities
5. Create a comprehensive analysis issue

The network configuration allows access to multiple ecosystems:
- **defaults**: Basic infrastructure (certificates, Ubuntu mirrors)
- **python**: Python packages (pypi.org, files.pythonhosted.org)
- **node**: Node.js packages (registry.npmjs.org, yarnpkg.com)
- **go**: Go modules (proxy.golang.org, sum.golang.org)
- **containers**: Container registries (registry.hub.docker.com, ghcr.io)
- **github**: GitHub API for repository access

This demonstrates network configuration for full-stack projects using multiple languages and tools.
