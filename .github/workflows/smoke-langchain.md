---
private: true
emoji: "🦜"
description: Smoke test a Python LangChain program through a dynamically discovered AWF endpoint
on:
  slash_command:
    name: smoke-langchain
    strategy: centralized
    events: [issues, issue_comment, pull_request, pull_request_comment]
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["smoke"]
  status-comment: true
permissions:
  contents: read
  issues: read
  pull-requests: read
  copilot-requests: write
name: Smoke LangChain
model: copilot/gpt-5.4
engine: copilot
strict: true
runtimes:
  node:
    version: "20"
  python:
    version: "3.11"
network:
  allowed:
    - defaults
    - github
    - python
tools:
  bash:
    - "*"
safe-outputs:
  allowed-domains: [default-safe-outputs]
  add-comment:
    hide-older-comments: true
    max: 2
  messages:
    footer: "> 🦜 *[{workflow_name}]({run_url})*{ai_credits_suffix}{history_link}"
timeout-minutes: 10
features:
  gh-aw-detection: false
---

# Smoke Test: LangChain

Run this smoke test entirely inside the agent container. Keep all output concise.

1. Create `/tmp/gh-aw/agent/langchain-smoke` and a Python virtual environment there.
2. Install exactly `langchain==1.3.14` and `langchain-openai==1.4.1`.
3. Write `/tmp/gh-aw/agent/langchain-smoke/smoke.py` with exactly this program, then run it:
   ```python
   import json
   from urllib.request import urlopen
   from urllib.parse import urlparse, urlunparse

   from langchain_openai import ChatOpenAI

   def get_json(url):
       with urlopen(url, timeout=30) as response:
           return json.load(response)

   reflect = get_json("http://api-proxy:10000/reflect")
   endpoint = next(
       (
           item
           for item in reflect.get("endpoints", [])
           if item.get("configured") and item.get("provider") == "copilot"
       ),
       None,
   )
   if not endpoint or not endpoint.get("models_url"):
       raise RuntimeError("No configured Copilot endpoint was returned by /reflect")

   parsed = urlparse(endpoint["models_url"])
   if not parsed.path.endswith("/models"):
       raise RuntimeError("The reflected models_url does not end in /models")
   base_url = urlunparse(parsed._replace(path=parsed.path.removesuffix("/models"), params="", query="", fragment=""))

   models = endpoint.get("models") or [item.get("id") for item in get_json(endpoint["models_url"]).get("data", [])]
   model = next((item for item in models if item), None)
   if not model:
       raise RuntimeError("No model was returned by the reflected endpoint")

   response = ChatOpenAI(base_url=base_url, api_key="not-needed", model=model).invoke("Reply with the single word: smoke")
   if not str(response.content).strip():
       raise RuntimeError("LangChain returned an empty response")
   print(f"PASS provider={endpoint['provider']} model={model}")
   ```
   The program must derive `base_url` and model from `/reflect`; do not hard-code an api-proxy model endpoint, port, model, or API key.
4. Confirm the program completes successfully without printing the full response or any credentials.
5. Report PASS or FAIL, the reflected provider, and model. Do not print the full response or any credentials.

If triggered by a pull request, use `add_comment` to post the concise result. Otherwise, call `noop` with the result.
