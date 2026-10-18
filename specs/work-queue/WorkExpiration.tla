-------------------------- MODULE WorkExpiration --------------------------
EXTENDS Naturals, Sequences, FiniteSets, TLC

\* Focused bounded replay model, not a runtime refinement or a clock oracle.
\* Commit timestamps and exact terminal evidence are trusted inputs.
CONSTANTS LogBound, Broken
ASSUME Broken \in {"none", "grant", "release", "watermark", "checkpoint", "claim"}
Works == {"old", "fresh"}
Times == 0..2
Deadline(w) == IF w = "old" THEN 1 ELSE 3
Max(a, b) == IF a > b THEN a ELSE b
Effective(p, at) == IF Broken = "watermark" THEN at ELSE Max(p.time, at)

Empty == [work |-> [w \in Works |-> "available"],
          claim |-> [w \in Works |-> "none"],
          barrier |-> [w \in Works |-> "none"],
          retained |-> {}, charges |-> [w \in Works |-> 0],
          reservations |-> {}, terminal |-> {}, released |-> {},
          results |-> {}, time |-> 0]

Due(p, at) == {w \in Works :
  at >= Deadline(w) /\ p.barrier[w] \notin {"pending", "failed"} /\
  (p.work[w] = "available" \/ (Broken = "claim" /\ p.work[w] = "claimed"))}
Expire(p, at) ==
  LET due == Due(p, at)
  IN [p EXCEPT !.time = at,
      !.work = [w \in Works |-> IF w \in due THEN "cancelled" ELSE p.work[w]],
      !.claim = [w \in Works |->
        IF Broken = "claim" /\ w \in due THEN "cancelled" ELSE p.claim[w]],
      !.reservations = IF Broken = "release"
                       THEN p.reservations \ due ELSE p.reservations]

Apply(p, e) ==
  LET q == Expire(p, Effective(p, e.at))
      w == e.work
      changed ==
        CASE e.kind = "grant" ->
               [q EXCEPT !.work[w] = "claimed", !.claim[w] = "open",
                !.charges[w] = @ + 1, !.reservations = @ \cup {w}]
          [] e.kind = "finish" ->
               [q EXCEPT !.work[w] = "completed", !.claim[w] = "completed",
                !.barrier[w] = "pending"]
          [] e.kind = "retain" ->
               [q EXCEPT !.work[w] = "available", !.claim[w] = "completed",
                !.barrier[w] = "pending", !.retained = @ \cup {w}]
          [] e.kind = "cancel" ->
               [q EXCEPT !.work[w] = "available", !.claim[w] = "cancelled"]
          [] e.kind = "verify" ->
               [q EXCEPT !.barrier[w] = "verified",
                !.results = IF w \in q.retained THEN @ ELSE @ \cup {w}]
          [] e.kind = "terminal" -> [q EXCEPT !.terminal = @ \cup {w}]
          [] e.kind = "release" ->
               [q EXCEPT !.reservations = @ \ {w}, !.released = @ \cup {w}]
          [] OTHER -> q
  IN Expire(changed, changed.time)

RECURSIVE Replay(_, _)
Replay(p, events) ==
  IF Len(events) = 0 THEN p ELSE Replay(Apply(p, Head(events)), Tail(events))

VARIABLES ledger, projection, saved, cut
vars == <<ledger, projection, saved, cut>>
Init == /\ ledger = <<>> /\ projection = Empty /\ saved = Empty /\ cut = 0

Enabled(kind, w, at) ==
  LET q == Expire(projection, Effective(projection, at))
  IN CASE kind = "grant" ->
            /\ q.claim[w] = "none"
            /\ q.barrier[w] \notin {"pending", "failed"}
            /\ (q.work[w] = "available" \/
                (Broken = "grant" /\ q.work[w] = "cancelled"))
       [] kind \in {"finish", "retain", "cancel"} -> q.claim[w] = "open"
       [] kind = "verify" -> q.barrier[w] = "pending"
       [] kind = "terminal" ->
            w \in q.reservations /\ w \notin q.terminal
       [] kind = "release" ->
            w \in q.reservations /\ w \in q.terminal
       [] OTHER -> w = "old" /\ q # projection

Publish(kind, w, at) ==
  /\ Len(ledger) < LogBound /\ Enabled(kind, w, at)
  /\ LET e == [kind |-> kind, work |-> w, at |-> at]
     IN /\ ledger' = Append(ledger, e) /\ projection' = Apply(projection, e)
  /\ UNCHANGED <<saved, cut>>

Checkpoint ==
  /\ cut = 0 /\ Len(ledger) > 0
  /\ saved' = IF Broken = "checkpoint"
              THEN [projection EXCEPT !.time = 0] ELSE projection
  /\ cut' = Len(ledger)
  /\ UNCHANGED <<ledger, projection>>

Next == Checkpoint \/
  \E kind \in {"tick", "grant", "finish", "retain", "cancel",
                "verify", "terminal", "release"}, w \in Works, at \in Times :
    Publish(kind, w, at)
Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ projection.time \in Times /\ cut \in 0..Len(ledger)
  /\ projection.work \in [Works -> {"available", "claimed", "completed", "cancelled"}]
  /\ projection.claim \in [Works -> {"none", "open", "completed", "cancelled"}]
  /\ projection.barrier \in [Works -> {"none", "pending", "verified", "failed"}]
  /\ projection.charges \in [Works -> 0..1]
  /\ projection.reservations \subseteq Works /\ projection.results \subseteq Works

Events(kind, w) == {i \in 1..Len(ledger) :
  ledger[i].kind = kind /\ ledger[i].work = w}
ClaimAuthority == \A w \in Works :
  projection.claim[w] =
    IF Events("finish", w) # {} \/ Events("retain", w) # {} THEN "completed"
    ELSE IF Events("cancel", w) # {} THEN "cancelled"
    ELSE IF Events("grant", w) # {} THEN "open" ELSE "none"
GrantAuthority == \A i \in 1..Len(ledger) :
  ledger[i].kind = "grant" =>
    \A j \in 1..i : ledger[j].at < Deadline(ledger[i].work)
ReservationAuthority == projection.reservations =
  {w \in Works : Events("grant", w) # {} /\ Events("release", w) = {}}
ChargePersistence == \A w \in Works :
  projection.charges[w] = Cardinality(Events("grant", w))
CompletionPersistence == \A w \in Works :
  Events("finish", w) # {} => projection.work[w] = "completed"
ResultAuthority == \A w \in projection.results :
  Events("finish", w) # {} /\ Events("verify", w) # {}
CommittedTimeMonotonicity ==
  projection.time = IF Len(ledger) = 0 THEN 0
                    ELSE CHOOSE t \in Times :
                      (\E i \in 1..Len(ledger) : t = ledger[i].at) /\
                      (\A i \in 1..Len(ledger) : t >= ledger[i].at)
ProjectionSoundness ==
  /\ projection = Replay(Empty, ledger)
  /\ projection = Replay(saved, SubSeq(ledger, cut + 1, Len(ledger)))
ExpirationSafety == TypeOK /\ ClaimAuthority /\ GrantAuthority /\
  ReservationAuthority /\ ChargePersistence /\ CompletionPersistence /\
  ResultAuthority /\ CommittedTimeMonotonicity /\ ProjectionSoundness

\* Guarded witnesses demonstrate reachability, not eventual service.
NoRunningPastExpiry == ~(projection.time >= Deadline("old") /\
  projection.claim["old"] = "open" /\ "old" \in projection.reservations)
NoCheckpointExpiry == ~(cut > 0 /\ saved.work["old"] = "cancelled")
=============================================================================
