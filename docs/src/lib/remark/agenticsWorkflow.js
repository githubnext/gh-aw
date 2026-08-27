// @ts-check

import { readFileSync } from "node:fs";
import { join } from "node:path";

const directivePattern = /^<!-- agentics-workflow: ([a-z0-9][a-z0-9-]*\.md) -->$/;

/**
 * Replace agentics workflow directives with code blocks sourced from a checkout.
 *
 * @param {{ sourceDirectory: string }} options
 */
export default function remarkAgenticsWorkflow(options) {
  if (!options?.sourceDirectory) {
    throw new Error("remarkAgenticsWorkflow requires a sourceDirectory");
  }

  return function transform(tree) {
    visit(tree, options.sourceDirectory);
  };
}

/**
 * @param {any} node
 * @param {string} sourceDirectory
 */
function visit(node, sourceDirectory) {
  if (!node || typeof node !== "object") return;

  if (node.type === "html" && typeof node.value === "string") {
    const match = directivePattern.exec(node.value.trim());
    if (match) {
      const filename = match[1];
      const sourcePath = join(sourceDirectory, filename);

      let workflow;
      try {
        workflow = readFileSync(sourcePath, "utf8");
      } catch (error) {
        throw new Error(`Unable to read agentics workflow ${sourcePath}`, { cause: error });
      }

      node.type = "code";
      node.lang = "aw";
      node.meta = `wrap title=".github/workflows/${filename}"`;
      node.value = workflow;
      return;
    }
  }

  if (Array.isArray(node.children)) {
    for (const child of node.children) visit(child, sourceDirectory);
  }
}
