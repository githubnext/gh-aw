---
graders:
  # Recurrence Quantification Analysis (RQA) determinism (DET): the fraction
  # of recurrent points that fall on diagonal line structures (length >= 2)
  # in the state-recurrence matrix built from the canonical state/event
  # sequence, versus all recurrent points. High DET means the agent tends to
  # repeat the *same ordered subsequence* of states more than once (e.g.
  # re-running an identical multi-step probe), not just visiting the same
  # state in isolation. Lower is better: less repeated deterministic
  # structure means more novel, non-repetitive behavior.
  recurrence-determinism:
    name: Recurrence Determinism (RQA DET)
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
      // in execution order), same normalization as state-revisit-probability-rep.
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
        return { value: 0, unit: "ratio", passed: null, message: "not applicable: fewer than four canonical state visits" };
      }

      // Recurrence matrix R[i][j] = 1 iff sequence[i] === sequence[j] and i !== j
      // (the main diagonal / line of identity is excluded by definition).
      // Diagonal line minimum length for RQA DET.
      const lmin = 2;
      let totalRecurrentPoints = 0;
      let diagonalPoints = 0;
      const diagonalLineLengths = [];

      // Walk every diagonal offset d = j - i (d != 0), scanning each diagonal
      // once and grouping consecutive recurrent points into line lengths.
      for (let d = 1; d < n; d += 1) {
        let runLength = 0;
        for (let i = 0; i + d < n; i += 1) {
          const j = i + d;
          const recurrent = sequence[i] === sequence[j];
          if (recurrent) {
            totalRecurrentPoints += 2; // symmetric: (i,j) and (j,i)
            runLength += 1;
          } else {
            if (runLength > 0) diagonalLineLengths.push(runLength);
            runLength = 0;
          }
        }
        if (runLength > 0) diagonalLineLengths.push(runLength);
      }

      for (const length of diagonalLineLengths) {
        if (length >= lmin) diagonalPoints += length * 2; // symmetric contribution
      }

      if (totalRecurrentPoints === 0) {
        return {
          value: 0,
          unit: "ratio",
          details: `no recurrent state pairs found across ${n} state visits; DET is vacuously 0`,
        };
      }

      const det = helpers.ratio(diagonalPoints, totalRecurrentPoints);
      const longLines = diagonalLineLengths.filter(length => length >= lmin).sort((a, b) => b - a).slice(0, 5);
      return {
        value: det,
        unit: "ratio",
        details: `visits=${n} recurrentPoints=${totalRecurrentPoints} diagonalPoints=${diagonalPoints} lmin=${lmin}${longLines.length === 0 ? "" : `; longest diagonal lines: ${longLines.join(", ")}`}`,
      };
---

<!--
recurrence-determinism computes the RQA DET metric over the canonical state
sequence: the fraction of recurrent (i, j) pairs in the state-recurrence
matrix that belong to diagonal line structures of length >= 2, versus the
total number of recurrent pairs. It detects when an agent re-executes the
same ordered multi-step subsequence more than once (structural repetition),
which is distinct from state-revisit-probability-rep's simple revisit count
because DET specifically rewards/penalizes *ordered, repeated* structure
rather than isolated repeated states.
-->
