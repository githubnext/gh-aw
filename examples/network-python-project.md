---
on:
  workflow_dispatch:
    inputs:
      package_name:
        description: 'Python package to analyze'
        required: true
        default: 'requests'

permissions:
  contents: read

engine: copilot

network:
  allowed:
    - defaults    # Basic infrastructure (certificates, Ubuntu)
    - python      # PyPI, pip, conda, pythonhosted.org
    - github      # GitHub API and resources

tools:
  bash:

safe-outputs:
  create-issue:
    title-prefix: "[python-analysis] "
    labels: [automation, analysis]
---

Analyze the Python package "{{ inputs.package_name }}" by:

1. Using pip to fetch package information from PyPI
2. Checking the package's dependencies
3. Reviewing basic package metadata (version, author, license)
4. Creating a summary issue with your findings

The network configuration allows access to:
- **defaults**: Basic infrastructure for certificates and Ubuntu packages
- **python**: Full Python ecosystem (PyPI, pip, files.pythonhosted.org, conda)
- **github**: GitHub API for creating the issue

This demonstrates proper network configuration for Python development workflows.
