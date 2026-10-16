----------------------------- MODULE MCPCompile -----------------------------
EXTENDS TLC

CONSTANT Fault

Faults == {
    "none", "caller-opt-in", "actor-bypass", "option-bypass",
    "no-emit-omitted", "update-check-enabled", "scanner-before-gate",
    "missing-import-download", "diagnostic-write", "post-processing-write",
    "cache-hit-write", "cwd-cache-root", "read-only-denied",
    "validation-bypass", "write-output-omitted"
}
ASSUME Fault \in Faults

Tools == {"compile", "add", "update", "fix"}
Options == {"none", "fix", "dry-run", "scanner"}
Contexts == {"repository-root", "repository-subdirectory", "non-git"}
Mutations == {"source", "lock", "diagnostic", "action-pins", "attributes",
              "import-cache", "update-cache"}
Requests == {r \in [tool : Tools, allowWriteTools : BOOLEAN,
                   validateActor : BOOLEAN, actorWrite : BOOLEAN,
                   option : Options, valid : BOOLEAN,
                   imports : {"none", "cached", "missing"}, context : Contexts] :
             r.tool = "compile" \/ r.option = "none"}

VARIABLE s
vars == <<s>>

ActorAllowed == ~s.request.validateActor \/ s.request.actorWrite
EffectiveWriteTools == s.request.allowWriteTools \/ Fault = "caller-opt-in"
Authorized ==
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
             writeEnabled |-> FALSE, noEmit |-> TRUE, updateCheck |-> FALSE,
             scannerStarted |-> FALSE, downloaded |-> FALSE,
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
                             !.noEmit = ~writable /\ Fault # "no-emit-omitted",
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
           download == missing /\ (~s.noEmit \/ Fault = "missing-import-download")
           cacheWrite == download
                         \/ (s.request.imports = "cached" /\ Fault = "cache-hit-write")
       IN s' = [s EXCEPT !.phase = IF missing /\ ~download
                                  THEN "finalize" ELSE "validate",
                        !.result = IF missing /\ ~download THEN "error" ELSE @,
                        !.downloaded = download,
                        !.mutations = IF cacheWrite THEN @ \cup {"import-cache"} ELSE @,
                        !.lastEvent = IF missing /\ ~download
                                      THEN "UncachedImportError" ELSE "ResolveImports"]

Validate ==
    /\ s.phase = "validate"
    /\ s' = [s EXCEPT !.phase = "finalize",
                     !.result = IF s.request.valid \/ Fault = "validation-bypass"
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

Next == Authorize \/ Prepare \/ ResolveImports \/ Validate \/ Finalize
Spec == Init /\ [][Next]_vars

TypeOK ==
    s \in [request : Requests,
           phase : {"authorize", "prepare", "imports", "validate", "finalize", "done"},
           started : BOOLEAN, writeEnabled : BOOLEAN, noEmit : BOOLEAN,
           updateCheck : BOOLEAN, scannerStarted : BOOLEAN, downloaded : BOOLEAN,
           cacheRoot : {"unset", "repository", "subdirectory", "working-directory"},
           mutations : SUBSET Mutations, result : {"pending", "denied", "success", "error"},
           lastEvent : {"Init", "DenyBeforeExecution", "Authorize", "PrepareCompile",
                        "WriteManagementOperation", "UncachedImportError",
                        "ResolveImports", "ValidateWorkflow", "FinishCompile"}]
StartupCapability ==
    /\ (s.writeEnabled => s.request.allowWriteTools)
    /\ (s.started /\ s.request.tool # "compile" => s.request.allowWriteTools)
ActorAuthorization ==
    s.started /\ (s.request.tool # "compile" \/ s.request.allowWriteTools) => ActorAllowed
WritingOptionsGated ==
    s.started /\ s.request.tool = "compile" /\ s.request.option # "none"
    => s.request.allowWriteTools
ReadOnlyMode ==
    s.started /\ s.request.tool = "compile" /\ ~s.request.allowWriteTools
    => s.noEmit /\ ~s.updateCheck
NoExternalScanners ==
    ~s.request.allowWriteTools => ~s.scannerStarted
NoRemoteDownloads ==
    ~s.request.allowWriteTools => ~s.downloaded
ReadOnlyIsolation ==
    ~s.request.allowWriteTools => s.mutations = {}
CacheRootConforms ==
    s.cacheRoot # "unset" => s.cacheRoot = ExpectedCacheRoot
ReadOnlyAvailability ==
    /\ s.phase = "done" /\ s.request.tool = "compile"
    /\ ~s.request.allowWriteTools /\ s.request.option = "none"
    /\ s.request.valid /\ s.request.imports # "missing"
    => s.result = "success"
ValidationPreserved ==
    s.phase = "done" /\ s.request.tool = "compile" /\ s.started /\ ~s.request.valid
    => s.result = "error"
DeniedBeforeExecution ==
    s.result = "denied" => ~s.started /\ ~s.scannerStarted /\ s.mutations = {}
WriteEnabledEmission ==
    /\ s.phase = "done" /\ s.request.tool = "compile"
    /\ s.request.allowWriteTools /\ s.result = "success"
    => "lock" \in s.mutations

\* Negated reachability claims: appended individually by the runner.
NoReadOnlySuccess ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ ~s.request.allowWriteTools /\ s.result = "success")
NoCachedSubdirectorySuccess ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ ~s.request.allowWriteTools /\ s.request.imports = "cached"
      /\ s.request.context = "repository-subdirectory" /\ s.result = "success")
NoMissingImportError ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ ~s.request.allowWriteTools /\ s.request.imports = "missing"
      /\ s.result = "error" /\ s.mutations = {})
NoWriteEnabledSuccess ==
    ~(s.phase = "done" /\ s.request.tool = "compile"
      /\ s.request.allowWriteTools /\ s.result = "success" /\ "lock" \in s.mutations)
NoActorDenial ==
    ~(s.phase = "done" /\ s.request.allowWriteTools
      /\ s.request.validateActor /\ ~s.request.actorWrite /\ s.result = "denied")
=============================================================================
