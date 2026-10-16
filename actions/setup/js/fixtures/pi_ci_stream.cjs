// @ts-check

// Shape and order from Daily Repository Chronicle run 36884805242, agent artifact
// 11174126484/pi-streaming.jsonl. All text, IDs, timestamps and arguments are synthetic.
const user = { role: "user", content: [{ type: "text", text: "SANITIZED PRIVATE PROMPT" }], timestamp: 1 };
const call = { type: "toolCall", id: "sanitized-call", name: "bash", arguments: { command: "printf sanitized" } };
const first = {
  role: "assistant",
  content: [call],
  api: "openai-responses",
  provider: "openai",
  model: "gpt-5.4",
  timestamp: 2,
  responseId: "sanitized-response-1",
  stopReason: "toolUse",
  usage: { input: 6407, output: 140, cacheRead: 1536, cacheWrite: 0, totalTokens: 8083, cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0 } },
};
const result = { content: [{ type: "text", text: "sanitized output" }], structuredContent: { success: true, data: { count: 0 } } };
const toolResult = { role: "toolResult", toolCallId: call.id, toolName: call.name, content: result.content, isError: false, timestamp: 3 };
const final = {
  role: "assistant",
  content: [{ type: "text", text: "Sanitized complete.\n" }],
  model: "gpt-5.4",
  timestamp: 4,
  responseId: "sanitized-response-2",
  stopReason: "stop",
  usage: { input: 20, output: 5, cacheRead: 4, cacheWrite: 1, totalTokens: 30, cost: { total: 0 } },
};

const success = [
  { type: "session", version: 3, id: "sanitized-session", timestamp: "2026-01-01T00:00:00Z", cwd: "/sanitized/repo" },
  { type: "agent_start" },
  { type: "turn_start" },
  { type: "message_start", message: { role: "system", content: "SANITIZED SYSTEM PROMPT", sections: [], timestamp: 0, toolsAdded: [] } },
  { type: "message_end", message: { role: "system", content: "SANITIZED SYSTEM PROMPT", sections: [], timestamp: 0, toolsAdded: [] } },
  { type: "message_start", message: user },
  { type: "message_end", message: user },
  { type: "message_start", message: { ...first, content: [], usage: { input: 0, output: 0 } } },
  { type: "message_update", usage: { input: 0, output: 0 }, assistantMessageEvent: { type: "toolcall_start", contentIndex: 0, id: call.id, toolName: call.name } },
  { type: "message_update", assistantMessageEvent: { type: "toolcall_delta", contentIndex: 0, delta: '{"command":' } },
  { type: "message_update", assistantMessageEvent: { type: "toolcall_delta", contentIndex: 0, delta: '"printf sanitized"}' } },
  { type: "message_update", assistantMessageEvent: { type: "toolcall_end", contentIndex: 0, toolCall: call } },
  { type: "message_end", message: first },
  { type: "tool_execution_start", toolCallId: call.id, toolName: call.name, args: call.arguments },
  { type: "tool_execution_update", toolCallId: call.id, toolName: call.name, args: call.arguments, partialResult: { content: [{ type: "text", text: "sanitized" }] } },
  { type: "tool_execution_end", toolCallId: call.id, toolName: call.name, result, isError: false },
  { type: "message_start", message: toolResult },
  { type: "message_end", message: toolResult },
  { type: "turn_end", message: first, toolResults: [toolResult] },
  { type: "turn_start" },
  { type: "message_start", message: { ...final, content: [], usage: { input: 0, output: 0 } } },
  { type: "message_update", assistantMessageEvent: { type: "text_start", contentIndex: 0 } },
  { type: "message_update", assistantMessageEvent: { type: "text_delta", contentIndex: 0, delta: "Sanitized " } },
  { type: "message_update", assistantMessageEvent: { type: "text_delta", contentIndex: 0, delta: "complete.\n" } },
  { type: "message_update", assistantMessageEvent: { type: "text_end", contentIndex: 0, content: "Sanitized complete.\n" } },
  { type: "message_end", message: final },
  { type: "turn_end", message: final, toolResults: [] },
  { type: "agent_end", messages: [user, first, toolResult, final], willRetry: false },
  { type: "agent_settled" },
];

