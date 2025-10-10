# Documentation Audit - Diátaxis Categorization

## Executive Summary

This audit categorizes all 33 documentation files in `docs/src/content/docs/` according to the Diátaxis framework. The current structure has a reasonable foundation but requires reorganization and content splitting to fully align with Diátaxis principles.

**Key Findings:**
- **Mixed content is common**: 15 files contain multiple Diátaxis types that should be split
- **Strong reference section**: Most reference files are properly categorized
- **Guides need clarification**: Many guides mix How-To and Explanation content
- **Missing tutorial content**: Need beginner-focused learning paths
- **Samples are examples, not docs**: Sample pages should be reorganized

## Current Structure Analysis

### start-here/ (2 files)

| File | Primary Category | Secondary Types | Action Required |
|------|-----------------|-----------------|-----------------|
| `index.mdx` | **Explanation** | Tutorial (cards) | **Split** - Move tutorial cards to separate beginner tutorial |
| `quick-start.md` | **Tutorial** | Reference (steps) | **Keep** - Good tutorial, minor cleanup of reference details |
| `concepts.md` | **Explanation** | None | **Keep** - Pure explanation of concepts |

**Analysis:**
- The index.mdx serves as a landing page with mixed purposes
- quick-start.md is a solid tutorial but includes some reference material
- concepts.md is well-structured explanation content

### guides/ (8 files)

| File | Primary Category | Secondary Types | Action Required |
|------|-----------------|-----------------|-----------------|
| `chatops.md` | **How-To** | Explanation, Reference | **Split** - Extract explanation sections and reference details |
| `issueops.md` | **How-To** | Explanation, Reference | **Split** - Extract explanation sections and reference details |
| `labelops.md` | **How-To** | Explanation, Reference | **Split** - Extract explanation sections |
| `mcp-server.md` | **How-To** | Reference | **Keep with cleanup** - Good How-To but needs reference extraction |
| `mcps.md` | **How-To** | Explanation, Reference | **Split** - Large file mixing concepts, how-to, and reference |
| `packaging-imports.md` | **How-To** | Reference | **Split** - Extract spec syntax reference |
| `security.md` | **Explanation** | How-To | **Split** - Core is explanation, extract how-to implementation |
| `web-search.md` | **How-To** | None | **Keep** - Focused how-to guide |

**Analysis:**
- Most guides mix How-To instructions with conceptual explanations
- Many include reference tables/lists that belong in reference section
- security.md is primarily explanation with implementation patterns
- mcps.md is particularly large and unfocused, needs significant splitting

### reference/ (14 files)

| File | Primary Category | Secondary Types | Action Required |
|------|-----------------|-----------------|-----------------|
| `cache-memory.md` | **Reference** | How-To | **Keep with cleanup** - Mostly reference, minor how-to examples |
| `command-triggers.md` | **Reference** | How-To | **Split** - Extract how-to examples to guides |
| `concurrency.md` | **Reference** | None | **Keep** - Pure reference |
| `custom-safe-outputs.md` | **Reference** | How-To | **Keep with cleanup** - Good reference with examples |
| `engines.md` | **Reference** | How-To | **Split** - Extract setup how-tos |
| `frontmatter.md` | **Reference** | None | **Keep** - Comprehensive reference documentation |
| `include-directives.md` | **Reference** | How-To, Explanation | **Split** - Extract explanation and how-to content |
| `markdown.md` | **Reference** | None | **Keep** - Pure reference |
| `network.md` | **Reference** | Explanation | **Keep with cleanup** - Mostly reference |
| `safe-jobs.md` | **Reference** | Explanation | **Keep** - Technical reference with context |
| `safe-outputs.md` | **Reference** | How-To | **Split** - Extract how-to examples to guides |
| `spec-syntax.md` | **Reference** | None | **Keep** - Pure reference |
| `template-rendering.md` | **Reference** | None | **Keep** - Pure reference |
| `tools.md` | **Reference** | How-To | **Split** - Extract configuration how-tos |
| `workflow-structure.md` | **Reference** | None | **Keep** - Pure reference |

