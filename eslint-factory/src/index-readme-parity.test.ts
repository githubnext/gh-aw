import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import ts from "typescript";
import { describe, expect, it } from "vitest";

describe("ESLint rule documentation", () => {
  it("documents every exported rule exactly once", () => {
    const readme = readFileSync(resolve(__dirname, "../README.md"), "utf8");
    const index = readFileSync(resolve(__dirname, "index.ts"), "utf8");
    const documentedRuleNames = [...readme.matchAll(/^### `([^`]+)`$/gm)].map(match => match[1]);
    const sourceFile = ts.createSourceFile("index.ts", index, ts.ScriptTarget.Latest, true);
    const pluginDeclaration = sourceFile.statements
      .filter(ts.isVariableStatement)
      .flatMap(statement => statement.declarationList.declarations)
      .find(declaration => declaration.name.getText(sourceFile) === "plugin");
    const plugin = pluginDeclaration?.initializer;
    const rulesProperty = plugin && ts.isObjectLiteralExpression(plugin) ? plugin.properties.find(property => ts.isPropertyAssignment(property) && property.name.getText(sourceFile) === "rules") : undefined;
    const rules = rulesProperty?.initializer;
    const exportedRuleNames = rules && ts.isObjectLiteralExpression(rules) ? rules.properties.filter(ts.isPropertyAssignment).map(property => property.name.getText(sourceFile).replace(/^"|"$/g, "")) : [];

    expect(rules).toBeDefined();
    expect(documentedRuleNames).toHaveLength(exportedRuleNames.length);
    expect([...documentedRuleNames].sort()).toEqual([...exportedRuleNames].sort());
  });
});
