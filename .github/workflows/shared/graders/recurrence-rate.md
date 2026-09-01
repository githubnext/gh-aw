---
graders:
  # Recurrence Quantification Analysis (RQA) recurrence rate (RR): the overall
  # density of recurrent points in the state-recurrence matrix built from the
  # canonical state/event sequence, excluding the line of identity (i != j).
  # RR captures how much of the run revisits previously-seen states in any
  # form. Lower is better: less time spent in previously-visited states.
  recurrence-rate:
    name: Recurrence Rate (RQA RR)
    unit: ratio
    direction: lower_is_better
    min: 0.0
    max: 1.0
    script: |
      const isRecord = value => value !== null && typeof value === "object" && !Array.isArray(value);
      const candidates = [
        trace.trajectoryIR,
        trace.trajectoryIr,
        trace.ir,
        isRecord(trace.agentOutput) ? trace.agentOutput.trajectoryIR : null,
        isRecord(trace.agentOutput) ? trace.agentOutput.trajectoryIr : null,
        isRecord(trace.agentOutput) ? trace.agentOutput.trajectory : null,
        isRecord(trace.agentOutput) ? trace.agentOutput : null,
      ].filter(isRecord);

      let states = [];
      let events = [];
      for (const candidate of candidates) {
        if (Array.isArray(candidate.states)) {
          states = candidate.states;
          events = Array.isArray(candidate.events) ? candidate.events : [];
          break;
        }
      }

      const eventOrder = new Map();
      for (let i = 0; i < events.length; i += 1) {
        const event = events[i];
        if (!isRecord(event)) continue;
        const index = Number(event.index);
        if (Number.isFinite(index)) eventOrder.set(index, i);
      }

      // Build the ordered canonical state-id sequence (one entry per visit,
      // in execution order), same normalization as recurrence-determinism.
      const ordered = states.map((state, position) => {
        if (typeof state === "string") return { id: state, order: position };
        if (!isRecord(state) || typeof state.id !== "string" || state.id === "") return null;
        const firstEventIndex = Number(state.firstEventIndex);
        const hasFirstEventIndex = Number.isFinite(firstEventIndex);
        const eventPosition = hasFirstEventIndex && eventOrder.has(firstEventIndex) ? eventOrder.get(firstEventIndex) : null;
        const order = eventPosition !== null ? eventPosition : hasFirstEventIndex ? firstEventIndex : position;
        return { id: state.id, order };
      }).filter(entry => entry !== null).sort((a, b) => a.order - b.order);

      const sequence = ordered.map(entry => entry.id);
      const n = sequence.length;
      if (n < 2) {
        return { value: 0, unit: "ratio", passed: null, message: "not applicable: fewer than two canonical state visits" };
      }

      // RR = recurrentPoints / (n * (n - 1)), where recurrentPoints counts all
      // ordered pairs (i, j) with i != j and sequence[i] === sequence[j].
      const counts = new Map();
      for (const id of sequence) counts.set(id, (counts.get(id) || 0) + 1);

      let recurrentPoints = 0;
      for (const count of counts.values()) {
        if (count > 1) recurrentPoints += count * (count - 1);
      }

      const possiblePoints = n * (n - 1);
      const rr = helpers.ratio(recurrentPoints, possiblePoints);
      const repeatedStates = [...counts.entries()]
        .filter(([, count]) => count > 1)
        .sort((a, b) => b[1] - a[1])
        .slice(0, 5)
        .map(([id, count]) => `${id}:${count}`);

      return {
        value: rr,
        unit: "ratio",
        details: `visits=${n} recurrentPoints=${recurrentPoints} possiblePoints=${possiblePoints}${repeatedStates.length === 0 ? "" : `; repeated states (id:count): ${repeatedStates.join(", ")}`}`,
      };
---

<!--
recurrence-rate computes the RQA RR metric over the canonical state sequence:
the density of recurrent (i, j) pairs in the state-recurrence matrix where
i != j and state_i == state_j. Unlike recurrence-determinism (diagonal
structure) and recurrence-laminarity / recurrence-trapping-time (vertical
structure), RR is the base recurrence density itself: the aggregate share of
the run spent revisiting any previously-seen canonical state.
-->
