# Python webhook gateway reference

This executable reference exercises the numeric-v2 gateway contract at baseline
`527e59970ad970f26f8e1038b6f301cef80cf876`. It is **not a deployed CAO service,
a production HTTP server, or a fully conforming receiver**. It does not change the
destination's mandatory original-sender/original-actor App gate.

## Run the local experiment

From this directory, install into an isolated environment and run the synthetic
tests:

```bash
python3 -m venv .venv
.venv/bin/python -m pip install .
.venv/bin/python -m unittest discover -s tests -v
```

Python 3.9+ is supported. `cryptography` is the only runtime dependency; its
version and the build backend are pinned in `pyproject.toml`. Tests generate an
ephemeral RSA key, start loopback provider-ingress and fake GitHub HTTP servers,
verify actual RS256 signatures and API credentials, and stop their servers.
They perform no live dispatch, onboarding, secret/grant change, or workflow run.
The optional destination-interoperability test additionally uses installed Node.js
and this checkout's actual `validate_webhook.cjs`.

The standard-library `ThreadingHTTPServer` keeps the experiment self-contained
without introducing a framework. It binds **only `127.0.0.1`**. Any actual
deployment would require a separately reviewed TLS reverse proxy, request/header
and concurrency limits, process isolation, monitoring, private local storage, and
credential provisioning; this example does not establish those controls.

## Local administrative interface

These commands and configuration fields are **implementation-specific choices,
not a proposed shared enrollment/ingress wire contract**:

```bash
.venv/bin/webhook-gateway --db /private/path/gateway.sqlite install-policy /private/path/policy.json
.venv/bin/webhook-gateway --db /private/path/gateway.sqlite serve --port 8080
.venv/bin/webhook-gateway --db /private/path/gateway.sqlite status
.venv/bin/webhook-gateway --db /private/path/gateway.sqlite work-once
```

`GET /healthz` reports process liveness only, not enrollment, readiness, or
conformance. `POST /ingress/<source-id>` receives the provider's original raw JSON
body and authentication headers. A successful durable admission or duplicate
returns HTTP `200`; `not_matched` is an explicit nonactivating result. No network
administration, OIDC activation, token-exchange, or continuation endpoint exists.
Ingress never accepts an App credential, source grant, destination, or manifest.

The local administrator imports one protected JSON policy, with:

| Field | Required content |
| --- | --- |
| `schema_version` | Local policy version `1`, unrelated to the numeric-v2 dispatch schema. |
| `github` | Protected `api_url`, `server_url`, numeric `app_id` and `bot_id`, `app_slug`, and `private_key_env` naming the environment variable containing the server-only PEM key. |
| `sources` | Source-ID-to-enrollment map: `provider`, independent `approved` boolean, epoch-seconds `expires_at`, and `secret_env`. |
| GitHub source | `repository: {id, full_name, owner_id, organization}`; `organization` is independently verified organization login or `null` for user-owned repositories. Optional `projects` maps node IDs to `{owner_kind: "orgs", owner, owner_id, number}`. |
| Jira source | `origin` and required `project` key. Only signed admin webhooks using SHA-256 are supported. |
| Linear source | `organization_id`, reviewed `workspace` URL segment, optional `team_id`. |
| Azure source | `collection_id`, `project_id`, and `collection_url`; `secret_env` contains the configured Basic-auth `username:password`, not a token for outbound GitHub calls. |
| `destinations` | Destination-ID-to-enrollment map: independent `approved`, epoch-seconds `expires_at`, `repository`, numeric `repository_id`, numeric `installation_id`, numeric `account_id`, `account_login`, `default_branch`, approved 40-hex `revision`, `artifact` absolute file path, and `artifact_sha256` of its exact bytes. |
| Destination attestations | `gate_verified: true` and `identity_verified: true` assert **prior independent trusted review** of the destination gate and centrally supplied App/bot binding. Import does not conduct or substitute for that onboarding. |
| `bindings` | Explicit list of `{source: "<source-id>", destination: "<destination-id>"}` reviewed source-to-destination grants. Neither matching selectors nor authentication creates a binding. |

`read_policy` validates the artifact digest, extracts its single-line
`# gh-aw-manifest: ...` header, and validates the full compiled subscriptions.
Caller-supplied subscription arrays are not accepted by the CLI. There is no
discovery-to-approval auto-enrollment. Import uses the same SQLite transaction
fence as dispatch; the local DB is the sole authoritative active policy, not
subsequent edits to the source JSON. Every import changes the admission generation
unless its exact normalized bytes are unchanged. Jobs from any older generation
become `blocked`, even for unrelated policy changes.

