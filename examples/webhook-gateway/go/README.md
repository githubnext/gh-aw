# Go webhook receiver reference

This executable exercises the numeric-v2 receiver contract from baseline
`527e59970ad970f26f8e1038b6f301cef80cf876`, with the coordinator's subsequent
clarifications about exact configured content literals, canonical URLs, and
uncertain dispatch outcomes. It uses the repository's Go module and compiler
catalogs, with no nested module or additional dependencies.

**This is not a production CAO deployment or a fully conforming receiver.**
Label commands are rejected because occurrence-aware, independently authorized
source-label consumption is not implemented. The destination App gate remains
unchanged. This server neither provisions secrets nor implements an
activation/OIDC endpoint. See [implementation findings](FINDINGS.md).

## Run locally

Run commands from the repository root:

```bash
go test -race ./examples/webhook-gateway/go -count=1
go build -o /tmp/gh-aw-webhook-go ./examples/webhook-gateway/go
/tmp/gh-aw-webhook-go -subscription-digest /protected/notify.lock.yml
```

`-subscription-digest` prints the SHA-256 of the Go-serialized, typed
`webhook_subscriptions` array from a compiled workflow. It makes no network
requests and grants no authority. The digest format is local to this reference,
not a shared cross-language canonicalization or gateway wire contract.

Supply a private, administrator-owned JSON configuration file. This is the
implementation-specific enrollment format, **not** an API defined by the spec.
The following template is deliberately nonactivating. Replace the revision,
verified identity bindings, subscriptions, and digest through trusted enrollment;
discovery alone must never set either approval boolean.

```json
{
  "api_url": "https://api.github.com",
  "server_url": "https://github.com",
  "app_id": "123",
  "bot_id": "456",
  "sources": {
    "source": {
      "approved": false,
      "provider": "github",
      "secret_env": "GITHUB_SOURCE_WEBHOOK_SECRET",
      "repository": "source/repo",
      "repository_id": "10",
      "owner_id": "42",
      "organization": "source",
      "organization_id": "42"
    }
  },
  "destinations": {
    "destination": {
      "approved": false,
      "repository": "destination/repo",
      "repository_id": "789",
      "installation_id": "321",
      "account_id": "55",
      "revision": "0000000000000000000000000000000000000000"
    }
  },
  "routes": {
    "notify": {
      "destination": "destination",
      "workflow_id": "notify",
      "source_ids": ["source"],
      "manifest_sha256": "",
      "subscriptions": []
    }
  }
}
```

Extract the actual `webhook_subscriptions` array from the compiled manifest and
set it on the route. Set `manifest_sha256` to the digest printed above, not a hash
of the lock file or an independently reordered JSON serialization. All
subscriptions for one destination/workflow belong on one route. The server
requires the exact emitted context paths, route, projection, and both byte limits.

