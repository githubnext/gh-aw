---
tools:
  edit:
  bash:
    - "*"
  web-fetch:
safe-outputs:
  messages:
    footer: "> 🔥 *[{workflow_name}]({run_url}) — Powered by OpenCode*{ai_credits_suffix}{history_link}"
    run-started: "🔥 OpenCode initializing... [{workflow_name}]({run_url}) begins on this {event_type}..."
    run-success: "🚀 [{workflow_name}]({run_url}) **MISSION COMPLETE!** OpenCode delivered. 🔥"
    run-failure: "⚠️ [{workflow_name}]({run_url}) {status}. OpenCode encountered unexpected challenges..."
---

<!-- Shared OpenCode engine integration defaults for smoke and test workflows. -->
