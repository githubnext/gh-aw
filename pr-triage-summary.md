# PR Triage Run History

| Run ID | Date | Agent PRs | New | Carried Over | Closed Since | Notes |
|--------|------|-----------|-----|--------------|--------------|-------|
| §28395315609 | 2026-06-29T18:55Z | 10 | 6 | 4 | 4 merged | 4 fast_track ready; #41830 64h critical stale; bug-fixes-ready batch |
| §28376613466 | 2026-06-29T13:46Z | 8 | 5 | 3 | 7 (5 merged) | #41830 59.3h stale fast_track; 3 PRs changes-requested; #42222 fast_track merged |
| §28357644191 | 2026-06-29T08:07Z | 8 | — | — | — | prior run |
| §28258543430 | 2026-06-26T18:51Z | 14 | 13 | 1 | 10 | burst activity |
| §28239513901 | 2026-06-26T12:59Z | 11 | 9 | 2 | 4 | draft-heavy run |
| §28224135185 | 2026-06-26T07:37Z | 6 | 4 | 2 | 2 | 4 new PRs burst |
| §28210925990 | 2026-06-26T01:20Z | 3 | 2 | 1 | 5 | #41555 #41553 #41295 |
| §28193371345 | 2026-06-25T18:58Z | — | — | — | — | prior run |

## Persistent Observations
- All Copilot PRs use same-repo branches (`copilot/*`); no fork PRs seen in any run
- #41830 is the oldest open PR at 64h+ — has been critical stale since §28376613466
- #41824 (66h) has accumulated conflicting triage labels from multiple runs; needs manual cleanup
- safe_outputs JS handler PRs cluster naturally (#41830, #42318, #42313)
- Experimental features (#42314 Auggie, #42100 BinEval) tend to stay as long-lived drafts

## Batch History
| Batch | PRs | Status |
|---|---|---|
| bug-fixes-ready | #41830 · #42318 · #42313 · #42317 | active (§28395315609) |
| production-fixes | #41830 | carried from prior run |
| changes-requested | #41824 · #42235 · #42226 | #42235 and #42226 merged (§28376613466) |
