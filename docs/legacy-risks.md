# Legacy Verification Risks

## What Was Wrong With V1

V1 blended two different ideas into one product:

- deterministic SMTP verification
- speculative probe-email delivery monitoring

That coupling created logical and operational problems.

## Logical Weaknesses

- `no bounce` was treated as a positive signal even though it only means "no bounce was observed in the chosen window."
- Catch-all domains could be marked as valid because the mailbox server accepted both real and fake recipients.
- Temporary provider failures and policy blocks were pushed into a delayed workflow that still tended to converge to a positive answer.
- The old model over-promised certainty on inherently ambiguous cases.

## Operational Weaknesses

- Tor added latency and instability to SMTP connectivity.
- Gmail/app-password flows were a bad fit for verification and increased account-risk without improving recipient-side behavior.
- IMAP bounce watching required extra credentials, mailbox setup, and background scheduling.
- The product surface grew around maintaining that fragile loop: SMTP account pools, templates, Tor health, and bounce rechecks.

## Security And Product Risks

- Multi-step delayed logic made auditing and reasoning about outcomes harder.
- The old surface introduced extra configuration and secret-management burden.
- Users had to manage infrastructure unrelated to the real verification goal.

## V2 Direction

V2 keeps deterministic SMTP callouts for hard outcomes, explicitly identifies accept-all behavior, and adds evidence-backed scoring for ambiguous cases instead of pretending they are proven valid.
