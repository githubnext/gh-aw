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

  // Determine the SHA to use
  let sha = null;
  
  // Priority 1: Check if SHA is provided in the item
  const firstItem = createCommitStatusItems[0];
  if (firstItem.sha) {
    sha = firstItem.sha;
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
    core.setFailed("Could not determine commit SHA for status creation");
    return;
  }

  let statusCreated = false;
  let statusUrl = "";

  try {
    // Get repository info
    const owner = context.repo.owner;
    const repo = context.repo.repo;

    // Process the first item only (respecting max: 1 default)
    const item = createCommitStatusItems[0];

    // Validate required fields
    if (!item.state) {
      core.setFailed("Commit status 'state' is required");
      return;
    }

    if (!item.description) {
      core.setFailed("Commit status 'description' is required");
      return;
    }

    // Validate state enum
    const validStates = ["error", "failure", "pending", "success"];
    if (!validStates.includes(item.state)) {
      core.setFailed(
        `Invalid commit status state: ${item.state}. Must be one of: ${validStates.join(", ")}`
      );
      return;
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
            core.setFailed(
              `Target URL domain "${hostname}" is not in the allowed domains list. Allowed domains: ${allowedDomains}`
            );
            return;
          }
        } catch (error) {
          core.setFailed(`Invalid target_url format: ${item.target_url}`);
          return;
        }
      }
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

    statusCreated = true;
    statusUrl = response.data.url || "";

    core.info(`✓ Commit status created successfully`);
    core.info(`  Status URL: ${statusUrl}`);

    // Set outputs
    core.setOutput("status_created", "true");
    core.setOutput("status_url", statusUrl);

    // Add to step summary
    await core.summary
      .addHeading("Commit Status Created", 2)
      .addRaw(`**State:** ${statusData.state}\n\n`)
      .addRaw(`**Context:** ${statusData.context}\n\n`)
      .addRaw(`**Description:** ${statusData.description}\n\n`)
      .addRaw(`**SHA:** ${sha}\n\n`)
      .addRaw(statusUrl ? `**Status URL:** ${statusUrl}\n\n` : "")
      .write();
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.setFailed(`Failed to create commit status: ${errorMessage}`);
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
