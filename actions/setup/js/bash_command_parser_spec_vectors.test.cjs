// @ts-check

import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { splitOnPipelineOperators, extractCommandName, extractCommandNamesFromPipeline } = require("./bash_command_parser.cjs");
const vectors = require("./bash_command_parser_spec_vectors.json");

describe("bash_command_parser specification vectors", () => {
  describe("splitOnPipelineOperators vectors", () => {
    for (const vector of vectors.vectors.splitOnPipelineOperators) {
      it(`${vector.id} (${vector.source})`, () => {
        expect(splitOnPipelineOperators(vector.input)).toEqual(vector.expected);
      });
    }
  });

  describe("extractCommandName vectors", () => {
    for (const vector of vectors.vectors.extractCommandName) {
      it(`${vector.id} (${vector.source})`, () => {
        expect(extractCommandName(vector.input)).toBe(vector.expected);
      });
    }
  });

  describe("extractCommandNamesFromPipeline vectors", () => {
    for (const vector of vectors.vectors.extractCommandNamesFromPipeline) {
      it(`${vector.id} (${vector.source})`, () => {
        expect(extractCommandNamesFromPipeline(vector.input)).toEqual(vector.expected);
      });
    }
  });
});

describe("bash_command_parser specification metamorphic relations", () => {
  for (const relation of vectors.metamorphic) {
    it(`${relation.id} (${relation.relation})`, () => {
      if (relation.function === "splitOnPipelineOperators") {
        const left = splitOnPipelineOperators(relation.left);
        const right = splitOnPipelineOperators(relation.right);
        expect(left).toEqual(right);
        expect(left).toEqual(relation.expected);
        return;
      }

      if (relation.function === "extractCommandName") {
        const left = extractCommandName(relation.left);
        const right = extractCommandName(relation.right);
        expect(left).toBe(right);
        expect(left).toBe(relation.expected);
        return;
      }

      const left = extractCommandNamesFromPipeline(relation.left);
      const right = extractCommandNamesFromPipeline(relation.right);
      expect(left).toEqual(right);
      expect(left).toEqual(relation.expected);
    });
  }
});

describe("bash_command_parser specification type contract", () => {
  it("splitOnPipelineOperators accepts StringLike input without throwing", () => {
    const values = [null, undefined, 0, 1, true, false, {}, [], Symbol("x")];
    for (const value of values) {
      expect(() => splitOnPipelineOperators(value)).not.toThrow();
      expect(splitOnPipelineOperators(value)).toEqual([]);
    }
  });

  it("extractCommandName accepts StringLike input without throwing", () => {
    const values = [null, undefined, 0, 1, true, false, {}, [], Symbol("x")];
    for (const value of values) {
      expect(() => extractCommandName(value)).not.toThrow();
      expect(extractCommandName(value)).toBeNull();
    }
  });

  it("extractCommandNamesFromPipeline accepts StringLike input without throwing", () => {
    const values = [null, undefined, 0, 1, true, false, {}, [], Symbol("x")];
    for (const value of values) {
      expect(() => extractCommandNamesFromPipeline(value)).not.toThrow();
      expect(extractCommandNamesFromPipeline(value)).toEqual([]);
    }
  });

  it("extractCommandName does not overflow stack with long negation/group prefixes", () => {
    const input = `${"! ".repeat(20000)}ls /tmp`;
    expect(() => extractCommandName(input)).not.toThrow();
    expect(extractCommandName(input)).toBe("ls");
  });

  it("extractCommandName handles moderate negation/group prefixes correctly", () => {
    const input = `${"! ".repeat(64)}{ { ls /tmp; } }`;
    expect(() => extractCommandName(input)).not.toThrow();
    expect(extractCommandName(input)).toBe("ls");
  });
});
