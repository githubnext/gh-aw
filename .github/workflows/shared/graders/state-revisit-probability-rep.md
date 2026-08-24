---
graders:
  # Measures the fraction of visited canonical states that are redundant revisits:
  # (visited states - distinct states) / visited states. Lower is better.
  state-revisit-probability-rep:
    name: State Revisit Probability REP
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

      const ordered = states.map((state, position) => {
        if (typeof state === "string") return { id: state, order: position, eventIndex: null };
        if (!isRecord(state) || typeof state.id !== "string" || state.id === "") return null;
        const firstEventIndex = Number(state.firstEventIndex);
        const hasFirstEventIndex = Number.isFinite(firstEventIndex);
        const eventPosition = hasFirstEventIndex && eventOrder.has(firstEventIndex) ? eventOrder.get(firstEventIndex) : null;
        const order = eventPosition !== null ? eventPosition : hasFirstEventIndex ? firstEventIndex : position;
        return { id: state.id, order, eventIndex: hasFirstEventIndex ? firstEventIndex : null };
      }).filter(entry => entry !== null).sort((a, b) => a.order - b.order);

      const visited = ordered.length;
      if (visited < 2) {
        return { value: 0, unit: "ratio", passed: null, message: "not applicable: fewer than two state visits" };
      }

      const seen = new Set();
      const repeated = [];
      for (const entry of ordered) {
        if (seen.has(entry.id) && repeated.length < 5) {
          repeated.push(entry.eventIndex === null ? entry.id : `${entry.id} at event ${entry.eventIndex}`);
        }
        seen.add(entry.id);
      }
      const distinct = seen.size;
      const revisits = visited - distinct;
      return {
        value: helpers.ratio(revisits, visited),
        unit: "ratio",
        details: `visited=${visited} distinct=${distinct} revisits=${revisits}${repeated.length === 0 ? "" : `; repeated states: ${repeated.join(", ")}`}`,
      };
---

<!--
state-revisit-probability-rep measures structural exploration redundancy as
(visited states - distinct states) / visited states over canonical state visits.
It catches wasted exploration where an agent returns to known behavioral states
anywhere in the trajectory, not just adjacent loops.
-->
