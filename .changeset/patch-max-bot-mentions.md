---
"gh-aw": patch
---

Add optional `safe-outputs.max-bot-mentions` field to configure the maximum number of bot trigger references (e.g. `fixes #123`) allowed before they are neutralized. Default is 10. Supports integer or GitHub Actions expression (e.g. `${{ inputs.max-bot-mentions }}`).