Set protected `GH_AW_GATEWAY_APP_ID` and `GH_AW_GATEWAY_APP_BOT_ID` environment
variables consistently with the approved public binding. Set
`GH_AW_GATEWAY_PRIVATE_KEY_FILE` to the approved CAO App's private RSA PEM file
(PKCS#1 or PKCS#8, at least 2048 bits). Resolve each source's `secret_env`
separately. Never put secret values or keys in the JSON file, command arguments,
workflow, dispatch, or repository.

```bash
/tmp/gh-aw-webhook-go -config /protected/enrollment.json \
  -state /protected/state/gateway.json -check
/tmp/gh-aw-webhook-go -config /protected/enrollment.json \
  -state /protected/state/gateway.json -listen 127.0.0.1:8080
```

The listener accepts loopback addresses only. A reviewed HTTPS terminator must
preserve the exact uncompressed request body and authentication headers; do not
expose this HTTP listener or Basic authentication over a public plaintext hop.
The private local state directory must be on a filesystem supporting atomic
rename, file/directory `fsync`, and POSIX advisory locks. Linux and macOS are
supported; Windows, shared/network filesystems, and multiple active workers are
not supported. A second process using the same state file fails to start.

## Enrollment and ingress reference

The sole endpoint is `POST /ingress/{source-id}` with uncompressed
`Content-Type: application/json`. It is an implementation-specific choice, not a
new standard endpoint. Only configured sources can authenticate, and
authentication does not imply `approved: true` or route/destination approval.
Raw ingress is limited to 25 MiB, JSON nesting to 64, and headers to 16 KiB.
Malformed UTF-8, duplicate keys, multiple JSON documents, unsupported metadata,
unknown projection policies, and missing scopes fail closed.

| Provider | Authentication | Independently enrolled source data |
| --- | --- | --- |
| GitHub | Raw-body HMAC-SHA256, `X-Hub-Signature-256`; original `X-GitHub-Delivery` and `X-GitHub-Event` required for matching | Repository name/ID, owner ID, organization login/ID where applicable. Projects additionally map `projects` node IDs to `{kind: "orgs" or "users", owner, number}`. |
| Jira | Signed admin raw-body HMAC-SHA256, `X-Hub-Signature` | Approved HTTPS `origin` and exact `project` key. Connect JWT and unsigned admin webhooks are unsupported. |
| Linear | Raw-body HMAC-SHA256, `Linear-Signature`; signed `webhookTimestamp` within one minute | Exact `organization_id`, `workspace`, optional `team`. Top-level delivered `url` is preferred; a delivered `data.url` fallback must agree if both exist. |
| Azure DevOps | Configured HTTPS service-hook Basic user (`basic_user`) and password (`secret_env`) | `collection_id`, `collection_url`, exact `project` ID, optional URL-encoded `project_path` for an approved project name. |

Jira and Linear use a SHA-256 digest of the authenticated original bytes as their
delivery ID. Azure uses the original event `id`. All providers also retain a
source-bound raw-body digest and event discriminator to suppress replay with
changed unsigned delivery headers and reject conflicting unsigned event headers.
Body HMAC does not authenticate those headers. This does not promise semantic deduplication if a provider rewrites a
retry's body; see the findings.

Matching uses only delivered evidence, including complete Projects Status pairs
and provider lifecycle/label changes. No API lookup fills in missing source
fields. Source URL identities are bound to enrolled identities and the original
resource. Repository/organization selectors narrow those enrollments, never
authorize new sources. Azure accepts a validated delivered item URL or constructs
the documented item path from the protected enrolled collection base and original
work-item ID. It never strips an update URL or trusts its origin; conflicting
delivered resource links or revision IDs reject the delivery.

The canonical URL profile is intentionally conservative:
ASCII canonical host/scheme/port, no dot segments (including percent-encoded
dots), credentials, query delimiters, or noncanonical encoding. Only exact
catalogued comment/review anchors are emitted.

Each matched family is dispatched in its own one-delivery batch. Exact typed
matching literals, including catalogued body/tag-string selectors, are forwarded;
extra text and unselected values never are. Both serialized UTF-8 byte limits
are checked on the actual transmitted JSON.

## Durable operation and dispatch

HTTP `200` means the original delivery was durably admitted or already recorded,
**not** that it matched, dispatched, activated, or executed successfully. Linear
requires `200`, rather than a generic queue `202`. No body or credentials are
persisted: state holds hashes, minimal projected jobs, approval digest, attempts,
and fixed credential-safe errors.

One worker flushes immediately (one-second polling). Admission is serialized and
fsynced before acknowledgement, without holding its lock during GitHub calls.
The replay ledger retains up to 10,000 original deliveries without eviction;
capacity exhaustion rejects new deliveries. Protected configuration is reloaded
at ingress, before each attempt, and immediately before dispatch. Any change to
its bytes invalidates previously admitted work conservatively; approvals are
prospective, not a request to replay old unmatched deliveries.

For each attempt, the server creates an RS256 App JWT, resolves and verifies the
approved installation, and requests only the destination repository ID and
`contents: write`. It verifies the returned token scope, expiry, repository,
current default-branch SHA, and compiled subscription digest before dispatching.
Requests stay on the configured API origin/base path; redirects are never
followed. The exact minted token sends the dispatch and is revoked afterward,
including rejected scope and cancelled work, with an independent cleanup context.
No token is cached or persisted and no fallback credential is available.

The durable states are `pending`, `sending`, `sent`, `failed`, and `quarantined`.
An explicit dispatch `429`, or a retriable pre-dispatch `429`/`500`/`502`/`503`/`504`,
is retried with exponential backoff (five attempts maximum). A dispatch transport
failure, dispatch `5xx`, unexpected successful status, crashed `sending`
checkpoint, or token-cleanup failure is quarantined instead. Other errors are
terminal `failed`. `sent` means GitHub returned `204` and token cleanup completed,
not that an Actions run succeeded.

Operators must inspect the protected state file for job status/errors. There is
no administrative replay or approval HTTP endpoint. Do not delete replay state
or move quarantined jobs back to pending: resolve unknown GitHub outcomes and
current authorization first. Abrupt termination can prevent revocation; this
reference cannot claim confirmed cleanup or exactly-once execution in that case.
Runtime job tokens remain the destination controller's separate responsibility,
using the official SHA-pinned action and its post-job cleanup.
