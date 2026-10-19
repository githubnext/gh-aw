const { createRequire } = require("node:module");
const { readFileSync } = require("node:fs");
const path = require("node:path");

const expectedVersion = "7.0.0-dev.20260707.2";
const fields = [
  "name",
  "expression",
  "initializer",
  "declarationList",
  "body",
  "block",
  "left",
  "right",
  "operatorToken",
  "condition",
  "whenTrue",
  "whenFalse",
  "thenStatement",
  "elseStatement",
  "tryBlock",
  "catchClause",
  "finallyBlock",
  "variableDeclaration",
  "argumentExpression",
  "moduleSpecifier",
  "importClause",
  "namedBindings",
  "propertyName",
  "incrementor",
  "operand",
];
const lists = ["statements", "declarations", "arguments", "parameters", "elements", "properties", "templateSpans", "clauses"];

function main() {
  const load = createRequire(path.join(process.argv[1], "package.json"));
  if (load("@typescript/native-preview/package.json").version !== expectedVersion) {
    throw new Error("incompatible native compiler");
  }
  const { API } = load("@typescript/native-preview/unstable/sync");
  const { createVirtualFileSystem } = load("@typescript/native-preview/unstable/fs");
  const { SyntaxKind } = load("@typescript/native-preview/unstable/ast");
  const kinds = new Map(
    Object.entries(SyntaxKind)
      .filter(([name, value]) => typeof value === "number" && !name.startsWith("First") && !name.startsWith("Last"))
      .map(([name, value]) => [value, name])
  );
  const request = JSON.parse(readFileSync(0, "utf8"));
  if (!Array.isArray(request.sources) || request.sources.length > 64 || request.sources.some(source => typeof source !== "string" || source.length > 1024 * 1024)) {
    throw new Error("invalid JavaScript batch");
  }
  const filenames = request.sources.map((_, i) => `/gh-aw-taint/input_${i}.cjs`);
  const config = "/gh-aw-taint/tsconfig.json";
  const fs = createVirtualFileSystem({
    [config]: JSON.stringify({
      compilerOptions: { allowJs: true, noLib: true, noResolve: true, noEmit: true },
      files: filenames,
    }),
    ...Object.fromEntries(filenames.map((name, i) => [name, request.sources[i]])),
  });
  // Missing virtual files must never fall back to a repository tsconfig,
  // plugin, package, or real filesystem.
  const read = fs.readFile;
  fs.readFile = name => read(name) ?? null;
  const entries = fs.getAccessibleEntries;
  fs.getAccessibleEntries = name => entries(name) ?? { files: [], directories: [] };
  const api = new API({ cwd: "/gh-aw-taint", fs, runExternalCode: false });
  try {
    const program = api.updateSnapshot({ openProject: config }).getProject(config).program;
    let count;
    function project(node, source, depth = 0) {
      if (++count > 100000 || depth > 256) throw new Error("JavaScript AST limit");
      const result = { kind: kinds.get(node.kind) ?? "Unknown", line: source.getLineAndCharacterOfPosition(node.getStart(source)).line + 1 };
      if (typeof node.text === "string") result.text = node.text;
      if (node.kind === SyntaxKind.PrefixUnaryExpression && node.operator === SyntaxKind.ExclamationToken) result.text = "!";
      const included = new Set();
      for (const field of fields) {
        const child = node[field];
        if (child && typeof child.kind === "number") {
          result[field] = project(child, source, depth + 1);
          included.add(child);
        }
      }
      for (const field of lists) {
        if (node[field]) {
          result[field] = Array.from(node[field], child => {
            included.add(child);
            return project(child, source, depth + 1);
          });
        }
      }
      const other = [];
      node.forEachChild(child => {
        if (!included.has(child)) other.push(project(child, source, depth + 1));
      });
      if (other.length) result.other = other;
      return result;
    }
    const results = filenames.map(filename => {
      try {
        if (program.getSyntacticDiagnostics(filename).length) return { error: "syntax" };
        const source = program.getSourceFile(filename);
        if (!source) return { error: "missing" };
        count = 0;
        return { tree: project(source, source) };
      } catch {
        // Preserve other scripts when this file exceeds native AST limits.
        return { error: "unavailable" };
      }
    });
    process.stdout.write(JSON.stringify(results));
  } finally {
    api.close();
  }
}

try {
  main();
} catch {
  // Compiler errors can contain source snippets. The caller reports only
  // this fixed, value-free failure and keeps the analysis incomplete.
  process.stderr.write("Native TypeScript AST unavailable or source could not be parsed\n");
  process.exitCode = 1;
}
