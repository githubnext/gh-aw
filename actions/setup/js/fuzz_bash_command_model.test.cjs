// @ts-check

import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { generateShellProgram, checkShellProgramModel } = require("./fuzz_bash_command_model.cjs");

describe("model-based Bash parser fuzz generator", () => {
  it("is deterministic and bounded on empty and oversized inputs", async () => {
    expect(generateShellProgram([], "")).toEqual(generateShellProgram([], ""));
    expect((await checkShellProgramModel([], "")).names).toEqual([]);
    const bytes = Array(4096).fill(255);
    const payload = "'\"#\n\r\\; && $(python3) ".repeat(1000);
    const model = await checkShellProgramModel(bytes, payload);
    expect(model).toEqual(generateShellProgram(bytes, payload));
    expect(model.commandNames.length).toBeLessThanOrEqual(8);
    expect(model.source.length).toBeLessThan(5000);
  });

  it("checks all byte choices with reproducible generated programs", async () => {
    const names = new Set();
    const segments = new Set();
    for (let seed = 0; seed < 256; seed++) {
      // Fixed LCG: no Math.random, clock, shell execution or external services.
      let state = seed;
      const bytes = Array.from({ length: 256 }, () => {
        state = (Math.imul(state, 1664525) + 1013904223) >>> 0;
        return state >>> 24;
      });
      const model = await checkShellProgramModel(bytes, `seed=${seed} ' " \r # ; | & \\`);
      for (const name of model.names) names.add(name);
      for (const segment of model.segments) segments.add(segment);
    }
    expect([...names].sort()).toEqual(["cat", "echo", "git", "jq", "python3", "wc"]);
    expect(segments.size).toBeGreaterThan(500);
  });

  it("does not treat command text inside arbitrary comments as executable", async () => {
    for (const payload of ['"unclosed', "'unclosed", "$(python3)", "\rpython3; #", "\\\npython3", "\0\u00a0\v"]) {
      const model = await checkShellProgramModel([1], payload);
      expect(model.names).toEqual(["echo"]);
    }
  });

  it("deduplicates in first-occurrence order and denies a forbidden command at every position", async () => {
    for (let position = 0; position < 3; position++) {
      const bytes = [3];
      for (let i = 0; i < 3; i++) bytes.push(i === position ? 5 : 0, 0, 0, 0, 0, 4, 0);
      const model = await checkShellProgramModel(bytes, "python3 is only a comment here");
      expect(model.commandNames).toEqual(Array.from({ length: 3 }, (_, i) => (i === position ? "python3" : "echo")));
      expect(model.names).toEqual(position === 0 ? ["python3", "echo"] : ["echo", "python3"]);
    }
  });

  it("exercises each supported operator and unsupported suffix", async () => {
    const operators = [" && ", " || ", " | ", " |& ", "; ", " & ", "\n", "\r\n"];
    for (let i = 0; i < 11; i++) {
      // Two commands, no arguments; the final byte selects the unsafe suffix.
      const bytes = [2, 0, 0, 0, 0, 0, i % 8, 0, 1, 0, 0, 0, 0, 0, 0, i];
      const model = await checkShellProgramModel(bytes, "ignored");
      expect(model.names).toEqual(["echo", "cat"]);
      expect(model.segments).toEqual(["echo", "cat"]);
      expect(model.source).toContain(`echo${operators[i % 8]}#`);
    }
  });
});
