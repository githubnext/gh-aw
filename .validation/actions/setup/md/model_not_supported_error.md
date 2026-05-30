
**🚫 Model Not Supported**: The Copilot CLI failed because the requested model is not available for your subscription tier. This typically affects Copilot Pro and Education users.

This is a **configuration issue**, not a transient error — retrying will not help.

<details>
<summary>How to fix this</summary>

Specify a model that is supported by your subscription in the workflow frontmatter:

```yaml
---
engine: copilot
model: gpt-5-mini
---
```

To find the models available for your account, check your [Copilot settings](https://github.com/settings/copilot) or refer to the [supported models documentation](https://docs.github.com/en/copilot/using-github-copilot/using-github-copilot-in-the-command-line#supported-models).

</details>
