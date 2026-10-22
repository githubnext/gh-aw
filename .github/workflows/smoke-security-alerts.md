---
private: true
name: Smoke Security Alerts
description: Validate gateway security alert notifications and authenticated read access
intent: Give maintainers confidence that security alert notifications reach the workflow without exposing alert bodies or granting mutation authority.
on:
  gateway-dispatch:
    repos: [github/gh-aw]
    code-scanning-alert:
      types: [created]
    dependabot-alerts:
      types: [created]
permissions:
  contents: read
  security-events: read
  vulnerability-alerts: read
  copilot-requests: write
engine: copilot
checkout: false
network:
  allowed: []
tools:
  github:
    mode: gh-proxy
    toolsets: [context, code_security, dependabot]
safe-outputs:
  noop:
timeout-minutes: 5
---

# Smoke Security Alerts

Inspect every notification in ${{ toJSON(gateway.events) }}. Confirm the event
family is `code_scanning_alert` or `dependabot_alert`, with only the canonical
alert URL, selected activity, and any declared matching filter literals.

Read each alert using authenticated GitHub tools against the trusted GitHub API
endpoint for its repository and alert number. Never send credentials to arbitrary
notification URLs or redirects. If access is denied or the alert no longer
exists, report the specific incomplete check; do not substitute cached payloads.

Report only the family, alert number, activity, and read-access result. Do not
copy alert descriptions, code, dependency details, or credentials into reports.
Do not create, dismiss, reopen, or otherwise modify alerts or repository resources.
On success, call `noop` with a concise summary of the checked notifications.
