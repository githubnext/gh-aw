// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "dispatch_workflow";

const { getErrorMessage } = require("./error_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { resolveTargetRepoConfig, parseRepoSlug, validateTargetRepo } = require("./repo_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { buildAwContext } = require("./aw_context.cjs");
const { loadTemporaryIdMapFromResolved, resolveIssueNumber, replaceTemporaryIdReferences } = require("./temporary_id.cjs");

/**
 * Main handler factory for dispatch_workflow
 * Returns a message handler function that processes individual dispatch_workflow messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedWorkflows = config.workflows || [];
  const maxCount = config.max || 1;
  const workflowFiles = config.workflow_files || {}; // Map of workflow name to file extension
  const awContextWorkflows = new Set(config.aw_context_workflows || []); // Workflows that accept aw_context input
  const githubClient = await createAuthenticatedGitHubClient(config);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);

  // Resolve the dispatch destination repository from target-repo config, falling back to context.repo
  const contextRepoSlug = `${context.repo.owner}/${context.repo.repo}`;
  const normalizedTargetRepo = (defaultTargetRepo ?? "").toString().trim();

  let resolvedRepoSlug = contextRepoSlug;
  let repo = context.repo;

  if (normalizedTargetRepo) {
    const parsedRepo = parseRepoSlug(normalizedTargetRepo);
    if (!parsedRepo) {
      core.warning(`Invalid 'target-repo' configuration value '${normalizedTargetRepo}'; falling back to workflow context repository ${contextRepoSlug}.`);
    } else {
      resolvedRepoSlug = normalizedTargetRepo;
      repo = parsedRepo;
    }
  }

  const isCrossRepoDispatch = resolvedRepoSlug !== contextRepoSlug;

  // SEC-005: Enforce cross-repository allowlist per Safe Outputs Specification §3.2.6 (SP6).
  // Default-deny: cross-repo dispatch is only permitted when an explicit allowlist is configured
  // and the resolved target repo is present in that list. Uses validateTargetRepo from
  // repo_helpers.cjs for consistent slug validation and glob-pattern matching (e.g. "org/*").
  if (isCrossRepoDispatch) {
    if (allowedRepos.size === 0) {
      throw new Error(`E004: Cross-repository dispatch to '${resolvedRepoSlug}' is not permitted. No allowlist is configured. Define 'allowed_repos' to enable cross-repository dispatch.`);
    }
    const repoValidation = validateTargetRepo(resolvedRepoSlug, contextRepoSlug, allowedRepos);
    if (!repoValidation.valid) {
      throw new Error(`E004: ${repoValidation.error}`);
    }
    core.info(`Cross-repo allowlist check passed for ${resolvedRepoSlug}`);
  }

  core.info(`Dispatch workflow configuration: max=${maxCount}`);
  if (allowedWorkflows.length > 0) {
    core.info(`Allowed workflows: ${allowedWorkflows.join(", ")}`);
  }
  if (Object.keys(workflowFiles).length > 0) {
    core.info(`Workflow files: ${JSON.stringify(workflowFiles)}`);
  }
  if (isCrossRepoDispatch) {
    core.info(`Dispatching to target repo: ${resolvedRepoSlug}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;
  let lastDispatchTime = 0;
  const isStaged = isStagedMode(config);

  // Helper function to get the default branch of the dispatch target repository
  const getDefaultBranchRef = async () => {
    // Only use the context payload's default_branch when dispatching to the caller's own repo
    if (!isCrossRepoDispatch && context.payload.repository?.default_branch) {
      return `refs/heads/${context.payload.repository.default_branch}`;
    }

    // Fall back to querying the target repository
    try {
      const { data: repoData } = await githubClient.rest.repos.get({
        owner: repo.owner,
        repo: repo.repo,
      });
      return `refs/heads/${repoData.default_branch}`;
    } catch (error) {
      core.warning(`Failed to fetch default branch: ${getErrorMessage(error)}`);
      return "refs/heads/main";
    }
  };

  // When running in a PR context, GITHUB_REF points to the merge ref (refs/pull/{PR_NUMBER}/merge)
  // which is not a valid branch ref for dispatching workflows. Instead, we need to use
  // GITHUB_HEAD_REF which contains the actual PR branch name.
  // For cross-repo dispatch (workflow_call relay), the caller's GITHUB_REF has no meaning on
  // the target repository, so we use the compiler-injected target-ref instead.
  let ref;
  if (config["target-ref"]) {
    // Compiler-injected target ref for cross-repo dispatch (workflow_call relay pattern).
    // Takes precedence over all environment variables to avoid using the caller's ref.
    ref = config["target-ref"];
    core.info(`Using configured target-ref: ${ref}`);
  } else if (process.env.GITHUB_HEAD_REF) {
    // We're in a pull_request event, use the PR branch ref
    ref = `refs/heads/${process.env.GITHUB_HEAD_REF}`;
    core.info(`Using PR branch ref: ${ref}`);
  } else if (process.env.GITHUB_REF || context.ref) {
    // Use GITHUB_REF for non-PR contexts (push, workflow_dispatch, etc.)
    ref = process.env.GITHUB_REF || context.ref;
  } else {
    // Last resort: fetch the repository's default branch
    ref = await getDefaultBranchRef();
    core.info(`Using default branch ref: ${ref}`);
  }

  /**
   * Message handler function that processes a single dispatch_workflow message
   * @param {Object} message - The dispatch_workflow message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleDispatchWorkflow(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping dispatch_workflow: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    const workflowName = message.workflow_name;

    if (!workflowName || workflowName.trim() === "") {
      core.warning("Workflow name is empty, skipping");
      return {
        success: false,
        error: "Workflow name is empty",
      };
    }

    // Validate workflow is in allowed list
    if (allowedWorkflows.length > 0 && !allowedWorkflows.includes(workflowName)) {
      const error = `Workflow "${workflowName}" is not in the allowed workflows list: ${allowedWorkflows.join(", ")}`;
      core.warning(error);
      return {
        success: false,
        error: error,
      };
    }

    try {
      // Add 5 second delay between dispatches (except for the first one)
      if (lastDispatchTime > 0) {
        const timeSinceLastDispatch = Date.now() - lastDispatchTime;
        const delayNeeded = 5000 - timeSinceLastDispatch;
        if (delayNeeded > 0) {
          core.info(`Waiting ${Math.ceil(delayNeeded / 1000)} seconds before next dispatch...`);
          await new Promise(resolve => setTimeout(resolve, delayNeeded));
        }
      }

      core.info(`Dispatching workflow: ${workflowName}`);

      // Prepare inputs - convert all values to strings as required by workflow_dispatch
      // and resolve any #temporary_id references before dispatching
      /** @type {Record<string, string>} */
      const inputs = {};
      if (message.inputs && typeof message.inputs === "object") {
        // Build a Map from resolvedTemporaryIds for use with replaceTemporaryIdReferences
        const temporaryIdMap = loadTemporaryIdMapFromResolved(resolvedTemporaryIds);

        for (const [key, value] of Object.entries(message.inputs)) {
          // Convert value to string
          let strValue;
          if (value === null || value === undefined) {
            strValue = "";
          } else if (typeof value === "object") {
            strValue = JSON.stringify(value);
          } else {
            strValue = String(value);
          }

          // Resolve temporary ID references in string input values.
          // If the entire value is a pure temporary ID (e.g. "#aw_slosync"), resolve it
          // using the full temporaryIdMap (which retains repo context) so we can emit
          // just the number for same-repo targets and "owner/repo#number" for cross-repo,
          // consistent with how replaceTemporaryIdReferences handles embedded references.
          // For text values with embedded references (e.g. "See #aw_xxx"), replace each
          // reference with its cross-repository link (owner/repo#number).
          if (strValue !== "") {
            const pureIdResult = resolveIssueNumber(strValue, temporaryIdMap);
            if (pureIdResult.wasTemporaryId) {
              if (pureIdResult.errorMessage || !pureIdResult.resolved) {
                core.warning(`Unresolved temporary ID in input "${key}": ${pureIdResult.errorMessage}`);
              } else if (pureIdResult.resolved.repo === resolvedRepoSlug) {
                strValue = String(pureIdResult.resolved.number);
              } else {
                strValue = `${pureIdResult.resolved.repo}#${pureIdResult.resolved.number}`;
              }
            } else if (temporaryIdMap.size > 0) {
              strValue = replaceTemporaryIdReferences(strValue, temporaryIdMap, resolvedRepoSlug);
            }
          }

          inputs[key] = strValue;
        }
      }

      // Inject aw_context if the target workflow declares it as an input.
      // Only workflows listed in aw_context_workflows (populated at compile time) support this.
      if (awContextWorkflows.has(workflowName)) {
        inputs["aw_context"] = JSON.stringify(buildAwContext());
      }

      // Get the workflow file extension from compile-time resolution
      const extension = workflowFiles[workflowName];
      if (!extension) {
        return {
          success: false,
          error: `Workflow "${workflowName}" file extension not found in configuration. This workflow may not have been validated at compile time.`,
        };
      }

      const workflowFile = `${workflowName}${extension}`;
      core.info(`Dispatching workflow: ${workflowFile}`);

      // If in staged mode, preview the dispatch without executing it
      if (isStaged) {
        logStagedPreviewInfo(`Would dispatch workflow: ${workflowFile} in ${resolvedRepoSlug} with ref: ${ref}`);
        return {
          success: true,
          staged: true,
          workflow_name: workflowName,
          inputs: inputs,
        };
      }

      // Dispatch the workflow using the resolved file.
      // Request return_run_details for newer GitHub API support; fall back without it
      // for older GitHub Enterprise Server deployments that don't support the parameter.
      /** @type {{ data: { workflow_run_id?: number } }} */
      let response;
      try {
        response = await githubClient.rest.actions.createWorkflowDispatch({
          owner: repo.owner,
          repo: repo.repo,
          workflow_id: workflowFile,
          ref: ref,
          inputs: inputs,
          return_run_details: true,
        });
      } catch (dispatchError) {
        /** @type {any} */
        const err = dispatchError;
        const status = err && typeof err === "object" ? err.status : undefined;
        const dispatchErrMessage = typeof err?.response?.data?.message === "string" ? err.response.data.message : String(dispatchError);

        const isValidationStatus = status === 400 || status === 422;
        const mentionsReturnRunDetails = typeof dispatchErrMessage === "string" && dispatchErrMessage.toLowerCase().includes("return_run_details");

        if (isValidationStatus && mentionsReturnRunDetails) {
          core.info("Workflow dispatch failed due to unsupported 'return_run_details' parameter; retrying without it for GitHub Enterprise compatibility.");
          response = await githubClient.rest.actions.createWorkflowDispatch({
            owner: repo.owner,
            repo: repo.repo,
            workflow_id: workflowFile,
            ref: ref,
            inputs: inputs,
          });
        } else {
          throw err;
        }
      }

      const runId = response && response.data ? response.data.workflow_run_id : undefined;
      if (runId) {
        core.info(`✓ Successfully dispatched workflow: ${workflowFile} (run ID: ${runId})`);
      } else {
        core.info(`✓ Successfully dispatched workflow: ${workflowFile}`);
      }

      // Record the time of this dispatch for rate limiting
      lastDispatchTime = Date.now();

      return {
        success: true,
        workflow_name: workflowName,
        inputs: inputs,
        run_id: runId,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to dispatch workflow "${workflowName}": ${errorMessage}`);

      return {
        success: false,
        error: `Failed to dispatch workflow "${workflowName}": ${errorMessage}`,
      };
    }
  };
}

module.exports = { main };
