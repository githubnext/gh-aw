{{if .ProjectURL}}
# Project Update Instructions (Authoritative Write Contract)

## Project Board Integration

This file defines the ONLY allowed rules for writing to the GitHub Project board.
If any other instructions conflict with this file, THIS FILE TAKES PRECEDENCE for all project writes.

---

## 0) Hard Requirements (Do Not Deviate)

- Writes MUST use only the `update-project` safe-output.
- All writes MUST target exactly:
  - **Project URL**: `{{.ProjectURL}}`
- Every item MUST include:
  - `campaign_id: "{{.CampaignID}}"`

## Campaign ID

All campaign tracking MUST key off `campaign_id: "{{.CampaignID}}"`.

---

## 1) Required Project Fields (Must Already Exist)

| Field | Type | Allowed / Notes |
|---|---|---|
| `status` | single-select | `Todo` / `In Progress` / `Review required` / `Blocked` / `Done` |
| `campaign_id` | text | Must equal `{{.CampaignID}}` |
| `worker_workflow` | text | workflow ID or `"unknown"` |
| `repository` | text | `owner/repo` |
| `priority` | single-select | `High` / `Medium` / `Low` |
| `size` | single-select | `Small` / `Medium` / `Large` |
| `start_date` | date | `YYYY-MM-DD` |
| `end_date` | date | `YYYY-MM-DD` |

Field names are case-sensitive.

---

## 2) Content Identification (Mandatory)

Use **content number** (integer), never the URL as an identifier.

- Issue URL: `.../issues/123` → `content_type: "issue"`, `content_number: 123`
- PR URL: `.../pull/456` → `content_type: "pull_request"`, `content_number: 456`

---

## 3) Deterministic Field Rules (No Inference)

These rules apply to any time you write fields:

- `campaign_id`: always `{{.CampaignID}}`
- `worker_workflow`: workflow ID if known, else `"unknown"`
- `repository`: extract `owner/repo` from the issue/PR URL
- `priority`: default `Medium` unless explicitly known
- `size`: default `Medium` unless explicitly known
- `start_date`: issue/PR `created_at` formatted `YYYY-MM-DD`
- `end_date`:
  - if closed/merged → `closed_at` / `merged_at` formatted `YYYY-MM-DD`
  - if open → **today’s date** formatted `YYYY-MM-DD` (**required for roadmap view; do not leave blank**)

For open items, `end_date` is a UI-required placeholder and does NOT represent actual completion.

---

## 4) Read-Write Separation (Prevents Read/Write Mixing)

1. **READ STEP (no writes)** — validate existence and gather metadata
2. **WRITE STEP (writes only)** — execute `update-project`

Never interleave reads and writes.

---

## 5) Adding an Issue or PR (First Write)

### Adding New Issues

When first adding an item to the project, you MUST write ALL required fields.

```yaml
update-project:
  project: "{{.ProjectURL}}"
  campaign_id: "{{.CampaignID}}"
  content_type: "issue"              # or "pull_request"
  content_number: 123
  fields:
    status: "Todo"                   # "Done" if already closed/merged
    campaign_id: "{{.CampaignID}}"
    worker_workflow: "unknown"
    repository: "owner/repo"
    priority: "Medium"
    size: "Medium"
    start_date: "2025-12-15"
    end_date: "2026-01-03"
```

---

## 6) Updating an Existing Item (Minimal Writes)

### Updating Existing Items

Preferred behavior is minimal, idempotent writes:

- If item exists and `status` is unchanged → **No-op**
- If item exists and `status` differs → **Update `status` only**
- If any required field is missing/empty/invalid → **One-time full backfill** (repair only)

### Status-only Update (Default)

```yaml
update-project:
  project: "{{.ProjectURL}}"
  campaign_id: "{{.CampaignID}}"
  content_type: "issue"              # or "pull_request"
  content_number: 123
  fields:
    status: "Done"
```

### Full Backfill (Repair Only)

