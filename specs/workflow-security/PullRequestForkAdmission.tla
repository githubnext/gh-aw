------------------------- MODULE PullRequestForkAdmission -------------------------
EXTENDS FiniteSets, TLC

CONSTANT Fault
ASSUME Fault \in {"none", "default-allows-forks", "association-ignored",
                  "same-repository-denied", "non-pr-filtered",
                  "association-grants-permissions"}

Triggers == {"pull_request", "other"}
ForkModes == {"default", "patterns", "author-association"}
Associations == {"OWNER", "MEMBER", "COLLABORATOR", "CONTRIBUTOR",
                 "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MANNEQUIN", "NONE"}
Permissions == {"read", "write"}

VARIABLES trigger, sameRepository, forkMode, repositoryMatches, wildcard,
          authorAssociation, allowedAssociations, configuredPermissions
vars == <<trigger, sameRepository, forkMode, repositoryMatches, wildcard,
          authorAssociation, allowedAssociations, configuredPermissions>>

ExpectedAdmission ==
    trigger # "pull_request" \/
    sameRepository \/
    (forkMode = "patterns" /\ (wildcard \/ repositoryMatches)) \/
    (forkMode = "author-association" /\
      authorAssociation \in allowedAssociations)

ObservedAdmission ==
    IF trigger # "pull_request"
    THEN Fault # "non-pr-filtered"
    ELSE IF Fault = "default-allows-forks" /\ forkMode = "default" /\
            ~sameRepository
         THEN TRUE
         ELSE IF Fault = "same-repository-denied" /\ sameRepository
              THEN FALSE
              ELSE IF Fault = "association-ignored" /\
                      forkMode = "author-association" /\
                     authorAssociation \in allowedAssociations /\
                     ~sameRepository
                  THEN FALSE
                  ELSE ExpectedAdmission

ObservedPermissions ==
    IF Fault = "association-grants-permissions" /\
       trigger = "pull_request" /\ forkMode = "author-association" /\
       authorAssociation \in allowedAssociations
    THEN "write"
    ELSE configuredPermissions

Init ==
    /\ trigger \in Triggers
    /\ sameRepository \in BOOLEAN
    /\ forkMode \in ForkModes
    /\ repositoryMatches \in BOOLEAN
    /\ wildcard \in BOOLEAN
    /\ authorAssociation \in Associations
    /\ allowedAssociations \in SUBSET Associations
    /\ configuredPermissions \in Permissions

Next == UNCHANGED vars
Spec == Init /\ [][Next]_vars

TypeOK ==
    /\ trigger \in Triggers
    /\ sameRepository \in BOOLEAN
    /\ forkMode \in ForkModes
    /\ repositoryMatches \in BOOLEAN
    /\ wildcard \in BOOLEAN
    /\ authorAssociation \in Associations
    /\ allowedAssociations \in SUBSET Associations
    /\ configuredPermissions \in Permissions

AdmissionConforms == ObservedAdmission = ExpectedAdmission

SameRepositoryAlwaysAdmitted ==
    trigger = "pull_request" /\ sameRepository => ObservedAdmission

DefaultForksAreDenied ==
    trigger = "pull_request" /\ forkMode = "default" /\ ~sameRepository =>
      ~ObservedAdmission

EmptyAssociationListDeniesForks ==
    trigger = "pull_request" /\ forkMode = "author-association" /\
      allowedAssociations = {} /\ ~sameRepository => ~ObservedAdmission

ConfiguredForkRulesAdmitMatches ==
    trigger = "pull_request" /\ ~sameRepository /\
      ((forkMode = "patterns" /\ (wildcard \/ repositoryMatches)) \/
       (forkMode = "author-association" /\
         authorAssociation \in allowedAssociations)) => ObservedAdmission

UnmatchedForkRulesDeny ==
    trigger = "pull_request" /\ ~sameRepository /\
      ~((forkMode = "patterns" /\ (wildcard \/ repositoryMatches)) \/
        (forkMode = "author-association" /\
          authorAssociation \in allowedAssociations)) => ~ObservedAdmission

NonPullRequestEventsAreUnaffected ==
    trigger # "pull_request" => ObservedAdmission

AdmissionDoesNotChangePermissions ==
    ObservedPermissions = configuredPermissions

=============================================================================
