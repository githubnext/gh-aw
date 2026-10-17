----------------------------- MODULE MCPCompile -----------------------------
EXTENDS TLC

CONSTANT Fault

Faults == {
    "none", "caller-opt-in", "actor-bypass", "option-bypass",
    "no-emit-omitted", "update-check-enabled", "scanner-before-gate",
    "missing-import-download", "diagnostic-write", "post-processing-write",
    "cache-hit-write", "cwd-cache-root", "read-only-denied",
    "validation-bypass", "write-output-omitted", "read-only-mode-omitted",
    "docker-pull", "include-cache-bypass", "no-emit-cache-policy",
    "stale-lock-validation"
}
ASSUME Fault \in Faults

Tools == {"compile", "add", "update", "fix"}
Options == {"none", "fix", "dry-run", "scanner"}
Contexts == {"repository-root", "repository-subdirectory", "non-git"}
Mutations == {"source", "lock", "diagnostic", "action-pins", "attributes",
              "import-cache", "update-cache", "temporary-include"}
Requests == {r \in [entry : {"mcp", "cli-no-emit"},
                   tool : Tools, allowWriteTools : BOOLEAN,
                   validateActor : BOOLEAN, actorWrite : BOOLEAN,
                   option : Options, valid : BOOLEAN,
                   imports : {"none", "cached", "missing"}, context : Contexts,
                   includePath : {"frontmatter", "body", "nested"},
                   image : {"none", "cached", "missing", "required-missing"},
                   staleLock : BOOLEAN] :
             /\ (r.tool = "compile" \/ r.option = "none")
             /\ (r.entry = "cli-no-emit" =>
                    r.tool = "compile" /\ r.option = "none" /\ r.allowWriteTools
                    /\ ~r.validateActor /\ r.actorWrite)}

VARIABLE s
vars == <<s>>

ActorAllowed == ~s.request.validateActor \/ s.request.actorWrite
ReadOnlyRequest == s.request.entry = "mcp" /\ ~s.request.allowWriteTools
EffectiveWriteTools == s.request.allowWriteTools \/ Fault = "caller-opt-in"
Authorized ==
    s.request.entry = "cli-no-emit" \/
        /\ (s.request.tool = "compile" \/ EffectiveWriteTools)
        /\ (~EffectiveWriteTools \/ ActorAllowed \/ Fault = "actor-bypass")
OptionAllowed ==
    s.request.tool # "compile" \/ s.request.option = "none"
    \/ EffectiveWriteTools \/ Fault = "option-bypass"
ExpectedCacheRoot ==
    IF s.request.context = "non-git" THEN "working-directory" ELSE "repository"

Init ==
    \E r \in Requests :
        s = [request |-> r, phase |-> "authorize", started |-> FALSE,
             writeEnabled |-> FALSE, noEmit |-> TRUE, readOnly |-> FALSE,
             updateCheck |-> FALSE, scannerStarted |-> FALSE,
             downloaded |-> FALSE, pulled |-> FALSE,
             cacheRoot |-> "unset", mutations |-> {},
             result |-> "pending", lastEvent |-> "Init"]

Authorize ==
    /\ s.phase = "authorize"
    /\ LET denied == ~Authorized \/ ~OptionAllowed
                      \/ (Fault = "read-only-denied"
                          /\ ~s.request.allowWriteTools)
           scanner == s.request.tool = "compile" /\ s.request.option = "scanner"
           writable == EffectiveWriteTools
           update == s.request.tool = "compile"
                     /\ (writable \/ Fault = "update-check-enabled")
       IN IF denied
          THEN s' = [s EXCEPT !.phase = "done", !.result = "denied",
                             !.scannerStarted = scanner /\ Fault = "scanner-before-gate",
                             !.lastEvent = "DenyBeforeExecution"]
          ELSE s' = [s EXCEPT !.phase = "prepare", !.started = TRUE,
                             !.writeEnabled = writable,
                             !.noEmit = (s.request.entry = "cli-no-emit" \/ ~writable)
                                        /\ (~ReadOnlyRequest \/ Fault # "no-emit-omitted"),
                             !.readOnly = (ReadOnlyRequest /\ Fault # "read-only-mode-omitted")
                                          \/ (s.request.entry = "cli-no-emit"
                                              /\ Fault = "no-emit-cache-policy"),
                             !.updateCheck = update, !.scannerStarted = scanner,
                             !.mutations = IF update THEN {"update-cache"} ELSE {},
                             !.lastEvent = "Authorize"]

Prepare ==
    /\ s.phase = "prepare"
    /\ IF s.request.tool # "compile"
       THEN s' = [s EXCEPT !.phase = "done", !.result = "success",
                          !.mutations = @ \cup {"source"},
                          !.lastEvent = "WriteManagementOperation"]
       ELSE s' = [s EXCEPT !.phase = "imports",
                          !.cacheRoot = IF Fault = "cwd-cache-root"
                                          /\ s.request.context = "repository-subdirectory"
                                       THEN "subdirectory" ELSE ExpectedCacheRoot,
                          !.mutations = IF s.request.option = "fix"
                                        THEN @ \cup {"source"} ELSE @,
                          !.lastEvent = "PrepareCompile"]

