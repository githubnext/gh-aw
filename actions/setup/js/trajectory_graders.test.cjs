const { normalizeResult, BUILTIN_META } = require("./trace_graders.cjs");
const { TRAJECTORY_GRADERS, TRAJECTORY_GRADER_META, MAX_TRAJECTORY_ITEMS } = require("./trajectory_graders.cjs");
function makeTrace(overrides = {}) {
  return {
    tokenUsageEntries: [],
    agentUsage: null,
    mcpGatewayEntries: [],
    agentOutput: null,
    toolCalls: [],
    gatewayRequests: [],
    retryEvents: [],
    errorEvents: [],
    steps: [],
    totalInputTokens: 0,
    totalOutputTokens: 0,
    totalDurationMs: 0,
    totalRequests: 0,
    files: [],
    artifacts: [],
    ...overrides,
  };
}

function runTrajectoryBuiltin(id, trace, config) {
  return normalizeResult(id, TRAJECTORY_GRADERS[id](makeTrace(trace), config), { ...BUILTIN_META[id], source: "builtin" });
}

function runPolicyNearMiss(trace) {
  return runTrajectoryBuiltin("policy-near-miss", trace);
}

function runExplorationError(trace) {
  return runTrajectoryBuiltin("exploration-error", trace);
}

function runExploitationError(trace) {
  return runTrajectoryBuiltin("exploitation-error", trace);
}

function runSkillConstraintCoverage(trace, config) {
  return runTrajectoryBuiltin("skill-constraint-coverage", trace, config);
}

function runToolOutputConsumptionRate(trace) {
  return runTrajectoryBuiltin("tool-output-consumption-rate", trace);
}

function runBacklogGrader(id, trace) {
  return runTrajectoryBuiltin(id, trace);
}

