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
