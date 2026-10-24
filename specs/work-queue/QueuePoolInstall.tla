-------------------------- MODULE QueuePoolInstall --------------------------
EXTENDS Naturals, TLC

\* Bounded additive first-use PoolInstall for an existing AW-managed queue.
\* Host authorization, proposal validity and verified worker-route properties
\* are explicit trusted inputs; Git publication/CAS behavior is abstracted.
CONSTANTS ExistingPool, NewPool, CallerRole, HostAuthorized,
          ProposalPreservesPolicy, NewPoolValid, RoutesValid,
          RoutesImmutable, RepositoryScoped, RoutesApproved

Roles == {"administrator", "producer", "dispatcher"}
Phases == {"requested", "route-check", "ready", "done", "rejected"}

VARIABLES installedPools, acceptedWork, policyEpoch, phase,
          routesVerified, publications
vars == <<installedPools, acceptedWork, policyEpoch, phase,
          routesVerified, publications>>

Init ==
    /\ ExistingPool # NewPool
    /\ CallerRole \in Roles
    /\ HostAuthorized \in BOOLEAN
    /\ ProposalPreservesPolicy \in BOOLEAN
    /\ NewPoolValid \in BOOLEAN
    /\ RoutesValid \in BOOLEAN
    /\ RoutesImmutable \in BOOLEAN
    /\ RepositoryScoped \in BOOLEAN
    /\ RoutesApproved \in BOOLEAN
    /\ installedPools = {ExistingPool}
    /\ acceptedWork = {"accepted"}
    /\ policyEpoch = 1
    /\ phase = "requested"
    /\ routesVerified = FALSE
    /\ publications = 0

ValidateProposal ==
    /\ phase = "requested"
    /\ phase' = IF ProposalPreservesPolicy /\ NewPoolValid
                 THEN "route-check" ELSE "rejected"
    /\ UNCHANGED <<installedPools, acceptedWork, policyEpoch,
                   routesVerified, publications>>

VerifyRoutes ==
    /\ phase = "route-check"
    /\ routesVerified' = RoutesValid /\ RoutesImmutable /\ RepositoryScoped
                          /\ RoutesApproved
    /\ phase' = IF routesVerified' THEN "ready" ELSE "rejected"
    /\ UNCHANGED <<installedPools, acceptedWork, policyEpoch, publications>>

InstallPool ==
    /\ phase = "ready"
    /\ HostAuthorized
    /\ CallerRole \in Roles
    /\ routesVerified
    \* Replay independently checks authority, even for a direct ledger request.
    /\ RoutesApproved
    /\ ProposalPreservesPolicy /\ NewPoolValid
    /\ NewPool \notin installedPools
    /\ installedPools' = installedPools \cup {NewPool}
    /\ acceptedWork' = acceptedWork
    /\ policyEpoch' = policyEpoch
    /\ phase' = "done"
    /\ publications' = publications + 1
    /\ UNCHANGED routesVerified

Next == ValidateProposal \/ VerifyRoutes \/ InstallPool
Spec == Init /\ [][Next]_vars

TypeOK ==
    /\ installedPools \subseteq {ExistingPool, NewPool}
    /\ acceptedWork = {"accepted"}
    /\ policyEpoch \in Nat
    /\ phase \in Phases
    /\ routesVerified \in BOOLEAN
    /\ publications \in 0..1

ExistingPoolPreserved == ExistingPool \in installedPools
AcceptedWorkPreserved == acceptedWork = {"accepted"}
PolicyEpochPreserved == policyEpoch = 1
SingleAdditiveInstall ==
    /\ (NewPool \in installedPools) = (publications = 1)
    /\ publications \in 0..1
PublicationAuthority ==
    publications = 0 \/
        /\ HostAuthorized
        /\ CallerRole \in Roles
        /\ routesVerified
        /\ ProposalPreservesPolicy
        /\ NewPoolValid
        /\ RoutesValid
        /\ RoutesImmutable
        /\ RepositoryScoped
        /\ RoutesApproved
Safety ==
    /\ TypeOK
    /\ ExistingPoolPreserved
    /\ AcceptedWorkPreserved
    /\ PolicyEpochPreserved
    /\ SingleAdditiveInstall
    /\ PublicationAuthority

\* A deliberately false invariant demonstrates the producer path is reachable.
NoProducerInstall ==
    ~(CallerRole = "producer" /\ NewPool \in installedPools)
=============================================================================
