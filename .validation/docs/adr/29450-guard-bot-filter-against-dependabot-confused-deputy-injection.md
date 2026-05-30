# ADR-29450: Guard Bot-Filter System Against Dependabot Confused Deputy Injection

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Every compiled gh-aw workflow runs a `pre_activation` job that executes two JavaScript checks before the agent starts: `check_membership.cjs`, which verifies that `github.actor` holds the required repository permission (or appears in the bot allowlist), and `check_skip_bots.cjs`, which suppresses the workflow when `github.actor` is in the workflow's `skip-bots` list. Both scripts treated `github.actor` — the account that *triggered* the run — as the sole identity signal, without verifying that the triggering account is also the account that *authored* the underlying pull request or comment. GitHub Actions sets `github.actor` to whichever account caused the most-recent workflow trigger, which can differ from the PR author when a third party (such as Dependabot) reacts to a comment on a fork PR. This discrepancy is exploitable via the [Dependabot Confused Deputy Injection](https://labs.boostsecurity.io/articles/weaponizing-dependabot-pwn-request-at-its-finest/) technique, where an attacker manipulates Dependabot into re-triggering a run with `actor = dependabot[bot]` so that the actor's elevated permissions — not the attacker's — are used for the permission check, causing the agent to execute against the attacker's untrusted code.

### Decision

We will introduce a `isConfusedDeputyAttack(actor, eventName, payload)` pure helper in `check_permissions_utils.cjs` that detects when `github.actor` does not match the event-specific author field in the webhook payload. For `pull_request` events, the check is intentionally scoped to the `synchronize` action only — the documented attack vector — because other `pull_request` actions such as `labeled`, `unlabeled`, `assigned`, and `review_requested` legitimately have the actor differ from the PR author (the actor is whoever performed that action, not the person who opened the PR). For `pull_request_review` events, the check compares against `payload.review.user.login` (the reviewer). For `pull_request_review_comment` and `issue_comment` events, the check compares against `payload.comment.user.login` (the comment author). For all other event types the helper returns `false`. `check_membership.cjs` will call this helper immediately before the `checkRepositoryPermission` call and deny any run where the check returns `true`. `check_skip_bots.cjs` will call the same helper after the "no skip-bots" early return and, if a confused deputy is detected, will *not* suppress the workflow — allowing the run to proceed to `check_membership.cjs` where it will be denied explicitly rather than silently skipped. In addition, the compiled condition for the `"dependabot pull request"` trigger shorthand in `trigger_parser.go` will be tightened from `github.actor == 'dependabot[bot]'` to `github.actor == 'dependabot[bot]' && github.event.pull_request.user.login == 'dependabot[bot]'`, ensuring the shorthand only activates when the PR was genuinely authored by Dependabot.

### Alternatives Considered

#### Alternative 1: Allowlist-Only Defence (Opt-In per Repository)

Restrict `dependabot[bot]` to the bot allowlist only for repositories that explicitly opt in, requiring a configuration change for any repo that wants Dependabot PRs handled by the agent. This was rejected because it places the security burden on individual repository owners, does not address the `skip-bots` bypass variant (where the attacker impersonates Dependabot to suppress the workflow for their own PR), and still leaves unenrolled repositories vulnerable. It also provides no protection for `issue_comment` or `pull_request_review` confused-deputy variants.

#### Alternative 2: Use `GITHUB_TRIGGERING_ACTOR` Instead of `github.actor`

Replace all references to `github.actor` with the `GITHUB_TRIGGERING_ACTOR` environment variable, which GitHub sets to the account that initiated a re-run. This was rejected because `GITHUB_TRIGGERING_ACTOR` is only populated during re-run scenarios; it is absent for the original confused-deputy trigger (the first run that Dependabot initiates in response to a comment), making it an incomplete defence.

#### Alternative 3: Deny All `[bot]`-Suffixed Actors on PR Events

Reject any actor whose login ends in `[bot]` for `pull_request` and related events. This was rejected because legitimate bot-authored PRs — from Renovate, automated release tooling, or custom GitHub Apps — should remain functional without per-repository exemption lists, and the blanket denial would break those workflows.

#### Alternative 4: Compile-Time Fix Only (`trigger_parser.go`)

Tighten only the `"dependabot pull request"` compiled condition and make no runtime changes to `check_membership.cjs` or `check_skip_bots.cjs`. This was rejected because the compile-time condition protects only that specific trigger shorthand; it leaves `check_membership.cjs` vulnerable whenever the workflow's activation condition passes through a path other than the `"dependabot pull request"` shorthand, and it provides no coverage for `issue_comment` or `pull_request_review` events.

### Consequences

