# Campaign Flow Diagrams

## Campaign Lifecycle Flow

```mermaid
graph TB
    Start([User Creates Campaign Spec]) --> Compile[gh aw compile]
    Compile --> Lock[.campaign.lock.yml Generated]
    Lock --> Trigger[Schedule/Manual Trigger]
    
    Trigger --> Setup[Setup Actions]
    Setup --> Discovery[Discovery Precomputation]
    
    Discovery --> SearchAPI[Search GitHub API]
    SearchAPI --> Manifest[Generate Manifest JSON]
    Manifest --> SaveCursor[Save Cursor to Repo-Memory]
    
    SaveCursor --> Agent[AI Agent Execution]
    Agent --> ReadManifest[Read Discovery Manifest]
    ReadManifest --> Phase0{Has Workers?}
    
    Phase0 -->|Yes| Dispatch[Dispatch Worker Workflows]
    Phase0 -->|No| Phase1
    Dispatch --> Phase1[Phase 1: Read Project State]
    
    Phase1 --> Phase2[Phase 2: Plan Updates]
    Phase2 --> Budget{Within Budget?}
    
    Budget -->|Yes| Phase3[Phase 3: Write Updates]
    Budget -->|No| Defer[Defer Remaining Items]
    
    Phase3 --> Phase4[Phase 4: Status Update]
    Defer --> Phase4
    Phase4 --> End([Wait for Next Run])
    
    End --> Trigger
    
    style Start fill:#e1f5e1
    style End fill:#e1f5e1
    style Agent fill:#fff4e1
    style Discovery fill:#e1f0ff
    style Manifest fill:#e1f0ff
    style Budget fill:#ffe1e1
```

## Discovery Precomputation Detail

```mermaid
sequenceDiagram
    participant Orchestrator
    participant Discovery Script
    participant GitHub API
    participant Repo-Memory
    participant Manifest File
    
    Orchestrator->>Discovery Script: Run campaign_discovery.cjs
    Discovery Script->>Repo-Memory: Load cursor.json
    Repo-Memory-->>Discovery Script: cursor: {page: 3}
    
    loop For each workflow in spec.workflows
        Discovery Script->>GitHub API: Search "tracker-id: workflow-name"
        GitHub API-->>Discovery Script: Issues/PRs (page 3)
        Discovery Script->>Discovery Script: Normalize items
        Discovery Script->>Discovery Script: Check budget (items, pages)
    end
    
    Discovery Script->>Manifest File: Write ./.gh-aw/campaign.discovery.json
    Discovery Script->>Repo-Memory: Update cursor.json
    Discovery Script-->>Orchestrator: Discovery complete
    
    Orchestrator->>Agent: Start agent job
    Agent->>Manifest File: Read discovery manifest
    Manifest File-->>Agent: {items: [...], summary: {...}}
```

## Component Architecture

```mermaid
graph LR
    subgraph "Campaign Spec (.campaign.md)"
        Spec[YAML Frontmatter]
        Spec --> ID[id: security-q1-2025]
        Spec --> Workflows[workflows: [...]]
        Spec --> Governance[governance: {...}]
        Spec --> Project[project-url]
    end
    
    subgraph "Compilation"
        Compiler[BuildOrchestrator]
        Compiler --> GenWorkflow[Generate WorkflowData]
        GenWorkflow --> Tools[Configure Tools]
        GenWorkflow --> SafeOutputs[Configure Safe Outputs]
        GenWorkflow --> Prompt[Generate Prompt]
    end
    
    subgraph "Runtime"
        Discovery[Discovery Precomputation]
        Agent[AI Agent]
        Workers[Worker Workflows]
        ProjectBoard[GitHub Project Board]
        
        Discovery --> Manifest[Manifest JSON]
        Manifest --> Agent
        Agent --> Workers
        Agent --> ProjectBoard
    end
    
    Spec --> Compiler
    Compiler --> Lock[.campaign.lock.yml]
    Lock --> Discovery
    
    style Spec fill:#e1f5e1
    style Discovery fill:#e1f0ff
    style Agent fill:#fff4e1
    style ProjectBoard fill:#ffe1f0
```

## Governance Budget Flow

