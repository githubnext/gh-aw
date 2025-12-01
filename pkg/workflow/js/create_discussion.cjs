// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { closeOlderDiscussions } = require("./close_older_discussions.cjs");
const { replaceTemporaryIdReferences, loadTemporaryIdMap } = require("./temporary_id.cjs");

/**
 * Parse the allowed repos from environment variable
 * @returns {Set<string>} Set of allowed repository slugs
 */
function parseAllowedRepos() {
  const allowedReposEnv = process.env.GH_AW_ALLOWED_REPOS;
  const set = new Set();
  if (allowedReposEnv) {
    allowedReposEnv
      .split(",")
      .map(repo => repo.trim())
      .filter(repo => repo)
      .forEach(repo => set.add(repo));
  }
  return set;
}

/**
 * Get the default target repository
 * @returns {string} Repository slug in "owner/repo" format
 */
function getDefaultTargetRepo() {
  // First check if there's a target-repo override
  const targetRepoSlug = process.env.GH_AW_TARGET_REPO_SLUG;
  if (targetRepoSlug) {
    return targetRepoSlug;
  }
  // Fall back to context repo
  return `${context.repo.owner}/${context.repo.repo}`;
}

/**
 * Validate that a repo is allowed for discussion creation
 * @param {string} repo - Repository slug to validate
 * @param {string} defaultRepo - Default target repository
 * @param {Set<string>} allowedRepos - Set of explicitly allowed repos
 * @returns {{valid: boolean, error: string|null}}
 */
function validateRepo(repo, defaultRepo, allowedRepos) {
  // Default repo is always allowed
  if (repo === defaultRepo) {
    return { valid: true, error: null };
  }
  // Check if it's in the allowed repos list
  if (allowedRepos.has(repo)) {
    return { valid: true, error: null };
  }
  return {
    valid: false,
    error: `Repository '${repo}' is not in the allowed-repos list. Allowed: ${defaultRepo}${allowedRepos.size > 0 ? ", " + Array.from(allowedRepos).join(", ") : ""}`,
  };
}

/**
 * Parse owner and repo from a repository slug
 * @param {string} repoSlug - Repository slug in "owner/repo" format
 * @returns {{owner: string, repo: string}|null}
 */
function parseRepoSlug(repoSlug) {
  const parts = repoSlug.split("/");
  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    return null;
  }
  return { owner: parts[0], repo: parts[1] };
}

/**
 * Fetch repository ID and discussion categories for a repository
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @returns {Promise<{repositoryId: string, discussionCategories: Array<{id: string, name: string, slug: string, description: string}>}|null>}
 */
async function fetchRepoDiscussionInfo(owner, repo) {
  const repositoryQuery = `
    query($owner: String!, $repo: String!) {
      repository(owner: $owner, name: $repo) {
        id
        discussionCategories(first: 20) {
          nodes {
            id
            name
            slug
            description
          }
        }
      }
    }
  `;
  const queryResult = await github.graphql(repositoryQuery, {
    owner: owner,
    repo: repo,
  });
  if (!queryResult || !queryResult.repository) {
    return null;
  }
  return {
    repositoryId: queryResult.repository.id,
    discussionCategories: queryResult.repository.discussionCategories.nodes || [],
  };
}

/**
 * Resolve category ID for a repository
 * @param {string} categoryConfig - Category ID, name, or slug from config
 * @param {string} itemCategory - Category from agent output item (optional)
 * @param {Array<{id: string, name: string, slug: string}>} categories - Available categories
 * @returns {string|undefined} Resolved category ID
 */
function resolveCategoryId(categoryConfig, itemCategory, categories) {
  // Use item category if provided, otherwise use config
  const categoryToMatch = itemCategory || categoryConfig;

  if (categoryToMatch) {
    // Try to match against category IDs first
    const categoryById = categories.find(cat => cat.id === categoryToMatch);
    if (categoryById) {
      return categoryById.id;
    }
    // Try to match against category names
    const categoryByName = categories.find(cat => cat.name === categoryToMatch);
    if (categoryByName) {
      return categoryByName.id;
    }
    // Try to match against category slugs (routes)
    const categoryBySlug = categories.find(cat => cat.slug === categoryToMatch);
    if (categoryBySlug) {
      return categoryBySlug.id;
    }
  }

  // Fall back to first category if available
  if (categories.length > 0) {
    return categories[0].id;
  }

  return undefined;
}

