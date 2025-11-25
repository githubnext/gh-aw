// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

/**
 * Extract issue number from a number or URL string
 * @param {number|string} issueRef - Issue number or URL (e.g., 42 or "https://github.com/owner/repo/issues/42")
 * @returns {{number: number, owner?: string, repo?: string} | null}
 */
function parseIssueReference(issueRef) {
  if (typeof issueRef === "number") {
    return { number: issueRef };
  }

  if (typeof issueRef === "string") {
    // Try parsing as a number first
    const parsed = parseInt(issueRef, 10);
    if (!isNaN(parsed) && parsed > 0) {
      return { number: parsed };
    }

    // Try parsing as a GitHub URL
    // Matches: https://github.com/owner/repo/issues/123
    const urlMatch = issueRef.match(/github\.com\/([^/]+)\/([^/]+)\/issues\/(\d+)/);
    if (urlMatch) {
      return {
        owner: urlMatch[1],
        repo: urlMatch[2],
        number: parseInt(urlMatch[3], 10),
      };
    }
  }

  return null;
}

/**
 * Get the node ID for an issue using GraphQL
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<string|null>} - Issue node ID or null if not found
 */
async function getIssueNodeId(owner, repo, issueNumber) {
  const query = `
    query($owner: String!, $repo: String!, $number: Int!) {
      repository(owner: $owner, name: $repo) {
        issue(number: $number) {
          id
        }
      }
    }
  `;

  try {
    const result = await github.graphql(query, {
      owner,
      repo,
      number: issueNumber,
    });
    return result.repository?.issue?.id || null;
  } catch (error) {
    core.error(`Failed to get node ID for issue #${issueNumber}: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
}

/**
 * Add a sub-issue relationship (child becomes sub-issue of parent)
 * Uses the addSubIssue GraphQL mutation
 * @param {string} parentId - Parent issue node ID
 * @param {string} childId - Child issue node ID
 * @returns {Promise<boolean>} - Success status
 */
async function addSubIssue(parentId, childId) {
  const mutation = `
    mutation($parentId: ID!, $childId: ID!) {
      addSubIssue(input: {issueId: $parentId, subIssueId: $childId}) {
        issue {
          id
          number
        }
        subIssue {
          id
          number
        }
      }
    }
  `;

  try {
    await github.graphql(mutation, {
      parentId,
      childId,
    });
    return true;
  } catch (error) {
    core.error(`Failed to add sub-issue relationship: ${error instanceof Error ? error.message : String(error)}`);
    return false;
  }
}

/**
 * Add an issue dependency (blocking relationship) using the addIssueDependency GraphQL mutation
 * @param {string} dependsOnId - The issue ID that is blocked (depends on the blocking issue)
 * @param {string} dependencyId - The issue ID that is blocking
 * @returns {Promise<boolean>} - Success status
 */
async function addIssueDependency(dependsOnId, dependencyId) {
  const mutation = `
    mutation($dependsOnId: ID!, $dependencyId: ID!) {
      addIssueDependency(input: {issueId: $dependsOnId, dependsOnIssueId: $dependencyId}) {
        issue {
          id
          number
        }
      }
    }
  `;

  try {
    await github.graphql(mutation, {
      dependsOnId,
      dependencyId,
    });
    return true;
  } catch (error) {
    core.error(`Failed to add blocking relationship: ${error instanceof Error ? error.message : String(error)}`);
    return false;
  }
}

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all link_issues items
  const linkItems = result.items.filter(item => item.type === "link_issues");
  if (linkItems.length === 0) {
    core.info("No link_issues items found in agent output");
    return;
  }

  core.info(`Found ${linkItems.length} link_issues item(s)`);

  // Handle staged mode
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Link Issues",
      description: "The following issue relationships would be created if staged mode was disabled:",
      items: linkItems,
      renderItem: item => {
        const relationship = item.relationship || "sub";
        let content = "";
        if (relationship === "sub") {
          content += `**Relationship:** Sub-issue\n`;
          content += `**Parent Issue:** ${item.parent_issue}\n`;
          content += `**Sub-issue (child):** ${item.child_issue}\n`;
        } else {
          content += `**Relationship:** Blocking\n`;
          content += `**Blocking Issue:** ${item.parent_issue}\n`;
          content += `**Blocked Issue:** ${item.child_issue}\n`;
        }
        return content;
      },
    });
    return;
  }

  // Get target repository (for cross-repo support)
  const targetRepoSlug = process.env.GH_AW_TARGET_REPO_SLUG;
  let targetOwner = context.repo.owner;
  let targetRepo = context.repo.repo;

  if (targetRepoSlug) {
    const parts = targetRepoSlug.split("/");
    if (parts.length === 2) {
      targetOwner = parts[0];
      targetRepo = parts[1];
      core.info(`Using target repository: ${targetOwner}/${targetRepo}`);
    }
  }

  // Process each link_issues item
  let successCount = 0;
  let failCount = 0;
  const results = [];

  for (const item of linkItems) {
    const relationship = item.relationship || "sub";

    // Parse parent and child issue references
    const parentRef = parseIssueReference(item.parent_issue);
    const childRef = parseIssueReference(item.child_issue);

    if (!parentRef) {
      core.error(`Invalid parent_issue reference: ${item.parent_issue}`);
      failCount++;
      results.push({
        success: false,
        relationship,
        parent: item.parent_issue,
        child: item.child_issue,
        error: "Invalid parent_issue reference",
      });
      continue;
    }

    if (!childRef) {
      core.error(`Invalid child_issue reference: ${item.child_issue}`);
      failCount++;
      results.push({
        success: false,
        relationship,
        parent: item.parent_issue,
        child: item.child_issue,
        error: "Invalid child_issue reference",
      });
      continue;
    }

    // Determine owner/repo for each issue (fallback to target repo)
    const parentOwner = parentRef.owner || targetOwner;
    const parentRepo = parentRef.repo || targetRepo;
    const childOwner = childRef.owner || targetOwner;
    const childRepo = childRef.repo || targetRepo;

    core.info(`Processing ${relationship} relationship: parent #${parentRef.number} -> child #${childRef.number}`);

    // Get node IDs for both issues
    const parentNodeId = await getIssueNodeId(parentOwner, parentRepo, parentRef.number);
    const childNodeId = await getIssueNodeId(childOwner, childRepo, childRef.number);

    if (!parentNodeId) {
      core.error(`Could not find parent issue #${parentRef.number} in ${parentOwner}/${parentRepo}`);
      failCount++;
      results.push({
        success: false,
        relationship,
        parent: `${parentOwner}/${parentRepo}#${parentRef.number}`,
        child: `${childOwner}/${childRepo}#${childRef.number}`,
        error: `Parent issue #${parentRef.number} not found`,
      });
      continue;
    }

    if (!childNodeId) {
      core.error(`Could not find child issue #${childRef.number} in ${childOwner}/${childRepo}`);
      failCount++;
      results.push({
        success: false,
        relationship,
        parent: `${parentOwner}/${parentRepo}#${parentRef.number}`,
        child: `${childOwner}/${childRepo}#${childRef.number}`,
        error: `Child issue #${childRef.number} not found`,
      });
      continue;
    }

    // Create the relationship
    let success = false;
    if (relationship === "sub") {
      // Sub-issue: child becomes sub-issue of parent
      success = await addSubIssue(parentNodeId, childNodeId);
      if (success) {
        core.info(`Successfully made issue #${childRef.number} a sub-issue of #${parentRef.number}`);
      }
    } else if (relationship === "blocks") {
      // Blocking: parent blocks child (child depends on parent)
      success = await addIssueDependency(childNodeId, parentNodeId);
      if (success) {
        core.info(`Successfully marked issue #${parentRef.number} as blocking #${childRef.number}`);
      }
    }

    if (success) {
      successCount++;
      results.push({
        success: true,
        relationship,
        parent: `${parentOwner}/${parentRepo}#${parentRef.number}`,
        child: `${childOwner}/${childRepo}#${childRef.number}`,
      });
    } else {
      failCount++;
      results.push({
        success: false,
        relationship,
        parent: `${parentOwner}/${parentRepo}#${parentRef.number}`,
        child: `${childOwner}/${childRepo}#${childRef.number}`,
        error: "Failed to create relationship via GitHub API",
      });
    }
  }

  // Set outputs
  core.setOutput("links_created", successCount.toString());
  core.setOutput("links_failed", failCount.toString());

  // Generate summary
  let summaryContent = "## Link Issues\n\n";

  if (successCount > 0) {
    summaryContent += `Successfully created ${successCount} issue relationship(s):\n\n`;
    for (const result of results.filter(r => r.success)) {
      if (result.relationship === "sub") {
        summaryContent += `- ✅ Made ${result.child} a sub-issue of ${result.parent}\n`;
      } else {
        summaryContent += `- ✅ Marked ${result.parent} as blocking ${result.child}\n`;
      }
    }
    summaryContent += "\n";
  }

  if (failCount > 0) {
    summaryContent += `Failed to create ${failCount} relationship(s):\n\n`;
    for (const result of results.filter(r => !r.success)) {
      summaryContent += `- ❌ ${result.parent} → ${result.child}: ${result.error}\n`;
    }
    summaryContent += "\n";
  }

  if (successCount === 0 && failCount === 0) {
    summaryContent += "No issue relationships were processed.\n";
  }

  await core.summary.addRaw(summaryContent).write();

  // Fail the job if all items failed
  if (failCount > 0 && successCount === 0) {
    core.setFailed(`All ${failCount} issue relationship(s) failed to create`);
  }
}

await main();
