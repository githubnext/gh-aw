# Recorded AWF `/reflect` fixtures

These files are unchanged byte-for-byte copies of `sandbox/firewall/awf-reflect.json`
from public `github/gh-aw` workflow-run artifacts. They are excluded from
prettier so that their SHA-256 digests stay verifiable. Tests derive labelled
pending-fetch and malformed variants in memory; they never edit these files.

| File | Source run | Artifact | SHA-256 | Shape |
| --- | --- | --- | --- | --- |
| `run-38995410124-awf-reflect.json` | [smoke-claude-on-copilot 38995410124](https://github.com/github/gh-aw/actions/runs/38995410124), commit `12ba3ffcbaf5760b6094f52d8f7d61938e4e951e`, 2026-10-17 | `agent` (12087091651) | `9d59511015584d359b4159737d1d44ed12ad9426a1690e62148ab53b7543648a` | Five endpoints; configured Copilot with 40 models and 40 `routing_models`; four unconfigured endpoints with null `models` and `routing_models` |
| `run-29812420778-legacy-awf-reflect.json` | [Smoke Copilot 29812420778](https://github.com/github/gh-aw/actions/runs/29812420778), commit `fed7b8398300613d6f79474d75d3ffcdc6665d18`, 2026-07-21 | `agent` (8488185681) | `da2570969e0d62bd10195f029b5472f750404b515784d8c68e9b267e0975b410` | Older AWF reflection without `routing_models`; configured Copilot with 36 models and four unconfigured endpoints with null `models` |

The legacy file is the oldest retained public reflection found. It is not the
source of a `pkg/cli/testdata/model_routing_golden/` case: those source runs are
either private or expired (for example, run 37886511291 returns HTTP 404). Every
AWF `/reflect` version since its introduction supplies endpoint `configured`, so
no recorded reflection without `configured` exists to pin.
