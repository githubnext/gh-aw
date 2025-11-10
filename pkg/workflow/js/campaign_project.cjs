// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");

/**
 * Campaign Project Board Management
 *
 * This script manages GitHub Projects v2 boards for agentic workflows:
 * - Creates a project board if it doesn't exist
 * - Adds issues created by agents to the project board
 * - Tracks sub-issues and their relationship to parent issues
 * - Creates and populates custom fields for advanced analytics:
 *   * Number fields: For story points, effort estimates, hours
 *   * Single Select fields: For priority, status, team, component
 *   * Date fields: For due dates, completion dates, deadlines
 *   * Text fields: For tags, notes, additional metadata
 *   * Iteration fields: For sprint planning (must be created manually)
 * - Updates the item status based on workflow state
 * - Generates campaign insights (velocity, progress, bottlenecks)
 *
 * Custom fields enable rich analytics and charts via:
 * - GitHub Projects native charts
 * - Third-party tools like Screenful
 * - Custom GraphQL queries
 */

async function main() {
  // Initialize outputs
  core.setOutput("project_number", "");
  core.setOutput("project_url", "");
  core.setOutput("item_id", "");

  const result = loadAgentOutput();
  if (!result.success) {
    core.warning("No agent output available");
  }

  const projectName = process.env.GH_AW_PROJECT_NAME;
  if (!projectName) {
    core.error("GH_AW_PROJECT_NAME is required");
    throw new Error("Project name is required");
  }

  const statusField = process.env.GH_AW_PROJECT_STATUS_FIELD || "Status";
  const agentField = process.env.GH_AW_PROJECT_AGENT_FIELD || "Agent";
  const view = process.env.GH_AW_PROJECT_VIEW || "board";

  core.info(`Managing campaign project: ${projectName}`);
  core.info(`Status field: ${statusField}, Agent field: ${agentField}, View: ${view}`);

  // Get organization or user login for project operations
  const owner = context.repo.owner;

  // Determine if this is an organization or user
  let ownerType = "USER";
  let ownerId;

  try {
    const ownerQuery = `
      query($login: String!) {
        repositoryOwner(login: $login) {
          __typename
          id
        }
      }
    `;
    const ownerResult = await github.graphql(ownerQuery, { login: owner });
    ownerType = ownerResult.repositoryOwner.__typename === "Organization" ? "ORGANIZATION" : "USER";
    ownerId = ownerResult.repositoryOwner.id;
    core.info(`Owner type: ${ownerType}, ID: ${ownerId}`);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    // Check for insufficient scopes or permission errors
    if (
      errorMessage.includes("INSUFFICIENT_SCOPES") ||
      errorMessage.includes("read:project") ||
      errorMessage.includes("does not have permission") ||
      errorMessage.includes("Resource not accessible")
    ) {
      core.warning(`⚠️  GitHub token does not have the required 'project' scope. Project board features will be skipped.`);
      core.warning(`💡 To enable project boards, provide a personal access token with 'project' scope.`);
      core.warning(`   Visit: https://github.com/settings/tokens to add 'project' scope to your token.`);
      core.info(`✓ Workflow will continue without project board integration.`);
      return; // Exit gracefully
    }
    core.error(`Failed to get owner info: ${errorMessage}`);
    throw error;
  }

  // Find or create project
  let project;
  try {
    // Query for existing projects
    const projectsQuery = `
      query($login: String!, $first: Int!) {
        ${ownerType === "ORGANIZATION" ? "organization" : "user"}(login: $login) {
          projectsV2(first: $first) {
            nodes {
              id
              number
              title
              url
            }
          }
        }
      }
    `;

    const projectsResult = await github.graphql(projectsQuery, {
      login: owner,
      first: 100,
    });

    const projects = ownerType === "ORGANIZATION" ? projectsResult.organization.projectsV2.nodes : projectsResult.user.projectsV2.nodes;

    project = projects.find(p => p.title === projectName);

    if (project) {
      core.info(`Found existing project: ${project.title} (#${project.number})`);
    } else {
      core.info(`Creating new project: ${projectName}`);

      // Create new project
      const createProjectMutation = `
        mutation($ownerId: ID!, $title: String!) {
          createProjectV2(input: {
            ownerId: $ownerId,
            title: $title
          }) {
            projectV2 {
              id
              number
              title
              url
            }
          }
        }
      `;

      const createResult = await github.graphql(createProjectMutation, {
        ownerId: ownerId,
        title: projectName,
      });

      project = createResult.createProjectV2.projectV2;
      core.info(`Created project #${project.number}: ${project.url}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    // Check for insufficient scopes or permission errors
    if (
      errorMessage.includes("INSUFFICIENT_SCOPES") ||
      errorMessage.includes("read:project") ||
      errorMessage.includes("does not have permission") ||
      errorMessage.includes("Resource not accessible")
    ) {
      core.warning(`⚠️  Cannot create/access project board - insufficient permissions. Skipping project board features.`);
      core.warning(`💡 To enable: provide a personal access token with 'project' scope.`);
      return; // Exit gracefully
    }
    core.error(`Failed to find/create project: ${errorMessage}`);
    throw error;
  }

  // Parse custom fields configuration
  /** @type {Array<{name: string, type: string, value?: string, options?: string[], description?: string}>} */
  let customFieldsConfig = [];
  const customFieldsJSON = process.env.GH_AW_PROJECT_CUSTOM_FIELDS;
  if (customFieldsJSON) {
    try {
      customFieldsConfig = JSON.parse(customFieldsJSON);
      core.info(`Custom fields config: ${customFieldsConfig.length} field(s)`);
    } catch (error) {
      core.warning(`Failed to parse custom fields config: ${error instanceof Error ? error.message : String(error)}`);
    }
  }

  // Get project fields
  let statusFieldId;
  let agentFieldId;
  let statusOptions = [];
  /** @type {Map<string, {id: string, type: string, options?: Array<{id: string, name: string}>}>} */
  const existingFields = new Map();

  try {
    const fieldsQuery = `
      query($projectId: ID!) {
        node(id: $projectId) {
          ... on ProjectV2 {
            fields(first: 50) {
              nodes {
                __typename
                ... on ProjectV2FieldCommon {
                  id
                  name
                }
                ... on ProjectV2SingleSelectField {
                  id
                  name
                  options {
                    id
                    name
                  }
                }
              }
            }
          }
        }
      }
    `;

    const fieldsResult = await github.graphql(fieldsQuery, { projectId: project.id });
    const fields = fieldsResult.node.fields.nodes;

    // Find status field
    const statusFieldNode = fields.find(f => f.name === statusField);
    if (statusFieldNode) {
      statusFieldId = statusFieldNode.id;
      if (statusFieldNode.options) {
        statusOptions = statusFieldNode.options;
      }
      core.info(`Found status field: ${statusField} (${statusFieldId})`);
      core.info(`Status options: ${statusOptions.map(o => o.name).join(", ")}`);
    }

    // Find agent field
    const agentFieldNode = fields.find(f => f.name === agentField);
    if (agentFieldNode) {
      agentFieldId = agentFieldNode.id;
      core.info(`Found agent field: ${agentField} (${agentFieldId})`);
    }

    // Map existing fields for custom field creation
    for (const field of fields) {
      existingFields.set(field.name, {
        id: field.id,
        type: field.__typename,
        options: field.options,
      });
    }
  } catch (error) {
    core.error(`Failed to get project fields: ${error instanceof Error ? error.message : String(error)}`);
    throw error;
  }

  // Create custom fields if they don't exist
  for (const customField of customFieldsConfig) {
    if (!existingFields.has(customField.name)) {
      try {
        core.info(`Creating custom field: ${customField.name} (${customField.type})`);

        let mutation = "";
        let variables = {
          projectId: project.id,
          name: customField.name,
        };

        switch (customField.type) {
          case "number":
            mutation = `
              mutation($projectId: ID!, $name: String!) {
                createProjectV2Field(input: {
                  projectId: $projectId,
                  dataType: NUMBER,
                  name: $name
                }) {
                  projectV2Field {
                    ... on ProjectV2Field {
                      id
                      name
                    }
                  }
                }
              }
            `;
            break;

          case "date":
            mutation = `
              mutation($projectId: ID!, $name: String!) {
                createProjectV2Field(input: {
                  projectId: $projectId,
                  dataType: DATE,
                  name: $name
                }) {
                  projectV2Field {
                    ... on ProjectV2Field {
                      id
                      name
                    }
                  }
                }
              }
            `;
            break;

          case "text":
            mutation = `
              mutation($projectId: ID!, $name: String!) {
                createProjectV2Field(input: {
                  projectId: $projectId,
                  dataType: TEXT,
                  name: $name
                }) {
                  projectV2Field {
                    ... on ProjectV2Field {
                      id
                      name
                    }
                  }
                }
              }
            `;
            break;

          case "single_select":
            if (customField.options && customField.options.length > 0) {
              mutation = `
                mutation($projectId: ID!, $name: String!, $options: [ProjectV2SingleSelectFieldOptionInput!]!) {
                  createProjectV2Field(input: {
                    projectId: $projectId,
                    dataType: SINGLE_SELECT,
                    name: $name,
                    singleSelectOptions: $options
                  }) {
                    projectV2Field {
                      ... on ProjectV2SingleSelectField {
                        id
                        name
                        options {
                          id
                          name
                        }
                      }
                    }
                  }
                }
              `;
              variables.options = customField.options.map((/** @type {string} */ opt) => ({
                name: opt,
                color: "GRAY",
                description: "",
              }));
            } else {
              core.warning(`Skipping single_select field ${customField.name}: no options provided`);
              continue;
            }
            break;

          case "iteration":
            core.warning(`Iteration fields must be created manually in GitHub Projects UI`);
            continue;

          default:
            core.warning(`Unknown custom field type: ${customField.type}`);
            continue;
        }

        if (mutation) {
          const createResult = await github.graphql(mutation, variables);
          const newField = createResult.createProjectV2Field.projectV2Field;
          existingFields.set(newField.name, {
            id: newField.id,
            type: customField.type,
            options: newField.options,
          });
          core.info(`✓ Created custom field: ${newField.name} (${newField.id})`);
        }
      } catch (error) {
        core.warning(`Failed to create custom field ${customField.name}: ${error instanceof Error ? error.message : String(error)}`);
      }
    } else {
      core.info(`Custom field ${customField.name} already exists`);
    }
  }

  // Determine status based on workflow conclusion
  let status = "In Progress";
  const jobStatus = context.payload?.workflow_run?.conclusion || process.env.GITHUB_JOB_STATUS;

  if (jobStatus === "success") {
    status = "Done";
  } else if (jobStatus === "failure") {
    status = "Failed";
  } else if (jobStatus === "cancelled") {
    status = "Cancelled";
  }

  core.info(`Item status: ${status} (job status: ${jobStatus})`);

  // Collect issues and sub-issues created during the workflow
  /** @type {Array<{number: number, url: string, title: string, isSubIssue: boolean, parentIssue?: number}>} */
  const createdIssues = [];
  if (result.success && result.items.length > 0) {
    for (const output of result.items) {
      if (output.type === "create-issue" && output.issueNumber) {
        createdIssues.push({
          number: output.issueNumber,
          url: output.issueUrl,
          title: output.issueTitle || `Issue #${output.issueNumber}`,
          isSubIssue: output.parentIssue !== undefined,
          parentIssue: output.parentIssue,
        });
        core.info(`Found created issue: #${output.issueNumber} - ${output.issueTitle || "(no title)"}`);
      }
    }
  }

  // Get repository node ID for linking issues
  let repositoryId;
  try {
    const repoQuery = `
      query($owner: String!, $name: String!) {
        repository(owner: $owner, name: $name) {
          id
        }
      }
    `;
    const repoResult = await github.graphql(repoQuery, {
      owner: context.repo.owner,
      name: context.repo.repo,
    });
    repositoryId = repoResult.repository.id;
  } catch (error) {
    core.warning(`Failed to get repository ID: ${error instanceof Error ? error.message : String(error)}`);
  }

  // Add issues to project board
  /** @type {string[]} */
  const addedItemIds = [];
  if (createdIssues.length > 0 && repositoryId) {
    core.info(`Adding ${createdIssues.length} issue(s) to project board`);

    for (const issue of createdIssues) {
      try {
        // Get issue node ID
        const issueQuery = `
          query($owner: String!, $name: String!, $number: Int!) {
            repository(owner: $owner, name: $name) {
              issue(number: $number) {
                id
              }
            }
          }
        `;
        const issueResult = await github.graphql(issueQuery, {
          owner: context.repo.owner,
          name: context.repo.repo,
          number: issue.number,
        });
        const issueId = issueResult.repository.issue.id;

        // Add issue to project
        const addIssueMutation = `
          mutation($projectId: ID!, $contentId: ID!) {
            addProjectV2ItemById(input: {
              projectId: $projectId,
              contentId: $contentId
            }) {
              item {
                id
              }
            }
          }
        `;

        const addIssueResult = await github.graphql(addIssueMutation, {
          projectId: project.id,
          contentId: issueId,
        });

        const itemId = addIssueResult.addProjectV2ItemById.item.id;
        addedItemIds.push(itemId);
        core.info(`Added issue #${issue.number} to project (item ID: ${itemId})`);

        // Update status field if available
        if (statusFieldId) {
          // Use "Done" for successfully created issues, keep status for failed ones
          const issueStatus = jobStatus === "success" ? "Done" : status;
          const statusOption = statusOptions.find((/** @type {{id: string, name: string}} */ o) => o.name === issueStatus);
          if (statusOption) {
            const updateStatusMutation = `
              mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
                updateProjectV2ItemFieldValue(input: {
                  projectId: $projectId,
                  itemId: $itemId,
                  fieldId: $fieldId,
                  value: { 
                    singleSelectOptionId: $optionId
                  }
                }) {
                  projectV2Item {
                    id
                  }
                }
              }
            `;

            await github.graphql(updateStatusMutation, {
              projectId: project.id,
              itemId: itemId,
              fieldId: statusFieldId,
              optionId: statusOption.id,
            });

            core.info(`Updated issue #${issue.number} status to: ${issueStatus}`);
          }
        }

        // Set agent field if available
        if (agentFieldId) {
          const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agent Workflow";
          const runNumber = context.runNumber;
          const agentName = `${workflowName} #${runNumber}`;

          const updateAgentMutation = `
            mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $text: String!) {
              updateProjectV2ItemFieldValue(input: {
                projectId: $projectId,
                itemId: $itemId,
                fieldId: $fieldId,
                value: { 
                  text: $text
                }
              }) {
                projectV2Item {
                  id
                }
              }
            }
          `;

          await github.graphql(updateAgentMutation, {
            projectId: project.id,
            itemId: itemId,
            fieldId: agentFieldId,
            text: agentName,
          });

          core.info(`Set agent field to: ${agentName}`);
        }

        // Populate custom fields with configured values
        for (const customFieldConfig of customFieldsConfig) {
          if (!customFieldConfig.value) continue;

          const fieldInfo = existingFields.get(customFieldConfig.name);
          if (!fieldInfo) {
            core.warning(`Custom field ${customFieldConfig.name} not found in project`);
            continue;
          }

          try {
            let mutation = "";
            let fieldVariables = {
              projectId: project.id,
              itemId: itemId,
              fieldId: fieldInfo.id,
            };

            switch (customFieldConfig.type) {
              case "number":
                mutation = `
                  mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: Float!) {
                    updateProjectV2ItemFieldValue(input: {
                      projectId: $projectId,
                      itemId: $itemId,
                      fieldId: $fieldId,
                      value: { number: $value }
                    }) {
                      projectV2Item { id }
                    }
                  }
                `;
                fieldVariables.value = parseFloat(customFieldConfig.value);
                break;

              case "date":
                mutation = `
                  mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: Date!) {
                    updateProjectV2ItemFieldValue(input: {
                      projectId: $projectId,
                      itemId: $itemId,
                      fieldId: $fieldId,
                      value: { date: $value }
                    }) {
                      projectV2Item { id }
                    }
                  }
                `;
                // Parse date value (ISO format YYYY-MM-DD)
                const dateValue = new Date(customFieldConfig.value);
                fieldVariables.value = dateValue.toISOString().split("T")[0];
                break;

              case "text":
                mutation = `
                  mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $value: String!) {
                    updateProjectV2ItemFieldValue(input: {
                      projectId: $projectId,
                      itemId: $itemId,
                      fieldId: $fieldId,
                      value: { text: $value }
                    }) {
                      projectV2Item { id }
                    }
                  }
                `;
                fieldVariables.value = customFieldConfig.value;
                break;

              case "single_select":
                if (fieldInfo.options) {
                  const option = fieldInfo.options.find(
                    (/** @type {{id: string, name: string}} */ o) => o.name === customFieldConfig.value
                  );
                  if (option) {
                    mutation = `
                      mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
                        updateProjectV2ItemFieldValue(input: {
                          projectId: $projectId,
                          itemId: $itemId,
                          fieldId: $fieldId,
                          value: { singleSelectOptionId: $optionId }
                        }) {
                          projectV2Item { id }
                        }
                      }
                    `;
                    fieldVariables.optionId = option.id;
                  } else {
                    core.warning(`Option "${customFieldConfig.value}" not found in field ${customFieldConfig.name}`);
                    continue;
                  }
                }
                break;

              default:
                core.warning(`Cannot set value for field type: ${customFieldConfig.type}`);
                continue;
            }

            if (mutation) {
              await github.graphql(mutation, fieldVariables);
              core.info(`Set ${customFieldConfig.name} = ${customFieldConfig.value}`);
            }
          } catch (error) {
            core.warning(`Failed to set custom field ${customFieldConfig.name}: ${error instanceof Error ? error.message : String(error)}`);
          }
        }

        // Parse and set simple text fields if provided
        const customFieldsJSON = process.env.GH_AW_PROJECT_FIELDS;
        if (customFieldsJSON) {
          try {
            const customFields = JSON.parse(customFieldsJSON);
            core.info(`Setting custom fields: ${Object.keys(customFields).join(", ")}`);
            // Note: Simple text field updates - would need field IDs to update
          } catch (error) {
            core.warning(`Failed to parse custom fields: ${error instanceof Error ? error.message : String(error)}`);
          }
        }
      } catch (error) {
        core.warning(`Failed to update issue #${issue.number}: ${error instanceof Error ? error.message : String(error)}`);
      }
    }
  } else if (createdIssues.length === 0) {
    core.info("No issues created during workflow - creating tracking item");

    // Create draft issue item as fallback for workflows that don't create issues
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agent Workflow";
    const runNumber = context.runNumber;
    const itemTitle = `${workflowName} #${runNumber}`;

    try {
      const createItemMutation = `
        mutation($projectId: ID!, $title: String!) {
          addProjectV2DraftIssue(input: {
            projectId: $projectId,
            title: $title
          }) {
            projectItem {
              id
            }
          }
        }
      `;

      const createItemResult = await github.graphql(createItemMutation, {
        projectId: project.id,
        title: itemTitle,
      });

      const itemId = createItemResult.addProjectV2DraftIssue.projectItem.id;
      addedItemIds.push(itemId);
      core.info(`Created draft item: ${itemTitle} (${itemId})`);

      // Update status field
      if (statusFieldId) {
        const statusOption = statusOptions.find((/** @type {{id: string, name: string}} */ o) => o.name === status);
        if (statusOption) {
          const updateStatusMutation = `
            mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
              updateProjectV2ItemFieldValue(input: {
                projectId: $projectId,
                itemId: $itemId,
                fieldId: $fieldId,
                value: { 
                  singleSelectOptionId: $optionId
                }
              }) {
                projectV2Item {
                  id
                }
              }
            }
          `;

          await github.graphql(updateStatusMutation, {
            projectId: project.id,
            itemId: itemId,
            fieldId: statusFieldId,
            optionId: statusOption.id,
          });

          core.info(`Updated status to: ${status}`);
        }
      }
    } catch (error) {
      core.error(`Failed to create draft item: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Generate insights if requested
  const insightsConfig = process.env.GH_AW_PROJECT_INSIGHTS;
  if (insightsConfig) {
    const insights = insightsConfig.split(",").map(i => i.trim());
    core.info(`Generating insights: ${insights.join(", ")}`);

    // Query project items for statistics
    /** @type {any[]} */
    let projectItems = [];
    try {
      const itemsQuery = `
        query($projectId: ID!, $first: Int!) {
          node(id: $projectId) {
            ... on ProjectV2 {
              items(first: $first) {
                nodes {
                  id
                  type
                  content {
                    ... on Issue {
                      number
                      title
                      url
                      state
                      createdAt
                      closedAt
                      labels(first: 10) {
                        nodes {
                          name
                        }
                      }
                    }
                  }
                  fieldValues(first: 20) {
                    nodes {
                      __typename
                      ... on ProjectV2ItemFieldSingleSelectValue {
                        name
                        field {
                          ... on ProjectV2SingleSelectField {
                            name
                          }
                        }
                      }
                      ... on ProjectV2ItemFieldTextValue {
                        text
                        field {
                          ... on ProjectV2Field {
                            name
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      `;

      const itemsResult = await github.graphql(itemsQuery, {
        projectId: project.id,
        first: 100,
      });

      projectItems = itemsResult.node.items.nodes;
      core.info(`Retrieved ${projectItems.length} project items for insights`);
    } catch (error) {
      core.warning(`Failed to query project items: ${error instanceof Error ? error.message : String(error)}`);
    }

    let summaryContent = "\n\n## 📊 Campaign Project Insights\n\n";
    summaryContent += `**Project:** [${project.title}](${project.url})\n\n`;
    summaryContent += `**Issues Added:** ${createdIssues.length}\n\n`;

    if (createdIssues.length > 0) {
      summaryContent += "### Created Issues\n\n";
      for (const issue of createdIssues) {
        const badge = issue.isSubIssue ? "🔗" : "📝";
        summaryContent += `- ${badge} [#${issue.number}](${issue.url}) - ${issue.title}\n`;
        if (issue.isSubIssue && issue.parentIssue) {
          summaryContent += `  ↳ Sub-issue of #${issue.parentIssue}\n`;
        }
      }
      summaryContent += "\n";

      // Calculate sub-issue statistics
      const mainIssues = createdIssues.filter(i => !i.isSubIssue);
      const subIssues = createdIssues.filter(i => i.isSubIssue);
      if (subIssues.length > 0) {
        summaryContent += `**Issue Breakdown:** ${mainIssues.length} main issue(s), ${subIssues.length} sub-issue(s)\n\n`;
      }
    }

    if (projectItems.length > 0) {
      // Calculate status distribution
      /** @type {Record<string, number>} */
      const statusCounts = {};
      for (const item of projectItems) {
        for (const fieldValue of item.fieldValues.nodes) {
          if (fieldValue.__typename === "ProjectV2ItemFieldSingleSelectValue" && fieldValue.field?.name === statusField) {
            statusCounts[fieldValue.name] = (statusCounts[fieldValue.name] || 0) + 1;
          }
        }
      }

      if (insights.includes("campaign-progress")) {
        summaryContent += "### Campaign Progress\n\n";
        const total = projectItems.length;
        for (const [statusName, count] of Object.entries(statusCounts)) {
          const percentage = Math.round((count / total) * 100);
          summaryContent += `- **${statusName}:** ${count}/${total} (${percentage}%)\n`;
        }
        summaryContent += "\n";
      }

      if (insights.includes("agent-velocity")) {
        summaryContent += "### Agent Velocity\n\n";
        const completedItems = projectItems.filter((/** @type {any} */ item) => {
          if (!item.content?.closedAt) return false;
          for (const fieldValue of item.fieldValues.nodes) {
            if (fieldValue.__typename === "ProjectV2ItemFieldSingleSelectValue" && fieldValue.field?.name === statusField) {
              return fieldValue.name === "Done";
            }
          }
          return false;
        });

        if (completedItems.length > 0) {
          const durations = completedItems
            .filter((/** @type {any} */ item) => item.content?.createdAt && item.content?.closedAt)
            .map((/** @type {any} */ item) => {
              const created = new Date(item.content.createdAt).getTime();
              const closed = new Date(item.content.closedAt).getTime();
              return (closed - created) / 1000 / 60; // minutes
            });

          if (durations.length > 0) {
            const avgDuration = durations.reduce((/** @type {number} */ sum, /** @type {number} */ d) => sum + d, 0) / durations.length;
            const hours = Math.floor(avgDuration / 60);
            const minutes = Math.round(avgDuration % 60);
            summaryContent += `**Average Completion Time:** ${hours}h ${minutes}m\n`;
            summaryContent += `**Completed Items:** ${completedItems.length}\n\n`;
          }
        } else {
          summaryContent += "_No completed items yet_\n\n";
        }
      }

      if (insights.includes("bottlenecks")) {
        summaryContent += "### Bottlenecks\n\n";
        const inProgressItems = projectItems.filter((/** @type {any} */ item) => {
          for (const fieldValue of item.fieldValues.nodes) {
            if (fieldValue.__typename === "ProjectV2ItemFieldSingleSelectValue" && fieldValue.field?.name === statusField) {
              return fieldValue.name === "In Progress";
            }
          }
          return false;
        });

        if (inProgressItems.length > 0) {
          summaryContent += `**Currently In Progress:** ${inProgressItems.length} item(s)\n`;
          for (const item of inProgressItems.slice(0, 5)) {
            if (item.content?.title && item.content?.url) {
              const ageMinutes = (Date.now() - new Date(item.content.createdAt).getTime()) / 1000 / 60;
              const hours = Math.floor(ageMinutes / 60);
              const minutes = Math.round(ageMinutes % 60);
              summaryContent += `- [#${item.content.number}](${item.content.url}) - ${item.content.title} (${hours}h ${minutes}m)\n`;
            }
          }
          summaryContent += "\n";
        } else {
          summaryContent += "_No items in progress_\n\n";
        }
      }
    }

    await core.summary.addRaw(summaryContent).write();
  }

  // Set outputs
  core.setOutput("project_number", project.number);
  core.setOutput("project_url", project.url);
  core.setOutput("item_id", addedItemIds.length > 0 ? addedItemIds[0] : "");
  core.setOutput("item_count", addedItemIds.length);
  core.setOutput("issue_count", createdIssues.length);

  core.info(`✓ Successfully managed campaign project board`);
}

await main();
