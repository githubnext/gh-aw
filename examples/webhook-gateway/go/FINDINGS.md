# Implementation-driven Go findings

Evidence baseline: `527e59970ad970f26f8e1038b6f301cef80cf876`,
`docs/src/content/docs/specs/webhook-gateway-specification.md`. Line references
below refer to that commit, not subsequent coordinator edits. The shared spec and
destination gate were not edited by this implementation.

Reproduce the focused evidence with:

```bash
go test -race ./examples/webhook-gateway/go -count=1
```

## 1. Content-bearing selector contradiction, subsequently clarified

**Confirmed contradictory prose:** provider configuration, line 80, prohibits
resource bodies and complete transition evidence; compiled projection, line 237,
prohibits descriptions beyond minimal references/literals. However, the normative
Projects catalog allows `body.from`/`body.to`
(`pkg/workflow/data/webhook_events.json:451-452`), and the provider catalog allows
the entire native `tags.from`/`tags.to` strings
(`pkg/workflow/data/provider_issue_webhooks.json:76-87`). Both are compiler-accepted
and require projecting those exact literal values to satisfy their conditions.

**Consolidated decision implemented:** retain these advertised selectors as a
narrow exception for exact configured scalar literals proven to match. Never
forward an arbitrary body, entire delta object, unselected current value, or a
configured string that only partially matches.

**Evidence:** `TestCatalogContentSelectorsProjectOnlyExactConfiguredLiterals` in
`projection_test.go` compiles both declarations, verifies exact-literal projection,
and rejects the same original text with extra suffix content. `matchedFilters` in
`projection.go` intersects every applicable typed condition.

**Clarification:** explicitly distinguish an exact configured body/tag-string
literal from copying general resource content. The coordinator accepted this
clarification; the initial Go refusal was replaced by positive/negative coverage.

## 2. No shared delivery-key or replay-retention profile

**Underspecified deployment choice:** dispatch envelope, line 543, requires a
stable original delivery identity; line 739 leaves replay/deduplication to the
receiver. Neither defines provider-specific delivery-key evidence, retention,
enrollment epoch boundaries, nor what constitutes a retry if signed timestamps
or serialization change.

**Choice:** preserve GitHub's original delivery ID and Azure's event ID. Use a
source-bound authenticated raw-body SHA-256 for Jira/Linear. Maintain an additional
raw-body replay index and event discriminator because provider delivery headers
are not covered by the body HMAC. Reject conflicting discriminators, including
under changed unsigned delivery IDs. Retain all records, reject admission at
10,000 records, and never evict automatically without a provably bounded horizon.

**Evidence:** `originalID` in `ingress.go`; `Store.admit` in `store.go`;
`TestConcurrentReplayAndConflictingIdentity`,
`TestReplayBindsUnsignedDiscriminatorAndRetainsTombstones`,
`TestIngressToScopedAppDispatchAndRevocation`, and
`TestAdmissionCapacityAndWriteFailureDoNotAcknowledge`.

**Limitation:** semantic retries whose authenticated bytes change can obtain a
different key. Reuse of one key with changed native bytes fails even if the
minimal projection is equal. This is deliberately stricter than runner-local
projected equality and is not a claim of provider-wide exactly-once identity.

**Clarification:** define per-provider retry-stable identity evidence and the
minimum replay horizon, or require a reviewed enrollment-specific profile.
Specify whether differing native bytes under one original identity may ever
merge and whether negative/unmatched decisions consume the replay key.

## 3. Ambiguous dispatch success cannot provide exactly-once execution

**Underspecified failure protocol:** token sequence, line 337, describes `204`
as acceptance rather than execution success; line 739 assigns queue/retries and
cross-run exactly-once behavior to the receiver without an atomic GitHub/local
commit protocol.

**Choice:** fsync `sending` before HTTP; persist `sent` only after `204` and
successful token revocation. Quarantine dispatch transport failures, `5xx`,
unexpected successful statuses, cleanup failures, and recovered `sending`
checkpoints. Retry explicit rate rejection and safe pre-dispatch failures only,
with current grants/digests rechecked and five attempts maximum.

**Evidence:** `Store.process`/`openStore`, `GitHub.useToken`, and
`TestUncertainSendingCheckpointDoesNotAutomaticallyReplay`,
`TestDispatchFailureMatrixIsCredentialSafe`,
`TestRetryRechecksApprovalAndPersistsAttempt`.

**Clarification:** explicitly require unknown-outcome quarantine and a reviewed
reconciliation protocol. Without destination idempotency/receipts, do not promise
exactly-once dispatch or run execution. The coordinator adopted quarantine.

## 4. Label consumption needs an occurrence-aware mutation protocol

**Underspecified protocol and implementation limitation:** gateway commands,
line 175, requires separate source write approval, durable fencing/idempotency,
and post-admission label consumption. It does not define an occurrence identity,
conditional mutation, reapplication semantics, or recovery from uncertain source
mutation/dispatch results.

