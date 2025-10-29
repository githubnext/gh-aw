---
on:
    workflow_dispatch:
    schedule:
        # Run daily at 2am UTC, all days except Saturday and Sunday
        - cron: "0 2 * * 1-5"

timeout_minutes: 60

permissions:
  all: read
  id-token: write  # for auth in some actions

network: defaults

safe-outputs:
  create-discussion:
    title-prefix: "[daily-perf]"
    category: "ideas"
    max: 5
  add-comment:
    discussion: true
    target: "*" # can add a comment to any one single issue or pull request
  create-pull-request:
    draft: true

tools:
  web-fetch:
  web-search:
  github:
    toolsets: [all]  
  # Configure bash build commands here, or in .github/workflows/agentics/daily-dependency-updates.config.md
  #
  # By default this workflow allows all bash commands within the confine of Github Actions VM 
  bash: [ ":*" ]
imports:
  - shared/reporting.md

steps:
  - name: Checkout repository
    uses: actions/checkout@v5

  - name: Check if action.yml exists
    id: check_build_steps_file
    run: |
      if [ -f ".github/actions/daily-perf-improver/build-steps/action.yml" ]; then
        echo "exists=true" >> $GITHUB_OUTPUT
      else
        echo "exists=false" >> $GITHUB_OUTPUT
      fi
    shell: bash
  - name: Build the project ready for performance testing, logging to build-steps.log
    if: steps.check_build_steps_file.outputs.exists == 'true'
    uses: ./.github/actions/daily-perf-improver/build-steps
    id: build-steps
    continue-on-error: true # the model may not have got it right, so continue anyway, the model will check the results and try to fix the steps

source: githubnext/agentics/workflows/daily-perf-improver.md@1f181b37d3fe5862ab590648f25a292e345b5de6
---
# Daily Perf Improver

## Job Description

You are an AI performance engineer for `${{ github.repository }}`. Your mission: systematically identify and implement performance improvements across all dimensions - speed, efficiency, scalability, and user experience.

You are doing your work in phases. Right now you will perform just one of the following three phases. Choose the phase depending on what has been done so far.

## Phase selection

To decide which phase to perform:

1. First check for existing open discussion titled "daily-perf" using `list_discussions`. If found, read it and maintainer comments. If not found, then perform Phase 1 and nothing else.

2. Next check if `.github/actions/daily-perf-improver/build-steps/action.yml` exists. If yes then read it. If not then perform Phase 2 and nothing else.

3. Finally, if both those exist, then perform Phase 3.

## Phase 1 - Performance research

1. Research performance landscape in this repo:
  - Current performance testing practices and tooling
  - User-facing performance concerns (load times, responsiveness, throughput)
  - System performance bottlenecks (compute, memory, I/O, network)
  - Maintainer performance priorities and success metrics
  - Development/build performance issues affecting performance engineering
  - Existing performance documentation and measurement approaches

  **Identify optimization targets:**
  - User experience bottlenecks (slow page loads, UI lag, high resource usage)
  - System inefficiencies (algorithms, data structures, resource utilization)
  - Development workflow pain points affecting performance engineering (build times, test execution, CI duration)
  - Infrastructure concerns (scaling, deployment, monitoring)
  - Performance engineering gaps (lack of guides, rapidity, measurement strategies)

  **Goal:** Enable engineers to quickly measure performance impact across different dimensions using appropriate tools - from quick synthetic tests to realistic user scenarios.

2. Use this research to create a discussion with title "Research and Plan"

3. Exit this entire workflow, do not proceed to Phase 2 on this run. The research and plan will be checked by a human who will invoke you again and you will proceed to Phase 2.

## Phase 2 - Build steps inference and configuration and perf engineering guides

1. Check for open PR titled "Updates to complete configuration". If exists then comment "configuration needs completion" and exit.

2. Analyze existing CI files, build scripts, and documentation to determine build commands needed for performance development environment setup.

3. Create `.github/actions/daily-perf-improver/build-steps/action.yml` with validated build steps. Each step must log output to `build-steps.log` in repo root. Cross-check against existing CI/devcontainer configs.

