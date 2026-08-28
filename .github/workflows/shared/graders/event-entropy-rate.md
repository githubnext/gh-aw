---
graders:
  # Normalized first-order (bigram, previous-symbol-conditioned) Shannon
  # entropy rate H(X_t | X_{t-1}) of the ordered canonical event-symbol
  # sequence, projected purely from the Trajectory IR's events[]. tool_call
  # events are qualified by their ref (finer alphabet granularity); other
  # events use their kind. Normalized by log2(alphabetSize) to stay in
  # [0, 1] and remain comparable across traces with different event
  # alphabets. Measures conditional unpredictability of the event process:
  # a trace bouncing unpredictably between few event kinds can score higher
  # than one with many kinds arranged in a strict repeating cycle.
  event-entropy-rate:
    name: Event Entropy Rate
    unit: ratio
    direction: higher_is_better
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

      let events = [];
      for (const candidate of candidates) {
        if (Array.isArray(candidate.events)) {
          events = candidate.events;
          break;
        }
      }

      // Build the ordered canonical event-symbol sequence: tool_call events
      // are qualified by their ref for finer alphabet granularity, other
      // events use their kind alone.
      const ordered = events.map((event, position) => {
        if (!isRecord(event) || typeof event.kind !== "string" || event.kind === "") return null;
        const index = Number(event.index);
        const order = Number.isFinite(index) ? index : position;
        const symbol = event.kind === "tool_call" && typeof event.ref === "string" && event.ref !== "" ? `tool_call:${event.ref}` : event.kind;
        return { symbol, order };
      }).filter(entry => entry !== null).sort((a, b) => a.order - b.order);

      const sequence = ordered.map(entry => entry.symbol);
      const n = sequence.length;
      if (n < 2) {
        return { value: 0, unit: "ratio", passed: null, message: "not applicable: fewer than two events" };
      }

      const alphabet = new Set(sequence);
      const alphabetSize = alphabet.size;
      if (alphabetSize < 2) {
        return { value: 0, unit: "ratio", details: `events=${n} alphabetSize=${alphabetSize}; entropy is vacuously 0 with a single symbol` };
      }

      // H(X_t | X_{t-1}): first-order (bigram) conditional Shannon entropy
      // of the symbol sequence, computed from observed transition
      // frequencies: sum over (prev, curr) pairs of p(prev, curr) * log2(total(prev) / count(prev, curr)).
      const nextCounts = new Map();
      const priorTotals = new Map();
      for (let i = 1; i < n; i += 1) {
        const prev = sequence[i - 1];
        const curr = sequence[i];
        if (!nextCounts.has(prev)) nextCounts.set(prev, new Map());
        const next = nextCounts.get(prev);
        next.set(curr, (next.get(curr) || 0) + 1);
        priorTotals.set(prev, (priorTotals.get(prev) || 0) + 1);
      }

      const transitionCount = n - 1;
      let conditionalEntropyBits = 0;
      for (const [prev, next] of nextCounts) {
        const total = priorTotals.get(prev);
        for (const count of next.values()) {
          const p = count / transitionCount;
          conditionalEntropyBits += p * Math.log2(total / count);
        }
      }

      const maxEntropyBits = Math.log2(alphabetSize);
      const normalized = helpers.clamp(conditionalEntropyBits / maxEntropyBits, 0, 1);

      return {
        value: normalized,
        unit: "ratio",
        details: `events=${n} alphabetSize=${alphabetSize} rawEntropyBits=${conditionalEntropyBits.toFixed(4)} maxEntropyBits=${maxEntropyBits.toFixed(4)}`,
      };
---

<!--
event-entropy-rate computes the normalized first-order (bigram) Shannon
entropy rate H(X_t | X_{t-1}) of the ordered canonical event-symbol
sequence built purely from the Trajectory IR's events[] (no states,
provenance, or objectives required). It measures conditional
unpredictability of the event process rather than raw distinct-event-kind
count or revisit density: a trace bouncing unpredictably between few event
kinds scores higher than one with many kinds arranged in a strict
repeating cycle, distinguishing it from the recurrence-* family of graders
which measure revisit structure, not information-theoretic diversity.
-->
