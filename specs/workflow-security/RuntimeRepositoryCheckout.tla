---------------------- MODULE RuntimeRepositoryCheckout ----------------------
EXTENDS Naturals, FiniteSets, Sequences, TLC

CONSTANT Fault
ASSUME Fault \in {"none", "untrusted-policy", "missing-allowlist",
                 "selector-wildcard", "unrestricted-policy", "raw-allowlist",
                 "repository-limit", "canonical-allowlist", "visibility", "path-bypass",
                 "partial-preflight", "recursive-fetch", "cleanup-fail-open",
                 "manifest-omitted", "fallback-token", "neighbor-token",
                 "write-credentials-stripped", "write-allowlist",
                 "staged-checkout", "failure-no-jobs"}

Jobs == {"agent", "safe_outputs"}
Repos == {"org/a", "org/b", "outside/c"}
None == "none"
EntryAuth == {"inherit", "pat", "read-app", "write-app", "read-and-write-app",
              "missing-read-app", "missing-write-app"}
Fallbacks == {"job-default", "operation-pat", "shared-app", "shared-pat"}
Tokens == Fallbacks \cup {"entry-pat", "entry-read-app", "entry-write-app",
                         "other-entry-app", None}
TrustedPolicy == {"literal", "variable"}

\* Finite input classes abstract the parsers, wildcard matcher, and API responses.
BaseInput ==
    [selector |-> "expression", policySource |-> "literal",
     allowed |-> {"org/a", "org/b"}, unrestricted |-> FALSE,
     values |-> <<"org/a">>, limit |-> 10, limitSyntaxValid |-> TRUE,
     prefix |-> "targets", prefixValid |-> TRUE, pathsSafe |-> TRUE,
     hostVisibility |-> "public",
     visibility |-> [r \in Repos |-> "public"],
     metadataVerified |-> TRUE, redirect |-> None,
     lfs |-> FALSE, submodules |-> FALSE, appScopeExplicit |-> TRUE,
     staged |-> FALSE, noRecordedJobs |-> TRUE,
     writeAllowed |-> {"org/a"}]

Inputs == {BaseInput,
    [BaseInput EXCEPT !.policySource = "variable"],
    [BaseInput EXCEPT !.policySource = "caller"],
    [BaseInput EXCEPT !.policySource = "composite"],
    [BaseInput EXCEPT !.policySource = "missing"],
    [BaseInput EXCEPT !.selector = "hosting", !.policySource = "missing",
                     !.prefixValid = FALSE],
    [BaseInput EXCEPT !.selector = "literal", !.policySource = "missing"],
    [BaseInput EXCEPT !.selector = "hosting"],
    [BaseInput EXCEPT !.selector = "wildcard"],
    [BaseInput EXCEPT !.values = <<"org/*">>],
    [BaseInput EXCEPT !.values = <<>>],
    [BaseInput EXCEPT !.values = <<"org/a", "ORG/A">>, !.limit = 1],
    [BaseInput EXCEPT !.values = <<"org/a", "org/b">>, !.limit = 2],
    [BaseInput EXCEPT !.values = <<"org/a", "org/b">>, !.limit = 1],
    [BaseInput EXCEPT !.limitSyntaxValid = FALSE],
    [BaseInput EXCEPT !.allowed = {}],
    [BaseInput EXCEPT !.values = <<"outside/c">>],
    [BaseInput EXCEPT !.redirect = "outside/c"],
    [BaseInput EXCEPT !.redirect = "org/b"],
    [BaseInput EXCEPT !.policySource = "variable", !.unrestricted = TRUE],
    [BaseInput EXCEPT !.unrestricted = TRUE],
    [BaseInput EXCEPT !.metadataVerified = FALSE],
    [BaseInput EXCEPT !.hostVisibility = "unknown"],
    [BaseInput EXCEPT !.visibility["org/a"] = "unknown"],
    [BaseInput EXCEPT !.values = <<"org/a", "org/b">>,
                     !.visibility["org/b"] = "private"],
    [BaseInput EXCEPT !.visibility["org/a"] = "internal"],
    [BaseInput EXCEPT !.hostVisibility = "private",
                     !.values = <<"org/b">>, !.visibility["org/b"] = "private"],
    [BaseInput EXCEPT !.hostVisibility = "internal",
                     !.visibility["org/a"] = "internal"],
    [BaseInput EXCEPT !.prefixValid = FALSE],
    [BaseInput EXCEPT !.pathsSafe = FALSE],
    [BaseInput EXCEPT !.prefix = "one/two/three/four/five/six/targets"],
    [BaseInput EXCEPT !.lfs = TRUE],
    [BaseInput EXCEPT !.submodules = TRUE],
    [BaseInput EXCEPT !.appScopeExplicit = FALSE],
    [BaseInput EXCEPT !.staged = TRUE],
    [BaseInput EXCEPT !.noRecordedJobs = FALSE]}

