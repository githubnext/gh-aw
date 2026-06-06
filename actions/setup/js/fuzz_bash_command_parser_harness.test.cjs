// @ts-check
/**
 * Fuzz / property-based tests for bash_command_parser.cjs
 *
 * Validates security invariants and correctness properties:
 *
 *  Security invariants
 *    - Operators inside quoted strings are never treated as separators
 *    - The parser never throws on arbitrary / malformed input
 *    - Empty / whitespace-only input always yields empty arrays (safe default)
 *
 *  Correctness properties
 *    - Known-good pipeline patterns split into the expected number of segments
 *    - Command names extracted from staged pipelines match the expected identifiers
 *    - Deduplication: the same command appearing multiple times is returned once
 *    - All operators (&&, ||, |, ;) produce consistent splitting behaviour
 *    - Env-var assignments are always skipped before the command name
 *    - Redirection-only segments yield null from extractCommandName
 *
 *  Exhaustive operator × quoting matrix
 *    - For every pipeline operator, for both quote styles, the operator is never
 *      treated as a separator (safe inside quotes)
 */

import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { testSplitOnPipelineOperators, testExtractCommandName, testExtractCommandNamesFromPipeline, quotedOperatorIsNotSplit, noThrowInvariant, emptyInputYieldsEmptyArrays } = require("./fuzz_bash_command_parser_harness.cjs");

// ─────────────────────────────────────────────────────────────────────────────
// Fixtures
// ─────────────────────────────────────────────────────────────────────────────

/** Pipeline operators that should split segments */
const OPERATORS = ["&&", "||", "|", ";"];

/** Commands that should be recognised as safe (typically in the workflow allow-list) */
const SAFE_COMMANDS = ["ls", "cat", "echo", "grep", "wc", "find", "jq", "printf", "pwd", "date", "head", "tail", "sort", "uniq", "yq", "safeoutputs", "gh", "git"];

/** Common shell flags and arguments that may appear after a command */
const COMMON_ARGS = ["-la", "-n", "-r", "-e", "2>/dev/null", ">/dev/null", "2>&1", "--help", "/tmp/file.json", "'hello world'", '"hello world"'];

