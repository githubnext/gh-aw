## Trend Data (2026-08-10)

- Prior recorded snapshot in this memory (as read from the old broken path) was 2026-04-03: tokens/day 99.5M, 123 runs, 78.4% PR merge rate, safe-output 100%, WHM 72/100, 19 stale locks, issues 56 open/444 closed. Treat as stale/unverified — 4+ months old and the persistence path was broken the whole time, so no continuous trend line can be drawn from Apr to Aug.
- Fresh baseline established this cycle (2026-08-09/10, from cross-referenced discussion reports):
  - Fleet success: 96.6% raw / 96.9% adjusted (323 runs, 2026-08-08→09, discussion #51643).
  - Safe-outputs job success: 98.4% of attempted (187), discussion #51688.
  - Tokens: 12.5M in the 2026-08-09 24h window (audit report) — highest single day in a mostly-gapped 30-day lookback (only ~11 days have data).
  - Lockfiles: 284 compiled, 39.3MB total, ~7.33 jobs/file avg (discussion #51638, 2026-08-09).
  - Issues (500-sample, 7d window): 134 open / 366 closed (per issues-analyst subagent, 2026-08-10); separately the fleet's own Daily Issues Report (1000-sample) showed 147 open / 853 closed, 13h55m avg close time.
  - Firewall: 91 firewall-enabled runs, 5,775 requests, 99.3% allowed (discussion #51613, 7d window ending 2026-08-10).
  - Firewall/MCP raw log coverage: 0.0% (0/15 access.log, 0/2 gateway.jsonl) — discussion #51654.
  - Driver-exit failure rate: 10% in a fresh 50-run sample pulled this cycle via `agenticworkflows logs`.
- Going forward: use 2026-08-10 as the new trend baseline. Next cycle should diff against these numbers, not the pre-2026-04-03 ones.
