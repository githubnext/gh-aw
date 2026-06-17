#!/usr/bin/env node

/**
 * prepare-objective-impact-pr-objective.cjs
 *
 * Enriches merged and closed-unmerged PR datasets with root issue labels and
 * objective values by using the outcome root-level resolver:
 *
 *   1. Reads merged-prs-linked.json and closed-unmerged-prs-linked.json.
 *   2. Collects all unique linked issue numbers referenced via closingIssuesReferences
 *      (the authoritative GraphQL resolver), falling back to body-extracted numbers
 *      only when the GraphQL field is empty.
 *   3. Batch-fetches labels for each unique linked issue using the GitHub GraphQL API
 *      (100 issues per query) to avoid N+1 REST calls.
 *   4. Applies the objective mapping (label_to_value) to compute objective_value and
 *      objective_labels for each PR based on its root issue labels.
 *   5. Writes:
 *      - merged-prs-with-objective.json
 *      - closed-unmerged-prs-with-objective.json
 *
 * Mirrors the resolver logic in pkg/cli/outcome_eval.go (resolveOutcomeIntent /
 * resolvePullRequestIntent) and pkg/github/label_objective_mapping.go.
 */

"use strict";

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const DATA_DIR = "/tmp/gh-aw/agent/objective-impact-report";
const MERGED_INPUT = path.join(DATA_DIR, "merged-prs-linked.json");
const CLOSED_INPUT = path.join(DATA_DIR, "closed-unmerged-prs-linked.json");
const MERGED_OUTPUT = path.join(DATA_DIR, "merged-prs-with-objective.json");
const CLOSED_OUTPUT = path.join(DATA_DIR, "closed-unmerged-prs-with-objective.json");
const MAPPING_FILE = path.join(DATA_DIR, "objective-mapping.json");

// Maximum issues to include in a single GraphQL batch query.
const GRAPHQL_BATCH_SIZE = 100;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * @param {string} filePath
 * @param {any} fallback
 * @returns {any}
 */
function readJSON(filePath, fallback) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return fallback;
  }
}

/**
 * Run a `gh` CLI command, returning stdout as a string.
 * Returns null on failure.
 * @param {string[]} args
 * @returns {string | null}
 */