**Analysis:**
- Strong foundation of reference material
- Several files correctly separate reference from other content
- Some files like engines.md and tools.md mix reference specs with setup instructions
- frontmatter.md, markdown.md, and spec-syntax.md are exemplary pure reference

### samples/ (4 files)

| File | Primary Category | Current Purpose | Action Required |
|------|-----------------|-----------------|-----------------|
| `coding-development.md` | **N/A (Index)** | Sample listing | **Move** - These are not documentation but sample listings |
| `quality-testing.md` | **N/A (Index)** | Sample listing | **Move** - Should link to external samples |
| `research-planning.md` | **N/A (Index)** | Sample listing | **Move** - Consider removing or converting to tutorial |
| `triage-analysis.md` | **N/A (Index)** | Sample listing | **Move** - Not instructional content |

**Analysis:**
- These files are indexes of sample workflows, not documentation
- They list samples from the Agentics collection with descriptions
- Should either be removed, converted to tutorial material, or kept as simple navigation pages
- Don't fit cleanly into Diátaxis framework as they're promotional/navigational

### tools/ (3 files)

| File | Primary Category | Secondary Types | Action Required |
| ------|-----------------|-----------------|-----------------|
| `agentic-authoring.md` | **How-To** | Reference | **Keep with cleanup** - Focused how-to with tool reference |
| `cli.md` | **Reference** | How-To | **Split** - Primarily reference but includes usage examples |
| `vscode.md` | **How-To** | None | **Keep** - Focused how-to guide |

**Analysis:**
- cli.md is primarily a command reference with usage examples
- agentic-authoring.md is a focused how-to
- vscode.md is a straightforward how-to guide

### index.mdx (Landing Page)

| Category | Purpose | Action Required |
|----------|---------|-----------------|
| **Mixed** | Landing/Overview | **Restructure** - Separate marketing from navigation |

**Analysis:**
- Serves as splash page with hero section
- Contains feature overview and navigation cards
- Not pure documentation but necessary landing page
- Keep but ensure it points to proper tutorial/guide/reference sections

## Files Requiring Content Split

### High Priority Splits

These files urgently need separation into multiple Diátaxis types:

1. **`mcps.md`** (guides/)
   - **Extract to:** How-To guide (basic setup), Explanation (what is MCP), Reference (server types, configuration options)
   - **Reason:** 393 lines mixing concepts, procedures, and technical specs
   - **Suggested split:**
     - Keep: How-To for basic MCP setup
     - New: Explanation document on MCP concepts
     - Move: Reference table to reference/mcps.md

2. **`engines.md`** (reference/)
   - **Extract to:** How-To guide (engine setup), Reference (engine configuration options)
   - **Reason:** Mixes setup procedures with configuration reference
   - **Suggested split:**
     - Move: Setup steps to guides/engine-setup.md
     - Keep: Configuration reference in reference/engines.md

3. **`tools.md`** (reference/)
   - **Extract to:** How-To guide (tool configuration), Reference (tool specifications)
   - **Reason:** Mixes configuration how-tos with technical reference
   - **Suggested split:**
     - Move: Configuration examples to guides/tool-configuration.md
     - Keep: Tool specifications in reference/tools.md

4. **`cli.md`** (tools/)
   - **Extract to:** Reference (command reference), Tutorial (CLI getting started)
   - **Reason:** 735 lines mixing command reference with usage tutorials
   - **Suggested split:**
     - Move: Command reference to reference/cli-commands.md
     - Create: Tutorial for CLI first-time usage
     - Keep: Tool overview in tools/cli.md

5. **`chatops.md`** (guides/)
   - **Extract to:** How-To (implementing ChatOps), Explanation (ChatOps concepts)
   - **Reason:** Mixes conceptual explanation with implementation
   - **Suggested split:**
     - Keep: Implementation guide in guides/chatops.md
     - New: ChatOps explanation in start-here/ or dedicated explanation section

6. **`issueops.md`** (guides/)
   - **Extract to:** How-To (implementing IssueOps), Explanation (IssueOps concepts)
   - **Reason:** Similar to chatops.md, mixes concepts and implementation
   - **Suggested split:**
     - Keep: Implementation guide in guides/issueops.md
     - New: IssueOps explanation

