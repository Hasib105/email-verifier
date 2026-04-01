# ADR 0001: Retire Probe Email, Bounce Watching, and Tor-Routed Verification

## Status

Accepted

## Context

The legacy verifier treated "probe email sent and no bounce observed" as positive evidence. That model was unreliable for several reasons:

- A missing bounce is not proof of mailbox existence.
- Delayed bounce, silent drop, greylisting, and policy filtering make the signal non-deterministic.
- Catch-all domains accept arbitrary recipients, so the probe result can look positive for the wrong reasons.
- Tor only obscured the app-to-provider hop. When mail was submitted through Gmail or another relay, the recipient still saw the relay provider as the sender.
- Routing verification through Tor increased provider suspicion and operational fragility without changing recipient-visible delivery semantics.

## Decision

Verifier V2 removes:

- Probe-email sending
- IMAP bounce watching
- Delayed verification finalization
- Tor-routed verification
- Gmail/app-password submission as a verification primitive

Verifier V2 uses direct SMTP RCPT callouts against recipient MX hosts and a cached per-domain control recipient baseline. Ambiguous outcomes remain ambiguous and are enriched with evidence rather than force-fit into valid/invalid.

## Consequences

Positive:

- The verifier only claims deterministic results when protocol evidence supports them.
- The core path is simpler, faster, and easier to reason about.
- Catch-all handling becomes explicit instead of hidden behind false positives.
- Infrastructure is easier to operate because it relies on direct callout egress rather than Tor plus mailbox monitoring.

Negative:

- V2 intentionally breaks legacy API semantics.
- Deliverability scoring for ambiguous domains now depends on enrichment quality instead of a delayed bounce heuristic.
- Stable outbound port 25 egress and correct verifier DNS/PTR setup become operational requirements.