**Choice:** reject every `label_command` subscription at configuration validation,
before admission, mutation, token minting, or dispatch. No source-write credential
is implemented. This path is explicitly unsupported/nonconforming.

**Evidence:** `validateSubscription` in `config.go` and
`TestLabelCommandUnsupportedRatherThanUnfenced`.

**Clarification:** require durable admission, confirmed correctly fenced
consumption of that application occurrence, then dispatch. An uncertain removal
must quarantine; retrying an unconditional DELETE can consume a newly reapplied
label. The coordinator adopted this ordering/limitation.

## 5. Canonical equality needs a named URL algorithm

**Underspecified original profile:** expected payloads, line 559, requires exact
URL parser canonical equality without naming the parser. Go's `net/url` round
trip can preserve uppercase hosts, explicit default ports, encoded dot segments,
and other forms normalized differently by the destination's WHATWG parser.

**Choice after consolidation:** enforce a conservative ASCII subset of WHATWG
canonical serialization, plus exact family-specific URL/anchor binding.
Noncanonical host/scheme/default ports, dot segments including encoded dots,
empty query/fragment delimiters, alternate IPv4 representations, credentials,
and encoded slashes are rejected. No parser-normalized input is silently rewritten.

**Evidence:** `canonicalResource`, `notificationURL`, and
`TestCanonicalURLConservatism` in `projection_test.go`; canonical family coverage
in `TestAllGitHubNotificationFamilies`.

**Limitation:** this is not a general-purpose WHATWG parser. It intentionally
rejects some URLs outside its conservative profile. Canonical Unicode must
already use ASCII/percent-encoded representations supported by the family.

**Clarification:** name WHATWG URL serialization, forbid empty `?`/`#`, and
publish cross-language acceptance vectors. The coordinator adopted that profile.

## 6. Linear and Azure native URL evidence needs an explicit source path

**Underspecified native input:** expected Linear payloads, lines 671-683, requires
binding a workspace-qualified URL to the native issue, but does not identify its
native URL field. Linear's official webhook contract documents top-level `url`.
Azure, lines 694-698, distinguishes work-item/update IDs without specifying which
delivered URL field supplies the work-item reference for update events.

**Choice:** prefer Linear's top-level `url`, accept delivered `data.url` only as a
fallback, and reject conflicting values. Require immutable `data.id`, matching
`data.identifier`, and reviewed workspace/org/team. For Azure, validate delivered
work-item references or construct the documented item path using only protected
collection enrollment and the authenticated original work-item ID. Never strip an
update URL or trust a payload origin. Reject contradictory delivered item links
and revision IDs. Bind optional project URL segments to the enrolled project ID
or explicit encoded project name.

**Evidence:** `linearProjection`/`azureProjection` in `providers.go`;
`TestLinearNativeTopLevelURLAndCompleteTerminalEvidence`,
`TestAzureWorkItemIdentityAndSelectedTerminalProjection`,
`TestAzureProtectedConstructionAndContradictoryNativeEvidence`.

**Clarification:** catalog the accepted native URL paths and consistency rules;
missing canonical evidence is nonactivating unless protected Azure construction
is possible. The coordinator confirmed Linear's top-level field and consistency
rule and Azure's narrowly bounded construction alternative.

## 7. Approved revision binding has a check-to-dispatch window

**Underspecified deployment choice:** approved binding, lines 258-260; default
branch/current grants, line 337; and boundary, line 739, require current reviewed
artifacts without defining discovery, approval digests, or execution-revision
handshake. `repository_dispatch` accepts no immutable execution SHA.

**Choice:** protected local configuration separately binds immutable source and
destination identities, source-to-route authority, approved default-branch SHA,
and the exact compiled subscription array digest. Verify installation metadata,
minted scope, current repository/default branch, and the lock file's manifest
through authenticated, redirect-free API calls. Reload approval before dispatch.
Any protected-config byte change invalidates queued work.

**Evidence:** `Config.validate`, `Gateway.process`, `GitHub.useToken`;
`TestApprovalWithdrawnBetweenMintAndDispatch`,
`TestRetryRechecksApprovalAndPersistsAttempt`, and revision-drift failure coverage.

**Limitations:** there is still a check-to-dispatch race if the default branch
moves after the verified read. Trusted destination onboarding and deployment
must prevent that race or add independent execution-revision verification; this
reference cannot certify hosted activation. Public App/bot binding approval is
an onboarding input, not a claim that this server performed onboarding.

**Clarification:** distinguish protected implementation-specific grants/digests
from the shared notification wire format, and explicitly specify where the
approved executing revision is verified. Do not invent an activation/OIDC
endpoint or modify the mandatory original-sender/original-actor gate.
