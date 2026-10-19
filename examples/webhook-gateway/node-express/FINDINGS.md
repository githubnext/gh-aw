# Express implementation-driven specification findings

Baseline: **`527e59970ad970f26f8e1038b6f301cef80cf876`**, `docs/src/content/docs/specs/webhook-gateway-specification.md`. All specification line numbers below refer to that immutable baseline, not the coordinator's later consolidation. Implementation paths are relative to this directory. No shared specification/reference file was edited by this child.

## Confirmed contradictions

### 1. Notification fragments versus the blanket fragment prohibition

**Requirement:** "GitHub notification subscriptions", lines 115-128, requires catalogued comment/review anchors. "Expected per-delivery payloads", line 559, permits only command comment anchors. Both cannot hold for an ordinary review notification.

**Choice and evidence:** Follow the normative notification catalog. `src/matching.mjs:32-58` binds parent and comment/review IDs to the original canonical URL; `src/core.mjs:45-54` rejects parser rewrites and bare delimiters. `node --test --test-name-pattern='all ten notification|WHATWG canonical' test/gateway.test.mjs` reproduces accepted `#pullrequestreview-123` and rejected bare `?`/`#`, uppercase scheme/host, default-port and dot-segment rewrites. The baseline destination already permits notification fragments in `actions/setup/js/webhook_resource.cjs:125`.

**Proposed clarification:** Explicitly permit exactly the selected catalogued notification **or** command anchor; state WHATWG serialized equality, including rejection of otherwise empty query/fragment delimiters. The coordinator confirmed this resolution.

### 2. Absolute body/delta exclusions versus normative matching-scalar selectors

**Requirement:** Lines 227-239 and 557/727 exclude source bodies/delta objects while allowing declared matching scalars. The normative Projects catalog explicitly exposes `body.from`/`body.to` (`pkg/workflow/data/webhook_events.json:451-452`); the Azure catalog exposes entire `tags.from`/`tags.to` strings (`pkg/workflow/data/provider_issue_webhooks.json:76-86`). Interpreting the exclusions as forbidding even configured matching strings makes valid catalogued subscriptions impossible.

**Choice and evidence:** Preserve catalogued exact scalar matching, never source objects or unselected text. `src/matching.mjs:132-147` intersects original scalars with configured literals; the test `catalogued body/tag scalar selectors forward only exact configured matching literals` verifies the narrow exception and rejects unselected text.

**Proposed clarification:** Distinguish source bodies and delta objects from a scalar already explicitly configured and proven equal. Such a selector is not authority to copy other body text or the entire delta. The coordinator confirmed this resolution.

## Underspecified protocol and deployment decisions

### 3. Stable identity, authenticated replay binding, scope and retention

**Requirement:** "Dispatch envelope", lines 541-543, requires original/stable delivery IDs and receiver-side durable deduplication, without a provider identity algorithm, signed-header binding, retention horizon or reset policy. A body HMAC does not authenticate GitHub/Jira delivery headers.

**Choice and evidence:** `src/auth.mjs:27-42` uses the original GitHub ID, Jira identifier (or content hash), Azure event `id`, and a Linear authenticated-content hash including original event time but excluding the transport timestamp. Linear `webhookId` is not assumed to be an event ID. `src/store.mjs:39-62` binds ID to authenticated-content digest and independently deduplicates content under rewritten unsigned IDs. The durable/concurrency test in `test/gateway.test.mjs:84-102` demonstrates both cases and conflicting-ID rejection.

**Proposed clarification:** Name provider identity/scope requirements, require authenticated-content binding where headers are unsigned, and define an approved replay horizon with retained tombstones or an explicit nonactivating capacity boundary. This implementation retains receipts indefinitely and fails admission at 100,000 receipts/jobs or 10,000 outstanding jobs. Its conservative content fallback can suppress two indistinguishable legitimate events; it does not claim lossless reconstruction of an absent provider event identity.

### 4. Acknowledgment loss and crash boundaries cannot establish exactly-once execution

**Requirement:** "CAO installation-token dance", lines 337-341, acknowledges dispatch with 204 and requires current grants on retries. It does not define a dispatch idempotency key, proof of nonacceptance, recovery after a committed-but-unrecorded send, or whether a 5xx may follow acceptance.

**Choice and evidence:** `src/github.mjs:46-73` disables redirects, retries only explicit negative/retryable responses, and marks dispatch transport failure/5xx uncertain. `src/store.mjs:65-81` persists a claim before external work and never automatically reclaims a surviving `in_flight` claim. Tests `explicit dispatch rate limit retries...`, `parallel workers...`, and `cleanup failure preserves an uncertain dispatch outcome...` reproduce bounded 429 retries, connection-loss/5xx quarantine and concurrent claims.

**Proposed clarification:** Define acknowledgment as request acceptance only, require uncertain sends/crashed sending claims to remain quarantined absent independently established nonacceptance, and forbid exactly-once claims from receiver deduplication alone. There is deliberately no automated operator replay command in this experiment.

### 5. Label removal lacks a safe occurrence/generation protocol

**Requirement:** "Gateway command subscriptions", lines 171-175, requires post-admission label consumption using separate source-write authority plus durable fencing/idempotency. No occurrence identity, conditional compare-and-remove API or crash reconciliation decision is defined. Retrying a DELETE after a lost response can consume a newer reapplication of the same label.

**Choice and evidence:** `src/contracts.mjs:59-63` rejects **all** `label_command` enrollment with `label-consumption-unsupported`; the closed-subscription test proves rejection. No label-write credential, mutation, bypass or runner-side substitute exists. This is an explicit implementation limitation, not label-command conformance.

