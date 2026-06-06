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
        expect(splitOnPipelineOperators(relation.left)).toEqual(splitOnPipelineOperators(relation.right));
        return;
      }

      if (relation.function === "extractCommandName") {
        expect(extractCommandName(relation.left)).toBe(extractCommandName(relation.right));
        return;
      }

      expect(extractCommandNamesFromPipeline(relation.left)).toEqual(extractCommandNamesFromPipeline(relation.right));
    });
  }
});
