---
title: Model Routing
description: Configure experimental per-run Copilot model and reasoning-effort selection for Copilot, Claude, Codex, and pi engines.
sidebar:
  order: 605
---

> [!WARNING]
> Model routing is experimental. Review the [known limitations](#known-limitations) before enabling it.

## How routing works

`engine.model-routing` chooses a Copilot model and reasoning effort once per run, before the agent starts, instead of using a fixed model. It supports the Copilot, Claude, Codex, and pi engines, each using the Copilot API. When routing is enabled, `engine.model` and any configured effort setting are ignored.

AWF's API proxy makes a small LLM call to classify the workflow's task text from `user.txt`, not gh-aw's system instructions. This text includes anything interpolated into the workflow prompt, such as issue bodies. The classifier treats it as untrusted data, not instructions to follow.

The public [githubnext/gh-aw-router](https://github.com/githubnext/gh-aw-router) then ranks the allowed model-and-effort choices using precompiled routing tables fitted on benchmark results and model capability data. Models not covered by the tables are never selected, even if allowed. The [router README](https://github.com/githubnext/gh-aw-router/blob/main/README.md) describes the classification labels, profiles, `/classify` and `/route` APIs, and table format. The router does not itself call an LLM; AWF executes the classifier call.

The selection is advisory. Requests for other policy-allowed models are admitted and logged as deviations, rather than rejected merely for differing from the selection.

`allowed-models` is the router's candidate list for the main agent. The compiler applies the workflow's model policy to it and writes the result to AWF `routing.candidateModels`. Declared sub-agent models are allowed automatically ([gh-aw#66234](https://github.com/github/gh-aw/pull/66234)) and do not need to be listed; listing one also makes it a candidate for the main agent. AWF admits requests for the candidates plus the declared sub-agent models.

`allowed-models` is not the workflow's request policy. That policy is set by `models.allowed` and `models.blocked` (see [Model Policy](/gh-aw/reference/glossary/#model-policy-modelsallowed-modelsblocked)), and both the candidates and the sub-agent models must satisfy it.

## Configuration

All three fields are required:

| Field | Values | Meaning |
|---|---|---|
| `goal` | `cost`, `cost-speed` | `cost` prefers the cheapest model-and-effort choice that meets the quality bar. `cost-speed` also weighs execution time. |
| `mode` | `economy`, `balanced`, `robust`, `auto` | `economy`, `balanced`, and `robust` set increasing quality bars, generally with increasing cost. `auto` lets the classifier recommend one of these three profiles for the task. |
| `allowed-models` | Non-empty list of Copilot model IDs | The router's candidates for the main agent. Declared sub-agent models are allowed automatically and need not be listed. Claude accepts Claude models only; Codex accepts GPT models; pi accepts any Copilot model. Each runtime verifies that the selected model advertises support for its API. The organization must have access to the models, and the routing tables must cover them for selection. |

```aw wrap
engine:
  id: copilot
  model-routing:
    goal: cost
    mode: auto
    allowed-models:
      - gpt-5.4-mini
      - gpt-5.4
      - gpt-5.6-luna
```

Candidate lists from imported workflows using the same engine are merged. Routing adds one classifier call per run, typically costing a fraction of a credit.

## Requirements

Routing requires the AWF firewall enabled and GitHub Copilot inference. It supports `copilot`, `claude`, `codex`, and `pi`; the compiler rejects other engines, non-Copilot providers, and incompatible literal model candidates. Claude Code uses the native Messages API and Codex uses the Responses API, so their candidate lists are restricted by model family. Expressions in `allowed-models` cannot be checked by the compiler. For non-Copilot engines, AWF's selected endpoint is advisory: Claude and Codex verify through `/reflect` `routing_models.supported_endpoints` that the selected model supports the engine's API, then use that API. Pi resolves its API from the Pi/AWF model catalog and verifies the corresponding endpoint against the same reflected metadata. These checks fail closed when candidate metadata is incomplete or the model does not advertise a compatible endpoint.

The provider must resolve to GitHub Copilot inference. Copilot and pi default to `github`. Claude defaults to Anthropic and Codex to OpenAI, so routed Claude and Codex workflows must set `engine.model-provider: github`. Any other provider, whether a default or set explicitly, fails compilation.

```aw wrap
engine:
  id: codex
  model-provider: github
  model-routing:
    goal: cost
    mode: economy
    allowed-models: [gpt-5.6-luna]
```

| AWF version | Capability |
|---|---|
| v0.28.29 or later | Task-level routing |
| v0.28.33 or later | Separate router candidates from admitted sub-agent models ([gh-aw#66234](https://github.com/github/gh-aw/pull/66234)) |
| v0.28.35 or later | Model-routing audit log |

Since [gh-aw#66291](https://github.com/github/gh-aw/pull/66291), routed workflows use gh-aw's default AWF version (currently **v0.28.56**) and router **0.1.3**, with version-matched, digest-pinned images. Most authors need no version or image overrides. An override of `sandbox.agent.version` must name a published AWF release, not just a Git tag.

Organizations requiring approved registries can override the router image through `sandbox.agent.images.router`. Use a compatible, digest-pinned image; see [Sandbox image overrides](/gh-aw/reference/sandbox/).

`allowed-models` must satisfy the workflow's `models.allowed` and `models.blocked` policy. GitHub Actions expressions in `allowed-models` are rejected when either policy is set, because the compiler cannot validate their runtime values. Models must also be available to the organization's Copilot plan; unavailable models are skipped.

For AWF-side routing configuration, including `routing.candidateModels`, see the [AWF configuration specification](https://github.com/github/gh-aw-firewall/blob/main/docs/awf-config-spec.md#13a-task-level-model-routing). The [API proxy sidecar reference](https://github.com/github/gh-aw-firewall/blob/main/docs/api-proxy-sidecar.md) describes discovery, selection, request policy, and logging.

## Sub-agents

Since [gh-aw#66234](https://github.com/github/gh-aw/pull/66234), inline and imported sub-agents declaring a `model:` are admitted by the request policy, even when that model is not in `allowed-models`. This does not expand router candidates: routing still selects only from `allowed-models`. Workflow model policies still apply.

Whether a sub-agent may use a different model family (Claude versus GPT) from the main agent depends on the engine. For the Copilot engine, it depends on the execution mode; see [CLI mode vs SDK mode](#cli-mode-vs-sdk-mode). The pi engine already runs sub-agents from both families in one run. Claude and Codex require sub-agent models compatible with their engine's API.

## CLI mode vs SDK mode

Model routing works with the Copilot engine in both Copilot CLI mode (the default) and Copilot SDK mode (`engine.copilot-sdk: true`). Only SDK mode supports declared sub-agents from a different model family in the same run.

- **CLI mode**: one wire API serves the whole session. If the main agent is a GPT model and a sub-agent is a Claude model (or the reverse), the sub-agent's requests fail; the main model's subsequent response is nondeterministic and may report the missing result or attempt another approach ([copilot-cli#5103](https://github.com/github/copilot-cli/issues/5103)). Since [gh-aw#67487](https://github.com/github/gh-aw/pull/67487), the run logs a startup warning when a declared custom agent's model does not support the main model's session endpoint.
- **SDK mode**: since [gh-aw#67509](https://github.com/github/gh-aw/pull/67509), the runner builds one provider per wire API (Responses and Chat Completions), so each model uses its own endpoint.

For mixed-family candidate pools, this means:

- In SDK mode, any routing pick works with any declared sub-agent.
- In CLI mode, a mixed pool is fine without sub-agents. With sub-agents, every candidate must be in the sub-agents' family, because the router may pick any of them as the main model.

CLI mode, with candidates and the sub-agent in the same family:

```aw wrap
---
on: workflow_dispatch
permissions:
  contents: read
  copilot-requests: write
engine:
  id: copilot
  model-routing:
    goal: cost
    mode: auto
    allowed-models: [gpt-5.4-mini, gpt-5.6-luna, gpt-5.6-sol]
---

Summarize the repository README and use the `summarizer` agent for the first draft.

## agent: `summarizer`
---
description: Drafts a short summary
model: gpt-5.4-mini
---
Draft a three-sentence summary of the text you are given.
```

Without the sub-agent, the same CLI-mode workflow can route over a mixed pool.

SDK mode, with a mixed pool and a Claude sub-agent:

```aw wrap
---
on: workflow_dispatch
permissions:
  contents: read
  copilot-requests: write
engine:
  id: copilot
  copilot-sdk: true
  model-routing:
    goal: cost
    mode: auto
    allowed-models: [gpt-5.4-mini, gpt-5.6-luna, gpt-5.6-sol, claude-haiku-4.5, claude-sonnet-5, claude-opus-5]
---

Review the open issues and use the `triager` agent to suggest a label for each one.

## agent: `triager`
---
description: Suggests one label for an issue
model: claude-haiku-4.5
---
Reply with the single best label for the issue you are given.
```

All six candidates are covered by the routing tables of router 0.1.3.

### Tested reference workflows

These smoke workflows run every 2 days, with `mode: economy` unless noted. They check main-agent request models against the candidate models and use the unified session and AWF's api-proxy `token-usage.jsonl` as model/endpoint evidence, including a successful request on a supported or declared endpoint:

- [`smoke-copilot-routed.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-copilot-routed.md): Copilot CLI mode, routing over three models (`gpt-5.4-mini`, `gpt-5.6-luna`, `claude-haiku-4.5`) with no sub-agents.
- [`smoke-claude-routed.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-claude-routed.md): Claude Code with `model-provider: github`, routing over `claude-sonnet-5` with no sub-agents. AWF selects `/chat/completions`; the main agent must use `/v1/messages`.
- [`smoke-copilot-sdk-routed.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-copilot-sdk-routed.md): Copilot SDK mode, routing over `gpt-5.6-luna` with a `claude-haiku-4.5` sub-agent.
- [`smoke-pi-routed.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-pi-routed.md): pi, routing over `gpt-5.6-luna` with `claude-haiku-4.5` (`/v1/messages`) and `gpt-5.4-mini` (`/responses`) sub-agents.
- [`smoke-copilot-routed-claude.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-copilot-routed-claude.md): Copilot CLI mode with `mode: auto`, routing over `claude-sonnet-5` with no sub-agents. The main agent must use `/chat/completions`.
- [`smoke-pi-routed-claude.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-pi-routed-claude.md): pi with `mode: robust`, routing over `claude-sonnet-5` with no sub-agents. The main agent must use `/v1/messages`.
- [`smoke-copilot-sdk-routed-claude.md`](https://github.com/github/gh-aw/blob/main/.github/workflows/smoke-copilot-sdk-routed-claude.md): Copilot SDK mode, routing over `claude-haiku-4.5` (`/chat/completions`, no reasoning effort) with a `gpt-5.4-mini` sub-agent (`/responses`).

The three Claude smokes also check the selection record: the requested mode matches the workflow's `mode`, the recommended profile is `economy`, `balanced` or `robust`, and the selected effort is one the selected model supports (or none when it supports no efforts). The recommended profile and effort vary between runs, so they are only logged.

The SDK and pi smokes route over a single model only to keep their checks deterministic: they attribute traffic to sub-agents by model, so sub-agent models must not appear in `allowed-models`. This is not a recommendation for real workflows.

## Known limitations

### Selection can vary between runs

The classifier's recommendation is not deterministic, especially with `mode: auto`. Runs of the same task can select different models or efforts. An explicit profile fixes the quality bar, but does not make classification or selection deterministic.

### Cross-family sub-agents in Copilot CLI mode

With the Copilot engine in CLI mode, sub-agents from a different model family than the main model fail. Use [SDK mode](#cli-mode-vs-sdk-mode) or keep every candidate in the sub-agents' family.

## Observability

With AWF v0.28.35 or later, the `agent` artifact contains:

```text
sandbox/firewall/logs/api-proxy-logs/model-routing.jsonl
```

It records classification labels and mode, ranked choices, the selection, router version, and one terminal record per inference request. Request records report `as_selected` or `deviated` and the outcome; requests whose model or effort cannot be observed can be marked `unobserved`.

The adjacent `token-usage.jsonl` contains credits per request. Classifier usage is marked with `purpose: "routing_classification"` and joined to routing requests by `request_id`.

Threat-detection jobs are not routed; they use their configured detection model ([gh-aw#66670](https://github.com/github/gh-aw/pull/66670)).

### Steering notices

With AWF v0.28.51 or later, the run's AI-credit budget and timeout produce advisory steering notices, which `gh aw audit` lists as `steering_notices` ([gh-aw#67597](https://github.com/github/gh-aw/pull/67597)). This matters for routing because runs routed to expensive models can approach the default 1,000-credit budget. Since gh-aw's default AWF version is v0.28.56, routed workflows get these notices by default; workflows that pin an older `sandbox.agent.version` don't. See [Token steering](/gh-aw/reference/sandbox/#token-steering-sandboxagenttoken-steering).

Claude, Codex, and pi log the selected model, effort, and effective endpoint before starting inference. When a runtime uses its own API instead of AWF's selected endpoint, `selected_endpoint` records AWF's original choice. Copilot CLI retains its existing model-and-effort log:

| Engine | Example log line |
|---|---|
| Copilot | `inference routing: mode=awf-routed model=gpt-5.6-sol effort=high` |
| Claude | `inference routing: mode=awf-routed model=claude-opus-5 effort=xhigh endpoint=/v1/messages selected_endpoint=/chat/completions` |
| Codex | `inference routing: mode=awf-routed model=gpt-5.6-sol effort=high endpoint=/responses` |
| pi | `inference routing: mode=awf-routed model=claude-haiku-4.5 effort=none endpoint=/v1/messages selected_endpoint=/chat/completions` |

Effort support and runtime mapping are engine-specific. AWF currently ranks its model-and-effort choices without filtering efforts by engine, so a routed choice the runtime cannot represent still fails closed rather than being clamped or replaced. Engine-specific pre-ranking effort filtering requires an AWF configuration capability; the runtime check remains the backstop:

| Engine | Accepted routed efforts | Runtime setting |
|---|---|---|
| Copilot | `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` | Passed through unchanged |
| Claude | `low`, `medium`, `high`, `xhigh`, `max` | Passed through unchanged; `none` and `minimal` fail |
| Codex | `minimal`, `low`, `medium`, `high`, `xhigh` | Passed through unchanged; `none` and `max` fail |
| pi | `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` | `none` maps to `off`; all others pass through unchanged |

`gh aw audit <run-id>` reports whether routing selected a route or failed, the objective and classification, selected model/effort/endpoint, top ranked choices, router version and latency, request outcomes and deviations, and a cost split for classifier, selected-model, and deviated traffic. The comparison notes model, effort, mode, or router-version changes even when total cost is similar. With `gh aw logs --format pretty|markdown` (or `--json`), the Model Routing section aggregates label-to-selection routes across runs, including run counts, total and average AIC, classifier cost, and deviation share. For example:

```text
Model Routing
  classifier_aic=0.059 deviated_requests=0/4 (0.0%)
  explain/local/trivial mode=economy selected=gpt-5.6-luna:high router=0.1.3 runs=1 aic_total=0.760 aic_avg=0.760
```

The per-run audit JSON includes `model_routing` with the same decision, request, failure, and cost details; logs JSON includes the cross-run `model_routing.routes` aggregate. See the [AWF routing audit-log schema](https://github.com/github/gh-aw-firewall/blob/main/docs/api-proxy-sidecar.md#model-routing-audit-log) for the complete record definition.

## Troubleshooting

| Signal | Meaning and checks |
|---|---|
| Exit code `78` | Routing failed, rather than silently falling back to a fixed model. The audit routing section shows failure `code` and `detail`; check for `no_route`, unavailable classifier/router, or configuration errors. |
| `degraded_reason` | The audit routing section reports why classification degraded and whether the router continued with fallback classification or failed. |
| `routed: "deviated"` | Audit shows request counts and the requested/selected models and efforts for deviations. An allowed sub-agent using its declared model can be a normal deviation; deviation is not itself a policy violation. For records from AWF before v0.28.39, an endpoint-only deviation is counted as selected-model traffic and the report notes this normalization. |
| `model ... advertises endpoints [...], none compatible with this engine API [...]` | The selected model does not advertise the Messages or Responses API required by Claude or Codex. Check the `/reflect` routing model metadata and remove the incompatible model from `allowed-models`. |
| `candidate metadata is incomplete` | AWF `/reflect` did not provide complete routing endpoint metadata. The runtime refuses to guess an API; upgrade AWF to a version that returns complete candidate metadata. |
| Upstream HTTP or stream error | Inspect `sandbox/firewall/logs/api-proxy-logs/upstream-errors.jsonl` in the `agent` artifact alongside the routing log. In particular, a sub-agent HTTP 400 can indicate the API-family mismatch rather than a model-policy rejection. |

Use the [API proxy sidecar reference](https://github.com/github/gh-aw-firewall/blob/main/docs/api-proxy-sidecar.md) and [AWF configuration specification](https://github.com/github/gh-aw-firewall/blob/main/docs/awf-config-spec.md) for the upstream contracts and failure details.
