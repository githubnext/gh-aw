---
# Sentrux - Code Architecture Quality Sensor
# Installs the sentrux binary for codebase structure analysis and quality scoring.
# Pinned to a specific version for supply chain security.
#
# Documentation: https://github.com/sentrux/sentrux
#
# Available CLI commands (use via bash):
#   sentrux check .               # check rules — exits 0 (pass) or 1 (fail)
#   sentrux gate --save .         # save baseline before a session
#   sentrux gate .                # compare after changes — detects degradation
#   sentrux --version             # print version
#
# To upgrade: update SENTRUX_VERSION in the install step to the desired release tag.
# Releases: https://github.com/sentrux/sentrux/releases
#
# Usage:
#   imports:
#     - shared/mcp/sentrux.md

steps:
  - name: Install sentrux
    run: |
      SENTRUX_VERSION="0.9.2"
      curl -fsSL -o /tmp/sentrux "https://github.com/sentrux/sentrux/releases/download/v${SENTRUX_VERSION}/sentrux-linux-x86_64"
      chmod +x /tmp/sentrux
      sudo mv /tmp/sentrux /usr/local/bin/sentrux
      sentrux --version
---

<!--

# Sentrux
# Architecture quality sensor for AI-assisted development
# Documentation: https://github.com/sentrux/sentrux

Sentrux scans codebase structure and computes a continuous quality signal (0–10000) from
5 root-cause metrics: modularity, acyclicity, depth, equality, and redundancy.

The binary is installed as a setup step and available in PATH. Invoke it directly with
the sentrux CLI to measure quality, check architectural rules, and detect drift.

Supported languages: 52 via tree-sitter plugins (Go, Rust, TypeScript, JavaScript, Python,
Java, C, C++, C#, Ruby, PHP, Swift, Kotlin, Scala, and more).

Note: sentrux uses stdio MCP transport which is not supported by the MCP Gateway.
Use sentrux CLI commands directly in workflow prompts.

-->

Use the `sentrux` binary to analyze the codebase at `${{ github.workspace }}`. The binary is installed and available in `PATH`.

**Check architectural rules (exits 0 on pass, 1 on failure):**
```bash
cd ${{ github.workspace }} && sentrux check .
```

**Save a quality baseline (before changes):**
```bash
cd ${{ github.workspace }} && sentrux gate --save .
```

**Compare quality after changes (detects degradation):**
```bash
cd ${{ github.workspace }} && sentrux gate .
```

Capture the JSON output for reporting:
```bash
cd ${{ github.workspace }} && sentrux check . --json 2>/dev/null || true
```
