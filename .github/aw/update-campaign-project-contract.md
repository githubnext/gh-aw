# Project Update Contract — Machine Check Checklist

This checklist is designed to validate LLM outputs before executing project writes.

## A) Output Structure Checks

- [ ] All writes use `update-project:` blocks (no other write mechanism).
- [ ] Each `update-project` block includes:
  - [ ] `project: "{{.ProjectURL}}"`
  - [ ] `campaign_id: "{{.CampaignID}}"` (top-level)
  - [ ] `content_type` ∈ {`issue`, `pull_request`}
  - [ ] `content_number` is an integer
  - [ ] `fields` object is present

## B) Field Validity Checks

- [ ] `fields.status` ∈ {`Todo`, `In Progress`, `Review required`, `Blocked`, `Done`}
- [ ] `fields.campaign_id` is present on first-add/backfill and equals `{{.CampaignID}}`
- [ ] `fields.worker_workflow` is present on first-add/backfill and is either a known workflow ID or `"unknown"`
- [ ] `fields.repository` matches `owner/repo`
- [ ] `fields.priority` ∈ {`High`, `Medium`, `Low`}
- [ ] `fields.size` ∈ {`Small`, `Medium`, `Large`}
- [ ] `fields.start_date` matches `YYYY-MM-DD`
- [ ] `fields.end_date` matches `YYYY-MM-DD`

## C) Update Semantics Checks

- [ ] For existing items, payload is **status-only** unless explicitly doing a backfill repair.
- [ ] Backfill is used only when required fields are missing/empty/invalid.
- [ ] No payload overwrites `priority`/`size`/`worker_workflow` with defaults during a normal status update.

## D) Read-Write Separation Checks

- [ ] All reads occur before any writes (no read/write interleaving).
- [ ] Writes are batched separately from discovery.

## E) Epic/Hierarchy Checks (Policy-Level)

- [ ] Exactly one Epic exists for the campaign board.
- [ ] Epic is on the board and uses `worker_workflow: "unknown"`.
- [ ] All campaign work issues are sub-issues of the Epic (if supported by environment/tooling).
- [ ] PRs are linked to issues via GitHub linking (e.g., “Closes #123”).

## F) Failure Handling Checks

- [ ] Invalid/deleted/inaccessible items are logged as failures and processing continues.
- [ ] Idempotency is delegated to the `update-project` tool; no pre-filtering by board presence.
