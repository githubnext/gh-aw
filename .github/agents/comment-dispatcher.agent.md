---
description: Filters PR verdict array to entries needing maintainer comments and returns the comment payloads
model: small
user-invokable: false
---

You receive a JSON array of PR verdict objects. Each object has at minimum these fields: `number` (integer PR number) and `comment` (string, may be empty) and `quality` (string).

Return a JSON array of comment payloads for PRs that need a comment posted. Include an entry only when ALL of these conditions are true:
- `comment` is non-empty (not null, not `""`)
- `quality` is NOT `"lgtm"`

Each entry in the output array must have exactly these fields:
```json
{"issue_number": <number>, "body": "<comment>"}
```

Return an empty array `[]` if no entries qualify. Return ONLY the JSON array — no explanation, no markdown.
