---
title: AW Admission Service Specification
description: Downstream gh-cao implementation contract for centralized Agentic Workflows admission and cost controls
---

# AW Admission Service Specification

**Version**: 1.0
**Status**: Downstream implementation contract; service implementation pending in gh-cao
**Date**: 2026-10-19

The terms MUST, MUST NOT, SHOULD, and MAY indicate normative requirements.

## 1. Scope and Ownership

gh-aw implements an optional client-side gate in the activation job, before agent execution. gh-cao implements the HTTPS admission service, trusted usage ingestion, policy evaluation, and protected operator diagnostics. This document specifies the service behavior required by that client.

The gate centrally decides whether an AW execution may start. It is not a billing API, spend-inspection endpoint, inference proxy, or mid-execution cancellation mechanism. A service decision alone cannot impose a hard upper bound on the cost of an already-running execution.

The initial protocol uses REST GET and the activation job's GitHub Actions token only. It MUST NOT require another caller credential, OIDC token, PAT, Copilot token, GitHub App token, API key, or a custom activation token.

## 2. Client Configuration

```json
{
  "admission": {
    "url": "https://cao.example.com/api/admission",
    "on_outage": "ignore"
  }
}
```

The configuration resides in `.github/workflows/aw.json`. The URL is deployment-selected; `/api/admission` is illustrative, not a hardcoded route. It MUST be an absolute HTTPS URL without embedded credentials, query parameters, fragments, or GitHub Actions expressions. Compiled workflows embed the configuration and require recompilation after changes.

Omitting `admission` disables the client. `on_outage` accepts `ignore` or `block` and defaults to `ignore`. It controls exhausted transient failures only; an explicit denial always blocks. The policy is local client configuration and is not transmitted to the service.

## 3. Request Contract

The service MUST accept `GET` at its configured endpoint, with no request body:

```http
GET /api/admission?repo=acme%2Fproject&workflow_id=triage&workflow_call_id=1234-2%3Atriage&run_id=1234&run_attempt=2&ref=refs%2Fheads%2Fmain&event_type=workflow_dispatch HTTP/1.1
Authorization: Bearer <ephemeral GitHub Actions token>
Cache-Control: no-store
```

The token is `${{ github.token }}` from the activation job. It MUST appear only in the Authorization header, never in the URL, body, diagnostics, or response. No other caller secret is transmitted.

### Query Parameters

All seven parameters are required exactly once and are UTF-8 URL-encoded strings.

| Parameter | Meaning and validation |
|-----------|------------------------|
| `repo` | Current execution's `owner/repository` slug; resolve to GitHub's canonical repository ID before authorization |
| `workflow_id` | Compiled AW logical workflow ID, matching `aw_info.workflow_id`; not a numeric GitHub workflow database ID or caller workflow ref |
| `workflow_call_id` | Exactly `<run_id>-<run_attempt>:<workflow_id>`, matching `aw_info.workflow_call_id` |
| `run_id` | Positive decimal GitHub Actions run ID; preserve as a string, without floating-point conversion |
| `run_attempt` | Positive decimal attempt number; the current client uses `1` when the runner omits it |
| `ref` | Current GitHub Actions ref, such as `refs/heads/main` or a pull-request ref |
| `event_type` | Current GitHub Actions event name, including `workflow_dispatch` for dispatched workflows |

The client derives current execution fields from the runner's context and its compiled workflow ID, not from caller-supplied `aw_context`. It does not send the full event, actor identity, arbitrary `aw_context`, usage, balances, or policy values. Original triggering-event lineage is not part of this protocol.

The service MUST reject missing, duplicate, unknown, malformed, or oversized parameters rather than choosing an ambiguous value. It MUST impose documented request-size limits and MUST NOT interpret query values as executable expressions or outbound URLs. It SHOULD support at least an 8 KiB request target. Empty values, invalid repository slugs, control characters, non-positive run numbers, and inconsistent invocation IDs MUST receive the uniform denial response.

The implementation MUST preserve opaque identifiers and ref data rather than lowercasing or splitting them unnecessarily. Canonicalization of the repository for authorization MUST NOT change the invocation ID supplied by the caller.

