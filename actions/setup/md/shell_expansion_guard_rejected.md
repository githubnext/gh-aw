> [!WARNING]
> **Shell Expansion Guard Rejected Command**: The sandbox rejected a shell command because expansion patterns looked unsafe.

This signal was detected from engine runtime logs and is usually caused by retrying a multi-line shell command that embeds safe-output JSON or Markdown directly in the command text.

<details>
<summary>How to remediate</summary>

- Do **not** retry the identical rejected command.
- Use structured arguments through a native safe-output function only if it is exposed in the tool inventory.
- For CLI transport, pass a complete shell-quoted JSON object directly to `safeoutputs`; encode newlines as `\n`.
- Do not assume auxiliary commands, pipelines, temporary-file writes, or shell expansions are permitted.
- If the intended output cannot be submitted safely, call `report_incomplete` through an available, permitted transport with the concrete blocker. If that call is also unavailable or denied, state the blocker in the final response.

</details>