```yaml
update-project:
  project: "{{.ProjectURL}}"
  campaign_id: "{{.CampaignID}}"
  content_type: "issue"              # or "pull_request"
  content_number: 123
  fields:
    status: "Done"
    campaign_id: "{{.CampaignID}}"
    worker_workflow: "WORKFLOW_ID"
    repository: "owner/repo"
    priority: "Medium"
    size: "Medium"
    start_date: "2025-12-15"
    end_date: "2026-01-02"
```

---

## 7) Idempotency Rules

- Matching status already set → **No-op**
- Different status → **Status-only update**
- Invalid/deleted/inaccessible URL → **Record failure and continue**

## Write Operation Rules

All writes MUST conform to this file and use `update-project` only.

---

## 8) Logging + Failure Handling (Mandatory)

For every attempted item, record:

- `content_type`, `content_number`, `repository`
- action taken: `noop | add | status_update | backfill | failed`
- error details if failed

Failures must not stop processing remaining items.

---

## 9) Worker Workflow Policy

- Workers are campaign-agnostic.
- Orchestrator populates `worker_workflow`.
- If `worker_workflow` cannot be determined, it MUST remain `"unknown"` unless explicitly reclassified by the orchestrator.

---

## 10) Parent / Sub-Issue Rules (Campaign Hierarchy)

- Each project board MUST have exactly **one Epic issue** representing the campaign.
- The Epic issue MUST:
  - Be added to the project board
  - Use the same `campaign_id`
  - Use `worker_workflow: "unknown"`

- All campaign work issues (non-epic) MUST be created as **sub-issues of the Epic**.
- Issues MUST NOT be re-parented based on worker assignment.

- Pull requests cannot be sub-issues:
  - PRs MUST reference their related issue via standard GitHub linking (e.g. “Closes #123”).

- Worker grouping MUST be done via the `worker_workflow` project field, not via parent issues.

- The Epic issue is narrative only.
- The project board is the sole authoritative source of campaign state.

---

## Appendix — Machine Check Checklist (Optional)

This checklist is designed to validate outputs before executing project writes.

### A) Output Structure Checks

- [ ] All writes use `update-project:` blocks (no other write mechanism).
- [ ] Each `update-project` block includes:
  - [ ] `project: "{{.ProjectURL}}"`
  - [ ] `campaign_id: "{{.CampaignID}}"` (top-level)
  - [ ] `content_type` ∈ {`issue`, `pull_request`}
  - [ ] `content_number` is an integer
  - [ ] `fields` object is present

### B) Field Validity Checks

- [ ] `fields.status` ∈ {`Todo`, `In Progress`, `Review required`, `Blocked`, `Done`}
- [ ] `fields.campaign_id` is present on first-add/backfill and equals `{{.CampaignID}}`
- [ ] `fields.worker_workflow` is present on first-add/backfill and is either a known workflow ID or `"unknown"`
- [ ] `fields.repository` matches `owner/repo`
- [ ] `fields.priority` ∈ {`High`, `Medium`, `Low`}
- [ ] `fields.size` ∈ {`Small`, `Medium`, `Large`}
- [ ] `fields.start_date` matches `YYYY-MM-DD`
- [ ] `fields.end_date` matches `YYYY-MM-DD`

### C) Update Semantics Checks

- [ ] For existing items, payload is **status-only** unless explicitly doing a backfill repair.
- [ ] Backfill is used only when required fields are missing/empty/invalid.
- [ ] No payload overwrites `priority`/`size`/`worker_workflow` with defaults during a normal status update.

### D) Read-Write Separation Checks

- [ ] All reads occur before any writes (no read/write interleaving).
- [ ] Writes are batched separately from discovery.

### E) Epic/Hierarchy Checks (Policy-Level)

- [ ] Exactly one Epic exists for the campaign board.
- [ ] Epic is on the board and uses `worker_workflow: "unknown"`.
- [ ] All campaign work issues are sub-issues of the Epic (if supported by environment/tooling).
- [ ] PRs are linked to issues via GitHub linking (e.g. “Closes #123”).

### F) Failure Handling Checks

- [ ] Invalid/deleted/inaccessible items are logged as failures and processing continues.
- [ ] Idempotency is delegated to the `update-project` tool; no pre-filtering by board presence.

{{end}}