```mermaid
graph TD
    Start[Campaign Run Starts] --> DiscoveryBudget{Discovery Budget}
    DiscoveryBudget -->|Items Limit| MaxItems[Max Items Per Run: 100]
    DiscoveryBudget -->|Pages Limit| MaxPages[Max Pages Per Run: 10]
    
    MaxItems --> DiscoveryLoop[Discovery Loop]
    MaxPages --> DiscoveryLoop
    
    DiscoveryLoop --> BudgetCheck{Budget Reached?}
    BudgetCheck -->|Yes| SaveCursor[Save Cursor]
    BudgetCheck -->|No| Continue[Continue Discovery]
    Continue --> DiscoveryLoop
    
    SaveCursor --> AgentBudget{Agent Budget}
    AgentBudget -->|New Items| MaxNewItems[Max New Items: 25]
    AgentBudget -->|Updates| MaxUpdates[Max Project Updates: 10]
    AgentBudget -->|Comments| MaxComments[Max Comments: 10]
    
    MaxNewItems --> Process[Process Items]
    MaxUpdates --> Process
    MaxComments --> Process
    
    Process --> AgentCheck{Budget Reached?}
    AgentCheck -->|Yes| Defer[Defer Remaining]
    AgentCheck -->|No| ProcessMore[Process More]
    ProcessMore --> Process
    
    Defer --> NextRun[Next Scheduled Run]
    
    style DiscoveryBudget fill:#ffe1e1
    style AgentBudget fill:#ffe1e1
    style BudgetCheck fill:#fff4e1
    style AgentCheck fill:#fff4e1
```

## Worker-Orchestrator Communication

```mermaid
sequenceDiagram
    participant Worker
    participant Issue/PR
    participant Discovery
    participant Orchestrator
    participant Project
    
    Worker->>Issue/PR: Create with tracker-id
    Note over Issue/PR: gh-aw-tracker-id: worker-name<br/>Label: campaign:security-q1
    
    Orchestrator->>Discovery: Run precomputation
    Discovery->>Issue/PR: Search "tracker-id: worker-name"
    Issue/PR-->>Discovery: Matching items
    
    Discovery->>Manifest: Write manifest JSON
    Discovery-->>Orchestrator: Discovery complete
    
    Orchestrator->>Manifest: Read items
    Manifest-->>Orchestrator: Items with metadata
    
    Orchestrator->>Project: Add items to board
    Orchestrator->>Issue/PR: Apply campaign label
    
    Note over Issue/PR: Protected from other<br/>workflows by campaign label
```

## Validation Layers

```mermaid
graph TB
    CampaignSpec[Campaign Spec] --> Layer1[Layer 1: JSON Schema]
    
    Layer1 --> TypeCheck[Type Checking]
    Layer1 --> RequiredFields[Required Fields]
    Layer1 --> EnumValidation[Enum Validation]
    
    TypeCheck --> Layer2[Layer 2: Semantic Rules]
    RequiredFields --> Layer2
    EnumValidation --> Layer2
    
    Layer2 --> IDFormat[ID Format: lowercase-hyphen]
    Layer2 --> URLStructure[URL Structure: GitHub Project]
    Layer2 --> KPIConsistency[KPI Consistency: 1 primary]
    Layer2 --> GovernanceLimits[Governance: non-negative]
    
    IDFormat --> Layer3[Layer 3: Context Checks]
    URLStructure --> Layer3
    KPIConsistency --> Layer3
    GovernanceLimits --> Layer3
    
    Layer3 --> WorkflowExists[Workflows Exist]
    Layer3 --> LockFilesCompiled[Lock Files Compiled]
    Layer3 --> RepoMemoryValid[Repo-Memory Paths Valid]
    
    WorkflowExists --> Result{Valid?}
    LockFilesCompiled --> Result
    RepoMemoryValid --> Result
    
    Result -->|Pass| Success[✅ Validation Passed]
    Result -->|Fail| Errors[❌ Actionable Errors]
    
    style Layer1 fill:#e1f0ff
    style Layer2 fill:#fff4e1
    style Layer3 fill:#ffe1f0
    style Success fill:#e1f5e1
    style Errors fill:#ffe1e1
```

