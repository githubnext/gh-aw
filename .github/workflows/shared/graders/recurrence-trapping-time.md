---
graders:
  # Recurrence Quantification Analysis (RQA) trapping time (TT): the mean
  # length of vertical line structures (length >= 2) in the state-recurrence
  # matrix built from the canonical state/event sequence. A vertical line means
  # consecutive steps all match one fixed previously visited state (stagnation).
  # TT complements recurrence-laminarity: LAM measures how much recurrent
  # structure is vertical, while TT measures how long each vertical episode
  # lasts on average once it starts. Lower is better: shorter stagnation runs.
  recurrence-trapping-time:
    name: Recurrence Trapping Time (RQA TT)
    unit: steps
    direction: lower_is_better
    min: 0.0
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
      // in execution order), same normalization as recurrence-laminarity.
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
      if (n < 4) {
        return { value: 0, unit: "steps", passed: null, message: "not applicable: fewer than four canonical state visits" };
      }

      // Recurrence matrix R[i][j] = 1 iff sequence[i] === sequence[j] and i !== j
      // (the main diagonal / line of identity is excluded by definition).
      // Vertical line minimum length for RQA TT.
      const vmin = 2;
      const verticalLineLengths = [];

      // Scan every column j once, grouping consecutive recurrent rows i into
      // vertical line lengths. TT is the mean length among vertical lines with
      // length >= vmin.
      for (let j = 0; j < n; j += 1) {
        let runLength = 0;
        for (let i = 0; i < n; i += 1) {
          const recurrent = i !== j && sequence[i] === sequence[j];
          if (recurrent) {
            runLength += 1;
          } else {
            if (runLength > 0) verticalLineLengths.push(runLength);
            runLength = 0;
          }
        }
        if (runLength > 0) verticalLineLengths.push(runLength);
      }

      const qualifyingLines = verticalLineLengths.filter(length => length >= vmin);
      if (qualifyingLines.length === 0) {
        return {
          value: 0,
          unit: "steps",
          details: `no vertical recurrence lines (vmin=${vmin}) found across ${n} state visits; TT is vacuously 0`,
        };
      }

      let totalLength = 0;
      for (const length of qualifyingLines) totalLength += length;
      const tt = totalLength / qualifyingLines.length;
      const longestLines = [...qualifyingLines].sort((a, b) => b - a).slice(0, 5);
      return {
        value: tt,
        unit: "steps",
        details: `visits=${n} verticalLines=${qualifyingLines.length} totalVerticalLength=${totalLength} vmin=${vmin}; longest vertical lines: ${longestLines.join(", ")}`,
      };
---

<!--
recurrence-trapping-time computes the RQA TT metric over the canonical state
sequence: the mean length of vertical line structures of length >= 2 in the
state-recurrence matrix. A vertical line means consecutive steps all matched
one previously visited state, i.e. a stagnation episode. This complements
recurrence-laminarity (how much recurrent structure is vertical) by measuring
how long each vertical stagnation episode lasts once it starts.
-->
