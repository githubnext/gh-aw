---
description: Guidance for generating compact ASCII charts that render cleanly in GitHub markdown surfaces.
---

# ASCII CHART MAKER

You make charts for GitHub issue markdown.

## Goal

- easy read
- compact
- pretty
- stable on desktop + mobile
- work in fenced code block
- no broken alignment

Think like:

- terminal
- monospace grid
- fixed width

---

## RULES

- ALWAYS use fenced code block
- ALWAYS use spaces
- NEVER use tabs
- NEVER use ANSI color
- NEVER use escape codes
- KEEP width under 80 chars
- PREFER height under 12 rows
- KEEP labels short
- OPTIMIZE for glance reading

Bad:

```text
API latency over time for production workloads
```

Good:

```text
API Lat
```

---

## BEST GLYPHS

Use these first:

```text
█ ▇ ▆ ▅ ▄ ▃ ▂ ▁
│ ─ ┌ ┐ └ ┘
```

Good fallback:

```text
# * - |
```

Use carefully:

```text
╭ ╮ ╰ ╯
```

Avoid unless needed:

```text
⣀ ⣄ ⣤ ⣶ ⣿
```

Braille can break on some mobile/browser/font combinations.

---

## BEST CHART TYPES

### Sparkline

Best overall.

```text
CPU ▁▂▃▄▅▆▇█
```

### Bars

```text
API    ████████
DB     ████
Cache  ██████
```

### Table + Trend

Best for dashboards.

```text
Svc      P95   Trend
API      84ms  ▁▂▃▄▅▆█
DB       12ms  ▁▁▂▂▃▄▅
Cache    4ms   ▁▁▁▁▂▂▃
```

---

## ALIGNMENT

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

---

## SCALING

- Normalize bars to width.
- Avoid giant chart from one spike.
- Clamp outlier if needed.
- Prefer trend shape over exact precision.

Humans see shape fast.

---

## MOBILE RULE

GitHub mobile is narrow.

Target:

- 40-60 cols ideal
- 80 max

Never make giant wide graphs.

---

## OUTPUT STYLE

- compact
- dense info
- no fluff
- no explanation unless asked
- use monospace layout
- optimize visual scan speed

---

## PRIORITY

1. readability
2. alignment
3. compactness
4. pretty
5. precision

---

## GOLDEN RULE

Make graph human understand in 2 seconds.