### Medium Priority Splits

7. **`safe-outputs.md`** (reference/)
   - **Extract to:** How-To guide (using safe outputs), Reference (safe output types)
   - **Reason:** Mixes usage examples with reference tables

8. **`packaging-imports.md`** (guides/)
   - **Extract to:** How-To (importing workflows), Reference (spec syntax)
   - **Reason:** Contains detailed spec syntax that duplicates spec-syntax.md

9. **`security.md`** (guides/)
   - **Extract to:** Explanation (security concepts), How-To (security implementation)
   - **Reason:** Primarily explanation with scattered implementation examples

10. **`command-triggers.md`** (reference/)
    - **Extract to:** How-To (using command triggers), Reference (trigger syntax)
    - **Reason:** Mixes procedural examples with reference specifications

11. **`include-directives.md`** (reference/)
    - **Extract to:** How-To (using imports), Reference (import syntax), Explanation (why imports)
    - **Reason:** Contains explanation, procedural, and reference content

### Lower Priority (Cleanup Only)

12. **`cache-memory.md`** (reference/)
    - **Action:** Remove how-to examples, keep pure reference
    - **Reason:** Mostly reference, minor cleanup needed

13. **`agentic-authoring.md`** (tools/)
    - **Action:** Minor cleanup of reference details
    - **Reason:** Focused how-to with minimal reference mixing

14. **`quick-start.md`** (start-here/)
    - **Action:** Minor cleanup of reference details
    - **Reason:** Good tutorial, just needs small adjustments

15. **`mcp-server.md`** (guides/)
    - **Action:** Extract reference configuration details
    - **Reason:** Mostly focused how-to

## Gaps Identified

### Critical Gaps (High Priority)

1. **Missing: "Your First Workflow" Tutorial**
   - **Type:** Tutorial
   - **Purpose:** Step-by-step guide for complete beginners
   - **Content:** Building a simple workflow from scratch, not just adding a sample
   - **Location:** `tutorials/first-workflow.md`

2. **Missing: "How Agentic Workflows Work" Explanation**
   - **Type:** Explanation
   - **Purpose:** Deep dive into architecture, compilation process, execution model
   - **Content:** Technical explanation of how the system works under the hood
   - **Location:** `explanation/how-it-works.md`

3. **Missing: "Workflow Design Patterns" Explanation**
   - **Type:** Explanation
   - **Purpose:** Understanding common patterns and when to use them
   - **Content:** ChatOps, IssueOps, scheduled research, etc. at conceptual level
   - **Location:** `explanation/design-patterns.md`

4. **Missing: "Debugging Failed Workflows" How-To**
   - **Type:** How-To
   - **Purpose:** Solve the problem of workflow failures
   - **Content:** Step-by-step troubleshooting guide
   - **Location:** `guides/debugging-workflows.md`

5. **Missing: "Security Best Practices" How-To**
   - **Type:** How-To
   - **Purpose:** Implement secure workflows
   - **Content:** Practical steps for securing workflows (security.md is explanation)
   - **Location:** `guides/security-implementation.md`

### Important Gaps (Medium Priority)

6. **Missing: "Error Messages" Reference**
   - **Type:** Reference
   - **Purpose:** Quick lookup for error messages
   - **Content:** Common errors and their meanings
   - **Location:** `reference/error-messages.md`

7. **Missing: "Workflow Examples" Tutorial**
   - **Type:** Tutorial
   - **Purpose:** Learn by example with annotations
   - **Content:** Annotated example workflows explaining each part
   - **Location:** `tutorials/workflow-examples.md`

8. **Missing: "Testing Workflows Locally" How-To**
   - **Type:** How-To
   - **Purpose:** Test workflows before deployment
   - **Content:** Local testing strategies and tools
   - **Location:** `guides/testing-locally.md`

9. **Missing: "Migration from GitHub Actions" How-To**
   - **Type:** How-To
   - **Purpose:** Convert existing Actions to Agentic Workflows
   - **Content:** Step-by-step conversion process
   - **Location:** `guides/migrating-from-actions.md`