async function main() {
  // Initialize outputs to empty strings to ensure they're always set
  core.setOutput("discussion_number", "");
  core.setOutput("discussion_url", "");

  // Load the temporary ID map from create_issue job
  const temporaryIdMap = loadTemporaryIdMap();
  if (temporaryIdMap.size > 0) {
    core.info(`Loaded temporary ID map with ${temporaryIdMap.size} entries`);
  }

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const createDiscussionItems = result.items.filter(item => item.type === "create_discussion");
  if (createDiscussionItems.length === 0) {
    core.warning("No create-discussion items found in agent output");
    return;
  }
  core.info(`Found ${createDiscussionItems.length} create-discussion item(s)`);

  // Parse allowed repos and default target
  const allowedRepos = parseAllowedRepos();
  const defaultTargetRepo = getDefaultTargetRepo();
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${Array.from(allowedRepos).join(", ")}`);
  }

  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    let summaryContent = "## 🎭 Staged Mode: Create Discussions Preview\n\n";
    summaryContent += "The following discussions would be created if staged mode was disabled:\n\n";
    for (let i = 0; i < createDiscussionItems.length; i++) {
      const item = createDiscussionItems[i];
      summaryContent += `### Discussion ${i + 1}\n`;
      summaryContent += `**Title:** ${item.title || "No title provided"}\n\n`;
      if (item.repo) {
        summaryContent += `**Repository:** ${item.repo}\n\n`;
      }
      if (item.body) {
        summaryContent += `**Body:**\n${item.body}\n\n`;
      }
      if (item.category) {
        summaryContent += `**Category:** ${item.category}\n\n`;
      }
      summaryContent += "---\n\n";
    }
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Discussion creation preview written to step summary");
    return;
  }

  // Cache for repository info to avoid redundant API calls
  /** @type {Map<string, {repositoryId: string, discussionCategories: Array<{id: string, name: string, slug: string, description: string}>}>} */
  const repoInfoCache = new Map();

  // Get configuration for close-older-discussions
  const closeOlderEnabled = process.env.GH_AW_CLOSE_OLDER_DISCUSSIONS === "true";
  const titlePrefix = process.env.GH_AW_DISCUSSION_TITLE_PREFIX || "";
  const configCategory = process.env.GH_AW_DISCUSSION_CATEGORY || "";
  const labelsEnvVar = process.env.GH_AW_DISCUSSION_LABELS || "";
  const labels = labelsEnvVar
    ? labelsEnvVar
        .split(",")
        .map(l => l.trim())
        .filter(l => l.length > 0)
    : [];
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const runId = context.runId;
  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository
    ? `${context.payload.repository.html_url}/actions/runs/${runId}`
    : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

  const createdDiscussions = [];
  const closedDiscussionsSummary = [];

  for (let i = 0; i < createDiscussionItems.length; i++) {
    const createDiscussionItem = createDiscussionItems[i];

    // Determine target repository for this discussion
    const itemRepo = createDiscussionItem.repo ? String(createDiscussionItem.repo).trim() : defaultTargetRepo;

    // Validate the repository is allowed
    const repoValidation = validateRepo(itemRepo, defaultTargetRepo, allowedRepos);
    if (!repoValidation.valid) {
      core.warning(`Skipping discussion: ${repoValidation.error}`);
      continue;
    }

    // Parse the repository slug
    const repoParts = parseRepoSlug(itemRepo);
    if (!repoParts) {
      core.warning(`Skipping discussion: Invalid repository format '${itemRepo}'. Expected 'owner/repo'.`);
      continue;
    }

    // Get repository info (cached)
    let repoInfo = repoInfoCache.get(itemRepo);
    if (!repoInfo) {
      try {
        const fetchedInfo = await fetchRepoDiscussionInfo(repoParts.owner, repoParts.repo);
        if (!fetchedInfo) {
          core.warning(`Skipping discussion: Failed to fetch repository information for '${itemRepo}'`);
          continue;
        }
        repoInfo = fetchedInfo;
        repoInfoCache.set(itemRepo, repoInfo);
        core.info(`Fetched discussion categories for ${itemRepo}: ${JSON.stringify(repoInfo.discussionCategories.map(cat => ({ name: cat.name, id: cat.id })))}`);
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        if (
          errorMessage.includes("Not Found") ||
          errorMessage.includes("not found") ||
          errorMessage.includes("Could not resolve to a Repository")
        ) {
          core.warning(`Skipping discussion: Discussions are not enabled for repository '${itemRepo}'`);
          continue;
        }
        core.error(`Failed to get discussion categories for ${itemRepo}: ${errorMessage}`);
        throw error;
      }
    }

    // Resolve category ID for this discussion
    const categoryId = resolveCategoryId(configCategory, createDiscussionItem.category, repoInfo.discussionCategories);
    if (!categoryId) {
      core.warning(`Skipping discussion in ${itemRepo}: No discussion category available`);
      continue;
    }

    core.info(
      `Processing create-discussion item ${i + 1}/${createDiscussionItems.length}: title=${createDiscussionItem.title}, bodyLength=${createDiscussionItem.body?.length || 0}, repo=${itemRepo}`
    );

    // Replace temporary ID references in title
    let title = createDiscussionItem.title ? replaceTemporaryIdReferences(createDiscussionItem.title.trim(), temporaryIdMap, itemRepo) : "";
    // Replace temporary ID references in body (with defensive null check)
    const bodyText = createDiscussionItem.body || "";
    let bodyLines = replaceTemporaryIdReferences(bodyText, temporaryIdMap, itemRepo).split("\n");
    if (!title) {
      title = replaceTemporaryIdReferences(bodyText, temporaryIdMap, itemRepo) || "Agent Output";
    }
    if (titlePrefix && !title.startsWith(titlePrefix)) {
      title = titlePrefix + title;
    }

    // Add tracker-id comment if present
    const trackerIDComment = getTrackerID("markdown");
    if (trackerIDComment) {
      bodyLines.push(trackerIDComment);
    }

    bodyLines.push(``, ``, `> AI generated by [${workflowName}](${runUrl})`, "");
    const body = bodyLines.join("\n").trim();
    core.info(`Creating discussion in ${itemRepo} with title: ${title}`);
    core.info(`Category ID: ${categoryId}`);
    core.info(`Body length: ${body.length}`);
    try {
      const createDiscussionMutation = `
        mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!) {
          createDiscussion(input: {
            repositoryId: $repositoryId,
            categoryId: $categoryId,
            title: $title,
            body: $body
          }) {
            discussion {
              id
              number
              title
              url
            }
          }
        }
      `;
      const mutationResult = await github.graphql(createDiscussionMutation, {
        repositoryId: repoInfo.repositoryId,
        categoryId: categoryId,
        title: title,
        body: body,
      });
      const discussion = mutationResult.createDiscussion.discussion;
      if (!discussion) {
        core.error(`Failed to create discussion in ${itemRepo}: No discussion data returned`);
        continue;
      }
      core.info(`Created discussion ${itemRepo}#${discussion.number}: ${discussion.url}`);
      createdDiscussions.push({ ...discussion, _repo: itemRepo });
      if (i === createDiscussionItems.length - 1) {
        core.setOutput("discussion_number", discussion.number);
        core.setOutput("discussion_url", discussion.url);
      }

      // Close older discussions if enabled and title prefix or labels are set
      // Note: close-older-discussions only works within the same repository
      const hasMatchingCriteria = titlePrefix || labels.length > 0;
      if (closeOlderEnabled && hasMatchingCriteria) {
        core.info("close-older-discussions is enabled, searching for older discussions to close...");
        try {
          const closedDiscussions = await closeOlderDiscussions(
            github,
            repoParts.owner,
            repoParts.repo,
            titlePrefix,
            labels,
            categoryId,
            { number: discussion.number, url: discussion.url },
            workflowName,
            runUrl
          );

          if (closedDiscussions.length > 0) {
            closedDiscussionsSummary.push(...closedDiscussions);
            core.info(`Closed ${closedDiscussions.length} older discussion(s) as outdated`);
          }
        } catch (closeError) {
          // Log error but don't fail the workflow - closing older discussions is a nice-to-have
          core.warning(`Failed to close older discussions: ${closeError instanceof Error ? closeError.message : String(closeError)}`);
        }
      } else if (closeOlderEnabled && !hasMatchingCriteria) {
        core.warning("close-older-discussions is enabled but no title-prefix or labels are set - skipping close older discussions");
      }
    } catch (error) {
      core.error(`✗ Failed to create discussion "${title}" in ${itemRepo}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }
  if (createdDiscussions.length > 0) {
    let summaryContent = "\n\n## GitHub Discussions\n";
    for (const discussion of createdDiscussions) {
      const repoLabel = discussion._repo !== defaultTargetRepo ? ` (${discussion._repo})` : "";
      summaryContent += `- Discussion #${discussion.number}${repoLabel}: [${discussion.title}](${discussion.url})\n`;
    }

    // Add closed discussions to summary
    if (closedDiscussionsSummary.length > 0) {
      summaryContent += "\n### Closed Older Discussions\n";
      for (const closed of closedDiscussionsSummary) {
        summaryContent += `- Discussion #${closed.number}: [View](${closed.url}) (marked as outdated)\n`;
      }
    }

    await core.summary.addRaw(summaryContent).write();
  }
  core.info(`Successfully created ${createdDiscussions.length} discussion(s)`);
}
await main();
