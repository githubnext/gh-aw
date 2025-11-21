---
description: Smoke test workflow for Sandbox Runtime (SRT) with custom configuration
on:
  workflow_dispatch:
permissions:
  contents: read
name: Smoke SRT Custom Config
engine: copilot
network:
  allowed:
    - defaults
    - github
    - "*.npmjs.org"
sandbox:
  type: sandbox-runtime
  config:
    network:
      allowedDomains:
        - "*.githubusercontent.com"
        - "*.github.com"
        - "*.githubcopilot.com"
        - "api.enterprise.githubcopilot.com"
        - "api.github.com"
        - "github.com"
        - "example.com"
        - "*.npmjs.org"
        - "npmjs.org"
        - "registry.npmjs.org"
        - "*.npmmirror.com"
        - "registry.npmmirror.com"
        - "*.yarnpkg.com"
        - "registry.yarnpkg.com"
        - "*.cloudflare.com"
        - "*.cloudfront.net"
        - "*.fastly.net"
        - "*.akamaized.net"
        - "*.edgekey.net"
        - "*.edgesuite.net"
        - "*.azureedge.net"
        - "*.azure.com"
        - "*.windows.net"
        - "*.live.com"
        - "*.microsoft.com"
        - "*.visualstudio.com"
        - "*.vscode-cdn.net"
        - "*.vscode.dev"
        - "*.anaconda.org"
        - "*.anaconda.com"
        - "*.conda.io"
        - "*.pypi.org"
        - "pypi.org"
        - "files.pythonhosted.org"
        - "*.python.org"
        - "python.org"
        - "*.rubygems.org"
        - "rubygems.org"
        - "*.crates.io"
        - "crates.io"
        - "static.crates.io"
        - "index.crates.io"
        - "*.rust-lang.org"
        - "rust-lang.org"
        - "*.golang.org"
        - "golang.org"
        - "proxy.golang.org"
        - "sum.golang.org"
        - "*.go.dev"
        - "go.dev"
        - "pkg.go.dev"
        - "*.maven.org"
        - "maven.org"
        - "repo.maven.apache.org"
        - "repo1.maven.org"
        - "*.gradle.org"
        - "gradle.org"
        - "services.gradle.org"
        - "*.jitpack.io"
        - "jitpack.io"
        - "*.sonatype.org"
        - "oss.sonatype.org"
        - "*.digicert.com"
        - "digicert.com"
        - "*.letsencrypt.org"
        - "letsencrypt.org"
        - "acme-v02.api.letsencrypt.org"
      deniedDomains: []
      allowUnixSockets:
        - "/var/run/docker.sock"
      allowLocalBinding: true
    filesystem:
      denyRead: []
      allowWrite:
        - "."
        - "/tmp"
        - "/home/runner/.copilot"
        - "/home/runner"
      denyWrite: []
    ignoreViolations: {}
    enableWeakerNestedSandbox: true
tools:
  bash:
  github:
timeout-minutes: 5
strict: true
---

You are testing the Sandbox Runtime (SRT) with custom configuration. Perform the following tasks:

1. Run `echo "Testing SRT with custom config"` using bash
2. Verify you can access GitHub by running a simple github tool operation
3. Check the environment with `env | grep -i copilot`

Report your findings. This validates that SRT works with custom configuration.
