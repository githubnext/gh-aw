# Express webhook gateway reference

An executable Node.js/Express receiver experiment against specification commit `527e59970ad970f26f8e1038b6f301cef80cf876`. It is **not a deployed CAO service or a claim of complete conformance**. In particular, label commands cannot enroll, ambiguous sends cannot be safely replayed automatically, and numeric source identities outside JavaScript's safe-integer range reject. [FINDINGS.md](FINDINGS.md) records material specification gaps and the conservative choices.

The runtime and tests are self-contained in this directory. The three local catalog snapshots are derived from that commit; `catalogs/provenance.json` records their source hashes. No runtime import reaches outside this directory.

## Run the local tests

Requires Node.js 24 or newer and npm:

```bash
cd examples/webhook-gateway/node-express
npm ci --ignore-scripts
npm test
```

Tests start only loopback ingress and GitHub HTTP servers, generate an ephemeral synthetic RSA key, and use synthetic non-secret authentication fixtures. No provider/GitHub credentials, enrollment changes, live dispatches, Actions runs, or network services are needed. Dependency versions and integrity hashes are locked.

## Executable service configuration

`npm start` requires protected local enrollment JSON, a private SQLite path, a TLS certificate/key, and centrally administered environment credentials:

| Variable | Value |
| --- | --- |
| `GATEWAY_CONFIG` | Absolute path to the protected enrollment JSON below, reread before admission and each credentialed operation |
| `GATEWAY_DATABASE` | Absolute SQLite filename in a private durable directory; never a shared/network filesystem |
| `GATEWAY_TLS_CERT`, `GATEWAY_TLS_KEY` | Absolute TLS certificate/key filenames |
| `GH_AW_GATEWAY_APP_ID`, `GH_AW_GATEWAY_APP_BOT_ID` | Canonical decimal public identities, identical to the independently verified destination binding |
| `GH_AW_GATEWAY_APP_PRIVATE_KEY` | Server-only RSA PEM, at least 2048 bits, supplied through the service's secret manager |
| The source's `credentialEnv` | Provider signing secret, or Azure's configured `username:password` |
| `HOST`, `PORT` | Optional listener, default `127.0.0.1:8443` |

The listener uses HTTPS directly; it does not trust forwarded headers or add an HTTP reverse-proxy authentication convention. Express's raw parser preserves original uncompressed request bytes. The process sets umask `077`; SQLite uses WAL, `synchronous=FULL`, immediate transactions and a 5-second busy timeout. Protect the enrollment file, its parent directory, TLS files, database directory and service environment from untrusted writers; perform enrollment changes by atomic file replacement. These files are operator-owned controls, not data accepted from webhook callers. Do not give this dispatch App's key to other automation.

The following is an intentionally **nonactivating template**, not enrollment approval. Replace public identity placeholders, independently review source/destination grants, verify the destination gate/effective identity configuration, and insert the **unchanged compiled `webhook_subscriptions` array** before approval. No provisioning or administrative HTTP endpoint exists.

```json
{
  "github": {
    "apiUrl": "https://api.github.com",
    "serverUrl": "https://github.com",
    "appId": "123",
    "botId": "456",
    "privateKeyEnv": "GH_AW_GATEWAY_APP_PRIVATE_KEY"
  },
  "sources": [
    {
      "id": "source",
      "provider": "github",
      "approved": false,
      "revision": "independently-reviewed-source-grant",
      "credentialEnv": "SOURCE_WEBHOOK_SECRET",
      "scope": {
        "repositoryId": "12",
        "repository": "source/repo",
        "ownerId": "11",
        "organization": { "id": "11", "login": "source" }
      }
    }
  ],
  "destinations": [
    {
      "id": "destination-workflow",
      "approved": false,
      "revision": "independently-reviewed-destination-grant",
      "repository": "destination/repo",
      "repositoryId": "789",
      "installationId": "321",
      "accountId": "99",
      "accountLogin": "destination",
      "installationPermissions": { "contents": "write", "metadata": "read" },
      "defaultBranch": "main",
      "workflowRevision": "0000000000000000000000000000000000000000",
      "artifactSha256": "0000000000000000000000000000000000000000000000000000000000000000",
      "gateVerified": false,
      "sourceIds": ["source"],
      "subscriptions": []
    }
  ]
}
```

Each destination enrollment represents one compiled workflow. Different workflows require separate destination IDs; a family may occur only once within a workflow. `installationPermissions` is the exact independently approved installation permission map, not the requested token map. Each minted token always requests only `contents: write` for exactly `repositoryId`; implicit `metadata: read` is accepted. IDs in enrollment are strings; unsafe numeric API identities fail closed rather than rounding.

`workflowRevision` pins the full approved default-branch commit. `artifactSha256` covers the exact compiled lock-file bytes at that revision, including its manifest. Before dispatch, authenticated repository metadata, default-branch revision, artifact hash and embedded subscriptions must match. GitHub's dispatch API cannot atomically pin the workflow execution revision: these reads detect drift but do not remove the default-branch time-of-check/time-of-use race. `gateVerified` is an operator attestation from independently reviewed onboarding, **not** an automatic proof of the workflow's gate or provisioning. An unrelated default-branch commit conservatively requires reapproval. Changing any source/destination/public App binding invalidates queued work's approval fingerprint.

