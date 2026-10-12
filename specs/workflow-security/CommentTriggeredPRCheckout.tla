------------------------- MODULE CommentTriggeredPRCheckout -------------------------
EXTENDS FiniteSets, TLC

CONSTANT Fault
ASSUME Fault \in {"none", "event-bypass", "sender-type-bypass",
                  "author-bypass", "actor-bypass", "allowlist-bypass",
                  "repository-bypass"}

Events == {"issue_comment", "pull_request_review_comment",
           "pull_request_review", "workflow_dispatch", "other"}
CommentEvents == {"issue_comment", "pull_request_review_comment"}
ReviewEvents == {"pull_request_review"}
CommentOrReviewEvents == CommentEvents \cup ReviewEvents
Actors == {"my-app", "my-app[bot]", "other", "other[bot]"}
SenderTypes == {"Bot", "User"}
RepositoryRelations == {"same", "fork", "missing-head", "missing-base"}
Permissions == {"none", "read", "write", "maintain", "admin"}

VARIABLES event, actor, sender, commentAuthor, reviewAuthor, senderType,
         allowListed, repositoryRelation, permission
vars == <<event, actor, sender, commentAuthor, reviewAuthor, senderType,
         allowListed, repositoryRelation, permission>>

CanonicalBotIdentity(login) ==
    IF login = "my-app[bot]" THEN "my-app"
    ELSE IF login = "other[bot]" THEN "other"
    ELSE login

CommentEvent == event \in CommentEvents
CommentOrReviewEvent == event \in CommentOrReviewEvents
AuthorMatchesSender ==
    IF CommentEvent THEN sender = commentAuthor
    ELSE IF event \in ReviewEvents THEN sender = reviewAuthor
    ELSE FALSE
ActorMatchesSender ==
    CanonicalBotIdentity(actor) = CanonicalBotIdentity(sender)
CommonBypassGuards ==
    CommentOrReviewEvent /\ senderType = "Bot" /\ AuthorMatchesSender /\
      ActorMatchesSender /\ allowListed /\ repositoryRelation = "same"

ExpectedBotBypass == CommonBypassGuards
ObservedBotBypass ==
    ExpectedBotBypass \/
    (Fault = "event-bypass" /\ ~CommentOrReviewEvent /\ senderType = "Bot" /\
      sender = commentAuthor /\ ActorMatchesSender /\ allowListed /\
      repositoryRelation = "same") \/
    (Fault = "sender-type-bypass" /\ CommentOrReviewEvent /\ senderType # "Bot" /\
      AuthorMatchesSender /\ ActorMatchesSender /\ allowListed /\
      repositoryRelation = "same") \/
    (Fault = "author-bypass" /\ CommentOrReviewEvent /\ senderType = "Bot" /\
      ~AuthorMatchesSender /\ ActorMatchesSender /\ allowListed /\
      repositoryRelation = "same") \/
    (Fault = "actor-bypass" /\ CommentOrReviewEvent /\ senderType = "Bot" /\
      AuthorMatchesSender /\ ~ActorMatchesSender /\ allowListed /\
      repositoryRelation = "same") \/
    (Fault = "allowlist-bypass" /\ CommentOrReviewEvent /\ senderType = "Bot" /\
      AuthorMatchesSender /\ ActorMatchesSender /\ ~allowListed /\
      repositoryRelation = "same") \/
    (Fault = "repository-bypass" /\ CommentOrReviewEvent /\ senderType = "Bot" /\
      AuthorMatchesSender /\ ActorMatchesSender /\ allowListed /\
      repositoryRelation # "same")

CollaboratorAuthorized ==
    permission \in {"write", "maintain", "admin"}
ExpectedCheckoutAuthorized == ExpectedBotBypass \/ CollaboratorAuthorized
ObservedCheckoutAuthorized == ObservedBotBypass \/ CollaboratorAuthorized

Init ==
    /\ event \in Events
    /\ actor \in Actors
    /\ sender \in Actors
    /\ commentAuthor \in Actors
    /\ reviewAuthor \in Actors
    /\ senderType \in SenderTypes
    /\ allowListed \in BOOLEAN
    /\ repositoryRelation \in RepositoryRelations
    /\ permission \in Permissions

Next == UNCHANGED vars
Spec == Init /\ [][Next]_vars

TypeOK ==
    /\ event \in Events
    /\ actor \in Actors
    /\ sender \in Actors
    /\ commentAuthor \in Actors
    /\ reviewAuthor \in Actors
    /\ senderType \in SenderTypes
    /\ allowListed \in BOOLEAN
    /\ repositoryRelation \in RepositoryRelations
    /\ permission \in Permissions

NoUnauthorizedBotBypass ==
    ObservedBotBypass => ExpectedBotBypass

CheckoutAuthorizationMatchesPolicy ==
    ObservedCheckoutAuthorized = ExpectedCheckoutAuthorized

=====================================================================================
