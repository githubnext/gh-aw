# Local Inference with Self-Hosted Runners

This guide explains how to run GitHub Agentic Workflows on self-hosted runners with local or privately-hosted model inference.

## Why This Approach Is Valuable

Running `gh-aw` agentic workflows on self-hosted infrastructure gives you:

- **Cost control** — local models (e.g. Qwen 2.5 via Ollama) eliminate per-token API costs. Models small enough to run on a MacBook Air can still complete useful tasks.
- **Data privacy** — prompts and context never leave your network when using a local inference endpoint.
- **Custom execution environments** — install internal tooling, mount private caches, attach GPUs, or add private network routes that are unavailable on hosted runners.
- **Model flexibility** — point the workflow at any OpenAI-compatible gateway: local Ollama, OpenRouter, an internal model server, or a future inference host.

## Runner Requirements

`gh-aw` agent jobs require a **Linux** runner with:

- Docker
- Passwordless sudo for the runner service account
- `iptables` support
- Outbound HTTPS to GitHub, GHCR, and the selected engine endpoint
- Access to any domains listed in the workflow network allowlist

> **macOS note**: A macOS host cannot satisfy the Linux runner requirements directly. Use a local Linux VM (e.g. Lima) to register as the runner while the Mac hosts the Ollama model endpoint.

## Example Scenarios

### Scenario A: Local Mac host + Lima VM + Ollama (fully local)

The Mac runs Ollama and Qwen locally. A Lima Linux VM on the same Mac registers as the GitHub Actions self-hosted runner. The agent container calls the Mac-hosted model endpoint via `host.docker.internal`.

Quick start (from repository root on the Mac):

```bash
scripts/run-local-macrunner-qwenollama.sh "Check the local agent lane."
```

The launcher installs dependencies, registers the runner, dispatches the workflow, and watches the run.

### Scenario B: Remote VM runner + OpenRouter inference

Run the agent job on a Linux VM (e.g. Azure) while routing model calls through [OpenRouter](https://openrouter.ai/). This separates runner infrastructure from model provider and gives access to a wider model catalog.

```bash
gh aw secrets set OPENROUTER_API_KEY --value "$OPENROUTER_API_KEY"
infra/azure-vm/create-runner-vm.sh
gh aw run azure-vm-openrouter
```

## Adapting the Patterns

The example scenarios in `githubnext/self-hosted-aw` are starting points. Common adaptations:

- Change runner labels to target a different host class.
- Change `OPENAI_BASE_URL` and the engine config to point at another OpenAI-compatible gateway.
- Change the model IDs (e.g. swap Qwen 2.5 0.5B for Qwen 3 27B) and update smoke tests accordingly.
- Add GPU resources, private network routes, or internal tool installations to the bootstrap scripts.

## Further Reading

Full setup guides, architecture diagrams, and runnable workflow sources are in the reference repository:

**[githubnext/self-hosted-aw](https://github.com/githubnext/self-hosted-aw)** — demonstrates `gh-aw` with self-hosted runners, model-routing platforms, and local inference across multiple example scenarios.
