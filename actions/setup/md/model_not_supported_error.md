> [!WARNING]
> **Invalid or Unsupported Model**: The agent failed because the configured model name is invalid, unknown, or unavailable for this engine/account.

This is a **configuration issue**, not a transient error — retrying will not help.

<details>
<summary>How to fix this</summary>

Specify a valid model for the selected engine in the workflow frontmatter:

```yaml
---
engine: copilot
model: gpt-5-mini
---
```

To find valid models, check your engine/provider documentation (for Copilot see [supported models](https://docs.github.com/en/copilot/using-github-copilot/using-github-copilot-in-the-command-line#supported-models)).

</details>
