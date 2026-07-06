---
description: Guidance for generating compact ASCII charts that render cleanly in GitHub markdown surfaces.
---

# ASCII CHART MAKER

Make charts for GitHub issue markdown: easy to read, compact, stable on desktop + mobile, inside a fenced code block. Think monospace grid, fixed width.

## RULES

- ALWAYS fenced code block; ALWAYS spaces, NEVER tabs
- NEVER ANSI color or escape codes
- Width under 80 chars; prefer height under 12 rows
- KEEP labels short; optimize for glance reading

Bad: `API latency over time for production workloads` — Good: `API Lat`

## BEST GLYPHS

Use first:

```text
█ ▇ ▆ ▅ ▄ ▃ ▂ ▁
│ ─ ┌ ┐ └ ┘
```

Fallback: `# * - |`. Use carefully: `╭ ╮ ╰ ╯`. Avoid unless needed: `⣀ ⣄ ⣤ ⣶ ⣿` (braille breaks on some mobile/browser/font combos).

## BEST CHART TYPES

### Sparkline (best overall)

```text
CPU ▁▂▃▄▅▆▇█
```

### Bars

```text
API    ████████
DB     ████
Cache  ██████
```

### Table + Trend (best for dashboards)

```text
Svc      P95   Trend
API      84ms  ▁▂▃▄▅▆█
DB       12ms  ▁▁▂▂▃▄▅
Cache    4ms   ▁▁▁▁▂▂▃
```

## ALIGNMENT

Pad labels to equal width.

Good:

```text
API      ███████
Worker   ████
Cache    █████████
```

Bad:

```text
API ███████
Worker ████
Cache █████████
```

## SCALING

- Normalize bars to width; clamp outliers so one spike doesn't dominate.
- Prefer trend shape over exact precision — humans read shape fast.

## MOBILE

GitHub mobile is narrow. Target 40-60 cols ideal, 80 max. Never make giant wide graphs.

## GOLDEN RULE

Make a graph a human understands in 2 seconds. Priority: readability > alignment > compactness > pretty > precision.
