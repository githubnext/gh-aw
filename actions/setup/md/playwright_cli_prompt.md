<playwright-cli>
<description>Playwright CLI is available for browser automation tasks. Use bash to run playwright-cli commands directly on the runner.</description>
<usage>
playwright-cli is a session-based tool. You must open a browser session first, then run commands within that session.

To take a screenshot, pipe commands to `playwright-cli open` using a heredoc or echo:

```bash
playwright-cli open --browser firefox https://example.com <<'EOF'
screenshot /tmp/gh-aw/screenshot.png
exit
EOF
```

Other commands available within a session (pipe after `open`):
- `goto <url>` — navigate to a URL
- `snapshot` — capture accessibility snapshot
- `click <ref>` — click an element
- `screenshot [ref]` — screenshot of current page or element

Output files should be saved to /tmp/gh-aw/ for access by subsequent steps.

Use `--browser firefox` (the default installed browser). Chromium requires SUID sandbox setup not available on standard runners.

Refer to `playwright-cli --help` for the full list of commands and options available.
</usage>
</playwright-cli>
