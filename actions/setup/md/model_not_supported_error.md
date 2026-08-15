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

If the error text is `No model available. Check policy enablement under GitHub Settings > Copilot`, the model is not disabled in the workflow but by Copilot policy. Enable the model under **GitHub Settings > Copilot > Policies** for the org/repo, or pick a model that is already enabled. This can also be triggered by a subagent (`task` tool) dispatch requesting a model that the policy does not allow, even when the main agent's model is enabled.

</details>
