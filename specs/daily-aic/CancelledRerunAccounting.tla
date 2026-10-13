------------------------- MODULE CancelledRerunAccounting -------------------------
EXTENDS FiniteSets, TLC

Outcomes == {"cancelled", "success", "failure", "timed_out"}
ProducerStates == {"verified", "missing", "invalid"}
\* "verified" means one earlier executed attempt has a unique, in-bounds,
\* unexpired producer artifact. "ambiguous" means multiple prior jobs compete.
PriorStates == {"verified", "missing", "invalid", "ambiguous"}
AccountingStates == {"complete", "missing"}
Decisions == {"pending", "current", "carried", "rejected"}

VARIABLES outcome, currentProducer, priorProducer, accounting, decision, warned
vars == <<outcome, currentProducer, priorProducer, accounting, decision, warned>>

Init ==
  /\ outcome \in Outcomes
  /\ currentProducer \in ProducerStates
  /\ priorProducer \in PriorStates
  /\ accounting \in AccountingStates
  /\ decision = "pending"
  /\ warned = FALSE

Evaluate ==
  /\ decision = "pending"
  /\ IF currentProducer = "verified"
        THEN /\ decision' = IF accounting = "complete" THEN "current" ELSE "rejected"
             /\ warned' = FALSE
        ELSE IF outcome = "cancelled"
                  /\ priorProducer = "verified"
                  /\ accounting = "complete"
             THEN /\ decision' = "carried"
                  /\ warned' = TRUE
             ELSE /\ decision' = "rejected"
                  /\ warned' = FALSE
  /\ UNCHANGED <<outcome, currentProducer, priorProducer, accounting>>

Next == Evaluate
Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ outcome \in Outcomes
  /\ currentProducer \in ProducerStates
  /\ priorProducer \in PriorStates
  /\ accounting \in AccountingStates
  /\ decision \in Decisions
  /\ warned \in BOOLEAN

CarriedRequiresVerifiedEvidence ==
  decision = "carried" =>
    /\ outcome = "cancelled"
    /\ currentProducer # "verified"
    /\ priorProducer = "verified"
    /\ accounting = "complete"
    /\ warned

OtherOutcomesRequireCurrentProducer ==
  decision # "pending" /\ outcome # "cancelled" =>
    IF currentProducer = "verified"
    THEN decision \in {"current", "rejected"}
    ELSE decision = "rejected"

AmbiguousPriorFailsClosed ==
  decision # "pending" /\ currentProducer # "verified"
    /\ priorProducer = "ambiguous" =>
      decision = "rejected"

MissingAccountingFailsClosed ==
  decision # "pending" /\ accounting = "missing" =>
    decision = "rejected"

CurrentDecisionRequiresCurrentProducer ==
  decision = "current" =>
    /\ currentProducer = "verified"
    /\ accounting = "complete"
=============================================================================