### Provider-specific source scopes

| Provider | `scope` |
| --- | --- |
| GitHub repository families | Immutable `repositoryId`, literal `repository`, immutable `ownerId`; optional enrolled `organization: {id, login}` must match both original organization and owner |
| Projects | Add `project: {nodeId, ownerKind: "orgs" or "users", owner, number}`; repository fields may be omitted for organization-level deliveries; organization-owned Projects require the enrolled organization |
| Jira | `{origin: "https://tenant.atlassian.net", project: "APP"}` |
| Linear | `{organizationId: "...", workspace: "workspace", teamId: "..."}`; `teamId` is optional |
| Azure DevOps | `{collectionId: "...", projectId: "...", collectionUrl: "https://dev.azure.com/organization"}`; optional reviewed `projectSegment` for canonical project-qualified URLs |

GitHub and Jira verify SHA-256 HMACs in their respective documented headers; unknown algorithms reject. Linear verifies `Linear-Signature` and a signed millisecond `webhookTimestamp` within 60 seconds, then checks entity, organization, optional team and canonical workspace/identifier binding. Azure verifies the configured Basic credential over HTTPS and checks the delivered collection/project; it does not invent a service-hook HMAC. Source authentication does not approve that source or any destination.

## Local endpoint and worker behavior

These endpoints, enrollment JSON and acknowledgments are **implementation choices**, not a shared gh-aw/CAO wire contract:

| Endpoint | Behavior |
| --- | --- |
| `GET /health` | Local service readiness |
| `POST /ingress/:sourceId` | Raw `application/json`, at most 1 MiB; standard provider authentication; returns HTTP 200 only after durable admission with `{duplicate, admitted}` |

Unmatched deliveries return `admitted: 0`, not an activation. Invalid signatures return 401; unapproved/mismatched sources return 403; conflicting delivery IDs return 409; oversized requests return 413; unavailable credentials/capacity/storage return 503. Other malformed requests fail closed. Logs contain fixed error codes and local job IDs, never request bodies, HTTP error bodies, authorization headers, private keys or tokens.

Matching checks the authenticated original fields and the complete compiled OR-of-AND predicates without hydration. Projection forwards only canonical URL, intersected activities and actual matching configured scalars. Notification/command anchors remain family-specific; bare `?`/`#` delimiters reject. Each admitted job creates a homogeneous **singleton batch**; both exact UTF-8 JSON byte limits include the wrapper. Coalescing/batch windows are deliberately absent, not replaced with an undocumented batch wire format.

The worker processes one claim at a time, polls every 250 ms, rechecks current approvals before each GitHub call, creates an RS256 App JWT, resolves the approved installation, mints a repository/permission-restricted token, checks the destination artifact, dispatches using that exact token, and revokes it in trusted cleanup. Redirects are disabled, requests time out after 15 seconds and JSON responses are capped at 2 MiB. No resource URL receives an App credential. No PAT, workflow token, App-user credential or fallback token is accepted.

| Durable state | Meaning |
| --- | --- |
| `pending` | Admitted; a retry is scheduled only for an explicit 429 or a pre-dispatch transient failure, at most three total attempts with exponential backoff |
| `in_flight` | Claimed; a surviving claim after restart remains quarantined and is never automatically reclaimed |
| `dispatched` | GitHub returned 204 and token revocation returned 204; no claim about workflow execution |
| `uncertain` | Connection loss or 5xx during dispatch; may already have been accepted; never automatically retried |
| `blocked` / `failed` | Approval/binding invalid or terminal failure; no automatic replay |
| `cleanup_failed` / `uncertain_cleanup_failed` / `dispatched_cleanup_failed` | Token revocation was not confirmed; dispatch uncertainty or acknowledgment remains distinguished, never presented as successful cleanup |

Use `Store.status()` from trusted local tooling to inspect outcomes; there is no unauthenticated status, replay or administration API. This experiment provides no operator replay/reset command. A crash may prevent revocation; no token is persisted, and its returned expiry remains the backstop rather than a false cleanup claim.

Receipts are source-scoped and retained indefinitely. Reusing an ID with different authenticated content rejects; replaying identical content under a rewritten unsigned delivery header collapses. Linear's fingerprint excludes its transport timestamp but includes the original event time. Conservative content deduplication can suppress indistinguishable legitimate events. Hard capacities are 100,000 receipts, 100,000 total jobs and 10,000 pending/in-flight jobs; reaching a bound fails admission explicitly, without evicting replay tombstones.

The destination App/original-actor gate is unchanged. Separate runtime App tokens remain the destination controller's SHA-pinned `actions/create-github-app-token` responsibility with post-job cleanup. There is no activation/OIDC/continuation endpoint, no runtime OIDC lifetime policy, and no destination private-key provisioning.