ResolveImports ==
    /\ s.phase = "imports"
    /\ LET missing == s.request.imports = "missing"
                      \/ (s.request.imports = "cached" /\ s.cacheRoot # ExpectedCacheRoot)
                      \/ (s.request.imports # "none" /\ s.request.includePath # "frontmatter"
                          /\ Fault = "include-cache-bypass")
           download == missing /\ (~s.readOnly \/ Fault = "missing-import-download"
                                    \/ Fault = "include-cache-bypass")
           cacheWrite == download
                         \/ (s.request.imports = "cached" /\ Fault = "cache-hit-write")
           tempWrite == download /\ s.request.includePath # "frontmatter"
                        /\ Fault = "include-cache-bypass"
       IN s' = [s EXCEPT !.phase = IF missing /\ ~download
                                  THEN "finalize" ELSE "images",
                        !.result = IF missing /\ ~download THEN "error" ELSE @,
                        !.downloaded = download,
                        !.mutations = @
                            \cup (IF cacheWrite THEN {"import-cache"} ELSE {})
                            \cup (IF tempWrite THEN {"temporary-include"} ELSE {}),
                        !.lastEvent = IF missing /\ ~download
                                      THEN "UncachedImportError" ELSE "ResolveImports"]

ResolveImages ==
    /\ s.phase = "images"
    /\ LET missing == s.request.image \in {"missing", "required-missing"}
           pull == missing /\ (~s.readOnly \/ Fault = "docker-pull")
           requiredError == s.request.image = "required-missing" /\ ~pull
       IN s' = [s EXCEPT !.phase = IF requiredError THEN "finalize" ELSE "validate",
                        !.result = IF requiredError THEN "error" ELSE @,
                        !.pulled = pull,
                        !.lastEvent = IF missing /\ ~pull
                                      THEN IF requiredError THEN "MissingLocalImageError"
                                                            ELSE "MissingLocalImageWarning"
                                      ELSE "InspectImages"]

Validate ==
    /\ s.phase = "validate"
    /\ s' = [s EXCEPT !.phase = "finalize",
                     !.result = IF (s.request.valid \/ Fault = "validation-bypass")
                                   /\ ~(ReadOnlyRequest /\ s.noEmit /\ s.request.staleLock
                                        /\ Fault = "stale-lock-validation")
                                THEN "success" ELSE "error",
                     !.lastEvent = "ValidateWorkflow"]

Finalize ==
    /\ s.phase = "finalize"
    /\ LET lock == s.result = "success" /\ ~s.noEmit
                   /\ Fault # "write-output-omitted"
           diagnostic == s.result = "error"
                         /\ (~s.noEmit \/ Fault = "diagnostic-write")
           post == ~s.noEmit \/ Fault = "post-processing-write"
       IN s' = [s EXCEPT !.phase = "done",
                        !.mutations = @
                            \cup (IF lock THEN {"lock"} ELSE {})
                            \cup (IF diagnostic THEN {"diagnostic"} ELSE {})
                            \cup (IF post THEN {"action-pins", "attributes"} ELSE {}),
                        !.lastEvent = "FinishCompile"]

Next == Authorize \/ Prepare \/ ResolveImports \/ ResolveImages \/ Validate \/ Finalize
Spec == Init /\ [][Next]_vars