Runtime(i) == i.policySource # "missing" \/ i.selector = "expression"
Normalize(r) == IF r = "ORG/A" THEN "org/a" ELSE r
Selected(i) == {Normalize(i.values[n]) : n \in 1..Len(i.values)}
Canonical(i, r) == IF r = "org/a" /\ i.redirect # None THEN i.redirect ELSE r
CanonicalRepos(i) == {Canonical(i, r) : r \in Selected(i)}
RepoPath(i, r) == i.prefix \o "/" \o r
Manifest(i, rs) ==
    {[repository |-> r, path |-> RepoPath(i, r), default_branch |-> "main"] : r \in rs}

ExpectedCompile(i) ==
    /\ i.selector # "wildcard"
    /\ IF Runtime(i)
       THEN /\ i.policySource \in TrustedPolicy
            /\ (i.policySource = "variable" \/ ~i.unrestricted)
            /\ i.limitSyntaxValid /\ i.prefixValid /\ i.appScopeExplicit
            /\ ~i.lfs /\ ~i.submodules
       ELSE i.selector \in {"hosting", "literal"}
ObservedCompile(i) ==
    ExpectedCompile(i) \/
    (Fault = "untrusted-policy" /\ i.policySource \in {"caller", "composite"}
        /\ ExpectedCompile([i EXCEPT !.policySource = "literal"])) \/
    (Fault = "missing-allowlist" /\ i.selector = "expression"
        /\ i.policySource = "missing"
        /\ ExpectedCompile([i EXCEPT !.policySource = "literal"])) \/
    (Fault = "selector-wildcard" /\ i.selector = "wildcard"
        /\ ExpectedCompile([i EXCEPT !.selector = "literal"]))

ExpectedToken(job, entry, fallback) ==
    CASE job = "safe_outputs" /\ entry \in {"write-app", "read-and-write-app"}
            -> "entry-write-app"
      [] entry = "pat" -> "entry-pat"
      [] entry = "read-app" \/ (job = "agent" /\ entry = "read-and-write-app")
            -> "entry-read-app"
      [] entry = "missing-read-app" -> "job-default"
      [] job = "safe_outputs" -> fallback
      [] OTHER -> "job-default"
ObservedToken(job, entry, fallback) ==
    IF job = "safe_outputs" /\ entry = "inherit" /\ Fault = "fallback-token"
    THEN "job-default"
    ELSE IF job = "safe_outputs" /\ entry = "inherit" /\ Fault = "neighbor-token"
    THEN "other-entry-app"
    ELSE ExpectedToken(job, entry, fallback)

VisibilityAllowed(i) ==
    /\ i.hostVisibility \in {"public", "private", "internal"}
    /\ \A r \in CanonicalRepos(i):
        /\ i.visibility[r] \in {"public", "private", "internal"}
        /\ (i.hostVisibility = "public" => i.visibility[r] = "public")
PreflightAllowed(i) ==
    /\ Selected(i) \subseteq Repos
    /\ (~i.unrestricted \/ Fault = "unrestricted-policy")
    /\ i.allowed # {}
    /\ (Selected(i) \subseteq i.allowed \/ Fault = "raw-allowlist")
    /\ (Cardinality(Selected(i)) <= i.limit \/ Fault = "repository-limit")
MetadataAllowed(i) ==
    /\ i.metadataVerified /\ (i.pathsSafe \/ Fault = "path-bypass")
    /\ (CanonicalRepos(i) \subseteq i.allowed \/ Fault = "canonical-allowlist")
    /\ (VisibilityAllowed(i) \/ Fault = "visibility")

