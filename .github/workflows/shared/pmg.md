---
pre-steps:
  - name: Install PMG (Package Manager Guard)
    uses: safedep/pmg@v1
---
<!--
# PMG — Package Manager Guard

This shared workflow installs [PMG](https://github.com/safedep/pmg) — a supply chain security
tool that intercepts `npm`, `pip`, `poetry`, `yarn`, `bun`, `uv`, and other package managers to
block malicious packages **before** they execute.

PMG provides:
- **Malware blocking** — checks every package against SafeDep's real-time threat intelligence
- **Dependency cooldown** — blocks package versions published within a configurable window
- **Transparent interception** — wraps the package managers you already use (no workflow changes)

## Usage

Add as the **first** `imports:` entry in any workflow that installs third-party packages so
PMG's shims are in place before any `steps:` package installs run:

```yaml
imports:
  - shared/pmg.md       # must be first
  - shared/other.md
```

## References

- https://github.com/safedep/pmg
- https://github.com/safedep/pmg/blob/main/docs/github-action.md
-->
