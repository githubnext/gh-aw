<playwright-awf-policy>
This workflow runs Playwright CLI inside the AWF (agentic workflow firewall)
sandbox. The following rules take precedence over any generic Playwright CLI
skill guidance, including suggestions to `npm install`, use `npx` to fetch
missing packages, or navigate to arbitrary example domains.

- The Playwright CLI and Chromium browser are already staged for this sandbox.
  Never install packages, browsers, or system dependencies at runtime, and
  never run commands like `npm install -g @playwright/cli`, `npx playwright
  install`, or similar. If the CLI or browser appears unavailable, report the
  failure instead of trying to install anything.
- If you need to exercise a local application, start its server **inside this
  sandbox** (for example with `run` in the background) before using
  `playwright-cli`. Bind the server only to `127.0.0.1` (or `localhost`),
  preferably on an ephemeral port. Never bind to `0.0.0.0`, publish host
  ports, or rely on an external preview/tunnel service.
- Before navigating with `playwright-cli`, poll the loopback URL until it
  responds (a readiness/health check) instead of using a fixed sleep.
- Navigate only to the loopback URL of the server you started, or to domains
  explicitly present in this workflow's allowed domains list. Do not navigate
  to arbitrary public example sites.
- Traffic to `localhost`/`127.0.0.1` is served directly; all other HTTP/HTTPS
  browser traffic is routed through the sandbox's proxy and is subject to the
  workflow's domain allowlist. Do not attempt to change browser proxy
  settings, bypass rules, or the proxy environment variables — they are
  enforced by the sandbox, not by this agent.
- Never treat the API/model-provider proxy as a browser proxy; it exists only
  to reach the model provider, not for browser navigation.
- Always clean up: close the browser/page, stop any server process you
  started, and verify there are no leftover background processes before
  finishing the task.
</playwright-awf-policy>