VARIABLE s
vars == <<s>>
Init ==
    \E i \in Inputs, j \in Jobs, entry \in EntryAuth, fallback \in Fallbacks:
        s = [input |-> i, job |-> j, entry |-> entry, fallback |-> fallback,
             phase |-> "compile", compiled |-> FALSE, metadataRead |-> FALSE,
             validated |-> {}, checkouts |-> {}, token |-> None,
             retained |-> FALSE, finalized |-> FALSE, manifest |-> {},
             engineStarted |-> FALSE, handlerStarted |-> FALSE,
             writes |-> {}, indirectReads |-> FALSE,
             failureOutput |-> FALSE, failureIssue |-> FALSE, lastEvent |-> "Init"]

Compile ==
    /\ s.phase = "compile"
    /\ LET accepted == ObservedCompile(s.input)
       IN s' = [s EXCEPT
           !.compiled = accepted,
           !.phase = IF ~accepted THEN "compile-rejected"
                     ELSE IF s.job = "safe_outputs" /\ s.input.staged
                             /\ Fault # "staged-checkout" THEN "skipped"
                     ELSE IF ~Runtime(s.input) THEN "static" ELSE "preflight",
           !.lastEvent = "Compile"]
Preflight ==
    /\ s.phase = "preflight"
    /\ LET accepted == PreflightAllowed(s.input)
       IN s' = [s EXCEPT
           !.phase = IF accepted THEN "metadata" ELSE "failure-pending",
           !.metadataRead = accepted,
           !.token = IF accepted THEN ObservedToken(s.job, s.entry, s.fallback) ELSE @,
           !.failureOutput = ~accepted, !.lastEvent = "Preflight"]
ValidateMetadata ==
    /\ s.phase = "metadata"
    /\ LET accepted == MetadataAllowed(s.input)
       IN s' = [s EXCEPT
           !.phase = IF accepted THEN "checkout" ELSE "failure-pending",
           !.validated = IF accepted THEN CanonicalRepos(s.input) ELSE {},
           !.failureOutput = ~accepted, !.lastEvent = "ValidateMetadata"]
Checkout(r) ==
    /\ s.phase = "checkout" /\ r \in s.validated \ s.checkouts
    /\ s' = [s EXCEPT !.checkouts = @ \cup {r}, !.retained = TRUE,
                     !.lastEvent = "Checkout:" \o r]
EarlyCheckout(r) ==
    /\ Fault = "partial-preflight" /\ s.phase = "metadata"
    /\ r \in CanonicalRepos(s.input) \ s.checkouts
    /\ s' = [s EXCEPT !.checkouts = @ \cup {r}, !.retained = TRUE,
                     !.lastEvent = "EarlyCheckout:" \o r]
CheckoutComplete ==
    /\ s.phase = "checkout" /\ s.checkouts = s.validated
    /\ s' = [s EXCEPT !.phase = "finalize", !.lastEvent = "CheckoutComplete"]
Finalize(success) ==
    /\ s.phase = "finalize"
    /\ LET accepted == success \/ (s.job = "agent" /\ Fault = "cleanup-fail-open")
       IN s' = [s EXCEPT
           !.phase = IF accepted THEN "manifest" ELSE "failure-pending",
           !.finalized = accepted,
           !.retained = IF s.job = "agent" /\ success THEN FALSE
                        ELSE IF s.job = "safe_outputs" /\ Fault = "write-credentials-stripped"
                             THEN FALSE ELSE @,
           !.indirectReads = accepted /\ s.checkouts # {} /\ Fault = "recursive-fetch",
           !.failureOutput = ~accepted, !.lastEvent = "Finalize"]
PublishManifest ==
    /\ s.phase = "manifest"
    /\ s' = [s EXCEPT
        !.manifest = IF Fault = "manifest-omitted" THEN {} ELSE Manifest(s.input, s.checkouts),
        !.phase = "consumer", !.lastEvent = "PublishManifest"]
Consume(r) ==
    /\ s.phase = "consumer" /\ r \in Repos
    /\ s' = [s EXCEPT
        !.engineStarted = s.job = "agent", !.handlerStarted = s.job = "safe_outputs",
        !.writes = IF s.job = "safe_outputs" /\ r \in s.checkouts
                     /\ (r \in s.input.writeAllowed \/ Fault = "write-allowlist")
                   THEN {r} ELSE {},
        !.phase = "complete", !.lastEvent = "Consume:" \o r]