describe("trajectory projections", () => {
  describe("policy-near-miss built-in grader", () => {
    it("discovers objectives after an events-only candidate", () => {
      const result = runPolicyNearMiss({
        trajectoryIR: { events: [{ kind: "safe_output" }] },
        ir: { objectives: [{ id: "guard", description: "Verify approval", satisfiedAtEventIndex: null }] },
      });

      expect(result.value).toBe(1);
      expect(result.details).toContain("guardObjectives=1 unmet=1");
    });

    it("identifies guard objectives and treats event index zero as satisfied", () => {
      const result = runPolicyNearMiss({
        trajectoryIR: {
          events: [{ kind: "safe_output" }],
          objectives: [
            { id: "met", description: "Check authorization", satisfiedAtEventIndex: 0 },
            { id: "unmet", description: "Verification required", satisfiedAtEventIndex: null },
            { id: "not-a-guard", description: "Complete checkbox", satisfiedAtEventIndex: null },
          ],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("guardObjectives=2 unmet=1");
    });

    it.each([
      ["no objectives", { trajectoryIR: { events: [{ kind: "safe_output" }], objectives: [] } }],
      ["no outcome", { trajectoryIR: { events: [], objectives: [{ description: "Check approval" }] } }],
      ["no guard objective", { trajectoryIR: { events: [{ kind: "safe_output" }], objectives: [{ description: "Write report" }] } }],
    ])("normalizes %s as unavailable", (_name, trace) => {
      const result = runPolicyNearMiss(trace);

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.status).toBe("unavailable");
    });
  });

  describe("exploration-error built-in grader", () => {
    it("reports unavailable when no objectives are declared", () => {
      const result = runExplorationError({ trajectoryIR: { events: [{ kind: "state_change", ref: "a" }] } });

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.message).toContain("no declared objectives");
    });

    it("returns zero when all objectives are already satisfied", () => {
      const result = runExplorationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Read all files", satisfiedAtEventIndex: 0 }],
        },
      });

      expect(result.value).toBe(0);
      expect(result.details).toContain("all objectives satisfied");
    });

    it("prefers the objective-bearing IR candidate over unrelated agentOutput observations", () => {
      const result = runExplorationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          events: [{ kind: "state_change", ref: "repo-root" }],
        },
        agentOutput: {
          observations: [{ id: "obs-1" }],
        },
      });

      expect(result.value).toBe(1);
      expect(result.details).toContain("observations=0");
    });

    it("counts distinct state refs from state_change events", () => {
      const result = runExplorationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          events: [
            { kind: "state_change", ref: "repo-root" },
            { kind: "state_change", ref: "repo-readme" },
            { kind: "state_change", ref: "repo-root" },
          ],
          observations: [{ id: "obs-1" }],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("distinctStatesVisited=2");
    });

    it("falls back to declared states[] when no state_change events exist", () => {
      const result = runExplorationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          states: [{ id: "a" }, { id: "b" }],
          observations: [{ id: "obs-1" }],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("from declared states[]");
    });

    it("clamps the ratio at zero when observations exceed the visited-state count", () => {
      const result = runExplorationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          states: [{ id: "a" }],
          observations: [{ id: "obs-1" }, { id: "obs-2" }, { id: "obs-3" }],
        },
      });

      expect(result.value).toBe(0);
    });

    it("is unavailable without state_change events or declared states", () => {
      const result = runExplorationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          observations: [{ id: "obs-1" }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.message).toContain("no state_change events or declared states");
    });
  });

  describe("exploitation-error built-in grader", () => {
    it("reports unavailable when no objectives are declared", () => {
      const result = runExploitationError({ trajectoryIR: { observations: [{ id: "obs-1" }] } });

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.message).toContain("no declared objectives");
    });

    it("returns zero when all objectives are already satisfied", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Read all files", satisfiedAtEventIndex: 0 }],
        },
      });

      expect(result.value).toBe(0);
      expect(result.details).toContain("all objectives satisfied");
    });

    it("is unavailable without state_change events or declared states", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          observations: [{ id: "obs-1" }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.message).toContain("no state_change events or declared states");
    });

    it("is unavailable when the trace records no observations", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          states: [{ id: "a" }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.message).toContain("no observations");
    });

    it("defers to exploration-error when exploration was insufficient", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          events: [
            { kind: "state_change", ref: "repo-root" },
            { kind: "state_change", ref: "repo-readme" },
          ],
          observations: [{ id: "obs-1", consumedByActionIds: ["act-1"] }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.message).toContain("exploration was insufficient");
      expect(result.message).toContain("exploration-error");
    });

    it("scores the unused fraction of observations when exploration was sufficient", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          events: [
            { kind: "state_change", ref: "repo-root" },
            { kind: "state_change", ref: "repo-root" },
          ],
          observations: [
            { id: "obs-1", consumedByActionIds: ["act-1"] },
            { id: "obs-2", consumedByActionIds: [] },
          ],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("observations=2 unused=1");
      expect(result.details).toContain("distinctStatesVisited=1");
      expect(result.details).toContain("unmet objectives: goal");
    });

    it("falls back to declared states[] when no state_change events exist", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          states: [{ id: "a" }, { id: "b" }],
          observations: [{ id: "obs-1" }, { id: "obs-2" }],
        },
      });

      expect(result.value).toBe(1);
      expect(result.details).toContain("from declared states[]");
    });

    it("prefers the objective-bearing IR candidate over unrelated agentOutput observations", () => {
      const result = runExploitationError({
        trajectoryIR: {
          objectives: [{ id: "goal", description: "Inspect repo", satisfiedAtEventIndex: null }],
          states: [{ id: "a" }],
          observations: [{ id: "obs-1", consumedByActionIds: ["act-1"] }],
        },
        agentOutput: {
          observations: [{ id: "obs-x" }],
        },
      });

      expect(result.value).toBe(0);
      expect(result.details).toContain("observations=1 unused=0");
    });
  });

  describe("skill-constraint-coverage built-in grader", () => {
    it("reports full coverage when every constraint is exercised and succeeds", () => {
      const result = runSkillConstraintCoverage(
        {
          trajectoryIR: {
            toolCalls: [
              { name: "read_file", arguments: { path: "a.md" }, success: true },
              { name: "run_lint", arguments: {}, success: true },
            ],
          },
        },
        {
          constraints: [
            { id: "reads-before-write", pattern: "read_file", description: "must read before writing" },
            { id: "lints", pattern: "run_lint", description: "must lint" },
          ],
        }
      );

      expect(result.value).toBe(1);
      expect(result.details).toContain("constraints=2 exercised=2 covered=2");
    });

    it("counts an exercised-but-failing constraint as uncovered", () => {
      const result = runSkillConstraintCoverage(
        {
          trajectoryIR: {
            toolCalls: [{ name: "run_lint", arguments: {}, success: false }],
          },
        },
        {
          constraints: [{ id: "lints", pattern: "run_lint" }],
        }
      );

      expect(result.value).toBe(0);
      expect(result.details).toContain("exercised=1 covered=0");
      expect(result.details).toContain("unmet: lints");
    });

    it("ignores success when requireSuccess is false", () => {
      const result = runSkillConstraintCoverage(
        {
          trajectoryIR: {
            actions: [{ type: "comment", target: "issue-1", validAtIssueTime: false }],
          },
        },
        {
          constraints: [{ id: "comments", pattern: "comment", requireSuccess: false }],
        }
      );

      expect(result.value).toBe(1);
      expect(result.details).toContain("covered=1");
    });

    it("uses the preprocessed trace as a fallback candidate when trajectoryIR is absent", () => {
      const result = runSkillConstraintCoverage(
        {
          toolCalls: [{ name: "read_file", arguments: { path: "a.md" }, success: true }],
        },
        {
          constraints: [{ id: "reads-before-write", pattern: "read_file" }],
        }
      );

      expect(result.value).toBe(1);
      expect(result.details).toContain("constraints=1 exercised=1 covered=1");
    });

    it("counts non-object constraint entries against the denominator instead of dropping them", () => {
      const result = runSkillConstraintCoverage(
        {
          trajectoryIR: {
            toolCalls: [{ name: "run_lint", arguments: {}, success: true }],
          },
        },
        {
          constraints: [null, { id: "lints", pattern: "run_lint" }],
        }
      );

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("constraints=2 exercised=1 covered=1 invalidPattern=1");
    });

    it("counts a constraint with an invalid pattern against the denominator instead of dropping it", () => {
      const result = runSkillConstraintCoverage(
        {
          trajectoryIR: {
            toolCalls: [{ name: "run_lint", arguments: {}, success: true }],
          },
        },
        {
          constraints: [
            { id: "lints", pattern: "run_lint" },
            { id: "broken", pattern: "(" },
          ],
        }
      );

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("constraints=2 exercised=1 covered=1 invalidPattern=1");
    });

    it.each([
      ["no constraints configured", { trajectoryIR: { toolCalls: [{ name: "x", success: true }] } }, {}],
      ["no constraints with a valid pattern", { trajectoryIR: { toolCalls: [{ name: "x", success: true }] } }, { constraints: [{ id: "c", pattern: "" }] }],
      ["no toolCalls/actions in trace", { trajectoryIR: {} }, { constraints: [{ id: "c", pattern: "x" }] }],
    ])("normalizes %s as unavailable", (_name, trace, config) => {
      const result = runSkillConstraintCoverage(trace, config);

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.status).toBe("unavailable");
    });
  });

  describe("tool-output-consumption-rate built-in grader", () => {
    it("scores the fraction of matching tool observations that were consumed", () => {
      const result = runToolOutputConsumptionRate({
        trajectoryIR: {
          toolCalls: [{ id: "tc-1" }, { id: "tc-2" }],
          observations: [
            { id: "obs-1", sourceToolCallId: "tc-1", consumedByActionIds: ["act-1"] },
            { id: "obs-2", sourceToolCallId: "tc-2", consumedByActionIds: [] },
            { id: "obs-3", sourceToolCallId: null, consumedByActionIds: ["act-2"] },
            { id: "obs-4", sourceToolCallId: "unknown", consumedByActionIds: ["act-3"] },
          ],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("toolObservations=2 consumed=1");
      expect(result.details).toContain("unconsumed: obs-2");
    });

    it("treats malformed consumption metadata as unconsumed", () => {
      const result = runToolOutputConsumptionRate({
        trajectoryIR: {
          toolCalls: [{ id: "tc-1" }],
          observations: [{ id: "obs-1", sourceToolCallId: "tc-1", consumedByActionIds: "act-1" }],
        },
      });

      expect(result.value).toBe(0);
      expect(result.details).toContain("toolObservations=1 consumed=0");
    });

    it("treats consumedByActionIds arrays containing only non-string/empty entries as unconsumed", () => {
      const result = runToolOutputConsumptionRate({
        trajectoryIR: {
          toolCalls: [{ id: "tc-1" }, { id: "tc-2" }],
          observations: [
            { id: "obs-1", sourceToolCallId: "tc-1", consumedByActionIds: [null] },
            { id: "obs-2", sourceToolCallId: "tc-2", consumedByActionIds: [42, ""] },
          ],
        },
      });

      expect(result.value).toBe(0);
      expect(result.details).toContain("toolObservations=2 consumed=0");
      expect(result.details).toContain("unconsumed: obs-1, obs-2");
    });

    it("reads observations/toolCalls from the root trace object", () => {
      const result = runToolOutputConsumptionRate({
        toolCalls: [{ id: "tc-1" }],
        observations: [{ id: "obs-1", sourceToolCallId: "tc-1", consumedByActionIds: ["act-1"] }],
      });

      expect(result.value).toBe(1);
    });

    it("reads a complete IR nested in agentOutput", () => {
      const result = runToolOutputConsumptionRate({
        agentOutput: {
          trajectoryIR: {
            toolCalls: [{ id: "tc-1" }],
            observations: [{ id: "obs-1", sourceToolCallId: "tc-1", consumedByActionIds: ["act-1"] }],
          },
        },
      });

      expect(result.value).toBe(1);
    });

    it.each([
      ["no observations", { trajectoryIR: { toolCalls: [{ id: "tc-1" }] } }, "no observations"],
      ["no tool calls", { trajectoryIR: { observations: [{ id: "obs-1", sourceToolCallId: "tc-1", consumedByActionIds: ["act-1"] }] } }, "no tool-originated observations"],
      [
        "no matching tool call",
        {
          trajectoryIR: {
            toolCalls: [{ id: "tc-1" }],
            observations: [{ id: "obs-1", sourceToolCallId: "unknown", consumedByActionIds: ["act-1"] }],
          },
        },
        "no tool-originated observations",
      ],
    ])("normalizes %s as unavailable", (_name, trace, message) => {
      const result = runToolOutputConsumptionRate(trace);

      expect(result.value).toBeNull();
      expect(result.passed).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain(message);
    });
  });

  describe("provenance backlog built-in graders", () => {
    const trace = {
      trajectoryIR: {
        events: [
          { index: 0, kind: "observation", ref: "obs-1" },
          { index: 1, kind: "safe_output", ref: "output-1" },
          { index: 2, kind: "safe_output", ref: "output-2" },
        ],
        actions: [
          { id: "action-1", type: "edit", target: "a.go", consequential: true },
          { id: "action-2", type: "comment", target: "issue-1", consequential: true },
          { id: "action-3", type: "read", target: "README.md", consequential: false },
        ],
        toolCalls: [{ id: "tool-1" }],
        observations: [{ id: "obs-1" }],
        provenanceEdges: [
          { from: "obs-1", to: "derived-1", relation: "informed" },
          { from: "derived-1", to: "output-1", relation: "derived-from" },
          { from: "tool-1", to: "action-1", relation: "informed" },
        ],
      },
    };

    it("follows multi-hop lineage from final outputs to evidence roots", () => {
      const result = runBacklogGrader("end-to-end-lineage-completeness", trace);

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("outputs=2 traceable=1");
      expect(result.details).toContain("output-2");
    });

    it("scores consequential actions only", () => {
      const result = runBacklogGrader("action-provenance-coverage", trace);

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("consequentialActions=2 covered=1");
      expect(result.details).toContain("action-2");
    });

    it.each([
      ["end-to-end-lineage-completeness", { trajectoryIR: { events: [], provenanceEdges: [] } }],
      ["action-provenance-coverage", { trajectoryIR: { actions: [{ id: "read-1", type: "read", consequential: false }], provenanceEdges: [], toolCalls: [], observations: [] } }],
    ])("normalizes non-applicable %s evidence as unavailable", (id, unavailableTrace) => {
      const result = runBacklogGrader(id, unavailableTrace);

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
    });

    it("rejects an action provenance graph missing the canonical consequential flag", () => {
      const result = runBacklogGrader("action-provenance-coverage", {
        trajectoryIR: {
          actions: [{ id: "action-1", type: "edit" }],
          toolCalls: [],
          observations: [],
          provenanceEdges: [],
        },
      });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("consequential flag");
    });

    it("rejects a malformed non-record action entry instead of silently excluding it", () => {
      const result = runBacklogGrader("action-provenance-coverage", {
        trajectoryIR: {
          actions: [{ id: "action-1", type: "edit", consequential: true }, "not-a-record"],
          toolCalls: [],
          observations: [],
          provenanceEdges: [],
        },
      });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("not a record");
    });
  });

  describe("completion backlog built-in graders", () => {
    it("counts every unsatisfied objective at termination", () => {
      const result = runBacklogGrader("premature-termination-gap", {
        trajectoryIR: {
          objectives: [{ id: "done", satisfiedAtEventIndex: 1 }, { id: "missing", satisfiedAtEventIndex: null }, { id: "also-missing" }],
        },
      });

      expect(result.value).toBe(2);
      expect(result.details).toContain("objectives=3 unsatisfied=2");
    });

    it("counts events after the last objective became satisfied", () => {
      const result = runBacklogGrader("evidence-saturation-stopping-lag", {
        trajectoryIR: {
          objectives: [
            { id: "first", satisfiedAtEventIndex: 1 },
            { id: "last", satisfiedAtEventIndex: 3 },
          ],
          events: [{ index: 0 }, { index: 1 }, { index: 3 }, { index: 4 }, { index: 8 }],
        },
      });

      expect(result.value).toBe(2);
      expect(result.details).toContain("saturationEventIndex=3 eventsAfterSaturation=2");
    });

    it("does not fabricate a stopping lag before evidence saturation", () => {
      const result = runBacklogGrader("evidence-saturation-stopping-lag", {
        trajectoryIR: {
          objectives: [{ id: "missing", satisfiedAtEventIndex: null }],
          events: [{ index: 0 }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("never saturated");
    });

    it("rejects an earlier satisfaction index absent from the trace even when the latest one is present", () => {
      const result = runBacklogGrader("evidence-saturation-stopping-lag", {
        trajectoryIR: {
          objectives: [
            { id: "first", satisfiedAtEventIndex: 2 },
            { id: "last", satisfiedAtEventIndex: 3 },
          ],
          events: [{ index: 0 }, { index: 1 }, { index: 3 }, { index: 4 }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("absent from the trace");
    });
  });

  describe("dependency-order-violation-rate built-in grader", () => {
    it("scores completed dependent objectives with any late prerequisite as violations", () => {
      const result = runBacklogGrader("dependency-order-violation-rate", {
        trajectoryIR: {
          objectives: [
            { id: "prerequisite-a", satisfiedAtEventIndex: 1, dependsOn: [] },
            { id: "prerequisite-b", satisfiedAtEventIndex: 5, dependsOn: [] },
            { id: "ordered", satisfiedAtEventIndex: 3, dependsOn: ["prerequisite-a"] },
            { id: "violating", satisfiedAtEventIndex: 4, dependsOn: ["prerequisite-a", "prerequisite-b"] },
            { id: "not-executed", satisfiedAtEventIndex: null, dependsOn: ["prerequisite-a"] },
          ],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("completedDependentObjectives=2 violations=1");
      expect(result.details).toContain("violating");
    });

    it("rejects incomplete dependency graphs", () => {
      const result = runBacklogGrader("dependency-order-violation-rate", {
        trajectoryIR: {
          objectives: [{ id: "dependent", satisfiedAtEventIndex: 1, dependsOn: ["missing"] }],
        },
      });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("unknown objective");
    });

    it("rejects a malformed non-array dependsOn instead of treating it as no dependencies", () => {
      const result = runBacklogGrader("dependency-order-violation-rate", {
        trajectoryIR: {
          objectives: [
            { id: "prerequisite-a", satisfiedAtEventIndex: 1, dependsOn: [] },
            { id: "malformed", satisfiedAtEventIndex: 2, dependsOn: "prerequisite-a" },
          ],
        },
      });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("dependsOn must be an array");
    });
  });

  describe("objective and grounding backlog built-in graders", () => {
    it("scores the fraction of declared objectives completed", () => {
      const result = runBacklogGrader("objective-coverage", {
        trajectoryIR: {
          objectives: [{ id: "done", satisfiedAtEventIndex: 1 }, { id: "done-too", satisfiedAtEventIndex: 0 }, { id: "missing", satisfiedAtEventIndex: null }, { id: "absent" }],
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("objectives=4 completed=2");
      expect(result.details).toContain("missing, absent");
    });

    it("rejects invalid objective satisfaction indexes", () => {
      const result = runBacklogGrader("objective-coverage", { trajectoryIR: { objectives: [{ id: "bad", satisfiedAtEventIndex: -1 }] } });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("invalid objective satisfaction index");
    });

    it("scores actions that were valid when issued", () => {
      const result = runBacklogGrader("grounding-accuracy", {
        trajectoryIR: {
          actions: [
            { id: "a1", validAtIssueTime: true },
            { id: "a2", validAtIssueTime: false },
            { id: "a3", validAtIssueTime: true },
            { id: "a4", validAtIssueTime: true },
          ],
        },
      });

      expect(result.value).toBeCloseTo(0.75);
      expect(result.details).toContain("actions=4 valid=3; ungrounded: a2");
    });

    it("does not guess grounding when validity is missing", () => {
      const result = runBacklogGrader("grounding-accuracy", { trajectoryIR: { actions: [{ id: "a1", validAtIssueTime: true }, { id: "a2" }] } });

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
    });
  });

  describe("reference-trajectory backlog built-in graders", () => {
    it("scores the correct tool prefix with parameter credit", () => {
      const result = runBacklogGrader("tool-wise-score", {
        trajectoryIR: {
          toolCalls: [
            { id: "t2", eventIndex: 2, name: "view", arguments: { path: "b.go", range: [1, 2] } },
            { id: "t1", eventIndex: 0, name: "grep", arguments: { pattern: "x", path: "." } },
            { id: "t3", eventIndex: 4, name: "bash", arguments: {} },
          ],
          reference: {
            toolCalls: [{ name: "grep", arguments: { path: ".", pattern: "x" } }, { name: "view", arguments: { path: "a.go", range: [1, 2] } }, { name: "edit", arguments: { path: "a.go" } }, { name: "bash" }],
          },
        },
      });

      expect(result.value).toBeCloseTo((1 + 0.75) / 4);
      expect(result.details).toContain("referenceSteps=4 observedSteps=3 correctPrefix=2");
      expect(result.details).toContain("expected edit, got bash");
    });

    it("computes nDTW over state_change events", () => {
      const exact = runBacklogGrader("trajectory-ndtw", {
        trajectoryIR: {
          events: [
            { index: 0, kind: "state_change", ref: "s1" },
            { index: 1, kind: "tool_call", ref: "t1" },
            { index: 2, kind: "state_change", ref: "s2" },
            { index: 3, kind: "state_change", ref: "s2" },
          ],
          reference: { states: ["s1", "s2"] },
        },
      });
      const detour = runBacklogGrader("trajectory-ndtw", {
        trajectoryIR: {
          events: [
            { index: 0, kind: "state_change", ref: "s1" },
            { index: 1, kind: "state_change", ref: "s9" },
            { index: 2, kind: "state_change", ref: "s2" },
          ],
          reference: { states: ["s1", "s2"] },
        },
      });

      expect(exact.value).toBeCloseTo(1);
      expect(detour.value).toBeCloseTo(Math.exp(-1 / 2));
      expect(detour.details).toContain("referenceStates=2 observedStates=3 dtwDistance=1");
    });

    it("scores recall of reference patch files", () => {
      const result = runBacklogGrader("code-search-recall", {
        trajectoryIR: {
          resources: [
            { id: "r1", kind: "file", uri: "./pkg/a.go" },
            { id: "r2", kind: "url", uri: "pkg/b.go" },
            { id: "r3", kind: "file", uri: "pkg/c.go" },
          ],
          reference: { patch: { files: [{ path: "pkg/a.go" }, "pkg/b.go", "pkg/c.go", "pkg/d.go"] } },
        },
      });

      expect(result.value).toBeCloseTo(0.5);
      expect(result.details).toContain("referenceFiles=4 located=2; never located: pkg/b.go, pkg/d.go");
    });

    it.each([
      ["tool-wise-score", { trajectoryIR: { toolCalls: [] } }],
      ["trajectory-ndtw", { trajectoryIR: { events: [{ index: 0, kind: "state_change", ref: "s1" }], reference: null } }],
      ["code-search-recall", { trajectoryIR: { resources: [], reference: { patch: { files: [] } } } }],
      ["trajectory-ndtw", { trajectoryIR: { events: [{ index: 0, kind: "tool_call", ref: "t1" }], reference: { states: ["s1"] } } }],
      ["tool-wise-score", { trajectoryIR: { reference: { toolCalls: [{ name: "grep" }] } } }],
      ["code-search-recall", { trajectoryIR: { reference: { patch: { files: [{ path: "" }] } }, resources: [] } }],
      ["tool-wise-score", { trajectoryIR: { toolCalls: [{ name: "grep", eventIndex: 1 }, { name: "view" }], reference: { toolCalls: [{ name: "grep" }] } } }],
    ])("reports %s as unavailable without fabricating a score", (id, unavailableTrace) => {
      const result = runBacklogGrader(id, unavailableTrace);

      expect(result.value).toBeNull();
      expect(result.status).toBe("unavailable");
    });
  });

  describe("code-read-precision built-in grader", () => {
    const reference = { patch: { files: ["pkg/a.go", { path: "./pkg/a.go" }, { path: "/pkg/b.go" }, "pkg/unread.go"] } };
    const read = target => ({ type: "read", target, consequential: false });
    const run = ir => runTrajectoryBuiltin("code-read-precision", { trajectoryIR: ir });

    it("scores distinct file reads, not reference recall or action frequency", () => {
      const result = run({
        reference,
        actions: [
          read(" pkg/a.go "),
          read("./pkg/a.go"),
          { type: "inspect", target: "/pkg/b.go", consequential: false },
          read("pkg/unrelated.go"),
          { type: "search", target: "pkg/other.go", consequential: false },
          { type: "list", target: "pkg/listed.go", consequential: false },
          { type: "edit", target: "pkg/edited.go", consequential: true },
          { type: "read", target: "pkg/mutated.go", consequential: true },
          { type: "thread", target: "pkg/thread.go", consequential: false },
        ],
        resources: [{ kind: "file", uri: "pkg/unread.go" }],
      });
      expect(result.value).toBeCloseTo(2 / 3);
      expect(result).toMatchObject({ source: "builtin", status: "pass", unit: "ratio" });
      expect(TRAJECTORY_GRADER_META["code-read-precision"]).toMatchObject({ direction: "higher_is_better", min: 0, max: 1 });
      expect(result.details).toBe("referenceFiles=3 readFiles=3 relevant=2; unrelated reads: pkg/unrelated.go");
    });

    it.each([
      ["all relevant", [read("pkg/a.go"), read("pkg/b.go")], 1],
      ["none relevant", [read("pkg/A.go"), read("pkg/elsewhere.go")], 0],
    ])("scores %s reads", (_name, actions, value) => {
      expect(run({ reference, actions }).value).toBe(value);
    });

    it.each(["root", "trajectoryIR", "trajectoryIr", "ir", "agentOutput", "agentOutput.trajectoryIR", "agentOutput.trajectoryIr", "agentOutput.trajectory"])("accepts the %s canonical IR location", location => {
      const ir = { reference, actions: [read("pkg/a.go")] };
      const trace = location === "root" ? ir : location.split(".").reduceRight((value, key) => ({ [key]: value }), ir);
      expect(runTrajectoryBuiltin("code-read-precision", trace).value).toBe(1);
    });

    it.each([
      ["missing reference", { actions: [read("pkg/a.go")] }, "no reference patch files"],
      ["malformed reference", { reference: { patch: { files: "pkg/a.go" } }, actions: [] }, "no reference patch files"],
      ["empty reference", { reference: { patch: { files: [] } }, actions: [] }, "no reference patch files"],
      ["missing reference path", { reference: { patch: { files: [{}] } }, actions: [] }, "missing a path"],
      ["invalid reference path", { reference: { patch: { files: [{ path: 42 }] } }, actions: [] }, "missing a path"],
      ["empty reference path", { reference: { patch: { files: [" ./ "] } }, actions: [] }, "missing a path"],
      ["missing actions", { reference }, "well-formed actions"],
      ["non-array actions", { reference, actions: {} }, "well-formed actions"],
      ["non-record action", { reference, actions: [read("pkg/a.go"), null] }, "well-formed actions"],
      ["missing type", { reference, actions: [read("pkg/a.go"), { consequential: false }] }, "missing a type"],
      ["missing flag", { reference, actions: [{ type: "read", target: "pkg/a.go" }] }, "consequential flag"],
      ["invalid flag", { reference, actions: [{ ...read("pkg/a.go"), consequential: "false" }] }, "consequential flag"],
      ["missing target", { reference, actions: [read("pkg/a.go"), read(null)] }, "file target"],
      ["invalid target", { reference, actions: [read(42)] }, "file target"],
      ["empty target", { reference, actions: [read(" / ")] }, "file target"],
      ["no reads", { reference, actions: [] }, "no file reads"],
      ["searches only", { reference, actions: [{ type: "search", target: null, consequential: false }] }, "no file reads"],
    ])("reports %s as unavailable without fabricating a score", (_name, ir, reason) => {
      const result = run(ir);
      expect(result).toMatchObject({ value: null, passed: null, status: "unavailable", source: "builtin" });
      expect(result.message).toContain(reason);
    });

    it("does not merge evidence from unrelated IR candidates", () => {
      const result = runTrajectoryBuiltin("code-read-precision", { reference, agentOutput: { actions: [read("pkg/a.go")] } });
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("well-formed actions");
    });

    it.each(["actions", "reference.patch.files"])("rejects oversized %s", collection => {
      const ir = { reference, actions: [read("pkg/a.go")] };
      const oversized = Array.from({ length: MAX_TRAJECTORY_ITEMS + 1 }, () => (collection === "actions" ? read("pkg/a.go") : "pkg/a.go"));
      if (collection === "actions") ir.actions = oversized;
      else ir.reference = { patch: { files: oversized } };
      const result = run(ir);
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain(`${collection} exceeds ${MAX_TRAJECTORY_ITEMS} items`);
    });

    it("computes at the size limit with bounded diagnostic output", () => {
      const longPath = `pkg/${"x".repeat(10_000)}.go`;
      const actions = [read(longPath), ...Array.from({ length: MAX_TRAJECTORY_ITEMS - 1 }, (_, index) => read(`pkg/${index}.go`))];
      const files = actions.map(action => action.target);
      const started = performance.now();
      expect(run({ reference: { patch: { files } }, actions }).value).toBe(1);
      const unrelated = run({ reference, actions });
      expect(unrelated.value).toBe(0);
      expect(unrelated.details).toContain(`${longPath.slice(0, 117)}...`);
      expect(unrelated.details).not.toContain(longPath);
      expect(unrelated.details).toContain("pkg/3.go");
      expect(unrelated.details).not.toContain("pkg/4.go");
      expect(unrelated.details.length).toBeLessThan(1000);
      expect(performance.now() - started).toBeLessThan(1000);
    });
  });

  describe("code-edit-precision built-in grader", () => {
    const reference = { patch: { files: ["pkg/a.go", { path: "./pkg/a.go" }, { path: "/pkg/b.go" }, "pkg/untouched.go"] } };
    const edit = target => ({ type: "edit", target, consequential: true });
    const run = ir => runTrajectoryBuiltin("code-edit-precision", { trajectoryIR: ir });

    it("scores distinct edited files, not reference recall or action frequency", () => {
      const result = run({
        reference,
        actions: [
          edit(" pkg/a.go "),
          edit("./pkg/a.go"),
          edit("/pkg/b.go"),
          edit("pkg/unrelated.go"),
          { type: "read", target: "pkg/untouched.go", consequential: false },
          { type: "search", target: "pkg/other.go", consequential: false },
          { type: "comment", target: "pkg/commented.go", consequential: true },
        ],
      });
      expect(result).toMatchObject({ value: 2 / 3, source: "builtin", status: "pass", unit: "ratio" });
      expect(TRAJECTORY_GRADER_META["code-edit-precision"]).toMatchObject({ direction: "higher_is_better", min: 0, max: 1 });
      expect(result.details).toBe("referenceFiles=3 editedFiles=3 relevant=2; unrelated edits: pkg/unrelated.go");
    });

    it.each([
      ["all relevant", [edit("pkg/a.go"), edit("pkg/b.go")], 1],
      ["none relevant", [edit("pkg/A.go"), edit("pkg/elsewhere.go")], 0],
    ])("scores %s edits", (_name, actions, value) => {
      expect(run({ reference, actions }).value).toBe(value);
    });

    it.each([
      ["missing reference", { actions: [edit("pkg/a.go")] }, "no reference patch files"],
      ["malformed reference", { reference: { patch: { files: "pkg/a.go" } }, actions: [] }, "no reference patch files"],
      ["empty reference", { reference: { patch: { files: [] } }, actions: [] }, "no reference patch files"],
      ["missing reference path", { reference: { patch: { files: [{}] } }, actions: [] }, "missing a path"],
      ["empty reference path", { reference: { patch: { files: [" ./ "] } }, actions: [] }, "missing a path"],
      ["missing actions", { reference }, "well-formed actions"],
      ["non-array actions", { reference, actions: {} }, "well-formed actions"],
      ["non-record action", { reference, actions: [edit("pkg/a.go"), null] }, "well-formed actions"],
      ["missing type", { reference, actions: [edit("pkg/a.go"), { consequential: false }] }, "missing a type"],
      ["missing flag", { reference, actions: [{ type: "edit", target: "pkg/a.go" }] }, "consequential flag"],
      ["non-consequential edit", { reference, actions: [{ ...edit("pkg/a.go"), consequential: false }] }, "not consequential"],
      ["missing target", { reference, actions: [edit("pkg/a.go"), edit(null)] }, "file target"],
      ["empty target", { reference, actions: [edit(" / ")] }, "file target"],
      ["no edits", { reference, actions: [] }, "no file edits"],
      ["reads only", { reference, actions: [{ type: "read", target: "pkg/a.go", consequential: false }] }, "no file edits"],
    ])("reports %s as unavailable without fabricating a score", (_name, ir, reason) => {
      const result = run(ir);
      expect(result).toMatchObject({ value: null, passed: null, status: "unavailable", source: "builtin" });
      expect(result.message).toContain(reason);
    });

    it("does not merge evidence from unrelated IR candidates", () => {
      const result = runTrajectoryBuiltin("code-edit-precision", { reference, agentOutput: { actions: [edit("pkg/a.go")] } });
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain("well-formed actions");
    });

    it.each(["actions", "reference.patch.files"])("rejects oversized %s", collection => {
      const ir = { reference, actions: [edit("pkg/a.go")] };
      const oversized = Array.from({ length: MAX_TRAJECTORY_ITEMS + 1 }, () => (collection === "actions" ? edit("pkg/a.go") : "pkg/a.go"));
      if (collection === "actions") ir.actions = oversized;
      else ir.reference = { patch: { files: oversized } };
      const result = run(ir);
      expect(result.status).toBe("unavailable");
      expect(result.message).toContain(`${collection} exceeds ${MAX_TRAJECTORY_ITEMS} items`);
    });

    it("bounds diagnostics for long unrelated paths", () => {
      const longPath = `pkg/${"x".repeat(10_000)}.go`;
      const result = run({ reference, actions: [edit(longPath)] });
      expect(result.value).toBe(0);
      expect(result.details).toContain(`${longPath.slice(0, 117)}...`);
      expect(result.details).not.toContain(longPath);
    });
  });

  describe("process-alignment-fitness built-in grader", () => {
    const model = { start: "search", end: "edit", transitions: { search: ["read"], read: ["edit"] } };
    const run = (toolCalls, config = { model }, location = "trajectoryIR") => runTrajectoryBuiltin("process-alignment-fitness", { [location]: { toolCalls } }, config);
    const call = (name, eventIndex) => ({ name, eventIndex });

    it("scores an ordered conforming trajectory including both boundaries", () => {
      const result = run([call("edit", 5), call("search", 1), call("read", 3)]);
      expect(result).toMatchObject({ value: 1, status: "pass", source: "builtin", unit: "ratio" });
      expect(result.details).toBe("observedSteps=3 allowedChecks=4 totalChecks=4");
      expect(TRAJECTORY_GRADER_META["process-alignment-fitness"]).toMatchObject({ direction: "higher_is_better", min: 0, max: 1 });
    });

    it.each(["trajectoryIr", "ir"])("reads the %s IR", location => {
      expect(run([call("search", 0), call("read", 1), call("edit", 2)], { model }, location).value).toBe(1);
    });

    it("counts forbidden transitions and wrong start/end independently", () => {
      expect(run([call("search", 0), call("edit", 1)]).value).toBe(2 / 3);
      expect(run([call("read", 0), call("search", 1), call("read", 2)]).value).toBe(1 / 4);
      expect(run([call("other", 0)]).value).toBe(0);
    });

    it("handles a one-call process", () => {
      expect(run([call("search", 0)], { model: { start: "search", end: "search", transitions: {} } }).value).toBe(1);
    });

    it("reports missing models as not applicable", () => {
      const result = run([call("search", 0)], {});
      expect(result).toMatchObject({ value: null, passed: null, status: "unavailable" });
      expect(result.message).toContain("not applicable");
    });

    it.each([
      ["missing calls", undefined],
      ["empty calls", []],
      ["non-array calls", {}],
      ["missing name", [{ eventIndex: 0 }]],
      ["missing event index", [{ name: "search" }]],
      ["negative index", [call("search", -1)]],
      ["duplicate index", [call("search", 1), call("read", 1)]],
    ])("rejects %s", (_reason, toolCalls) => {
      expect(run(toolCalls)).toMatchObject({ value: null, status: "unavailable" });
    });

    it.each([
      ["non-record", []],
      ["missing start", { end: "edit", transitions: {} }],
      ["missing end", { start: "search", transitions: {} }],
      ["invalid transitions", { start: "search", end: "edit", transitions: { search: "read" } }],
      ["invalid destination", { start: "search", end: "edit", transitions: { search: [""] } }],
    ])("rejects a %s model", (_reason, malformed) => {
      expect(run([call("search", 0)], { model: malformed })).toMatchObject({ value: null, status: "unavailable" });
    });

    it("bounds the number of model transitions", () => {
      const oversized = { start: "search", end: "edit", transitions: { search: Array(MAX_TRAJECTORY_ITEMS + 1).fill("read") } };
      expect(run([call("search", 0)], { model: oversized })).toMatchObject({ value: null, status: "unavailable" });
      const manySources = Object.fromEntries(Array.from({ length: MAX_TRAJECTORY_ITEMS + 1 }, (_, index) => [`tool-${index}`, []]));
      expect(run([call("search", 0)], { model: { start: "search", end: "edit", transitions: manySources } })).toMatchObject({ value: null, status: "unavailable" });
    });
  });

  // --- Hostile data ---
  describe("trajectory built-in graders", () => {
    const states = ["a", "b", "a", "b", "c", "c", "c", "a"].map((id, index) => ({ id, firstEventIndex: index }));
    const events = ["tool_call", "observation", "tool_call", "observation", "state_change", "message", "tool_call", "safe_output"].map((kind, index) => ({ index, kind }));

    it("computes state recurrence graders over the canonical IR", () => {
      const trace = { ir: { states, events } };
      expect(runTrajectoryBuiltin("state-revisit-probability-rep", trace).value).toBeCloseTo((8 - 3) / 8);
      expect(runTrajectoryBuiltin("recurrence-rate", trace).value).toBeCloseTo(14 / 56);
      for (const id of ["recurrence-determinism", "recurrence-laminarity", "recurrence-trapping-time"]) {
        const result = runTrajectoryBuiltin(id, trace);
        expect(result.status).toBe("pass");
        expect(typeof result.value).toBe("number");
      }
    });

    it("computes event sequence complexity graders over the canonical IR", () => {
      const trace = { ir: { events } };
      for (const id of ["event-entropy-rate", "lempel-ziv-trajectory-complexity"]) {
        const result = runTrajectoryBuiltin(id, trace);
        expect(result.status).toBe("pass");
        expect(result.value).toBeGreaterThan(0);
        expect(result.value).toBeLessThanOrEqual(1);
      }
    });

    it("reports every trajectory grader as unavailable or not applicable on a trace without IR", () => {
      for (const id of Object.keys(TRAJECTORY_GRADERS)) {
        const result = runTrajectoryBuiltin(id, {});
        expect(result.status).not.toBe("error");
        expect(result.status).toBe("unavailable");
        expect(result.value).toBeNull();
      }
    });

    it("reports oversized IR collections as unavailable without running the grader", () => {
      const oversized = Array.from({ length: MAX_TRAJECTORY_ITEMS + 1 }, (_, index) => ({ id: `s${index % 7}`, firstEventIndex: index }));
      for (const id of Object.keys(TRAJECTORY_GRADERS)) {
        const result = runTrajectoryBuiltin(id, { ir: { states: oversized } });
        expect(result.status).toBe("unavailable");
        expect(result.message).toContain(`states exceeds ${MAX_TRAJECTORY_ITEMS} items`);
      }
    });

    it("does not treat a long preprocessed toolCalls list as an oversized IR", () => {
      const toolCalls = Array.from({ length: MAX_TRAJECTORY_ITEMS + 1 }, () => ({ name: "bash", success: true }));
      const result = runTrajectoryBuiltin("skill-constraint-coverage", { toolCalls }, { constraints: [{ id: "uses-bash", pattern: "bash" }] });
      expect(result.value).toBe(1);
    });

    it("stays cheap at the maximum supported trajectory size", () => {
      const n = MAX_TRAJECTORY_ITEMS;
      const bigStates = Array.from({ length: n }, (_, index) => ({ id: `s${index % 11}`, firstEventIndex: index }));
      const bigEvents = Array.from({ length: n }, (_, index) => ({ index, kind: index % 3 === 0 ? "state_change" : "tool_call", ref: `s${index % 11}` }));
      const trace = { ir: { states: bigStates, events: bigEvents, reference: { states: bigStates.slice(0, 200).map(state => state.id) } } };
      const started = performance.now();
      for (const id of Object.keys(TRAJECTORY_GRADERS)) {
        expect(runTrajectoryBuiltin(id, trace).status).not.toBe("error");
      }
      expect(performance.now() - started).toBeLessThan(5000);
    });
  });

  // --- GRADER_VERSION ---
});