10. **Missing: "Understanding Safe Outputs" Explanation**
    - **Type:** Explanation
    - **Purpose:** Understand why and how safe outputs work
    - **Content:** Architecture and security model explanation
    - **Location:** `explanation/safe-outputs-architecture.md`

### Nice-to-Have Gaps (Lower Priority)

11. **Missing: "Performance Optimization" How-To**
    - **Type:** How-To
    - **Purpose:** Optimize slow workflows
    - **Content:** Practical optimization techniques
    - **Location:** `guides/performance-optimization.md`

12. **Missing: "Advanced MCP Usage" Tutorial**
    - **Type:** Tutorial
    - **Purpose:** Learn advanced MCP features
    - **Content:** Step-by-step advanced MCP scenarios
    - **Location:** `tutorials/advanced-mcp.md`

13. **Missing: "Workflow Lifecycle" Explanation**
    - **Type:** Explanation
    - **Purpose:** Understand workflow states and transitions
    - **Content:** Conceptual model of workflow lifecycle
    - **Location:** `explanation/workflow-lifecycle.md`

## Recommended Restructure

### Proposed New Directory Structure

```
docs/src/content/docs/
├── index.mdx                          # Landing page (keep)
├── tutorials/                         # NEW - Learning-oriented
│   ├── first-workflow.md             # NEW - Build your first workflow
│   ├── workflow-examples.md          # NEW - Annotated examples  
│   └── advanced-mcp.md               # NEW - Advanced MCP tutorial
├── guides/                           # Goal-oriented (how-to)
│   ├── chatops.md                    # KEEP - Focused how-to (cleaned)
│   ├── issueops.md                   # KEEP - Focused how-to (cleaned)
│   ├── labelops.md                   # KEEP - Focused how-to (cleaned)
│   ├── web-search.md                 # KEEP - Already focused
│   ├── packaging-imports.md          # KEEP - Focused how-to (cleaned)
│   ├── mcp-server.md                 # KEEP - Focused how-to (cleaned)
│   ├── mcp-setup.md                  # NEW - Basic MCP setup (from mcps.md)
│   ├── engine-setup.md               # NEW - Engine setup (from engines.md)
│   ├── tool-configuration.md         # NEW - Tool config (from tools.md)
│   ├── security-implementation.md    # NEW - Security practices
│   ├── debugging-workflows.md        # NEW - Troubleshooting
│   ├── testing-locally.md            # NEW - Local testing
│   ├── migrating-from-actions.md     # NEW - Migration guide
│   └── performance-optimization.md   # NEW - Optimization
├── reference/                        # Information-oriented
│   ├── frontmatter.md                # KEEP - Comprehensive reference
│   ├── engines.md                    # KEEP - Engine options (cleaned)
│   ├── tools.md                      # KEEP - Tool specs (cleaned)
│   ├── safe-outputs.md               # KEEP - Safe output types (cleaned)
│   ├── safe-jobs.md                  # KEEP - Already reference
│   ├── command-triggers.md           # KEEP - Trigger syntax (cleaned)
│   ├── network.md                    # KEEP - Network options
│   ├── concurrency.md                # KEEP - Concurrency options
│   ├── cache-memory.md               # KEEP - Cache options (cleaned)
│   ├── include-directives.md         # KEEP - Import syntax (cleaned)
│   ├── markdown.md                   # KEEP - Already reference
│   ├── spec-syntax.md                # KEEP - Already reference
│   ├── template-rendering.md         # KEEP - Already reference
│   ├── workflow-structure.md         # KEEP - Already reference
│   ├── custom-safe-outputs.md        # KEEP - Already reference
│   ├── cli-commands.md               # NEW - CLI reference (from tools/cli.md)
│   ├── mcps.md                       # NEW - MCP reference (from guides/mcps.md)
│   └── error-messages.md             # NEW - Error reference
├── explanation/                      # NEW - Understanding-oriented
│   ├── concepts.md                   # MOVE - From start-here/
│   ├── how-it-works.md              # NEW - Architecture deep-dive
│   ├── design-patterns.md           # NEW - Pattern explanations
│   ├── safe-outputs-architecture.md # NEW - Safe outputs explained
│   ├── workflow-lifecycle.md        # NEW - Lifecycle explanation
│   ├── security-model.md            # NEW - Security concepts (from guides/security.md)
│   ├── chatops-concepts.md          # NEW - ChatOps explained (from guides/chatops.md)
│   └── mcp-concepts.md              # NEW - MCP explained (from guides/mcps.md)
├── tools/                            # Tool documentation
│   ├── cli.md                        # KEEP - CLI overview (cleaned, reference moved)
│   ├── vscode.md                     # KEEP - VS Code integration
│   └── agentic-authoring.md          # KEEP - Authoring tool
└── samples/                          # Example listings
    ├── coding-development.md         # KEEP or REMOVE - Sample listing
    ├── quality-testing.md            # KEEP or REMOVE - Sample listing
    ├── research-planning.md          # KEEP or REMOVE - Sample listing
    └── triage-analysis.md            # KEEP or REMOVE - Sample listing
```

