---
description: Configure and use the built-in Playwright CLI integration in GitHub Agentic Workflows.
---

# Playwright

Use the built-in `playwright` tool for browser automation, accessibility checks,
end-to-end flows, and visual regression testing. The integration uses
`@playwright/cli`; it does not expose Playwright MCP tools.

## Configure the tool

Enable Playwright in workflow frontmatter:

```yaml
tools:
  playwright:
```

The compiler installs the pinned default `@playwright/cli` package and its agent
skills. Do not add installation steps to the workflow. Pin `version` only when
reproducible browser output is required, such as for visual baselines:

```yaml
tools:
  playwright:
    version: "0.1.18"
```

`mode: cli` is accepted but unnecessary. `mode: mcp` is not supported by the
built-in tool. If MCP is required, configure and pin `@playwright/mcp` explicitly
under `mcp-servers` and allow only the required tools.

## Configure network access

Playwright can reach `localhost` and `127.0.0.1` by default. Add only the
ecosystems and external domains the browser needs:

```yaml
network:
  allowed:
    - defaults
    - playwright
    - "docs.example.com"
```

The `playwright` ecosystem permits browser downloads. An explicit domain also
permits its subdomains. Prefer a local server over an external preview, and
avoid broad wildcard domains.

## Use Playwright CLI

Run `playwright-cli` through bash. Start with a snapshot and use its element refs
for later actions:

```bash
playwright-cli open "https://docs.example.com"
playwright-cli snapshot
playwright-cli click e15
playwright-cli fill e22 "search text" --submit
playwright-cli screenshot --filename=/tmp/docs.png
playwright-cli close
```

Useful commands include:

| Goal | Command |
|---|---|
| Open a browser and URL | `playwright-cli open <url>` |
| Navigate the open page | `playwright-cli goto <url>` |
| Inspect the page and get refs | `playwright-cli snapshot` |
| Limit snapshot size | `playwright-cli snapshot --depth=4` |
| Click or fill an element | `playwright-cli click <ref>` / `playwright-cli fill <ref> <text>` |
| Evaluate JavaScript | `playwright-cli eval "() => document.title"` |
| Capture a screenshot | `playwright-cli screenshot --filename=<path>` |
| Return only the command result | `playwright-cli --raw <command>` |
| Close the browser | `playwright-cli close` |

Prefer refs from the latest snapshot over brittle CSS selectors. Use `--raw`
when piping a result or comparing snapshots so page status output does not
pollute the data.

## Run against a local application

Start the application inside the agent sandbox, bind it only to `127.0.0.1`,
and poll a health endpoint before opening the page. Do not use a fixed sleep.

```yaml
steps:
  - name: Start application
    working-directory: ./web
    run: |
      npm ci
      npm run dev -- --host 127.0.0.1 &
      for attempt in $(seq 1 30); do
        curl --fail --silent http://127.0.0.1:4321/ >/dev/null && exit 0
        sleep 1
      done
      exit 1

tools:
  playwright:
  bash:
    - "curl http://127.0.0.1:*"

network:
  allowed:
    - defaults
    - local
    - node
    - playwright
```

Then direct the agent to use the loopback URL:

```bash
playwright-cli open "http://127.0.0.1:4321/"
playwright-cli resize 1440 900
playwright-cli screenshot --filename=/tmp/home.png
playwright-cli close
```

## Follow the AWF sandbox policy

When Playwright runs in the AWF sandbox:

- Never install packages, browsers, or system dependencies at runtime. Report a
  missing CLI or browser instead.
- Navigate only to loopback URLs or domains listed in `network.allowed`.
- Do not bind local servers to `0.0.0.0`, publish ports, or use preview tunnels.
- Do not change browser proxy settings, proxy environment variables, or the
  `localhost`/`127.0.0.1` proxy bypass.
- Close the browser and stop any server process started during the task.

These rules differ from using the standalone `awf` command to wrap a host-side
Playwright test. Standalone AWF uses `--allow-domains localhost` to expose
selected host ports to its container. In a gh-aw agent sandbox, start the server
inside the sandbox and keep it on loopback instead.

## Sample workflow

This workflow checks a public documentation site and reports actionable
accessibility findings:

```markdown
---
on:
  workflow_dispatch:

permissions:
  contents: read

tools:
  playwright:

network:
  allowed:
    - defaults
    - playwright
    - "docs.example.com"

safe-outputs:
  create-issue:
    title-prefix: "[accessibility] "
    labels: [accessibility]
    max: 3
  noop:
---

# Accessibility review

Open https://docs.example.com with `playwright-cli`. Inspect the page snapshot,
keyboard navigation, form labels, image alternatives, and heading structure.
Create focused issues for actionable findings. If there are none, call `noop`.
Always close the browser before finishing.
```

For visual comparisons, pin the Playwright CLI version, define the baseline
source explicitly, keep screenshots under `/tmp`, and use `cache-memory` when
baselines must persist across runs.

## Related guidance

- [`visual-regression.md`](visual-regression.md) for baseline storage and
  comparison patterns
- [`network.md`](network.md) for domain allowlisting
- [`mcp-clis.md`](mcp-clis.md) for CLI-mounted MCP servers, which are separate
  from the built-in Playwright CLI integration
- [Playwright CLI](https://github.com/microsoft/playwright-cli) for the complete
  command reference
