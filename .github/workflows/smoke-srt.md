---
description: Smoke test workflow for Sandbox Runtime (SRT) - validates SRT functionality with Copilot
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["test-srt"]
permissions:
  contents: read
  issues: read
name: Smoke SRT
engine: copilot
network:
  allowed:
    - defaults
    - github
    - "*.githubcopilot.com"
    - "example.com"
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
    enableWeakerNestedSandbox: true
tools:
  bash:
  github:
timeout-minutes: 5
strict: true
---

You are testing the Sandbox Runtime (SRT) integration. Perform the following tasks:

1. Run `echo "Hello from SRT!"` using bash
2. Check the current directory with `pwd`
3. List files in the current directory with `ls -la`

Report the results in a friendly summary. This is just a smoke test to validate that SRT is working correctly.
