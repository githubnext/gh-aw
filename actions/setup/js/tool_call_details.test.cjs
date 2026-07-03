// @ts-check

import { describe, it, expect } from "vitest";
import { extractShellCommandFromToolData } from "./tool_call_details.cjs";

describe("extractShellCommandFromToolData", () => {
  // --- null / non-object inputs ---

  it("returns empty string for null", () => {
    expect(extractShellCommandFromToolData(null)).toBe("");
  });

  it("returns empty string for undefined", () => {
    expect(extractShellCommandFromToolData(undefined)).toBe("");
  });

  it("returns empty string for a string value", () => {
    expect(extractShellCommandFromToolData("ls -la")).toBe("");
  });

  it("returns empty string for a number value", () => {
    expect(extractShellCommandFromToolData(42)).toBe("");
  });

  it("returns empty string for an empty object", () => {
    expect(extractShellCommandFromToolData({})).toBe("");
  });

  // --- top-level string fields (priority order) ---

  it("returns trimmed value from top-level `command` field", () => {
    expect(extractShellCommandFromToolData({ command: "  echo hello  " })).toBe("echo hello");
  });

  it("returns value from top-level `input` field when `command` is absent", () => {
    expect(extractShellCommandFromToolData({ input: "cat /etc/hosts" })).toBe("cat /etc/hosts");
  });

  it("returns value from top-level `arguments` field", () => {
    expect(extractShellCommandFromToolData({ arguments: "npm install" })).toBe("npm install");
  });

  it("returns value from top-level `args` field", () => {
    expect(extractShellCommandFromToolData({ args: "go test ./..." })).toBe("go test ./...");
  });

  it("returns value from top-level `toolInput` field", () => {
    expect(extractShellCommandFromToolData({ toolInput: "make build" })).toBe("make build");
  });

  it("returns value from top-level `parameters` field", () => {
    expect(extractShellCommandFromToolData({ parameters: "pip install -r requirements.txt" })).toBe("pip install -r requirements.txt");
  });

  // --- priority ordering ---

  it("prefers `command` over `input` when both are present", () => {
    expect(extractShellCommandFromToolData({ command: "first", input: "second" })).toBe("first");
  });

  it("prefers `input` over `arguments` when `command` is absent", () => {
    expect(extractShellCommandFromToolData({ input: "second", arguments: "third" })).toBe("second");
  });

  it("skips empty `command` and falls back to `input`", () => {
    expect(extractShellCommandFromToolData({ command: "   ", input: "fallback" })).toBe("fallback");
  });

  // --- nested object candidates ---

  it("extracts `command` from a nested object in the `input` field", () => {
    expect(extractShellCommandFromToolData({ input: { command: "docker ps" } })).toBe("docker ps");
  });

  it("extracts `cmd` from a nested object in the `input` field", () => {
    expect(extractShellCommandFromToolData({ input: { cmd: "ls -la" } })).toBe("ls -la");
  });

  it("extracts `command` from a nested object in the `arguments` field", () => {
    expect(extractShellCommandFromToolData({ arguments: { command: "git status" } })).toBe("git status");
  });

  it("extracts `cmd` from a nested object in the `toolInput` field", () => {
    expect(extractShellCommandFromToolData({ toolInput: { cmd: "curl -s https://example.com" } })).toBe("curl -s https://example.com");
  });

  it("prefers nested `command` over nested `cmd`", () => {
    expect(extractShellCommandFromToolData({ input: { command: "cmd-winner", cmd: "cmd-loser" } })).toBe("cmd-winner");
  });

  // --- edge cases ---

  it("returns empty string when all fields contain only whitespace strings", () => {
    expect(extractShellCommandFromToolData({ command: " ", input: "\t" })).toBe("");
  });

  it("returns empty string when nested objects have no command/cmd field", () => {
    expect(extractShellCommandFromToolData({ input: { foo: "bar" } })).toBe("");
  });

  it("skips non-object, non-string candidates gracefully (number)", () => {
    expect(extractShellCommandFromToolData({ command: 42, input: "fallback" })).toBe("fallback");
  });

  it("skips null nested candidate and falls through to next field", () => {
    expect(extractShellCommandFromToolData({ input: null, arguments: "next" })).toBe("next");
  });
});
