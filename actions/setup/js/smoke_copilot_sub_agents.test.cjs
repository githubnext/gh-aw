import fs from "node:fs";
import { createRequire } from "node:module";
import { describe, test, expect } from "vitest";

const require = createRequire(import.meta.url);
const workflow = fs.readFileSync(new URL("../../../.github/workflows/smoke-copilot-sub-agents.md", import.meta.url), "utf8");
const script = workflow
  .split("      script: |\n")[1]
  .split("\n---")[0]
  .replace(/^        /gm, "");
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;

function evidence() {
  const agents = [
    ["haiku-whoami", "copilot-completions/claude-haiku-4.5", "/chat/completions", "claude-haiku-4.5", "exact"],
    ["gpt5-whoami", "copilot-responses/gpt-5.6-luna", "/responses", "gpt-5", "resolved"],
    ["terra-whoami", "copilot-responses/gpt-5.6-terra", "/responses", "gpt-5.6-terra", "exact"],
  ];
  return {
    events: [
      ...agents.flatMap(([agentName, model]) => [
        { type: "subagent.started", data: { agentName, model, invocationId: agentName } },
        { type: "subagent.completed", data: { model, invocationId: agentName } },
      ]),
      ...agents.map(([role, model, , configured_model, status]) => ({
        type: "model.resolution",
        data: { engine: "copilot", role, configured_model, resolved_model: model.split("/")[1], provider: "copilot", status, cause: null },
      })),
    ],
    requests: agents.map(([, model, endpoint]) => ({ model: model.split("/")[1], path: endpoint, status: 200 })),
  };
}

async function assertEvidence({ events, requests }, execution = "success") {
  const mockRequire = name => {
    if (name.endsWith("unified_session.cjs")) return { writeUnifiedSession() {} };
    if (name === "node:fs") {
      return {
        existsSync: () => true,
        readFileSync: file => (file.endsWith("aw_session.jsonl") ? events : requests).map(record => JSON.stringify(record)).join("\n"),
      };
    }
    return require(name);
  };
  await new AsyncFunction("require", "process", script)(mockRequire, {
    env: { RUNNER_TEMP: "/tmp", SMOKE_EXECUTION: execution },
  });
}

describe("Copilot sub-agent smoke evidence", () => {
  test("accepts completed delegation with qualified models and matching proxy requests", async () => {
    await expect(assertEvidence(evidence())).resolves.toBeUndefined();
  });

  test("accepts the served dated Claude model identity", async () => {
    const data = evidence();
    data.events[1].data.model = "copilot-completions/claude-haiku-4-5-20251001";
    await expect(assertEvidence(data)).resolves.toBeUndefined();
  });

  test("rejects success without delegation", async () => {
    const data = evidence();
    await expect(assertEvidence({ events: data.events.filter(event => event.type === "model.resolution"), requests: data.requests })).rejects.toThrow("must actually run");
  });

  test("rejects a sub-agent without a model.resolution record", async () => {
    const data = evidence();
    data.events = data.events.filter(event => !(event.type === "model.resolution" && event.data.role === "gpt5-whoami"));
    await expect(assertEvidence(data)).rejects.toThrow("gpt5-whoami must have exactly one model.resolution record");
  });

  test("rejects a failed sub-agent model resolution", async () => {
    const data = evidence();
    Object.assign(data.events.find(event => event.type === "model.resolution" && event.data.role === "terra-whoami").data, { resolved_model: null, status: "failed", cause: "model_unavailable" });
    await expect(assertEvidence(data)).rejects.toThrow("terra-whoami model.resolution status is failed, expected exact");
  });

  test("expects the alias sub-agent to serve the model the harness resolved", async () => {
    const data = evidence();
    data.events.find(event => event.type === "model.resolution" && event.data.role === "gpt5-whoami").data.resolved_model = "gpt-6-luna";
    await expect(assertEvidence(data)).rejects.toThrow("gpt5-whoami changed or fell back from gpt-6-luna");
  });

  test("rejects a missing proxy request for only one GPT sub-agent", async () => {
    for (const model of ["gpt-5.6-luna", "gpt-5.6-terra"]) {
      const data = evidence();
      data.requests = data.requests.filter(request => request.model !== model);
      await expect(assertEvidence(data)).rejects.toThrow(`has no successful ${model} request on /responses`);
    }
  });

  test("rejects GPT sub-agents that resolve to the same served model", async () => {
    const data = evidence();
    data.events.find(event => event.type === "model.resolution" && event.data.role === "gpt5-whoami").data.resolved_model = "gpt-5.6-terra";
    for (const event of data.events.filter(event => event.data.invocationId === "gpt5-whoami")) event.data.model = "copilot-responses/gpt-5.6-terra";
    data.requests = data.requests.filter(request => request.model !== "gpt-5.6-luna");
    await expect(assertEvidence(data)).rejects.toThrow("terra-whoami resolved to gpt-5.6-terra, which another sub-agent also uses");
  });

  test("rejects a sub-agent that falls back to the main model", async () => {
    const data = evidence();
    data.events[1].data.model = "copilot-responses/gpt-5.3-codex";
    await expect(assertEvidence(data)).rejects.toThrow("changed or fell back");
  });

  test("rejects selection of a different model", async () => {
    const data = evidence();
    data.events.push({ type: "subagent.selected", data: { invocationId: "haiku-whoami", selectedModel: "copilot-responses/gpt-5.6-luna" } });
    await expect(assertEvidence(data)).rejects.toThrow("changed or fell back");
  });

  test("rejects requests using the wrong wire API", async () => {
    const data = evidence();
    data.requests[0].path = "/responses";
    await expect(assertEvidence(data)).rejects.toThrow("no successful");
  });

  test("rejects failed invocations even with successful model requests", async () => {
    const data = evidence();
    data.events.push({ type: "subagent.failed", data: { invocationId: "haiku-whoami" } });
    await expect(assertEvidence(data)).rejects.toThrow("failed");
  });

  test("reports a dead child pin's model.resolution cause before the failed execution", async () => {
    const data = evidence();
    data.events = data.events.filter(event => event.type === "model.resolution");
    Object.assign(data.events.find(event => event.data.role === "gpt5-whoami").data, { resolved_model: null, status: "failed", cause: "model_unavailable" });
    await expect(assertEvidence({ events: data.events, requests: [] }, "failure")).rejects.toThrow("gpt5-whoami model.resolution status is failed, expected resolved (cause: model_unavailable)");
  });
});
