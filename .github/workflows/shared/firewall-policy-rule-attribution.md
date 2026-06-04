#### Section 4: Policy Rule Attribution (Always Visible — when data available)

**Include this section when `policy_analysis` data was available for at least one run.**

This section provides rule-level insights that go beyond simple domain counts, showing *which policy rules* are handling traffic and *why* specific requests were denied.

**4a. Policy Configuration**

Show the policy summary from the most recent run:
- Number of rules, SSL Bump status, DLP status
- Example: "📋 Policy: 12 rules, SSL Bump disabled, DLP disabled"

**4b. Policy Rule Hit Table**

Show aggregated rule hit counts across all analyzed runs:

```markdown
| Rule | Action | Description | Total Hits |
|------|--------|-------------|------------|
| allow-github | 🟢 allow | Allow GitHub domains | 523 |
| allow-npm | 🟢 allow | Allow npm registry | 187 |
| deny-blocked-plain | 🔴 deny | Deny all other HTTP/HTTPS | 12 |
| deny-default | 🔴 deny | Default deny | 3 |
```

- Sort by hits (descending)
- Include all rules that had at least 1 hit
- Use 🟢 for allow rules and 🔴 for deny rules in the Action column

**4c. Denied Requests with Rule Attribution**

Show denied requests grouped by rule, with domain details:

```markdown
| Domain | Deny Rule | Reason | Occurrences |
|--------|-----------|--------|-------------|
| evil.com:443 | deny-blocked-plain | Domain not in allowlist | 5 |
| tracker.io:443 | deny-blocked-plain | Domain not in allowlist | 3 |
| unknown.host:80 | deny-default | Default deny | 1 |
```

- Group by domain + rule combination
- Sort by occurrences (descending)
- Show top 30 entries; wrap the full list in `<details>` if more than 30

**4d. Rule Effectiveness Summary**

Provide a brief analysis:
- Which deny rules are doing the most work (catching the most unauthorized traffic)
- Which allow rules handle the most traffic (busiest legitimate pathways)
- Any rules with zero hits that could be removed or indicate unused policy entries
- Any `(implicit-deny)` attributions that indicate gaps in the policy (traffic denied without matching any explicit rule)
