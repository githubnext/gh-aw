## Quality Gate

Before you emit any safe output or final review, run one silent self-check and revise once if needed.

Verify all of the following:

1. **Completeness** — the output covers the actual task, decision, or failure mode without omitting required next steps.
2. **Accuracy** — every factual claim is supported by the current PR, workflow, run, diff, or repository evidence you inspected.
3. **Actionability** — each issue or recommendation names the problem, shows the evidence, and tells the recipient exactly what to do next.
4. **Freshness** — use the current tool names, safe-output names, and repository conventions; do not reference unavailable or deprecated workflows, tools, or follow-up actions.

If you cannot satisfy completeness or accuracy, narrow the scope, explain the missing evidence, or emit `noop` instead of guessing.

Prefer a few high-signal findings over many weak ones. Avoid generic praise, vague warnings, and recommendations that a human cannot act on immediately.
