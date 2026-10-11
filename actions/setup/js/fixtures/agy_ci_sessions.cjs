// Sanitized subset of the native stream in the agent artifact of
// https://github.com/github/gh-aw/actions/runs/37848575608 (failed conformance,
// successful agent execution). Paths, conversation IDs and nonce payloads are
// replaced; states, missing fields, durations, text and accounting are retained.
// Rechecked against agent artifact 11581176766 on 2026-10-11. Native engine
// version: 1.3.1; workflow commit: badfd988eb7195722b6ec5e618c0064ce23564ca.
const conversation = "agy-ci-conversation";
const step = data => ({ event: "step_update", step_update: { conversation_id: conversation, ...data } });

const agySmoke = [
  { event: "init", conversation_id: conversation, init: { model: "gemini-3.8-flash-medium", cwd: "/workspace/gh-aw", tools: ["write_to_file", "call_mcp_tool"], permission_mode: "always-proceed" } },
  step({ step_index: 0, state: "DONE", step_type: "user_input" }),
  step({
    step_index: 1,
    state: "DONE",
    step_type: "agent_response",
    duration_seconds: 4.544596791,
    usage: { input_tokens: 17580, output_tokens: 544, thinking_tokens: 492, cache_read_tokens: 0, total_tokens: 18124 },
  }),
  step({
    step_index: 16,
    state: "ACTIVE",
    step_type: "tool",
    tool_name: "call_mcp_tool",
    tool_info: { name: "call_mcp_tool", parameters: { Arguments: { file_nonce: "fixture-nonce" }, ServerName: "agy-native", ToolName: "native_challenge" } },
  }),
  step({
    step_index: 16,
    state: "DONE",
    step_type: "tool",
    tool_name: "call_mcp_tool",
    duration_seconds: 0.07261689,
    tool_info: { name: "call_mcp_tool", parameters: { Arguments: { file_nonce: "fixture-nonce" }, ServerName: "agy-native", ToolName: "native_challenge" }, output: '{"toolNonce":"fixture-receipt"}' },
  }),
  step({ step_index: 20, state: "ACTIVE", step_type: "tool", tool_name: "write_to_file", tool_info: { name: "write_to_file", parameters: { TargetFile: "/workspace/result.json" } } }),
  step({
    step_index: 20,
    state: "DONE",
    step_type: "tool",
    tool_name: "write_to_file",
    duration_seconds: 0.020827639,
    tool_info: { name: "write_to_file", parameters: { TargetFile: "/workspace/result.json" } },
  }),
  ...["Engine confi", "guration conformance sui", "te and native Agy confor"].map(text_delta => step({ step_index: 33, state: "ACTIVE", step_type: "agent_response", text_delta })),
  step({
    step_index: 33,
    state: "DONE",
    step_type: "agent_response",
    text_delta: "mance gate executed successfully.\n",
    duration_seconds: 3.540337529,
    usage: { input_tokens: 5715, output_tokens: 371, thinking_tokens: 358, cache_read_tokens: 20916, total_tokens: 6086 },
  }),
  {
    event: "result",
    result: {
      conversation_id: conversation,
      status: "SUCCESS",
      response: "Engine configuration conformance suite and native Agy conformance gate executed successfully.\n",
      duration_seconds: 51.779892838,
      num_turns: 1,
      usage: { input_tokens: 79491, output_tokens: 4742, thinking_tokens: 3221, cache_read_tokens: 298918, total_tokens: 84233 },
    },
  },
];

// All native per-step accounting observations from the same trace, without
// prompt, response, paths or tool payloads. The appended legacy summary is not
// native accounting: it incorrectly marks input tokens as including cache.
const agyAccounting = [
  [17580, 544, 492, 0, 18124],
  [2095, 247, 159, 16425, 2342],
  [2502, 282, 154, 16416, 2784],
  [3166, 94, 17, 16472, 3260],
  [3417, 105, 27, 16463, 3522],
  [3763, 200, 75, 16453, 3963],
  [4352, 294, 220, 16444, 4646],
  [4945, 193, 89, 16518, 5138],
  [5259, 741, 654, 16509, 6000],
  [2125, 554, 276, 20807, 2679],
  [2824, 64, 12, 20901, 2888],
  [3298, 348, 274, 20884, 3646],
  [3828, 91, 15, 20926, 3919],
  [4171, 377, 293, 20911, 4548],
  [4900, 155, 86, 20946, 5055],
  [5551, 82, 20, 20927, 5633],
  [5715, 371, 358, 20916, 6086],
].map(([input_tokens, output_tokens, thinking_tokens, cache_read_tokens, total_tokens], index) =>
  step({ step_index: index * 2 + 1, state: "DONE", step_type: "agent_response", usage: { input_tokens, output_tokens, thinking_tokens, cache_read_tokens, total_tokens } })
);

const agyLegacySummary = {
  type: "result",
  num_turns: 1,
  usage: { input_tokens: 79491, output_tokens: 4742, cache_read_input_tokens: 298918, total_tokens: 84233, reasoning_output_tokens: 3221, input_tokens_include_cache: true },
};

module.exports = { agySmoke, agyAccounting, agyLegacySummary };