// ─────────────────────────────────────────────────────────────────────────────
// Security invariant: operators inside quotes are never separators
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – security: quoted operators not split", () => {
  for (const op of OPERATORS) {
    it(`should not split on ${op} inside single quotes`, () => {
      expect(quotedOperatorIsNotSplit(op)).toBe(true);
    });

    it(`should not split '${op}' when embedded in longer quoted string`, () => {
      const result = testSplitOnPipelineOperators(`echo 'prefix${op}suffix'`);
      expect(result.error).toBeNull();
      expect(result.segments).toHaveLength(1);
    });

    it(`should not split on ${op} inside double quotes`, () => {
      const result = testSplitOnPipelineOperators(`echo "prefix${op}suffix"`);
      expect(result.error).toBeNull();
      expect(result.segments).toHaveLength(1);
    });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Security invariant: no-throw on arbitrary input
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – security: no-throw on arbitrary input", () => {
  const arbitraryInputs = [
    "",
    "   ",
    "&&",
    "||",
    "|",
    ";",
    "&&&&",
    "||||",
    ";;;",
    "&&||&&",
    "'unclosed single quote",
    '"unclosed double quote',
    "$(unclosed subshell",
    "$((arithmetic))",
    "\\",
    "\n\r\t",
    "cmd\x00null",
    "a".repeat(10000),
    "'".repeat(100),
    '"'.repeat(100),
    "$($($(nested))))",
    "2>/dev/null",
    ">file",
    "<file",
    "{ echo hi; }",
    "! ls",
    "FOO=bar BAR=baz",
  ];

  for (const input of arbitraryInputs) {
    it(`should not throw for input: ${JSON.stringify(input).slice(0, 60)}`, () => {
      expect(noThrowInvariant(input)).toBe(true);
    });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Security invariant: empty / whitespace → empty arrays (safe default)
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – security: empty input yields empty arrays", () => {
  for (const input of ["", "   ", "\t", "\n", "\r\n", "  \t  \n  "]) {
    it(`should return empty arrays for ${JSON.stringify(input)}`, () => {
      expect(emptyInputYieldsEmptyArrays(input)).toBe(true);
    });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Correctness: known-good split counts
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – correctness: expected segment counts", () => {
  const cases = [
    { input: "ls /tmp", count: 1, desc: "single command" },
    { input: "ls && cat file", count: 2, desc: "&& two commands" },
    { input: "ls || echo fallback", count: 2, desc: "|| two commands" },
    { input: "ls | grep x", count: 2, desc: "| pipe two commands" },
    { input: "echo a; echo b", count: 2, desc: "; sequential two commands" },
    { input: "a && b && c", count: 3, desc: "three &&-chained commands" },
    { input: "a || b || c", count: 3, desc: "three ||-chained commands" },
    { input: "a | b | c", count: 3, desc: "three pipe-chained commands" },
    { input: "a; b; c", count: 3, desc: "three semicolon-separated commands" },
    { input: "a && b || c", count: 3, desc: "mixed && and ||" },
    { input: "a | b && c || d", count: 4, desc: "four-stage mixed pipeline" },
  ];

  for (const { input, count, desc } of cases) {
    it(`splits "${input}" into ${count} segment(s) (${desc})`, () => {
      const { segments, error } = testSplitOnPipelineOperators(input);
      expect(error).toBeNull();
      expect(segments).toHaveLength(count);
    });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Correctness: command name extraction across all safe commands
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – correctness: extractCommandName for safe commands", () => {
  for (const cmd of SAFE_COMMANDS) {
    it(`extracts "${cmd}" as the command name from "${cmd} ...args..."`, () => {
      const { name, error } = testExtractCommandName(`${cmd} --some-flag /some/path`);
      expect(error).toBeNull();
      expect(name).toBe(cmd);
    });

    it(`extracts "${cmd}" as the command name with a redirection suffix`, () => {
      const { name, error } = testExtractCommandName(`${cmd} /path 2>/dev/null`);
      expect(error).toBeNull();
      expect(name).toBe(cmd);
    });
  }

  it("extracts command after a single env-var assignment", () => {
    for (const cmd of SAFE_COMMANDS.slice(0, 5)) {
      const { name, error } = testExtractCommandName(`FOO=bar ${cmd} args`);
      expect(error).toBeNull();
      expect(name).toBe(cmd);
    }
  });

  it("extracts command after multiple env-var assignments", () => {
    for (const cmd of SAFE_COMMANDS.slice(0, 3)) {
      const { name, error } = testExtractCommandName(`A=1 B=2 C=3 ${cmd}`);
      expect(error).toBeNull();
      expect(name).toBe(cmd);
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Correctness: pipeline name extraction – expected sets
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – correctness: extractCommandNamesFromPipeline", () => {
  const cases = [
    {
      input: "ls && cat file",
      expected: ["ls", "cat"],
      desc: "two commands via &&",
    },
    {
      input: "ls || echo fallback",
      expected: ["ls", "echo"],
      desc: "two commands via ||",
    },
    {
      input: "grep pattern /tmp | wc -l",
      expected: ["grep", "wc"],
      desc: "two commands via |",
    },
    {
      input: "echo a; date; pwd",
      expected: ["echo", "date", "pwd"],
      desc: "three commands via ;",
    },
    {
      input: 'ls /tmp 2>/dev/null && echo "---" && cat file.json 2>/dev/null || echo "not found"',
      expected: ["ls", "echo", "cat"],
      desc: "GEO optimizer command 1",
    },
    {
      input: 'safeoutputs missing_data --help 2>/dev/null || echo "unavailable"',
      expected: ["safeoutputs", "echo"],
      desc: "GEO optimizer command 2",
    },
    {
      input: "pwd && ls -la && safeoutputs --help && printf '%s\\n' done",
      expected: ["pwd", "ls", "safeoutputs", "printf"],
      desc: "GEO optimizer command 3",
    },
  ];

  for (const { input, expected, desc } of cases) {
    it(`extracts [${expected.join(", ")}] from "${input.slice(0, 60)}..." (${desc})`, () => {
      const { names, error } = testExtractCommandNamesFromPipeline(input);
      expect(error).toBeNull();
      expect(names).toEqual(expected);
    });
  }

  it("deduplicates commands that appear multiple times", () => {
    for (const cmd of SAFE_COMMANDS.slice(0, 4)) {
      const input = `${cmd} && ${cmd} && ${cmd}`;
      const { names, error } = testExtractCommandNamesFromPipeline(input);
      expect(error).toBeNull();
      expect(names).toEqual([cmd]);
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Correctness: env-var assignments always skipped in pipelines
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – correctness: env-var assignments skipped in pipelines", () => {
  for (const cmd of SAFE_COMMANDS.slice(0, 6)) {
    it(`skips leading env-var assignment before "${cmd}" in a pipeline`, () => {
      const { names, error } = testExtractCommandNamesFromPipeline(`FOO=bar ${cmd} /path && echo done`);
      expect(error).toBeNull();
      expect(names).toContain(cmd);
      expect(names).toContain("echo");
    });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Correctness: redirection-only segments yield no command name
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – correctness: redirection-only segments", () => {
  const redirections = [">file.txt", "<input.txt", "2>/dev/null", "2>&1", ">/dev/null"];

  for (const redir of redirections) {
    it(`extractCommandName returns null for "${redir}"`, () => {
      const { name, error } = testExtractCommandName(redir);
      expect(error).toBeNull();
      expect(name).toBeNull();
    });
  }
});

// ─────────────────────────────────────────────────────────────────────────────
// Exhaustive operator × command matrix: every operator splits every command pair
// ─────────────────────────────────────────────────────────────────────────────

describe("fuzz_bash_command_parser – exhaustive operator × command-pair matrix", () => {
  /** Build every ordered pair (a, b) of distinct commands from the first N items. */
  function generateCommandPairs(commands, limit) {
    const pairs = [];
    for (const a of commands) {
      for (const b of commands) {
        if (a !== b) pairs.push([a, b]);
      }
    }
    return limit ? pairs.slice(0, limit) : pairs;
  }

  const cmdPairs = generateCommandPairs(SAFE_COMMANDS.slice(0, 4), 8);

  for (const op of OPERATORS) {
    for (const [a, b] of cmdPairs.slice(0, 8)) {
      it(`splits "${a} ${op} ${b}" into exactly 2 segments`, () => {
        const { segments, error } = testSplitOnPipelineOperators(`${a} ${op} ${b}`);
        expect(error).toBeNull();
        expect(segments).toHaveLength(2);
        expect(segments[0]).toContain(a);
        expect(segments[1]).toContain(b);
      });

      it(`extracts [${a}, ${b}] from "${a} ${op} ${b}"`, () => {
        const { names, error } = testExtractCommandNamesFromPipeline(`${a} ${op} ${b}`);
        expect(error).toBeNull();
        expect(names).toEqual([a, b]);
      });
    }
  }
});
