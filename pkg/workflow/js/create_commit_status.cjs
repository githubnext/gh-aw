// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("status_created", "");
  core.setOutput("status_url", "");

  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createCommitStatusItems = result.items.filter(item => item.type === "create_commit_status");
  if (createCommitStatusItems.length === 0) {
    core.info("No create-commit-status items found in agent output");
    return;
  }
  core.info(`Found ${createCommitStatusItems.length} create-commit-status item(s)`);

  if (isStaged) {
    await generateStagedPreview({
      title: "Create Commit Status",
      description: "The following commit statuses would be created if staged mode was disabled:",
      items: createCommitStatusItems,
      renderItem: (item, index) => {
        let content = `### Status ${index + 1}\n`;
        content += `**State:** ${item.state || "No state provided"}\n\n`;
        content += `**Description:** ${item.description || "No description provided"}\n\n`;
        if (item.context) {
          content += `**Context:** ${item.context}\n\n`;
        }
        if (item.target_url) {
          content += `**Target URL:** ${item.target_url}\n\n`;
        }
        if (item.sha) {
          content += `**SHA:** ${item.sha}\n\n`;
        }
        return content;
      },
    });
    return;
  }

  // Get default context from environment
  const defaultContext = process.env.GH_AW_COMMIT_STATUS_CONTEXT || "default";

  // Get repository info
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  // Process all items
  const createdStatuses = [];
  for (const item of createCommitStatusItems) {
    try {
      const result = await createCommitStatus(item, owner, repo, defaultContext);
      if (result) {
        createdStatuses.push(result);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to create commit status: ${errorMessage}`);
      // Continue processing other items
    }
  }

  // Set outputs based on results
  if (createdStatuses.length > 0) {
    core.setOutput("status_created", "true");
    core.setOutput("status_url", createdStatuses[0].url);

    // Add to step summary
    await core.summary
      .addHeading(`Created ${createdStatuses.length} Commit Status(es)`, 2);
    
    for (const status of createdStatuses) {
      await core.summary
        .addRaw(`**State:** ${status.state}\n\n`)
        .addRaw(`**Context:** ${status.context}\n\n`)
        .addRaw(`**Description:** ${status.description}\n\n`)
        .addRaw(`**SHA:** ${status.sha}\n\n`)
        .addRaw(status.url ? `**Status URL:** ${status.url}\n\n` : "")
        .addRaw("---\n\n");
    }
    
    await core.summary.write();
  } else {
    core.setFailed("No commit statuses were created successfully");
  }
}

/**
 * Creates a single commit status
 * @param {object} item - The commit status item
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} defaultContext - Default context string
 * @returns {Promise<object|null>} Created status info or null on failure
 */
async function createCommitStatus(item, owner, repo, defaultContext) {
  // Validate required fields
  if (!item.state) {
    core.error("Commit status 'state' is required");
    return null;
  }

  if (!item.description) {
    core.error("Commit status 'description' is required");
    return null;
  }

  // Validate state enum
  const validStates = ["error", "failure", "pending", "success"];
  if (!validStates.includes(item.state)) {
    core.error(
      `Invalid commit status state: ${item.state}. Must be one of: ${validStates.join(", ")}`
    );
    return null;
  }

  // Validate target_url against allowed domains if provided
  if (item.target_url) {
    const allowedDomains = process.env.GH_AW_COMMIT_STATUS_ALLOWED_DOMAINS;
    if (allowedDomains) {
      const domains = allowedDomains.split(",").map(d => d.trim());
      let isAllowed = false;

      try {
        const url = new URL(item.target_url);
        const hostname = url.hostname;

        // Check if the hostname matches any allowed domain
        for (const domain of domains) {
          if (domain.startsWith("*.")) {
            // Wildcard domain: *.example.com matches sub.example.com but not example.com
            const domainSuffix = domain.slice(1); // Remove the *
            if (hostname.endsWith(domainSuffix) && hostname !== domainSuffix.slice(1)) {
              isAllowed = true;
              break;
            }
          } else if (hostname === domain || hostname.endsWith("." + domain)) {
            // Exact domain or subdomain match
            isAllowed = true;
            break;
          }
        }

        if (!isAllowed) {
          core.error(
            `Target URL domain "${hostname}" is not in the allowed domains list. Allowed domains: ${allowedDomains}`
          );
          return null;
        }
      } catch (error) {
        core.error(`Invalid target_url format: ${item.target_url}`);
        return null;
      }
    }
  }

  // Determine the SHA to use
  let sha = null;
  
  // Priority 1: Check if SHA is provided in the item
  if (item.sha) {
    sha = item.sha;
    core.info(`Using SHA from agent output: ${sha}`);
  }
  
  // Priority 2: Get from pull request context
  if (!sha && context.payload?.pull_request?.head?.sha) {
    sha = context.payload.pull_request.head.sha;
    core.info(`Using SHA from pull request head: ${sha}`);
  }
  
  // Priority 3: Get from push context
  if (!sha && context.payload?.after) {
    sha = context.payload.after;
    core.info(`Using SHA from push event: ${sha}`);
  }
  
  // Priority 4: Get from current commit
  if (!sha && context.sha) {
    sha = context.sha;
    core.info(`Using SHA from workflow context: ${sha}`);
  }

  if (!sha) {
    core.error("Could not determine commit SHA for status creation");
    return null;
  }

  // Prepare the status data
  const statusData = {
    owner,
    repo,
    sha,
    state: item.state,
    description: item.description,
    context: item.context || defaultContext,
  };

  // Add optional target_url if provided
  if (item.target_url) {
    statusData.target_url = item.target_url;
  }

  core.info(`Creating commit status for SHA ${sha}...`);
  core.info(`  State: ${statusData.state}`);
  core.info(`  Context: ${statusData.context}`);
  core.info(`  Description: ${statusData.description}`);
  if (statusData.target_url) {
    core.info(`  Target URL: ${statusData.target_url}`);
  }

  // Create the commit status
  const response = await github.rest.repos.createCommitStatus(statusData);
  const statusUrl = response.data.url || "";

  core.info(`✓ Commit status created successfully`);
  core.info(`  Status URL: ${statusUrl}`);

  return {
    state: statusData.state,
    context: statusData.context,
    description: statusData.description,
    sha: sha,
    url: statusUrl,
  };
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
