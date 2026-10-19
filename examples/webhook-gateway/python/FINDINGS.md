# Python implementation findings

All specification line numbers refer to **baseline
`527e59970ad970f26f8e1038b6f301cef80cf876`**, in
`docs/src/content/docs/specs/webhook-gateway-specification.md`. This report does
not edit or redefine the shared specification.

## Confirmed wording contradiction

**Notification fragments:** section “Expected per-delivery payloads”, line 559,
forbids fragments except command-comment anchors. “GitHub notification
subscriptions”, lines 111–130, and notification examples at lines 607–645 require
`#issuecomment-`, `#pullrequestreview-`, `#discussion_r`, `#discussioncomment-`,
and `#commitcomment-` anchors for notifications without any command. The existing
destination `github_notification_webhook.cjs` also requires these catalog anchors.
Python follows the family-specific catalog, not the blanket prohibition:
`gateway/contracts.py::notification_url`; executable evidence:
`test_all_notification_families_identity_and_anchors`.
**Clarify:** permit precisely cataloged notification and command anchors, while
forbidding fragments on item/API/provider references.

## Underspecified integration decisions

**Dispatch uncertainty and exactly-once limits:** “CAO installation-token dance”,
line 337, defines `204` as acknowledgement but no idempotency key, request lookup,
or recovery protocol. “Dispatch envelope”, lines 541–543, delegates durable replay
to the receiver; the final responsibility boundary, line 739, disclaims
cross-run exactly-once behavior. A crash after GitHub accepts a POST but before
the receiver commits acknowledgement cannot safely distinguish acceptance from
non-delivery. Python persists `inflight` before HTTP, quarantines uncertain
dispatches, retries only known preparation failures/dispatch throttles, and never
automatically reclaims abandoned work: `gateway/store.py::work_once`,
`gateway/github.py::AppDispatcher.dispatch`;
`test_crash_after_dispatch_is_not_automatically_replayed` and
`test_dispatch_failure_unknown_outcome_not_retried`.
**Clarify:** explicitly disclaim exactly-once execution and require an explicit
uncertain-outcome state plus a separately approved reconciliation/replay policy.
Blind retries provide at-least-once effects, not durable exactly-once delivery.

**Label consumption needs a generation-aware fence:** “Gateway command
subscriptions”, line 175, requires durable fencing/idempotency and label removal
after admission, but does not establish mutation ordering relative to dispatch,
relabel-generation identity, conditional source mutation, or recovery after a
crash between removal and dispatch. Removing an old label again can delete a
newly applied label. Python rejects the subscription rather than treating a
receiver-side receipt ID as a source-mutation fence:
`gateway/contracts.py::validate_subscription`;
`test_label_and_user_project_limitations_are_explicit`.
**Clarify:** define required safety properties and evidence for separately
approved consumption protocols, including stale-worker/relabel races and
uncertain source mutations. Do not weaken consumption for demos. This is also an
explicit Python implementation limitation, not a claim that every integration
must use Python's rejection behavior.

**Original provider URL fields and identity evidence:** the Linear example and
binding rules, lines 681–683, require a workspace-qualified original Issue link
but do not name its source field or what original field binds its identifier.
The official linked webhook documentation supplies top-level `url`; older
examples can be unqualified. Azure rules, lines 697–699, prohibit update/revision
URLs but do not name the original canonical work-item URL field or explain
available native update deliveries containing only update/revision URLs.
Python chooses top-level Linear `url`, verifies original `data.identifier`
against the enrolled workspace, rejects contradictory `data.url`, and requires
Azure `resource.url` already to name the exact `workItemId`:
`gateway/contracts.py::resource_url`;
`test_linear_signature_signed_timestamp_and_evidence` and
`test_azure_basic_auth_update_id_and_terminal_projection`.
**Clarify:** specify accepted original URL/identifier paths or reviewed derivation
rules. Missing original evidence must remain nonactivating; do not silently
hydrate a URL with an API read or substitute an update ID. Python's narrow Azure
field choice is a declared limitation, not authority over all provider versions.

**Deployment/enrollment interchange is not specified:** “Approved identity
binding”, lines 247–257, requires independent approvals, while the final
responsibility boundary, line 739, leaves approved artifact/revision binding,
queue deployment, budgets and retries to integrations. No administrative policy
schema, revision withdrawal transaction, authentication-secret selection,
freshness lifetime, or ingress route is established. Python uses a local
protected policy import with expiring independent source/destination approvals,
explicit source/destination pairs, exact artifact digest, gate/binding review
attestations, and a SQLite policy generation. Any generation change blocks old
jobs: `gateway/policy.py::read_policy`, `gateway/store.py::install` and
`gateway/store.py::work_once`;
`test_artifact_approval_import_and_unknown_projection_fail_closed`,
`test_admission_fences_policy_withdrawal_and_expiry`, and
`test_policy_changes_are_serialized_with_inflight_dispatch`.
**Clarify:** distinguish deployment-specific administrative interfaces from
normative shared wire contracts; require integrations to document their protected
approval/revision/freshness and revocation fences rather than assuming discovery
or a well-shaped manifest activates authority.

**Default-branch execution revision race:** “CAO installation-token dance”,
line 337, requires availability on the default branch; conformance/adoption,
line 509, requires independently approved receiver and destination revisions.
GitHub dispatch chooses the default branch at execution, not an approved immutable
`ref` supplied in this transport. Python verifies the current branch SHA and
compiled artifact before dispatch, but cannot transactionally fence a concurrent
GitHub branch update: `gateway/github.py::AppDispatcher.dispatch`;
`test_installation_and_default_revision_must_still_match`.
**Clarify:** describe the assurance and remaining TOCTOU limit; require trusted
repository branch/deployment controls or a separately specified execution-revision
binding before stronger hosted readiness claims. Do not invent a payload proof,
OIDC activation endpoint, or permissive destination gate.

## Explicit implementation limitations

Python is not a fully conforming hosted receiver. Label commands and user-owned
Projects reject during policy import; Azure project-name URL segments and native
update URLs without an original canonical item link are unsupported. Jira
authentication supports SHA-256 signed admin webhooks only, not Connect/OAuth
webhook authentication or future unreviewed signature algorithms.

The loopback standard-library server requires separately reviewed TLS/proxy,
rate/concurrency and process-isolation controls. Its SQLite fence is local,
single-writer and deliberately low-throughput. Receipt retention is permanent
until its explicit finite budget rejects new work. Raw-fingerprint identity is
stricter than batch-local projected-equality deduplication, so a provider retry
that changes its body while retaining its delivery ID conflicts. Operator replay,
source-label writes, Actions-secret provisioning/rotation, and distributed
credential brokering are intentionally absent.

The reference does not independently conduct onboarding or rewrite the
destination gate. Its protected administrator attests the approved App/bot and
gate review. It does not claim to infer those approvals from subscription
discovery. Runtime credentials are neither minted by the server for an agent
job nor provisioned from the CAO dispatch key.
