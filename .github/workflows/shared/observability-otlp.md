---
observability:
  otlp:
    endpoint: ${{ secrets.GH_AW_OTEL_ENDPOINT }}
    headers: ${{ secrets.GH_AW_OTEL_HEADERS }}
---

## Experiment OTEL Span Attributes

If the file `/tmp/gh-aw/experiments/assignments.json` exists, read it and emit one OTEL span
attribute per experiment using the `otlp.cjs` `logSpan` helper:

```javascript
const fs = require('fs');
const path = require('path');
const assignmentsFile = '/tmp/gh-aw/experiments/assignments.json';
if (fs.existsSync(assignmentsFile)) {
  const assignments = JSON.parse(fs.readFileSync(assignmentsFile, 'utf8'));
  const names = Object.keys(assignments).sort();
  if (names.length > 0) {
    const attrs = { 'gh_aw.experiment.names': names.join(',') };
    for (const name of names) {
      attrs[`gh_aw.experiment.${name}`] = assignments[name];
    }
    const otlp = require('/tmp/gh-aw/actions/otlp.cjs');
    await otlp.logSpan('experiment', attrs);
  }
}
```

This enables filtering workflow runs by experiment variant in Datadog, Honeycomb, or any
OTLP-compatible backend. Attribute keys follow the pattern `gh_aw.experiment.<name>` with the
assigned variant as the value, plus `gh_aw.experiment.names` as a comma-separated index.