ReportFailure ==
    /\ s.phase = "failure-pending"
    /\ s' = [s EXCEPT
        !.failureIssue = ~(Fault = "failure-no-jobs" /\ s.input.noRecordedJobs),
        !.phase = "failed", !.lastEvent = "ReportFailure"]
Next ==
    \/ Compile \/ Preflight \/ ValidateMetadata
    \/ \E r \in Repos: Checkout(r) \/ EarlyCheckout(r) \/ Consume(r)
    \/ CheckoutComplete \/ \E success \in BOOLEAN: Finalize(success)
    \/ PublishManifest \/ ReportFailure
Spec == Init /\ [][Next]_vars

TypeOK ==
    /\ s.input \in Inputs /\ s.job \in Jobs /\ s.entry \in EntryAuth
    /\ s.fallback \in Fallbacks /\ s.token \in Tokens
    /\ s.phase \in {"compile", "compile-rejected", "static", "skipped",
                    "preflight", "metadata", "checkout", "finalize", "manifest",
                    "consumer", "complete", "failure-pending", "failed"}
    /\ s.validated \subseteq Repos /\ s.checkouts \subseteq Repos /\ s.writes \subseteq Repos
    /\ s.manifest \subseteq Manifest(s.input, Repos)
    /\ \A b \in {s.compiled, s.metadataRead, s.retained, s.finalized,
                 s.engineStarted, s.handlerStarted, s.indirectReads,
                 s.failureOutput, s.failureIssue}: b \in BOOLEAN
MandatoryAllowlist ==
    s.compiled /\ s.input.selector = "expression" => s.input.policySource # "missing"
PolicyProvenance ==
    s.compiled /\ Runtime(s.input) => s.input.policySource \in TrustedPolicy
ExplicitSelector == s.compiled => s.input.selector # "wildcard"
ScopedPolicy == s.metadataRead => ~s.input.unrestricted /\ s.input.allowed # {}
SelectedRepositoriesAllowed == s.metadataRead => Selected(s.input) \subseteq s.input.allowed
BoundedSelection == s.metadataRead => Cardinality(Selected(s.input)) <= s.input.limit
BatchValidatedBeforeCheckout ==
    s.checkouts # {} => s.validated = CanonicalRepos(s.input)
ValidatedPathsSafe == s.validated # {} => s.input.prefixValid /\ s.input.pathsSafe
CanonicalRepositoriesAllowed == s.validated \subseteq s.input.allowed
VisibilityIsolation == s.validated # {} => VisibilityAllowed(s.input) /\ s.input.metadataVerified
EntryAuthentication ==
    s.metadataRead => s.token = ExpectedToken(s.job, s.entry, s.fallback)
NoSecondaryReads == ~s.indirectReads
AgentCredentialsCleared ==
    (s.job = "agent" /\ s.finalized) \/ s.engineStarted => ~s.retained
ManifestPrecedesConsumers ==
    s.engineStarted \/ s.handlerStarted =>
        s.finalized /\ s.manifest = Manifest(s.input, s.checkouts)
PrivilegedCredentialsRetained ==
    s.handlerStarted /\ s.checkouts # {} => s.retained
SafeOutputAuthorization == s.writes \subseteq s.input.writeAllowed /\ s.writes \subseteq s.checkouts
StagedIsolation ==
    s.job = "safe_outputs" /\ s.input.staged =>
        ~s.metadataRead /\ s.token = None /\ s.checkouts = {} /\ s.manifest = {} /\ s.writes = {}
CheckoutFailureReported == s.phase = "failed" => s.failureOutput /\ s.failureIssue

\* False predicates establish non-vacuous reachability, not security requirements.
NoRuntimeCheckout == s.checkouts = {}
NoDeepManifestConsumer ==
    ~(s.input.prefix # "targets" /\ s.manifest # {} /\ s.handlerStarted /\ s.writes # {})
NoKnownCheckoutFailure == ~s.failureIssue
NoHostingException ==
    ~(s.phase = "static" /\ s.input.selector = "hosting" /\ s.input.policySource = "missing")
NoPrivateHostCheckout ==
    ~(s.input.hostVisibility = "private" /\ "org/b" \in s.checkouts)
=============================================================================