TypeOK ==
    s \in [request : Requests,
           phase : {"authorize", "prepare", "imports", "images", "validate", "finalize", "done"},
           started : BOOLEAN, writeEnabled : BOOLEAN, noEmit : BOOLEAN,
           readOnly : BOOLEAN, pulled : BOOLEAN,
           updateCheck : BOOLEAN, scannerStarted : BOOLEAN, downloaded : BOOLEAN,
           cacheRoot : {"unset", "repository", "subdirectory", "working-directory"},
           mutations : SUBSET Mutations, result : {"pending", "denied", "success", "error"},
           lastEvent : {"Init", "DenyBeforeExecution", "Authorize", "PrepareCompile",
                        "WriteManagementOperation", "UncachedImportError",
                        "ResolveImports", "InspectImages", "MissingLocalImageError",
                        "MissingLocalImageWarning",
                        "ValidateWorkflow", "FinishCompile"}]
StartupCapability ==
    /\ (s.writeEnabled => s.request.allowWriteTools)
    /\ (s.started /\ s.request.tool # "compile" => s.request.allowWriteTools)
ActorAuthorization ==
    s.request.entry = "mcp" /\ s.started
    /\ (s.request.tool # "compile" \/ s.request.allowWriteTools) => ActorAllowed
WritingOptionsGated ==
    s.started /\ s.request.tool = "compile" /\ s.request.option # "none"
    => s.request.allowWriteTools
ReadOnlyMode ==
    s.started /\ s.request.tool = "compile" /\ ReadOnlyRequest
    => s.noEmit /\ s.readOnly /\ ~s.updateCheck
NoExternalScanners ==
    ReadOnlyRequest => ~s.scannerStarted
NoRemoteDownloads ==
    ReadOnlyRequest => ~s.downloaded
NoImagePulls ==
    ReadOnlyRequest => ~s.pulled
ReadOnlyIsolation ==
    ReadOnlyRequest => s.mutations = {}
CacheRootConforms ==
    s.cacheRoot # "unset" => s.cacheRoot = ExpectedCacheRoot
ReadOnlyAvailability ==
    /\ s.phase = "done" /\ s.request.tool = "compile"
    /\ ReadOnlyRequest /\ s.request.option = "none"
    /\ s.request.valid /\ s.request.imports # "missing" /\ s.request.image # "required-missing"
    => s.result = "success"
ValidationPreserved ==
    s.phase = "done" /\ s.request.tool = "compile" /\ s.started /\ ~s.request.valid
    => s.result = "error"
DeniedBeforeExecution ==
    s.result = "denied" => ~s.started /\ ~s.scannerStarted /\ s.mutations = {}
WriteEnabledEmission ==
    /\ s.phase = "done" /\ s.request.tool = "compile" /\ s.request.entry = "mcp"
    /\ s.request.allowWriteTools /\ s.result = "success"
    => "lock" \in s.mutations
TrustedNoEmitAvailability ==
    s.phase = "done" /\ s.request.entry = "cli-no-emit" /\ s.request.valid
    => s.result = "success"

\* Negated reachability claims: appended individually by the runner.
NoReadOnlySuccess ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ ReadOnlyRequest /\ s.result = "success")
NoCachedSubdirectorySuccess ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ ReadOnlyRequest /\ s.request.imports = "cached"
      /\ s.request.context = "repository-subdirectory" /\ s.result = "success")
NoMissingImportError ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ ReadOnlyRequest /\ s.request.imports = "missing"
      /\ s.result = "error" /\ s.mutations = {})
NoWriteEnabledSuccess ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ s.request.allowWriteTools /\ s.result = "success" /\ "lock" \in s.mutations)
NoActorDenial ==
    ~(s.phase = "done" /\ s.request.entry = "mcp" /\ s.request.allowWriteTools
      /\ s.request.validateActor /\ ~s.request.actorWrite /\ s.result = "denied")
NoNestedIncludeSuccess ==
    ~(s.phase = "done" /\ ReadOnlyRequest /\ s.request.imports = "cached"
      /\ s.request.includePath = "nested" /\ s.result = "success")
NoTrustedNoEmitImportRefresh ==
    ~(s.phase = "done" /\ s.request.entry = "cli-no-emit"
      /\ s.request.imports = "missing" /\ s.noEmit /\ s.downloaded
      /\ s.result = "success" /\ "import-cache" \in s.mutations)
=============================================================================
