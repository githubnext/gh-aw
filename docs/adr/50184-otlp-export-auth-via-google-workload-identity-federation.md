# ADR-50184: OTLP Export Authentication via Google Workload Identity Federation

**Date**: 2026-08-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

gh-aw exports OpenTelemetry spans via OTLP to configurable backends. The existing authentication mechanism supports a static `Authorization` header (for bearer tokens) and a GitHub App token approach that mints a GitHub OIDC JWT and passes it as the OTLP credential. Google Cloud's managed telemetry endpoint (`telemetry.googleapis.com`) requires a Google-native access token rather than a GitHub JWT or a static API key. Storing a long-lived Google service account key as a Actions secret is undesirable from a security and operational standpoint. GitHub Actions already issues short-lived OIDC tokens that Google's Security Token Service (STS) can exchange for Google access tokens via Workload Identity Federation, making keyless authentication feasible within the existing runner environment.

### Decision

We will add a `workload-identity` block under `observability.otlp` that enables users to configure Google Workload Identity Federation as an OTLP authentication mechanism. When configured, gh-aw will generate two additional workflow steps before the setup action: one step that mints a GitHub Actions OIDC token (reusing the existing OIDC mint infrastructure) and a second step that exchanges that token for a Google access token via the Google STS API (`sts.googleapis.com/v1/token`). If a `service-account` is provided, a further impersonation call to `iamcredentials.googleapis.com` is made to obtain a service-account-scoped access token. The resulting access token is then passed to the gh-aw setup action as `otlp-oidc-token`, identical to the existing GitHub App flow, requiring no changes downstream of the setup action.

### Alternatives Considered

#### Alternative 1: Static Google Service Account Key as Actions Secret

Users could store a Google service account JSON key as a GitHub Actions secret and supply it as a static header value. This is straightforward to implement (no new code needed — the existing `headers` map already supports it) but requires long-lived credential management, periodic rotation, secret exposure risk if the repository is compromised, and violates the principle of least privilege by granting standing access.

#### Alternative 2: Use the Existing GitHub App Token as the OTLP Bearer Token, Relying on a Google Cloud Custom Token Validator

Google Cloud's OTLP endpoint could theoretically be fronted by a custom proxy or Cloud Endpoints configuration that validates GitHub JWTs and translates them to Google credentials. This avoids adding token exchange logic to gh-aw but requires users to deploy and maintain additional infrastructure, making it impractical as a default supported path. It also increases operational complexity significantly for a feature that should "just work."

### Consequences

#### Positive
- Keyless authentication: no long-lived secrets to store, rotate, or audit.
- Short-lived tokens: both the GitHub OIDC JWT and the resulting Google access token are scoped to the job and expire rapidly, minimizing blast radius if intercepted.
- Consistent user experience: the `workload-identity` config block follows the same frontmatter pattern as other auth configuration in gh-aw.
- No downstream changes to the setup action are required; it receives `otlp-oidc-token` as before.

#### Negative
- Only Google is supported as a WIF provider in this iteration; Azure, AWS, and other cloud providers would require additional implementation effort.
- Each span-emitting job incurs two extra HTTP round-trips (GitHub OIDC mint + Google STS exchange, plus an optional impersonation call) before the setup step, adding latency per job.
- Google STS and IAM Credentials API endpoints (`sts.googleapis.com`, `iamcredentials.googleapis.com`, `oauth2.googleapis.com`) are automatically added to the network allowlist, slightly expanding the job's outbound network surface.

#### Neutral
- The PR adds a validator (`validateOTLPWorkloadIdentity`) that prevents combining `workload-identity` with the GitHub App credential approach, making the two auth paths mutually exclusive.
- The `getOTLPAuthTokenStepID` helper abstracts which step ID produced the final access token, keeping downstream code DRY but adding a level of indirection for readers of `compiler_yaml_step_generation.go`.
- JSON schema (`main_workflow_schema.json`) is updated to formally define and validate the new `workload-identity` configuration block, ensuring users get validation errors for missing required fields.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
