# Workflow timing analysis — Run 21477851525

**Run:** https://github.com/githubnext/gh-aw/actions/runs/21477851525  
**Workflow:** Smoke Copilot  
**Total duration:** ~4m 57s (12:18:31 → 12:23:28 UTC)

## 5‑stage summary (major stages)

1. **Pre‑activation job** — **~7s** (12:18:35 → 12:18:42)
   - Runner setup, sparse checkout of `actions/`, and setup action prep.

2. **Activation job** — **~8s** (12:18:45 → 12:18:53)
   - Second checkout + setup action execution to prepare activation assets.

3. **Agent environment prep (containers + firewall + image pulls)** — **~1m 58s** (12:18:56 → 12:20:54)
   - Agent job starts, pulls images, configures firewall, and starts containers.

4. **Start coding agent (Claude Code CLI execution)** — **~1m 20s** (12:20:54.038 → 12:22:14.536)
   - Copilot/Claude session runs, MCP clients connect, tests execute, and results produced.

5. **Post‑agent pipeline** — **~53s** total
   - Agent teardown: **~19s** (12:22:14 → 12:22:33)
   - Threat detection job: **~24s** (12:22:36 → 12:23:00)
   - Safe outputs + cache update (overlap): **~11s / 6s** (start 12:23:03)
   - Conclusion job: **~9s** (12:23:18 → 12:23:27)

## Time‑to‑Start (TTS) breakdown

- **Workflow start → agent job start:** ~25s (12:18:31 → 12:18:56)  
  dominated by pre‑activation + activation jobs.

- **Agent job start → Claude Code CLI start:** **~1m 58s**  
  dominated by image pulls, firewall setup, and container startup.

- **Claude Code CLI session runtime:** **~1m 20s**

## Top 5 improvements (highest leverage)

1. **Pre‑warm or cache agent/container images** (Stage 3)
   - Largest single block before the coding agent starts (~1m 58s).
   - Use runner‑level image caching or a pre‑pull step (or a warm pool) to cut TTS materially.

2. **Reduce activation/pre‑activation duplication** (Stages 1–2)
   - Two similar jobs each do sparse checkout + setup action prep (~15s combined).
   - Consolidate or cache activation assets to remove one job or shorten both.

3. **Trim agent startup footprint** (Stage 3 → 4)
   - MCP client startup and tool catalog loading is fast individually but adds latency.
   - Disable unused toolsets for the smoke flow to reduce initialization work and token setup.

4. **Shorten the Claude Code CLI session** (Stage 4)
   - The session is ~1m 20s and includes extra tool calls (Playwright, build).
   - For TTS focus, split “smoke” into a minimal “start‑agent” pass vs. the full workflow,
     or defer heavy checks (Playwright/build) to post‑agent stages.

5. **Parallelize or skip post‑agent jobs when possible** (Stage 5)
   - Threat detection + safe outputs + conclusion add ~53s after agent completion.
   - If safe/allowed, run detection in parallel with safe outputs or gate it to changed runs.

## Key takeaways

- **TTS is dominated by container/image prep** in the agent job (nearly 2 minutes).
- **Claude Code CLI execution is ~1m 20s** and is the next biggest block.
- **Pre‑activation + activation** add ~15s before the agent even starts.
- **Post‑agent jobs** add ~53s after the agent finishes, which matters for end‑to‑end time.
