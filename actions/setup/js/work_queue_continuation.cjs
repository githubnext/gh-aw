// @ts-check
"use strict";

const { closed } = require("./work_queue_codec.cjs");

function createWorkQueueContinuationTool(snapshot, options = {}) {
  return {
    name: "work_queue_continue",
    description:
      "Experimental: finish this original Claim with outcome completed and keep_open true, retaining the same Work and its backing Issue for another run after independently verified delivery. Persist progress and handoff notes through the Claim's declared outputs before calling this tool. No child Work or workflow dispatch is created; normal queue policy selects the next Claim.",
    inputSchema: {
      type: "object",
      properties: { claim_handle: { type: "string", minLength: 1, maxLength: 256 } },
      additionalProperties: false,
    },
    handler: args => {
      closed(args, [], ["claim_handle"], "work_queue_continue");
      const { createWorkQueueFinishTool } = require("./work_queue_mcp_server.cjs");
      return createWorkQueueFinishTool({ ...options, snapshot }).handler({ ...args, outcome: "completed", keep_open: true });
    },
  };
}

module.exports = { createWorkQueueContinuationTool };
