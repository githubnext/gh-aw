> [!WARNING]
> **Shell Expansion Guard Rejected Command**: The sandbox rejected a shell command because expansion patterns looked unsafe.

This signal was detected from engine runtime logs and is usually caused by retrying a multi-line shell command that embeds safe-output JSON or Markdown directly in the command text.

<details>
<summary>How to remediate</summary>

- Do **not** retry the identical rejected command.
- Put multi-line content in a temporary file with a single-quoted heredoc.
- Use `jq -Rs` to JSON-escape the file contents before piping to `safeoutputs`.

```bash
cat <<'EOF' > /tmp/gh-aw/body.md
Title

Multi-line body content goes here.
EOF
jq -Rs '{title: "My title", body: .}' /tmp/gh-aw/body.md | safeoutputs create_discussion .
```

</details>