Only trusted administrators may write the DB, install policy, or access the
process environment. The database is required to have `0600` permissions; place
it in a private directory. Keys, HMAC secrets, JWTs, and installation credentials
are not persisted in it. Logs/status return fixed error codes, not input URLs,
raw bodies, upstream error text, signatures, or credentials.

## Implemented path

GitHub and Jira SHA-256 HMACs and Linear HMACs cover exact raw bytes; Linear also
requires a signed `webhookTimestamp` within 60 seconds. Azure verifies the
service hook's configured Basic authentication; it does not invent an Azure HMAC
scheme. Duplicate headers/JSON keys, malformed JSON, nonfinite numbers,
prototype keys, and JavaScript-unsafe numeric IDs/selectors reject.

The receiver checks independent source enrollment and source/destination grants,
then evaluates the original native event, selected types, typed catalog paths,
repository/organization selectors, and one complete `any_of` group. Missing data
is never hydrated. Only matching configured literals are projected; command
bodies/arguments, provider objects, and complete transitions never enter jobs.
Frozen normative catalog snapshots in `gateway/catalogs.json` come from the
baseline's notification catalog, three resource entries in `webhook_events.json`,
and provider catalog. Their baseline is recorded in the file.

Supported paths include every GitHub notification family, both security-alert
families, organization-owned Projects with reviewed URL metadata, slash commands,
and evidence-backed Jira/Linear/Azure issue normalization. External provider URL
limitations are explicit: Linear requires original top-level `url` and
`data.identifier`; Azure requires original `resource.url` already naming the work
item, optionally with its enrolled project-ID segment, and checks delivered
revision IDs, `System.Id`, and revision item links for contradictions. Azure update/revision
URLs and user-owned Projects are nonactivating. `label_command` import always
rejects with `unsupported_label_consumption`; no source-write credential is
provided to this process. An artifact containing an unsupported subscription is
rejected as a whole, not partially enabled.

SQLite atomically records the source-scoped original identity, raw fingerprint,
and projected queue entries before HTTP acknowledgement. Matching retries
collapse; conflicting identities and an identical raw body under a new ID reject.
Raw-body fingerprints are intentionally conservative: even different original
bytes that project identically conflict under the same delivery ID. Receipts
survive restarts and are not automatically evicted. The example hard-fails after
100,000 receipts or 1,000 pending/inflight jobs instead of dropping work.

Each dispatch has a single delivery, which is a valid homogeneous nonempty batch.
Both UTF-8 byte limits are checked on the exact compact bytes that will be sent.
There is no truncation, arbitrary URL fetching, or cross-source batching.

The worker commits an `inflight` marker, rechecks its admission generation and
current approvals, and serializes the bounded outgoing sequence with policy
withdrawal. The App client signs an RS256 JWT (`iat = now - 60`, `exp = now + 540`),
resolves and checks the destination installation, requests **only**
`repository_ids: [approved_id]` and `contents: write`, and verifies the returned
scope and actual expiry. It then verifies repository identity, default-branch
revision, and exact approved compiled artifact bytes with the minted token before
sending the emitted route with that **same** token. Source/destination expiry is
checked again immediately before dispatch. The client never follows redirects,
uses fallback credentials, requests resource URLs, or exposes credentials to
workflow payloads. Tokens are revoked after the attempt, including failed scope
validation; cleanup failures are explicit.

## Recovery and limitations

Mint/preparation transport failures and HTTP `429`/`5xx` can retry with exponential
backoff, at most five attempts, while approval remains current. A dispatch `429`
retries; dispatch `5xx` or transport loss becomes `uncertain`. Other dispatch
non-`204` responses fail. A crash after the durable `inflight` marker leaves that
state unresolved and it is **never automatically reclaimed**, including across
process restarts. No operator replay command pretends to prove whether GitHub
accepted the request. Status exposes the unresolved job for separate trusted
reconciliation.

A returned `204` is marked `sent`, not successful workflow execution. Successful
dispatch followed by failed token revocation is `sent_cleanup_failed` and never
redispatched. Abrupt process death may leave a server token alive until its
GitHub expiry: no confirmed revocation is claimed. Tokens are never cached or
reused. Runtime App tokens remain the responsibility of the separate SHA-pinned
`actions/create-github-app-token` action and owning-job cleanup, not this server.

The one-writer fence favors clarity and safety over throughput and can make
ingress return explicit `503` while network work holds the lock. Filesystem
policy compromise, distributed/multi-region storage, a credential broker,
provisioning/rotation of Actions secrets, capacity planning, and exactly-once
workflow execution are outside this reference. The independently approved
default-branch binding is rechecked, but GitHub can advance its default branch
between the final read and dispatch; this external TOCTOU cannot be fenced by
SQLite. These limitations preclude a production/hosted conformance claim.

Implementation-driven specification findings and proposed clarifications are
recorded in [FINDINGS.md](FINDINGS.md).