## 4. Authentication and Trust Boundary

The service MUST authenticate each request before reading tenant policy or usage. It MUST verify that the supplied token authorizes access to the claimed repository using the deployment's trusted GitHub API, then establish the run's repository and attempt from authoritative GitHub data. gh-aw grants `actions: read` to the activation job when admission is configured, including when stale-file checks are disabled.

The GitHub API host MUST be a trusted deployment configuration, not a request parameter or a URL derived from untrusted metadata. Forwarding the Actions token to arbitrary hosts, following credential-bearing redirects, or using it for repository writes is prohibited. The service MUST discard the token after request processing; it MUST NOT persist it, include it in queue messages, or record even a token hash.

### Limits of Token-Only Authentication

An Actions token is opaque and repository-scoped, not a signed assertion binding this exact job, workflow, ref, or run. A token prefix or successful repository/run read does not prove that the requester is a particular Actions invocation, nor reliably distinguish every GitHub App installation token from a GitHub Actions token. The implementation MUST NOT claim stronger attestation than this protocol provides.

The service MUST treat all query parameters as claims. It MUST use verified GitHub repository/run properties where available and reject detectable inconsistencies. A reusable workflow's logical AW ID may not be independently provable from the top-level run alone. Unverified workflow/ref/event claims MUST NOT grant access to a less restrictive tenant or budget; repository/organization controls remain authoritative. Unknown or unverifiable policy selectors MUST map to a conservative repository policy or denial, not unlimited execution.

Strict cryptographic binding to an individual workflow would require a separately versioned protocol and is outside this token-only contract. The service SHOULD bound probing and replays using authenticated repository scope, authoritative run state, request rate controls, and a bounded decision-record lifetime.

Service-owned GitHub App credentials or backend credentials MAY be used for trusted policy and usage ingestion. They are separate from caller authentication and MUST NOT be requested from, or returned to, the workflow.

## 5. Response Contract

The preferred admitted response is `204 No Content`. The current client also accepts `200 OK`. The service SHOULD use `204` consistently and return no response body.

| Status | Meaning | Client behavior |
|--------|---------|-----------------|
| `204`, or legacy-compatible `200` | Admitted | Continue activation |
| `403` | Uniform authentication, authorization, validation, or policy denial | Block immediately; no retry |
| `408`, `429`, `500`-`599` | Transient service failure | Retry, then apply local outage policy |
| Any other status, including redirects and other `2xx` | Not admitted or unsupported protocol | Block immediately; no retry |

The service MUST use the same `403` response for invalid credentials, inaccessible or unknown repositories/runs, disabled or unconfigured tenants, unknown policies, exhausted budgets, and all other durable non-admission reasons. It MUST NOT encode spend or denial reasons in the status, body, or headers.

When trusted GitHub, usage, policy, or storage dependencies are unavailable, the service MUST return a generic transient status, normally `503`, rather than a fabricated allow or policy-denial result. Rate limiting MAY return `429` only for genuine operational abuse/load controls independent of spend. A budget threshold, concurrency policy, or reset window MUST NOT be represented as `429`, `5xx`, or `Retry-After`: doing so activates fail-open behavior by default.

Every response MUST set `Cache-Control: no-store`, have an empty body, and omit budget, quota, reset, policy, and internal-correlation headers. The service MUST NOT redirect. It MUST NOT return a challenge requesting another credential.

## 6. Retries, Outages, and Availability

The current client makes at most three attempts. Each attempt has a ten-second timeout. Network errors, timeouts, `408`, `429`, and `5xx` are transient. Between attempts it waits 250 ms, then 500 ms, each plus random jitter from 0 through 249 ms. The same query identity is used on every attempt. Response headers do not affect retry timing.

An explicit denial after a transient failure terminates retries and blocks. After all transient attempts fail:

| Local policy | Result |
|--------------|--------|
| `ignore`, including omitted configuration | Emit a generic warning and continue without admission |
| `block` | Fail activation and do not start the agent |

Invalid local configuration, missing Actions credentials, and missing execution identity block without contacting the service. Redirects are not followed or treated as outages.

