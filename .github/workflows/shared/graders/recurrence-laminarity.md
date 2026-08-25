---
graders:
  # Recurrence Quantification Analysis (RQA) laminarity (LAM): the fraction
  # of recurrent points that fall on vertical line structures (length >= 2)
  # in the state-recurrence matrix built from the canonical state/event
  # sequence, versus all recurrent points. A vertical line in column j means
  # the agent stayed on states that all match one earlier fixed state across
  # several consecutive steps — i.e. it stagnated around that state instead
  # of moving on. This is distinct from recurrence-determinism (ordered,
  # repeated multi-step subsequences) and from
  # state-revisit-probability-rep (raw revisit count). Lower is better:
  # less laminar structure means less stagnation.
  recurrence-laminarity:
    name: Recurrence Laminarity (RQA LAM)
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
      if (n < 4) {
        return { value: 0, unit: "ratio", passed: null, message: "not applicable: fewer than four canonical state visits" };
      }

      // Recurrence matrix R[i][j] = 1 iff sequence[i] === sequence[j] and i !== j
      // (the main diagonal / line of identity is excluded by definition).
      // Vertical line minimum length for RQA LAM.
      const vmin = 2;
      let totalRecurrentPoints = 0;
      let verticalPoints = 0;
      const verticalLineLengths = [];

      // Scan every column j once, grouping consecutive recurrent rows i into
      // vertical line lengths. Scanning all columns visits every *ordered*
      // recurrent pair exactly once, so each unordered pair {i, j} counts
      // twice — the same ordered-pair convention recurrence-determinism uses,
      // and it applies to both the numerator and the denominator.
      for (let j = 0; j < n; j += 1) {
        let runLength = 0;
        for (let i = 0; i < n; i += 1) {
          const recurrent = i !== j && sequence[i] === sequence[j];
          if (recurrent) {
            totalRecurrentPoints += 1;
            runLength += 1;
          } else {
            if (runLength > 0) verticalLineLengths.push(runLength);
            runLength = 0;
          }
        }
        if (runLength > 0) verticalLineLengths.push(runLength);
      }

      for (const length of verticalLineLengths) {
        if (length >= vmin) verticalPoints += length;
      }

      if (totalRecurrentPoints === 0) {
        return {
          value: 0,
          unit: "ratio",
          details: `no recurrent state pairs found across ${n} state visits; LAM is vacuously 0`,
        };
      }

      const lam = helpers.ratio(verticalPoints, totalRecurrentPoints);
      const longLines = verticalLineLengths.filter(length => length >= vmin).sort((a, b) => b - a).slice(0, 5);
      return {
        value: lam,
        unit: "ratio",
        details: `visits=${n} recurrentPoints=${totalRecurrentPoints} verticalPoints=${verticalPoints} vmin=${vmin}${longLines.length === 0 ? "" : `; longest vertical lines: ${longLines.join(", ")}`}`,
      };
---

<!--
recurrence-laminarity computes the RQA LAM metric over the canonical state
sequence: the fraction of recurrent (i, j) pairs in the state-recurrence
matrix that belong to vertical line structures of length >= 2, versus the
total number of recurrent pairs. A vertical line means several consecutive
steps all matched one earlier fixed state, which identifies stagnation
(the agent stopped making progress and kept re-entering the same state).
It complements recurrence-determinism, which measures ordered repeated
subsequences (diagonal structure), and state-revisit-probability-rep,
which only counts how many visits were revisits.
-->