#### Positive
- Closes the confused deputy attack vector for `pull_request`, `issue_comment`, `pull_request_review`, and `pull_request_review_comment` events without any per-repository configuration change.
- The `"dependabot pull request"` trigger shorthand is made safe at compile time independently of the runtime check, providing defence-in-depth.
- `workflow_call`, `push`, `schedule`, `workflow_dispatch`, and `merge_group` events are entirely unaffected: `isConfusedDeputyAttack` returns `false` for all of them, producing no false positives.
- The confused deputy result is surfaced as an explicit `core.warning` and written to the job summary, making attacks visible in workflow logs rather than silently ignored.

#### Negative
- The `pull_request:synchronize` guard assumes that whenever a bot is the actor on a sync event, the bot should also be the PR author. Legitimate scenarios where a bot force-pushes to a PR it did not open (e.g., an automated rebase bot) would be denied. Such bots should open their own PRs rather than rebasing others' PRs; if they do not, they would need to be exempted via the bot allowlist.
- `check_permissions_utils.cjs` gains a new export (`isConfusedDeputyAttack`) and both `check_membership.cjs` and `check_skip_bots.cjs` gain a new import and call-site, modestly increasing the size and coupling of the pre-activation check surface.

#### Neutral
- The guard runs exclusively in the `pre_activation` job; the agent execution job is structurally unchanged.
- The `isConfusedDeputyAttack` function performs no network I/O and adds negligible latency to the pre-activation job.
- Existing compiled workflows do not need to be recompiled; the protection takes effect as soon as the updated `actions/setup/js/` helpers are deployed, because the JavaScript files are copied to `/tmp/gh-aw/actions` at runtime rather than embedded in compiled lock files.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### `isConfusedDeputyAttack` Helper

1. The `check_permissions_utils.cjs` module **MUST** export a pure function named `isConfusedDeputyAttack(actor, eventName, payload)` that returns a boolean.
2. For `pull_request` events, the function **MUST** return `true` if and only if `payload.action` strictly equals `"synchronize"` **and** `actor` does not strictly equal `payload.pull_request.user.login`. For all other `pull_request` actions (e.g., `labeled`, `unlabeled`, `opened`, `assigned`, `review_requested`), the function **MUST** return `false`, because for those actions the actor is legitimately the person who performed the action rather than the PR author.
3. For `pull_request_review` events, the function **MUST** return `true` if and only if `actor` does not strictly equal `payload.review.user.login`.
4. For `pull_request_review_comment` events, the function **MUST** return `true` if and only if `actor` does not strictly equal `payload.comment.user.login`.
5. For `issue_comment` events, the function **MUST** return `true` if and only if `actor` does not strictly equal `payload.comment.user.login`.
6. For all other event types (`push`, `schedule`, `workflow_dispatch`, `workflow_call`, `merge_group`, etc.), the function **MUST** return `false`.
7. If the event-specific author field is absent or `undefined` in the payload for an otherwise-covered event type, the function **MUST** return `false` to avoid false positives.
8. The function **MUST NOT** perform any network I/O, file I/O, or external side effects.

### Membership Check (`check_membership.cjs`)

1. After the safe-events bypass and before any call to `checkRepositoryPermission`, `check_membership.cjs` **MUST** call `isConfusedDeputyAttack` with the current `github.actor`, `github.event_name`, and `github.event` payload.
2. If `isConfusedDeputyAttack` returns `true`, the membership check **MUST** set `result` to `confused_deputy`, set `is_team_member` to `false`, emit a `core.warning` identifying the actor mismatch, write a denial summary to the job summary, and **MUST NOT** proceed to `checkRepositoryPermission`.
3. A run denied with `result = confused_deputy` **MUST** be treated identically to a `no_permission` denial by all downstream consumers of the `result` output.

### Skip-Bots Check (`check_skip_bots.cjs`)

1. After the "no skip-bots configured" early return and before the skip-bots matching loop, `check_skip_bots.cjs` **MUST** call `isConfusedDeputyAttack` with the current actor, event name, and payload.
2. If `isConfusedDeputyAttack` returns `true`, `check_skip_bots.cjs` **MUST** set `skip_bots_ok` to `true`, set `result` to `not_skipped`, and return immediately — **MUST NOT** suppress the workflow — so that the run proceeds to `check_membership.cjs` where it will be denied with an explicit result.
3. `check_skip_bots.cjs` **MUST NOT** suppress the workflow solely because `github.actor` appears in the `skip-bots` list when the confused deputy check has fired; suppressing at this stage **MUST NOT** be used as a substitute for the explicit denial in `check_membership.cjs`.

### Dependabot Pull Request Trigger Shorthand

1. The compiled condition for the `"dependabot pull request"` trigger shorthand **MUST** be `github.actor == 'dependabot[bot]' && github.event.pull_request.user.login == 'dependabot[bot]'`.
2. Implementations **MUST NOT** emit a condition that checks only `github.actor == 'dependabot[bot]'` without the corresponding `pull_request.user.login` guard for this shorthand.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [adr-writer agent]. The PR author must review, complete, and finalize this document before the PR can merge.*
