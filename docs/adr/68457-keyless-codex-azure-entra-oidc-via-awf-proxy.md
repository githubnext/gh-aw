# ADR-68457: Keyless Codex Azure Entra OIDC via the AWF Proxy

**Date**: 2026-02-10
**Status**: Draft
**Deciders**: Unknown [TODO: verify — PR author and maintainers]

---

### Context

Native Codex workflows always required `CODEX_API_KEY` or `OPENAI_API_KEY`, even when the workflow declared `engine.auth: {type: github-oidc, provider: azure}`. The AWF API proxy (agent firewall sidecar) already supports exchanging the GitHub Actions OIDC assertion for an Entra access token and refreshing it, but three independent layers still demanded a static key: the compiler's secret-validation step, the generated Codex `model_providers.openai-proxy` configuration (`env_key = "CODEX_API_KEY"`), and the `codex_harness.cjs` pre-flight key check. Organizations that forbid long-lived API-key secrets therefore could not run Codex against an Entra-protected Azure OpenAI or internal Responses gateway. Removing the key requirement must not weaken authentication: the Entra token and the Actions OIDC request credentials must never enter the agent sandbox, and keyless startup must be refused when the expected proxy is not actually in place. Tracked in issue #68026.

### Decision

We will allow Codex to run without any OpenAI API key *only* when the compiler can prove the workflow uses Azure GitHub OIDC behind the AWF firewall (`isCodexAzureOIDC`: `auth.type == github-oidc`, `auth.provider == azure`, resolved provider OpenAI, firewall enabled). In that case the compiler skips the secret-validation step and the `CODEX_API_KEY`/`OPENAI_API_KEY` env wiring, omits `env_key` from the managed proxy block (keeping `wire_api = "responses"` and `requires_openai_auth = false`), and emits a non-secret launch policy `GH_AW_CODEX_OIDC_AUTH=azure`. At runtime the harness defence-in-depth check requires a *fresh* `/reflect` response advertising a configured `openai` endpoint with `auth_type: github-oidc/azure` plus a successfully configured Codex proxy TOML; otherwise it refuses to spawn Codex. Token exchange and refresh remain entirely inside AWF, never in the Codex process.

### Alternatives Considered

#### Alternative 1: Inject a dummy API key (as done for Copilot BYOK)

The codebase already uses `CopilotBYOKDummyAPIKey` to satisfy clients that insist on a key. Reusing that pattern would have been a one-line change. It was rejected because a placeholder key makes the authentication mode indistinguishable from real key auth in generated lock files and logs, and it offers no point at which the runtime can refuse to start when the Entra proxy is absent — a misconfigured workflow would fail opaquely at the first request instead of failing fast with a clear message.

#### Alternative 2: Perform the OIDC → Entra token exchange inside the Codex harness

The harness could request the Actions ID token and exchange it for an Entra access token itself, then pass it to Codex as a bearer key. This keeps the change in one file, but it requires `ACTIONS_ID_TOKEN_REQUEST_TOKEN`/`ACTIONS_ID_TOKEN_REQUEST_URL` inside the agent sandbox (the PR explicitly excludes them via `--exclude-env`), places a refreshable credential in the agent's own environment, and duplicates exchange/refresh logic that AWF already owns. Rejected on security isolation grounds.

#### Alternative 3: Trust the persisted `/reflect` output file

A simpler runtime guard would read the existing reflection JSON from disk. Rejected because a previous run's file could assert `github-oidc/azure` for an endpoint that is no longer configured; the implementation deliberately requires a fresh fetch in the current run (`reflectOk: false` is covered by a test that asserts refusal).

### Consequences

#### Positive
- Codex workflows can run against Azure OpenAI or an internal Entra gateway with no long-lived API-key secret, using short-lived federated credentials instead.
- The Entra token and Actions OIDC request credentials stay outside the agent sandbox; compromise of the agent process does not yield a reusable credential.
- Failure modes are explicit and early: the harness exits with an actionable message when reflection or the proxy configuration does not match the expected keyless topology.
- Behaviour is additive — API-key auth, MCP credentials, Copilot BYOK, and non-OpenAI fallback providers are untouched.

#### Negative
- The keyless path is only valid with the AWF agent sandbox (firewall) enabled; `sandbox.agent: false` is unsupported, narrowing where the feature applies.
- The authentication decision is now expressed in four places (compiler secret list, proxy TOML generation, execution env, harness guard) that must stay consistent; a future change to one risks silently re-introducing a key requirement or weakening the guard.
- Operators must configure an Entra federated credential whose issuer, subject, and audience match the workflow, which is more setup than storing a secret and is harder to debug from CI logs alone.
- `isCodexAzureOIDC` calls `NewCodexEngine()` from a package-level helper, coupling a predicate to engine construction.

#### Neutral
- `GH_AW_CODEX_OIDC_AUTH=azure` is a non-secret launch policy intentionally visible in the generated lock file, and is inherited by threat detection and same-provider OpenAI fallback models.
- `configureCodexProviderFromReflect` now accepts pre-fetched `reflectData`, so the fresh reflection used by the guard is reused rather than re-read from disk.
- Documentation (`reference/auth.mdx`, `reference/engines.md`) now presents keyless Entra OIDC as a first-class Codex authentication option alongside API keys.
- The unrelated `go.sum` change (dropping a stale `charmbracelet/x/ansi v0.11.8` entry) carries no architectural weight.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