## Error Scenarios

```mermaid
graph TD
    Start[Campaign Run] --> Discovery[Discovery Phase]
    Discovery --> DiscoveryError{Error?}
    
    DiscoveryError -->|API Rate Limit| RateLimit[Save Cursor<br/>Retry Next Run]
    DiscoveryError -->|Network Failure| NetworkFail[Partial Results<br/>Don't Advance Cursor]
    DiscoveryError -->|No Error| Agent[Agent Phase]
    
    RateLimit --> NextRun[Next Scheduled Run]
    NetworkFail --> NextRun
    
    Agent --> WorkerDispatch{Dispatch Workers?}
    WorkerDispatch -->|Yes| DispatchError{Error?}
    WorkerDispatch -->|No| ProjectUpdate
    
    DispatchError -->|Workflow Missing| LogError1[Log Error<br/>Continue Other Workers]
    DispatchError -->|Permission Error| LogError2[Log Error<br/>Report in Status]
    DispatchError -->|No Error| ProjectUpdate[Project Update]
    
    LogError1 --> ProjectUpdate
    LogError2 --> ProjectUpdate
    
    ProjectUpdate --> UpdateError{Error?}
    UpdateError -->|Permission Error| StatusOnly[Skip Updates<br/>Report in Status]
    UpdateError -->|Budget Reached| Defer[Defer Items<br/>Process Next Run]
    UpdateError -->|No Error| StatusUpdate[Status Update]
    
    StatusOnly --> End[Run Complete]
    Defer --> End
    StatusUpdate --> End
    
    style DiscoveryError fill:#ffe1e1
    style DispatchError fill:#ffe1e1
    style UpdateError fill:#ffe1e1
```

## Multi-Repository Discovery (Proposed)

```mermaid
graph TB
    Config[Campaign Spec] --> AllowedRepos{allowed-repos?}
    AllowedRepos -->|Specified| RepoList[Explicit Repo List]
    AllowedRepos -->|Empty| CurrentRepo[Current Repo Only]
    
    Config --> AllowedOrgs{allowed-orgs?}
    AllowedOrgs -->|Specified| OrgList[Organization-Wide]
    AllowedOrgs -->|Empty| NoOrg[No Org Scope]
    
    RepoList --> Discovery[Discovery Script]
    CurrentRepo --> Discovery
    OrgList --> Discovery
    NoOrg --> Discovery
    
    Discovery --> ForEachRepo[For Each Repository]
    ForEachRepo --> Search[Search: repo:owner/name tracker-id]
    Search --> Items[Discovered Items]
    Items --> Normalize[Normalize Items]
    Normalize --> Dedupe[Deduplicate by URL]
    Dedupe --> Manifest[Manifest JSON]
    
    style Config fill:#e1f5e1
    style Discovery fill:#e1f0ff
    style Manifest fill:#fff4e1
```

---

## Legend

- 🟢 **Green**: User input, success states
- 🔵 **Blue**: Discovery/precomputation steps
- 🟡 **Yellow**: AI agent execution
- 🔴 **Red**: Decision points, budget checks
- 🟣 **Purple**: External systems (GitHub, Project)

## Usage

These diagrams illustrate:

1. **Campaign Lifecycle Flow** - Overall campaign execution from spec to status update
2. **Discovery Precomputation Detail** - How discovery runs before the agent
3. **Component Architecture** - Key components and their relationships
4. **Governance Budget Flow** - How budgets control campaign operations
5. **Worker-Orchestrator Communication** - Contract between workers and orchestrators
6. **Validation Layers** - Multi-layer validation approach
7. **Error Scenarios** - Common failure modes and recovery
8. **Multi-Repository Discovery** - How cross-repo discovery should work

## Tools

- **Mermaid** - All diagrams use Mermaid syntax for easy rendering
- **GitHub Markdown** - Renders automatically in GitHub
- **VS Code** - Use Mermaid preview extensions
- **Draw.io/Excalidraw** - Export to other formats if needed

## Next Steps

1. Add these diagrams to user-facing documentation
2. Create animated GIFs for complex flows
3. Add architecture decision records (ADRs)
4. Include in onboarding materials