The client discards response bodies without parsing them. Blocking failures expose only `ERR_ADMISSION` and the same generic message for every cause. Ignored outages emit a generic `ERR_ADMISSION` warning. Neither path prints statuses, headers, response bodies, tokens, or transport errors.

The service SHOULD complete within two seconds under its expected load and MUST bound dependency calls to finish below the client's ten-second deadline. It MUST remain safe when a caller times out or disconnects and retries reach another instance.

The default `ignore` policy explicitly trades enforcement for availability. Operators requiring cost controls during outages, rate limiting, or deliberate service disruption MUST configure `block`. No service-only implementation can override that client choice.

## 7. Cost-Control Decisions

gh-cao owns policy configuration and trusted usage aggregation. Policies MAY impose organization, repository, or verified-workflow controls. Exact spending limits and windows are operator configuration, not part of this public API.

An implementation MUST define its accounting unit, authoritative usage sources, observation freshness bounds, policy precedence, time-window boundaries, and handling of active executions. It MUST NOT mix AI Credit units with token counts or currency implicitly. Missing, stale, or unavailable usage MUST NOT silently become zero usage.

Disabled or unconfigured tenants and unsupported policy selectors are durable denials. Temporary inability to load an otherwise configured policy or sufficiently fresh accounting is an outage. The distinction MUST depend on operational state, not on whether a hidden spend threshold was crossed.

Repository and organization policies MUST apply even when workflow-level metadata cannot be verified. Applicable controls MUST be evaluated from a consistent policy/usage snapshot so concurrency does not cause contradictory decisions within one request.

### GET Safety and Accounting Limits

The GET request MUST NOT charge credits, decrement a budget, reserve funds, create a paid execution, or modify billing state. Request logs and short-lived decision caching are permitted implementation side effects, but repeated GET requests MUST NOT create repeated financial commitments.

Admission is a start-time check. Without independently maintained in-flight accounting or reservations, simultaneous executions can all observe the same remaining budget and overshoot it. The service MUST document its concurrency and overshoot model and MUST NOT advertise a hard spend cap based solely on this GET endpoint. A future reservation/settlement or stronger attestation protocol requires coordinated changes to this specification and the gh-aw client.

## 8. Idempotency and Consistency

The deduplication key MUST include the trusted GitHub deployment, canonical repository ID, `run_id`, `run_attempt`, and `workflow_id`. `workflow_call_id` MUST be checked for consistency but MUST NOT be the sole key across deployments. Client-supplied workflow identity retains the trust limitations in Section 4.

Retries and duplicate requests handled by multiple service instances MUST NOT multiply accounting, policy effects, or any backend observation of a started execution. A completed allow/deny decision SHOULD be memoized for the short client retry horizon, with bounded retention and an explicit policy-revocation invalidation strategy. Cached decisions MUST NOT admit a different invocation or a rerun; a new run attempt is a new key.

Transient failures MUST NOT be cached as policy denial or admission. After a timed-out response, another attempt MAY observe a newer policy/usage snapshot if no completed decision was retained. The implementation MUST test this ambiguity rather than assuming network delivery is exactly once.

## 9. Privacy and Abuse Resistance

The public result reveals only admitted versus not admitted. Full information-theoretic secrecy is impossible: repeated legitimate probes can reveal changes in eligibility. The service MUST authenticate and authorize before policy lookup, limit probes independently of spend, and document this residual side channel.

All durable denials MUST be externally indistinguishable across budget bands and policy reasons: the same status, empty body, header set, and response-size behavior. Denial timing MUST NOT intentionally depend on spend level. Implementations SHOULD use a common decision/response path and a bounded, spend-independent response envelope, and test distributions rather than claiming constant-time network behavior.

The service MUST NOT offer anonymous balance checks, threshold-specific status codes, amount-dependent retry timing, spend-based cache validators, or budget-reset headers. Reverse proxies, CDNs, API gateways, and tracing middleware MUST preserve the same rules. Shared intermediary caching is prohibited regardless of Authorization presence.

