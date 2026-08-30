---
graders:
  # Fraction of precompiled, workflow-supplied behavioral constraints
  # (config.constraints: { id, description, pattern, requireSuccess }) that
  # were both exercised (regex pattern matched at least one toolCalls/actions
  # entry) and passed (all matches succeeded/were valid at issue time, unless
  # requireSuccess: false) during the run. Constraints are stable across runs
  # of the same harness/skill, unlike per-run inferred objectives. Higher is
  # better: more declared requirements exercised and satisfied.
  skill-constraint-coverage:
    name: Skill Constraint Coverage
    unit: ratio
    direction: higher_is_better
    min: 0.0
    max: 1.0
    script: |
      const isRecord = value => value !== null && typeof value === "object" && !Array.isArray(value);
      const constraints = (Array.isArray(config.constraints) ? config.constraints : []).filter(isRecord);

      if (constraints.length === 0) {
        return { value: null, unit: "ratio", passed: null, message: "not applicable: no constraints configured" };
      }

      const candidates = [
        trace.trajectoryIR,
        trace.trajectoryIr,
        trace.ir,
        isRecord(trace.agentOutput) ? trace.agentOutput.trajectoryIR : null,
        isRecord(trace.agentOutput) ? trace.agentOutput.trajectoryIr : null,
        isRecord(trace.agentOutput) ? trace.agentOutput.trajectory : null,
        isRecord(trace.agentOutput) ? trace.agentOutput : null,
      ].filter(isRecord);

      const toolCalls = (candidates.find(value => Array.isArray(value.toolCalls))?.toolCalls ?? []).filter(isRecord);
      const actions = (candidates.find(value => Array.isArray(value.actions))?.actions ?? []).filter(isRecord);

      if (toolCalls.length === 0 && actions.length === 0) {
        return { value: null, unit: "ratio", passed: null, message: "not applicable: trace lacks toolCalls/actions" };
      }

      // Project toolCalls/actions into a flat list of { text, ok } entries: text is
      // matched against each constraint's pattern, ok reflects success/validity.
      const entries = [];
      for (const call of toolCalls) {
        const name = typeof call.name === "string" ? call.name : "";
        let argsText = "";
        try {
          argsText = call.arguments === undefined ? "" : JSON.stringify(call.arguments);
        } catch {
          argsText = "";
        }
        entries.push({ text: `${name} ${argsText}`, ok: call.success !== false });
      }
      for (const action of actions) {
        const type = typeof action.type === "string" ? action.type : "";
        const target = typeof action.target === "string" ? action.target : "";
        entries.push({ text: `${type} ${target}`, ok: action.validAtIssueTime !== false });
      }

      let validConstraints = 0;
      let exercised = 0;
      let covered = 0;
      const failing = [];
      for (const constraint of constraints) {
        const patternSource = typeof constraint.pattern === "string" ? constraint.pattern : "";
        if (patternSource === "") continue;
        let regex;
        try {
          regex = new RegExp(patternSource, "i");
        } catch {
          continue;
        }
        validConstraints += 1;
        const matches = entries.filter(entry => regex.test(entry.text));
        if (matches.length === 0) continue;
        exercised += 1;
        const requireSuccess = constraint.requireSuccess !== false;
        const passedConstraint = !requireSuccess || matches.every(entry => entry.ok);
        const label = typeof constraint.id === "string" && constraint.id !== "" ? constraint.id : patternSource;
        if (passedConstraint) {
          covered += 1;
        } else if (failing.length < 5) {
          failing.push(label);
        }
      }

      if (validConstraints === 0) {
        return { value: null, unit: "ratio", passed: null, message: "not applicable: no constraints with a valid pattern configured" };
      }

      return {
        value: helpers.ratio(covered, validConstraints),
        unit: "ratio",
        details: `constraints=${validConstraints} exercised=${exercised} covered=${covered}${failing.length === 0 ? "" : `; unmet: ${failing.join(", ")}`}`,
      };
---

<!--
skill-constraint-coverage converts a precompiled, workflow-supplied list of
behavioral constraints (config.constraints: { id, description, pattern,
requireSuccess }) into a harness-improvement signal: the fraction that were
both exercised (regex pattern matched at least one toolCalls/actions entry,
matched case-insensitively against "name arguments" for tool calls and "type
target" for actions) and passed (all matches succeeded/were valid at issue
time, unless requireSuccess: false) during the run. Unlike policy-near-miss
(keyword-matched guard objectives already declared as IR objectives) or the
not-yet-implemented objective-coverage (inferred per-run objectives),
constraints here are stable across runs of the same harness/skill, making the
metric trackable over time. Reports not-applicable (passed: null) when no
constraints are configured, none have a valid pattern, or the trace lacks
toolCalls/actions.
-->