// Provider error shape from Repository Tree Map Generator run 36447274044,
// agent artifact 10981486046/pi-streaming.jsonl. Prompt and identifiers replaced.
const error = {
  role: "assistant",
  content: [],
  api: "openai-responses",
  provider: "copilot",
  model: "auto",
  timestamp: 5,
  stopReason: "error",
  errorMessage: '400: {"message":"The requested model is not supported.","code":"model_not_supported","param":"model","type":"invalid_request_error"}',
  usage: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, totalTokens: 0, cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0 } },
};
const failure = [
  success[0],
  { type: "agent_start" },
  { type: "turn_start" },
  { type: "message_start", message: user },
  { type: "message_end", message: user },
  { type: "message_start", message: { ...error, usage: undefined, errorMessage: undefined } },
  { type: "message_end", message: error },
  { type: "turn_end", message: error, toolResults: [] },
  { type: "agent_end", messages: [user, error], willRetry: false },
  { type: "agent_settled" },
];

// Shape from Smoke Pi Sub Agents run 37735099783, agent artifact
// 11531640894/pi-streaming.jsonl. Names, IDs, timestamps and content replaced.
// This historical wrapper has no native invocation or parent tool identity.
const childObservations = [
  { type: "gh_aw_subagent_dispatch", agent: "sanitized-reader", requested_model: "gpt-5-mini", resolved_model: "gpt-5-mini" },
  {
    type: "gh_aw_subagent_event",
    agent: "sanitized-reader",
    event: {
      type: "message_end",
      message: {
        role: "assistant",
        content: [
          { type: "thinking", thinking: "Sanitized reasoning.", thinkingSignature: "sanitized-signature" },
          { type: "text", text: "Sanitized child answer.", textSignature: "sanitized-signature" },
        ],
        api: "openai-responses",
        provider: "aw-gateway",
        model: "gpt-5-mini",
        responseId: "sanitized-child-response",
        timestamp: 6,
        rawStopReason: "completed",
        stopReason: "stop",
        thinkingLevel: "minimal",
        usage: { input: 3085, output: 136, cacheRead: 0, cacheWrite: 0, reasoning: 64, totalTokens: 3221, cost: { input: 0.00077125, output: 0.000272, cacheRead: 0, cacheWrite: 0, total: 0.00104325 } },
      },
    },
  },
];

// Synthetic v1.3.0 release-contract observations, not captured workflow output.
// https://github.com/earendil-works/pi/blob/v1.3.0/packages/ai/src/types.ts
// https://github.com/earendil-works/pi/blob/v1.3.0/packages/coding-agent/src/core/agent-session.ts
const releaseMessage = {
  role: "assistant",
  responseId: "release-response",
  model: "configured-model",
  responseModel: "served-model",
  content: [{ type: "text", text: "Release fixture complete." }],
  thinkingLevel: "max",
  providerThinkingLevel: "max",
  stopReason: "stop",
  timestamp: 1,
  durationMs: 0,
  usage: { input: 10, output: 5, reasoning: 3, cacheRead: 2, cacheWrite: 0, totalTokens: 17, cost: { total: 0.1 } },
};
const release130 = [
  { type: "session", version: 3, id: "release-session", cwd: "/sanitized" },
  { type: "thinking_level_changed", level: "max" },
  { type: "message_end", message: releaseMessage },
  { type: "turn_end", message: releaseMessage, toolResults: [] },
  { type: "agent_end", messages: [releaseMessage], willRetry: false },
  { type: "agent_settled", aborted: false },
];

module.exports = { success, failure, childObservations, release130 };
