---
"gh-aw": minor
---

Rename firewall terminology from "denied" to "blocked" across code, JSON fields,
interfaces, JavaScript variables, table headers, and documentation. This updates
struct fields, JSON tags, method names, and user-facing text to use "blocked".

## Codemod

Update JSON payloads and workflow outputs using these before/after examples:

Before:

```json
{
  "firewall_log": {
    "denied_requests": 3,
    "denied_domains": ["blocked.example.com:443"],
    "requests_by_domain": {
      "blocked.example.com:443": {"allowed": 0, "denied": 2}
    }
  }
}
```

After:

```json
{
  "firewall_log": {
    "blocked_requests": 3,
    "blocked_domains": ["blocked.example.com:443"],
    "requests_by_domain": {
      "blocked.example.com:443": {"allowed": 0, "blocked": 2}
    }
  }
}
```

Update Go types and interfaces (examples):

Before:

```go
type FirewallLog struct {
    DeniedDomains []string `json:"denied_domains"`
}

func (f *FirewallLog) GetDeniedDomains() []string { ... }
```

After:

```go
type FirewallLog struct {
    BlockedDomains []string `json:"blocked_domains"`
}

func (f *FirewallLog) GetBlockedDomains() []string { ... }
```

Update JavaScript variables and table headers (examples):

Before: `deniedRequests`, `deniedDomains`, table header `| Domain | Allowed | Denied |`

After: `blockedRequests`, `blockedDomains`, table header `| Domain | Allowed | Blocked |`

This is a breaking change for any code that relied on the previous field names or
JSON tags; update integrations to use the new `blocked_*` fields and `blocked` in
per-domain stats.