### Migration Priority

**Phase 1: Critical Splits (Immediate)**
1. Split `mcps.md` → How-To, Explanation, Reference
2. Split `engines.md` → How-To, Reference
3. Split `tools.md` → How-To, Reference
4. Create critical gap content (first-workflow tutorial, how-it-works explanation)

**Phase 2: Important Splits (Near-term)**
5. Split `cli.md` → Reference, keep tool overview
6. Split `chatops.md` and `issueops.md` → How-To, Explanation
7. Split `security.md` → Explanation, How-To
8. Create medium-priority gap content (debugging, security-implementation)

**Phase 3: Cleanup and Polish (Later)**
9. Clean up remaining mixed-content files
10. Create nice-to-have gap content
11. Update cross-references across all documents
12. Review samples/ directory purpose

## Cross-Reference Updates Required

After restructuring, the following will need updating:

1. **Navigation structure** in `astro.config.mjs`
   - Add new sections: tutorials/, explanation/
   - Reorganize guides/ and reference/ sections

2. **Internal links** - Global search and replace needed for:
   - Links to split files (e.g., mcps.md → multiple files)
   - Links to moved files (e.g., concepts.md → explanation/)
   - Links to new files (tutorials, explanations)

3. **"Related Documentation" sections** - Every file with this section needs review

4. **Frontmatter sidebar order** - Renumber all sidebar orders after restructure

5. **Quick-start.md** - Update links to reference split files

## Content Quality Notes

### Well-Structured Files (Examples to Follow)

- **`frontmatter.md`** - Comprehensive, well-organized reference
- **`spec-syntax.md`** - Clean reference with clear examples
- **`workflow-structure.md`** - Focused reference documentation
- **`web-search.md`** - Concise, focused how-to guide
- **`concepts.md`** - Good explanation without mixing types

### Files Needing Significant Revision

- **`mcps.md`** - Too long (393 lines), unfocused, mixes all types
- **`cli.md`** - Too long (735+ lines), mixes reference and tutorial
- **`security.md`** - Unclear target audience, mixes explanation and how-to
- **`index.mdx`** - Landing page needs clearer separation of concerns

## Recommended Next Steps

1. **Create new directory structure** - Set up tutorials/ and explanation/ directories
2. **Create critical gap content** - First workflow tutorial, how-it-works explanation
3. **Begin high-priority splits** - Start with mcps.md, engines.md, tools.md
4. **Update navigation** - Modify astro.config.mjs to reflect new structure
5. **Migrate content incrementally** - One file at a time to avoid breaking changes
6. **Update cross-references** - Fix links as files are split/moved
7. **Review and iterate** - Get feedback on new structure before completing migration

## Conclusion

The current documentation has solid foundations but needs reorganization to align with Diátaxis:

- **Strengths:** Strong reference section, some good how-to guides
- **Weaknesses:** Extensive content mixing, missing tutorial and explanation sections
- **Priority:** Split large mixed files, create missing tutorial/explanation content
- **Timeline:** Phased approach over 3 phases

The audit identifies 15 files requiring content splits and 13 critical gaps in documentation coverage. Following the Diátaxis framework will improve documentation usability and help users find the right information at the right time.
