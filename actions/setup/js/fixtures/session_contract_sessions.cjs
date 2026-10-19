// @ts-check
"use strict";

/**
 * Unified-session fixtures for the U1–U3 session-contract smoke checks.
 *
 * Each file under `session-contract/` keeps, verbatim and in order, the
 * `session.format`, `session.collection_warning`, `model.inventory` and
 * `model.resolution` records of the published `usage/aw_session.jsonl` (usage
 * artifact, conclusion job) of one run. All other records are dropped.
 *
 * - smoke-copilot-run-39181772988: Smoke Copilot on main before #70013,
 *   https://github.com/github/gh-aw/actions/runs/39181772988 —
 *   `invalid_model_inventory` for sandbox/firewall/awf-reflect.json, no
 *   `model.inventory`, main resolved `auto` -> `claude-sonnet-5.5`.
 * - smoke-copilot-run-39176941475: Smoke Copilot after #70013,
 *   https://github.com/github/gh-aw/actions/runs/39176941475 — one agent-phase
 *   `model.inventory` (copilot configured, 40 models), main resolved.
 * - smoke-copilot-routed-run-39176939097: Smoke Copilot Routed after #70013,
 *   https://github.com/github/gh-aw/actions/runs/39176939097 — one agent-phase
 *   `model.inventory`, no `model.resolution`.
 * - smoke-pi-routed-run-39178909557: Smoke Pi Routed on main before #70013,
 *   https://github.com/github/gh-aw/actions/runs/39178909557 — exact
 *   `model.resolution` for haiku-whoami and mini-whoami, none for main, and
 *   `invalid_model_inventory` for sandbox/firewall/awf-reflect.json.
 */

const path = require("path");
const { readJsonlFile } = require("../jsonl_helpers.cjs");

const DIR = path.join(__dirname, "session-contract");

/**
 * @param {string} name
 * @returns {any[]} A fresh copy of the recorded session events.
 */
function load(name) {
  return [...readJsonlFile(path.join(DIR, `${name}.jsonl`), { strict: true })];
}

module.exports = {
  /** @returns {any[]} */
  preFixCopilotSession: () => load("smoke-copilot-run-39181772988"),
  /** @returns {any[]} */
  postFixCopilotSession: () => load("smoke-copilot-run-39176941475"),
  /** @returns {any[]} */
  postFixCopilotRoutedSession: () => load("smoke-copilot-routed-run-39176939097"),
  /** @returns {any[]} */
  piRoutedSession: () => load("smoke-pi-routed-run-39178909557"),
};