**Proposed clarification:** Require confirmed, occurrence-aware fenced consumption before dispatch; uncertain removal cannot be retried against a possibly newer occurrence. Keep receivers without an independently reviewed protocol nonactivating. The coordinator confirmed this resolution.

### 6. Approved artifact/revision binding is required without a shared enrollment format

**Requirement:** "Approved identity binding", lines 244-261; destination revision adoption, line 509; and workflow availability, lines 337/551, require independently approved configuration and a default-branch workflow. They do not specify a receiver administrative wire protocol, artifact digest representation or atomic revision guarantee.

**Choice and evidence:** `src/contracts.mjs:161-274` accepts a protected local enrollment file, not caller-provided grants. `src/server.mjs:78-109` checks the complete approval fingerprint before every credentialed operation. `src/github.mjs:115-136` verifies authenticated repository metadata, full default-branch SHA, exact lock-file SHA-256 and embedded subscriptions. Tests withdraw grants before/after minting and alter revision/artifact responses. `gateVerified` is an independently reviewed operator attestation, not a self-verifying boolean sent by a provider.

**Proposed clarification:** Keep administrative formats implementation-specific, require approved artifact/execution revision binding, and state that pre-dispatch discovery reads cannot atomically select `repository_dispatch`'s execution revision. This implementation conservatively blocks unrelated default-branch changes; GitHub can still change its branch between verification and acceptance. A trusted deployment must address that residual race rather than claim the read eliminated it.

### 7. Linear and Azure canonical source-URL selection

**Requirement:** "Linear issue", lines 681-683, requires binding a workspace-qualified URL to the original immutable Issue without naming a delivered URL field or the identifier evidence. "Azure DevOps work item", lines 695-697, forbids update/revision URLs but does not state which native field supplies an item URL when `resource.url` points at an update.

**Choice and evidence:** `src/matching.mjs:207-232` binds Linear organization/team, original `data.identifier` and protected workspace to top-level `url`, or `data.url` when necessary, rejecting contradictory delivered links. Azure constructs only the documented item path from protected collection/project enrollment plus authenticated original work-item ID, with the existing version-1.0 snapshot exception. `src/matching.mjs:120-130` validates any delivered item/update links against that same collection/project/item identity; contradictory revision IDs also reject. The provider HTTP tests exercise different issue IDs, missing update `workItemId`, wrong collections and contradictory links.

**Proposed clarification:** Document Linear's top-level `url`, optional equivalently bound fallback and contradictory-link rejection. Explicitly permit Azure construction from reviewed canonical enrollment plus original work-item identity, never by stripping arbitrary update URLs or trusting payload URL origins. The coordinator confirmed both resolutions.

### 8. Local service choices are not a shared protocol

**Requirement:** The introduction (lines 18-22) separates the receiver from hosted compatibility; line 739 leaves queues, budgets, retry policy and deployment outside the declarations. Provider authentication is required by line 78 without establishing an ingress URL or administrative endpoint.

**Choice and evidence:** `src/server.mjs:8-76` exposes implementation-specific `POST /ingress/:sourceId` and a durable 200 acknowledgment, not `/activate`, a CAO onboarding API or an OIDC token exchange. `src/main.mjs` requires direct TLS and protected local configuration. The service uses singleton homogeneous batches, SHA-256 GitHub/Jira authentication, Linear's 60-second timestamp window, Azure's configured HTTPS Basic credential, 1 MiB ingress and 2 MiB GitHub-response bounds. These are explicit capability limits. The separate HTTPS CLI smoke artifact verifies readiness and graceful SIGTERM without admissions or network dispatch.

**Proposed clarification:** Require integrations to publish authentication algorithms, limits, acknowledgment semantics and capability profiles without promoting those choices to a shared gh-aw wire contract. Provider variants needing other algorithms, content encodings or larger ingress remain unsupported here.

## Implementation limits that must not be called conformance

### 9. Numeric identity range and trusted cleanup failure windows

**Requirement:** Positive decimal URL/public identifiers are broadly permitted (lines 124-128, 256 and 695), while only selected numeric conditions and Projects source IDs are explicitly safe-integer bounded. Lines 341/414 require credential isolation and truthful revocation reporting.

**Choice and evidence:** `src/core.mjs:40-43,66-145` rejects out-of-range/rounded JSON numbers, duplicate keys and invalid UTF-8 rather than rounding identities; `src/github.mjs:96` rejects a repository ID that cannot be represented exactly in GitHub's numeric `repository_ids` array. Safe-integer source IDs are checked against their canonical URLs. IDs beyond this runtime's numeric range are nonactivating, not "supported as strings" while silently rounded. Malformed-number tests include `9007199254740993`, `9007199254740991.4` and `1.00000000000000001`.

`src/github.mjs:138-152` revokes each returned token in trusted cleanup and never persists credentials. A process kill, lost mint response or unreadable mint response can prevent learning/revoking the created token. Cleanup errors preserve acknowledged/uncertain dispatch distinctions and cannot trigger redispatch. Tests explicitly reject excessive permissions, wrong repository/selection, invalid expiry, failed mint and failed revocation; no fallback credential exists.

**Proposed clarification:** State any supported numeric ranges, require lossless comparison or explicit rejection, distinguish mint ambiguity from successful cleanup, and retain actual-expiry/operational reconciliation as the backstop for unconfirmed revocation. Full arbitrary-size GitHub numeric identity support and crash-proof revocation are not implemented.

## Preserved boundaries

The mandatory original-sender/original-actor destination gate is unchanged. The CAO private key stays server-side. Runtime App tokens remain the destination controller's SHA-pinned `actions/create-github-app-token` responsibility with cleanup; no action is run inside the server. The App JWT's official expiry is implemented, but no runtime OIDC lifetime/renewal policy or activation-to-CAO continuation protocol is introduced.
