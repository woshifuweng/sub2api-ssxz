# Unified chat workspace v1 security strategy

## Scope and product promise

The v1 workspace stores and displays text conversations for the authenticated
owner. Image, video, file-processing, tools, and other composer affordances are
not silently enabled: the UI must show them as unavailable until the backend has
an equally complete authorization, billing, retention, and deletion path.

## Trust boundaries

1. The browser is untrusted. Conversation, model, intent, attachment, price,
   user, group, and API-key identifiers supplied by it are hints only.
2. JWT authentication establishes a user identity; every conversation and
   message repository query must also constrain by that user ID. Knowing a
   conversation ID never grants access.
3. API keys and groups establish model/routing permission. The backend resolves
   the effective group and allowed models again when a message is sent.
4. Provider responses and streamed events are untrusted input. They must not be
   rendered as HTML without sanitization and must not become privileged browser
   instructions.
5. PostgreSQL is the durable conversation and billing record. Redis is only
   transient coordination/cache state and must not be the sole source of money,
   ownership, or message history truth.
6. Production and staging are separate security domains. Staging has distinct
   data, Redis, credentials, network policy, users, and service identity.

## Assets to protect

- User prompts, model responses, conversation titles, and timestamps.
- User identity, API keys, group membership, balance, and usage records.
- Provider credentials, routing details, internal account IDs, and internal
  error bodies.
- Billing correctness: one accepted message produces at most one charge, and a
  retry cannot create a duplicate paid operation.

## Required controls

- All workspace routes require JWT authentication, backend-mode policy, and the
  panel rate limiter. List/read/append operations enforce owner scope in the
  service or repository, not only in the frontend.
- Model and intent values use allowlists. Disabled capabilities fail closed with
  a stable error code and do not create placeholder paid work.
- Request bodies use configured size limits. Stored and displayed content uses
  bounded lengths; logs redact prompts, credentials, and provider bodies.
- Message submission has a durable idempotency key and a stable retry outcome
  before v1 can invoke a paid provider from this workspace.
- Billing admission uses the same output cap and group multiplier that will be
  sent upstream. Settlement records actual usage and a durable shortfall event.
- Frontend rendering treats message content as text/Markdown through the shared
  sanitizer. No `v-html` path may bypass it.
- Conversation deletion, export, and retention remain unavailable until their
  authorization and audit behavior are specified and tested.

## Abuse and failure cases

| Case | Required behavior |
| --- | --- |
| User guesses another conversation ID | 404-equivalent response; no existence leak |
| User changes model/group in the request | Backend recomputes permission and rejects mismatch |
| Duplicate send/retry | Same idempotency key returns the same result; no second charge |
| Balance becomes insufficient during a request | Bounded admission plus durable shortfall audit |
| Provider sends malformed/HTML content | Sanitized text rendering; no script execution |
| Redis unavailable | No loss of durable ownership or billing truth |
| Staging attempts an upstream call | Network/provider kill switch blocks it |

## Release evidence

The workspace cannot be released from a candidate that fails the backend unit
and contract suites, frontend full Vitest suite, typecheck, lint, production
build, staging isolation verifier, or the workspace rows in
`business-regression-matrix.md`. Production rollout remains a separate approved
operation after staging evidence is archived.
