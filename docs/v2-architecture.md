# Verifier V2 Architecture

## Core Model

Verifier V2 is a verifier-only system. It does not send probe emails, monitor IMAP inboxes, or use Tor. Every result comes from one of two layers:

1. deterministic SMTP and DNS evidence
2. enrichment evidence for ambiguous outcomes

## Request Flow

1. Normalize and parse the email address.
2. Resolve MX records, with A/AAAA fallback when no MX exists.
3. Run direct SMTP RCPT callouts against recipient MX hosts.
4. If the target recipient is accepted, run or reuse a cached control recipient check for the same domain fingerprint.
5. Classify the result as one of:
   - `deliverable`
   - `undeliverable`
   - `accept_all`
   - `unknown`
6. For `accept_all` and `unknown`, enqueue asynchronous enrichment.

## Domain Baseline

The control recipient is an obviously random address at the same domain. Its purpose is to answer: "does this domain accept arbitrary recipients?"

- target accepted + control rejected => `deliverable`
- target accepted + control accepted => `accept_all`
- target accepted + control inconclusive => `unknown`

The domain baseline is cached by domain fingerprint so repeated checks do not re-run the control probe unnecessarily.

## Enrichment Layer

The enrichment worker never upgrades a hard invalid result into valid. Instead it adjusts confidence and adds evidence for ambiguous classifications.

Current first-party evidence includes:

- disposable domain detection
- consumer mailbox provider detection
- role mailbox detection
- website reachability
- same-domain public email discovery from homepage/contact-page fetches
- exact public match detection
- local-part pattern heuristics

The codebase also ships disabled provider interfaces so optional third-party enrichment can be added later without making the classifier depend on vendors.

## Persistence

V2 stores:

- verification records
- SMTP callout attempts
- domain baseline profiles
- enrichment evidence

This makes every classification inspectable instead of depending on delayed mailbox side effects.

## Why This Is Safer

- deterministic outcomes are limited to deterministic evidence
- ambiguous outcomes stay ambiguous
- scoring is explicit and evidence-backed
- catch-all handling is first-class
- the verifier is easier to operate and audit
