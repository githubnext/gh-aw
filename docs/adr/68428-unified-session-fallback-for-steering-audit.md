# ADR-68428: Normalize Steering Events and Use the Unified Session as a Fallback Audit Source

**Date**: 2026-02-13
**Status**: Draft
**Deciders**: Unknown (drafted by the Design Decision Gate from PR #68428)

---

### Context

AWF steering notices (AI credit, token, and timeout warnings) are emitted by the API proxy and arrive in different event shapes: a flat `event` field, alternative keys (`type`, `event_name`, `eventName`), nested `payload.event` variants, and inconsistent casing such as `TOKEN_STEERING`. The unified session normalizer (`actions/setup/js/unified_session.cjs`) previously matched steering only against a fixed allow-list of lowercase names, so variant shapes were dropped. On the audit side, `pkg/cli/token_usage_steering.go` aggregated steering counts and notices exclusively from the raw proxy event log (`sandbox/firewall/logs/api-proxy-logs/events.jsonl`). When that raw log is not retained in a run's artifacts, audit reports showed zero steering activity even though the unified session file contained the same events. Both sources describe the same underlying proxy events, so naively reading both would double-count.

### Decision

We will normalize steering event names at the unified-session boundary — resolving nested and alternative payload keys via `getApiProxyEventName()` and classifying any name equal to `steering` or ending in `_steering` as `firewall.steering`, while persisting the canonical lowercase name back into the event `data`. We will also make the CLI audit path treat the unified session as a *fallback-only* source: `readUnifiedSteeringLog()` is consulted only when `findAPIProxyEventsLog()` returns no raw proxy log, so raw logs remain authoritative and the two sources are never merged.

### Alternatives Considered

#### Alternative 1: Extend the hard-coded steering name allow-list

Keep the existing `["ai_credit_steering", "token_steering", "timeout_steering"]` membership test and append new names as they appear. This is the smallest change and keeps classification explicit, but it requires a code change for every new steering type the proxy introduces and still does not address nested `payload.event` shapes or uppercase values, which were the actual failure modes observed in this PR.

#### Alternative 2: Always read both raw proxy logs and the unified session, de-duplicating by request ID

This would make the audit robust regardless of which artifact survives. It was rejected because steering records do not reliably carry a stable unique identifier (`request_id` is optional and the unified copy uses `requestId`), so de-duplication would be heuristic and could silently drop or double-count legitimate events. The fallback ordering gives deterministic behaviour with no de-duplication logic, at the cost of ignoring unified-session-only events when a partial raw log exists.

#### Alternative 3: Emit steering events into audit reports from the agent log pipeline instead

Rather than reading proxy artifacts, derive steering notices from the engine/agent logs that already surface the warning text. Rejected because the warning text alone lacks structured fields (threshold, provider, request ID) needed for the steering counts in audit output, and would couple audit fidelity to engine-specific log formats.

### Consequences

#### Positive
- Steering events are captured regardless of proxy payload shape or casing, including nested `payload.event` variants.
- Audit reports (`TotalSteeringEvents`, `SteeringEventCounts`, `SteeringNotices`, `GatewaySteeringEvents`) remain populated for runs where only the unified session artifact was retained.
- The fallback ordering is deterministic and explicitly documented in code, preventing double-counting between the two sources.

#### Negative
- Two parsing paths now exist for the same logical data (`parseAPIProxySteeringLog` and `readUnifiedSteeringLog`), which must be kept behaviourally consistent as the steering schema evolves.
- The suffix-based classifier (`endsWith("_steering")`) is broader than an allow-list and may classify future unrelated proxy events as steering if they adopt that naming suffix.
- When a raw proxy log exists but is truncated or partial, unified-session steering events are ignored entirely rather than filling the gap.

#### Neutral
- `collectUnifiedSteeringEvent` filters on `provenance.component == "firewall"` and `provenance.phase` in `{agent, detection}`, so steering events from other components or phases are intentionally excluded from audit aggregation.
- The unified-session reader accepts both `request_id` and `requestId` and re-hydrates the timestamp from the envelope, accommodating the normalizer's camelCase output.
- Line-length guarding (`readUnifiedSessionLine` oversized check) makes the fallback reader fail loudly on malformed sessions rather than silently skipping records.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
