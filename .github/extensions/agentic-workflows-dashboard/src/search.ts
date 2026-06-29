import type { WorkflowDefinition } from "./models.js";

export const DEFINITIONS_SEARCH_KEY = "awd-definitions-search";

export type SearchTerm = { field: "name" | "engine" | "label" | "text"; value: string };

export function parseDefinitionsSearch(query: string): SearchTerm[] {
  const terms: SearchTerm[] = [];
  const tokenRe = /(\w+):("([^"]*)"|\S+)|(\S+)/g;
  let match: RegExpExecArray | null;
  while ((match = tokenRe.exec(query)) !== null) {
    if (match[1]) {
      const field = match[1].toLowerCase();
      const rawValue = match[3] ?? match[2] ?? ""; // match[3]: content inside quotes; match[2]: unquoted value
      const value = rawValue.replace(/^"|"$/g, "").toLowerCase();
      if (field === "name" || field === "engine" || field === "label") {
        terms.push({ field, value });
      } else {
        // Unrecognized qualifier: fall back to text search for the value
        if (value) terms.push({ field: "text", value });
      }
    } else if (match[4]) {
      terms.push({ field: "text", value: match[4].toLowerCase() });
    }
  }
  return terms;
}

export function matchesDefinitionSearch(definition: WorkflowDefinition, query: string): boolean {
  if (!query.trim()) return true;
  const terms = parseDefinitionsSearch(query);
  if (terms.length === 0) return true;
  const name = (definition.workflow ?? "").toLowerCase();
  const engine = (definition.engine_id ?? "").toLowerCase();
  const labels = (definition.labels ?? []).map(l => l.toLowerCase());
  return terms.every(term => {
    if (term.field === "name") return name.includes(term.value);
    if (term.field === "engine") return engine.includes(term.value);
    if (term.field === "label") return labels.some(l => l.includes(term.value));
    return name.includes(term.value) || engine.includes(term.value) || labels.some(l => l.includes(term.value));
  });
}