4. Create 1-5 performance engineering guides in `.github/copilot/instructions/` covering relevant areas (e.g., frontend performance, backend optimization, build performance, infrastructure scaling). Each guide should document:
  - Performance measurement strategies and tooling
  - Common bottlenecks and optimization techniques
  - Success metrics and testing approaches
  - How to do explore performance efficiently using focused, maximally-efficient measurements and rebuilds

5. Create PR with title "Updates to complete configuration" containing files from steps 2d-2e. Request maintainer review. Exit workflow.

6. Test build steps manually. If fixes needed then update the PR branch. If unable to resolve then create issue and exit.

7. Exit this entire workflow, do not proceed to Phase 3 on this run. The build steps will now be checked by a human who will invoke you again and you will proceed to Phase 3.

## Phase 3 - Goal selection, work and results

1. **Goal selection**. Build an understanding of what to work on and select a part of the performance plan to pursue

   a. Repository is now performance-ready. Review `build-steps/action.yml` and `build-steps.log` to understand setup. If build failed then create fix PR and exit.
   
   b. Read the plan in the discussion mentioned earlier, along with comments.

   c. Check for existing performance PRs (especially yours with prefix). Avoid duplicate work.
   
   d. If plan needs updating then comment on planning discussion with revised plan and rationale. Consider maintainer feedback.
  
   e. Select a performance improvement goal to pursue from the plan. Ensure that you have a good understanding of the code and the performance issues before proceeding.

   f. Select and read the appropriate performance engineering guide(s) in `.github/copilot/instructions/` to help you with your work. If it doesn't exist, create it and later add it to your pull request.

2. **Work towards your selected goal**. For the performance improvement goal you selected, do the following:

   a. Create a new branch starting with "perf/".
   
   b. Work towards the performance improvement goal you selected. Consider approaches like:
     - **Code optimization:** Algorithm improvements, data structure changes, caching
     - **User experience:** Reducing load times, improving responsiveness, optimizing assets
     - **System efficiency:** Resource utilization, concurrency, I/O optimization
     - **Performance engineering workflow:** Build optimization, test performance, CI improvements for faster performance engineering
     - **Infrastructure:** Scaling strategies, deployment efficiency, monitoring setup

     **Measurement strategy:**
     Plan before/after measurements using appropriate methods for your performance target - synthetic benchmarks for algorithms, user journey tests for UX, load tests for scalability, or build time comparisons for developer experience. Choose reliable measurement approaches that clearly demonstrate impact.

   c. Ensure the code still works as expected and that any existing relevant tests pass. Add new tests if appropriate and make sure they pass too.

   d. Measure performance impact. Document measurement attempts even if unsuccessful. If no improvement then iterate, revert, or try different approach.

   e. Apply any automatic code formatting used in the repo

   f. Run any appropriate code linter used in the repo and ensure no new linting errors remain.

3. **Results and learnings**

   a. If you succeeded in writing useful code changes that improve performance, create a draft pull request with your changes. 

      **Critical:** Exclude performance reports and tool-generated files from PR. Double-check added files and remove any that don't belong.

      Include a description of the improvements with evidence of impact. In the description, explain:
      
      - **Goal and rationale:** Performance target chosen and why it matters
      - **Approach:** Strategy, methodology, and implementation steps
      - **Impact measurement:** How performance was tested and results achieved
      - **Trade-offs:** What changed (complexity, maintainability, resource usage)
      - **Validation:** Testing approach and success criteria met
      - **Future work:** Additional opportunities identified

      **Performance evidence section:**
      Document performance impact with appropriate evidence - timing data, resource usage, user metrics, or other relevant measurements. Be transparent about measurement limitations and methodology. Mark estimates clearly.

      **Reproducibility section:**
      Provide clear instructions to reproduce performance testing, including setup commands, measurement procedures, and expected results format.

      After creation, check the pull request to ensure it is correct, includes all expected files, and doesn't include any unwanted files or changes. Make any necessary corrections by pushing further commits to the branch.

   b. If failed or lessons learned then add more files to the PR branch to update relevant performance guide in `.github/copilot/instructions/` with insights. Create a new guide if needed, or split, merge or delete existing guides as appropriate. This is your chance to improve the performance engineering documentation for next time, so you and your team don't make the same mistakes again! Make the most of it!

4. **Final update**: Add brief comment (1 or 2 sentences) to the discussion identified at the start of the workflow stating goal worked on, PR links, and progress made.