function gh(args) {
  try {
    return execFileSync("gh", args, { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] }).trim();
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// Objective mapping
// ---------------------------------------------------------------------------

/**
 * Load the objective mapping from the precomputed file.
 * @param {string} filePath
 * @returns {{ label_to_value: Record<string, number>, multi_label_logic: string }}
 */
function loadObjectiveMapping(filePath) {
  const data = readJSON(filePath, {});
  const labelToValue = data.label_to_value || {};
  const logic = typeof data.multi_label_logic === "string" ? data.multi_label_logic : "max";
  return { label_to_value: labelToValue, multi_label_logic: logic };
}

/**
 * @param {string} label
 * @param {{ label_to_value: Record<string, number> }} mapping
 * @returns {boolean}
 */
function hasObjectiveLabel(label, mapping) {
  return Object.prototype.hasOwnProperty.call(mapping.label_to_value || {}, String(label).toLowerCase().trim());
}

/**
 * Compute the objective value for a set of labels.
 * Mirrors pkg/github/label_objective_mapping.go ComputeObjectiveValue.
 * @param {string[]} labels
 * @param {{ label_to_value: Record<string, number>, multi_label_logic: string }} mapping
 * @returns {number}
 */
function computeObjectiveValue(labels, mapping) {
  if (!Array.isArray(labels) || labels.length === 0) return 0;
  const lv = mapping.label_to_value || {};
  const matchingValues = [];
  for (const label of labels) {
    const normalized = String(label).toLowerCase().trim();
    if (Object.prototype.hasOwnProperty.call(lv, normalized)) {
      matchingValues.push(lv[normalized]);
    }
  }
  if (matchingValues.length === 0) return 0;
  if ((mapping.multi_label_logic || "max") === "sum") return matchingValues.reduce((a, b) => a + b, 0);
  return Math.max(...matchingValues);
}

/**
 * Return the subset of labels that have objective values.
 * Mirrors pkg/github/label_objective_mapping.go GetObjectiveLabels.
 * @param {string[]} labels
 * @param {{ label_to_value: Record<string, number> }} mapping
 * @returns {string[]}
 */
function getObjectiveLabels(labels, mapping) {
  if (!Array.isArray(labels) || labels.length === 0) return [];
  return labels.filter(label => hasObjectiveLabel(label, mapping));
}

// ---------------------------------------------------------------------------
// Root-level resolver: determine which issue numbers to use for objective lookup.
// Mirrors resolvePullRequestIntent in pkg/cli/outcome_eval.go:
//   - Use closingIssuesReferences when available (authoritative GraphQL source)
//   - Fall back to body-extracted numbers only when closingIssuesReferences is empty
// ---------------------------------------------------------------------------

/**
 * Return the canonical linked issue numbers for a PR, preferring
 * closingIssuesReferences (the proper resolver) over body text extraction.
 * @param {object} pr
 * @returns {number[]}
 */
function resolveLinkedIssueNumbers(pr) {
  // Extract numbers from the GraphQL closingIssuesReferences field.
  const graphqlNums = [];
  const refs = pr.closingIssuesReferences;
  if (refs) {
    const nodes = Array.isArray(refs.nodes) ? refs.nodes : Array.isArray(refs) ? refs : [];
    for (const node of nodes) {
      if (node && typeof node.number === "number" && node.number > 0) {
        graphqlNums.push(node.number);
      }
    }
  }

  if (graphqlNums.length > 0) {
    return [...new Set(graphqlNums)];
  }

  // Fall back to body-extracted numbers only when GraphQL resolver found nothing.
  if (Array.isArray(pr.linked_issue_numbers) && pr.linked_issue_numbers.length > 0) {
    return pr.linked_issue_numbers.filter(n => typeof n === "number" && n > 0);
  }

  return [];
}

// ---------------------------------------------------------------------------
// Batch GraphQL label fetching
// ---------------------------------------------------------------------------

/**
 * Escape a string for safe embedding inside a GraphQL string literal.
 * @param {string} value
 * @returns {string}
 */
function escapeGraphQLString(value) {
  return String(value)
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r");
}

/**
 * Build a batch GraphQL query to fetch labels for multiple issues in one call.
 * Uses field aliases (i${number}) to avoid conflicts.
 * @param {string} owner
 * @param {string} repoName
 * @param {number[]} issueNumbers
 * @returns {string}
 */
function buildBatchLabelQuery(owner, repoName, issueNumbers) {
  const escapedOwner = escapeGraphQLString(owner);
  const escapedRepo = escapeGraphQLString(repoName);
  const fields = issueNumbers
    .map(num => `  i${num}: issue(number: ${num}) { labels(first: 20) { nodes { name } } }`)
    .join("\n");
  return `query {\n  repository(owner: "${escapedOwner}", name: "${escapedRepo}") {\n${fields}\n  }\n}`;
}

/**
 * Fetch labels for a batch of issue numbers via a single GraphQL call.
 * Returns a map from issue number -> string[] of label names.
 * @param {string} owner
 * @param {string} repoName
 * @param {number[]} issueNumbers
 * @returns {Map<number, string[]>}
 */
function fetchBatchIssueLabels(owner, repoName, issueNumbers) {
  /** @type {Map<number, string[]>} */
  const result = new Map();
  if (issueNumbers.length === 0) return result;

  const query = buildBatchLabelQuery(owner, repoName, issueNumbers);
  const raw = gh(["api", "graphql", "-f", `query=${query}`]);
  if (!raw) {
    console.error(`[pr-objective] GraphQL batch query failed for ${owner}/${repoName} (${issueNumbers.length} issues)`);
    return result;
  }

  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch {
    console.error("[pr-objective] Failed to parse GraphQL response");
    return result;
  }

  const repository = parsed?.data?.repository || {};
  for (const num of issueNumbers) {
    const issueData = repository[`i${num}`];
    if (!issueData) continue;
    const nodes = issueData?.labels?.nodes || [];
    const labels = nodes.map(n => (n && typeof n.name === "string" ? n.name : "")).filter(Boolean);
    result.set(num, labels);
  }

  return result;
}

/**
 * Build a map of issue number -> labels for all linked issue numbers,
 * using batched GraphQL queries (GRAPHQL_BATCH_SIZE per call).
 * @param {string} owner
 * @param {string} repoName
 * @param {Set<number>} issueNumbers
 * @returns {Map<number, string[]>}
 */
function buildIssueLabelMap(owner, repoName, issueNumbers) {
  /** @type {Map<number, string[]>} */
  const labelMap = new Map();
  const nums = [...issueNumbers];
  for (let i = 0; i < nums.length; i += GRAPHQL_BATCH_SIZE) {
    const batch = nums.slice(i, i + GRAPHQL_BATCH_SIZE);
    const batchResult = fetchBatchIssueLabels(owner, repoName, batch);
    for (const [num, labels] of batchResult.entries()) {
      labelMap.set(num, labels);
    }
  }
  return labelMap;
}

// ---------------------------------------------------------------------------
// Main enrichment logic
// ---------------------------------------------------------------------------

/**
 * Parse "owner/repo" (or "host/owner/repo") from a repo slug.
 * Returns { owner, name } for the owner/name portion.
 * @param {string} repoSlug
 * @returns {{ owner: string, name: string } | null}
 */
function parseOwnerRepo(repoSlug) {
  if (!repoSlug) return null;
  const parts = repoSlug.split("/");
  if (parts.length === 2) return { owner: parts[0], name: parts[1] };
  if (parts.length === 3) return { owner: parts[1], name: parts[2] }; // HOST/owner/repo
  return null;
}

/**
 * Enrich a list of PRs with root issue labels and objective values.
 * @param {object[]} prs
 * @param {string} repoSlug
 * @param {{ label_to_value: Record<string, number>, multi_label_logic: string }} mapping
 * @returns {object[]}
 */
function enrichPRsWithObjective(prs, repoSlug, mapping) {
  const parsed = parseOwnerRepo(repoSlug);
  if (!parsed) {
    console.error(`[pr-objective] Cannot parse owner/repo from ${repoSlug}`);
    return prs.map(pr => ({ ...pr, objective_value: 0, objective_labels: [], root_issue_labels: [], attribution_source: "none" }));
  }

  const { owner, name } = parsed;

  // Collect all unique linked issue numbers across all PRs.
  const allLinkedNums = new Set();
  for (const pr of prs) {
    const nums = resolveLinkedIssueNumbers(pr);
    for (const n of nums) {
      allLinkedNums.add(n);
    }
  }

  console.log(`[pr-objective] Fetching labels for ${allLinkedNums.size} unique linked issues in ${owner}/${name}`);
  const labelMap = buildIssueLabelMap(owner, name, allLinkedNums);

  return prs.map(pr => {
    const linkedNums = resolveLinkedIssueNumbers(pr);

    // Determine attribution source: prefer GraphQL closing issues.
    let attributionSource = "none";
    const refs = pr.closingIssuesReferences;
    const graphqlNodes = refs && Array.isArray(refs.nodes) ? refs.nodes : Array.isArray(refs) ? refs : [];
    const graphqlNums = graphqlNodes.map(n => n?.number).filter(n => typeof n === "number" && n > 0);

    if (graphqlNums.length > 0) {
      attributionSource = "closing_issue"; // matches intent.SourceClosingIssue
    } else if (linkedNums.length > 0) {
      attributionSource = "body_text_fallback";
    }

    if (linkedNums.length === 0) {
      return { ...pr, objective_value: 0, objective_labels: [], root_issue_labels: [], root_issue_numbers: [], attribution_source: "none" };
    }

    // Aggregate labels across all linked root issues (union), using a Set for O(1) dedup.
    const rootLabelSet = new Set();
    const resolvedNums = [];
    for (const num of linkedNums) {
      const labels = labelMap.get(num);
      if (labels && labels.length > 0) {
        for (const label of labels) {
          rootLabelSet.add(label);
        }
        resolvedNums.push(num);
      }
    }
    const allRootLabels = [...rootLabelSet];

    const objectiveValue = computeObjectiveValue(allRootLabels, mapping);
    const objectiveLabels = getObjectiveLabels(allRootLabels, mapping);

    return {
      ...pr,
      root_issue_numbers: linkedNums,
      root_issue_labels: allRootLabels,
      objective_value: objectiveValue,
      objective_labels: objectiveLabels,
      attribution_source: resolvedNums.length > 0 ? attributionSource : "no_labels",
    };
  });
}

function main() {
  const repo = process.env.EXPR_GITHUB_REPOSITORY || process.env.GITHUB_REPOSITORY || "";
  if (!repo) {
    console.error("[pr-objective] EXPR_GITHUB_REPOSITORY or GITHUB_REPOSITORY is required");
    process.exit(1);
  }

  const mapping = loadObjectiveMapping(MAPPING_FILE);

  for (const { inputFile, outputFile, label } of [
    { inputFile: MERGED_INPUT, outputFile: MERGED_OUTPUT, label: "merged" },
    { inputFile: CLOSED_INPUT, outputFile: CLOSED_OUTPUT, label: "closed-unmerged" },
  ]) {
    const prs = readJSON(inputFile, []);
    if (!Array.isArray(prs) || prs.length === 0) {
      console.log(`[pr-objective] No ${label} PRs found in ${inputFile}`);
      fs.writeFileSync(outputFile, "[]\n");
      continue;
    }

    console.log(`[pr-objective] Enriching ${prs.length} ${label} PRs with objective values...`);
    const enriched = enrichPRsWithObjective(prs, repo, mapping);

    const mapped = enriched.filter(pr => (pr.objective_value || 0) > 0).length;
    const unmapped = enriched.filter(pr => !pr.root_issue_numbers || pr.root_issue_numbers.length === 0).length;
    console.log(`[pr-objective] ${label}: ${mapped} mapped, ${unmapped} with no linked issue, ${enriched.length - mapped - unmapped} with linked issue but no objective label`);

    fs.writeFileSync(outputFile, JSON.stringify(enriched, null, 2) + "\n");
    console.log(`[pr-objective] Wrote ${outputFile}`);
  }
}

main();