Authorization headers and tokens MUST be redacted before logging, tracing, exception capture, metrics, or request sampling. Request URLs contain repository/ref metadata; access to any retained metadata MUST be tenant-scoped and retention-bounded. Public responses and workflow-visible telemetry MUST NOT include usage or denial reasons.

Operators MAY retain protected decision reason codes and accounting diagnostics, without credentials, in a separate access-controlled system. These diagnostics MUST NOT be reflected into the workflow response, URL, artifact, or public metric labels. Cross-tenant reads are prohibited.

## 10. Acceptance Tests

The gh-cao implementation MUST provide automated tests for these cases before enabling the endpoint:

| ID | Case | Required result |
|----|------|-----------------|
| ADM-001 | Valid Actions token and permitted invocation | Empty `204`, `Cache-Control: no-store` |
| ADM-002 | Missing/invalid token, wrong repository, nonexistent run, or denied policy | Same empty `403` and denial headers |
| ADM-003 | At least three hidden spend bands and distinct durable denial reasons | No reason/amount differences in status, body, headers, size, or intentional timing |
| ADM-004 | Missing/duplicate/unknown parameter, invalid UTF-8/control data, or inconsistent invocation ID | Uniform `403`; no unsafe downstream calls |
| ADM-005 | URL-encoded slashes, Unicode identifiers, reserved characters, and large decimal run IDs | Parse once without truncation, injection, or precision loss |
| ADM-006 | Forged workflow/ref/event or cross-tenant metadata | No favorable bypass of authoritative repository/organization controls |
| ADM-007 | Authentication, usage, policy, or storage dependency outage | Empty generic transient response, not fabricated admission/denial |
| ADM-008 | Network glitch or `503`, followed by admission | Client retries with unchanged identity and continues on success |
| ADM-009 | Exhausted transient responses | Default/`ignore` continues with generic warning; `block` fails |
| ADM-010 | Explicit `403`, including after an outage | Client blocks without further retries under either policy |
| ADM-011 | Redirect to another origin | Client blocks; no credential sent to redirect destination |
| ADM-012 | Response contains sensitive diagnostics due to a faulty upstream | Client never reads/logs them; service sanitizes before production |
| ADM-013 | Concurrent duplicate GETs and lost responses across instances | No duplicate financial or execution-accounting effects |
| ADM-014 | Same workflow rerun and distinct reusable workflow IDs | Separate deduplication keys, within documented identity limits |
| ADM-015 | Stale accounting and concurrent new executions | Documented conservative/outage behavior; no implicit zero or hard-cap claim |
| ADM-016 | Public errors, proxy logs, tracing, exceptions, and metrics | No token/authorization leakage or public spend/reason leakage |
| ADM-017 | Spend-triggered denial versus genuine operational rate limiting | Budget denial is `403`; `429` does not encode budget state |
| ADM-018 | Default fail-open under sustained outage | Expected bypass is explicit; strict-policy deployment verifies `block` |

Tests MUST use synthetic tokens and accounting fixtures. They MUST NOT log production credentials or require real spend. Client compatibility tests MUST use the same status classification, parameter names, invocation-ID algorithm, timeout, retry count, and outage defaults as the gh-aw implementation.

## 11. Implementation References and Evolution

The current client contract is implemented by:

- [`pkg/parser/schemas/repo_config_schema.json`](../pkg/parser/schemas/repo_config_schema.json)
- [`pkg/workflow/compiler_admission.go`](../pkg/workflow/compiler_admission.go)
- [`actions/setup/js/check_admission.cjs`](../actions/setup/js/check_admission.cjs)
- [`actions/setup/js/check_admission.test.cjs`](../actions/setup/js/check_admission.test.cjs)
- [`pkg/workflow/compiler_admission_test.go`](../pkg/workflow/compiler_admission_test.go)

Operator-facing configuration is documented in [Enterprise Configuration](../docs/src/content/docs/reference/enterprise-configuration.md#central-admission-control).

There is no version header or negotiated capability in this initial request. Incompatible additions to required fields, authentication, admitted statuses, or outage semantics MUST use a separately configured/versioned endpoint and coordinated client changes; they MUST NOT silently change the existing endpoint.
