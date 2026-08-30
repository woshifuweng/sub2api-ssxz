# Full-site guardrails acceptance evidence — 2026-08-31

## Scope and release boundary

- Branch: `codex/staging-isolation-20260831`
- Final accepted commit: `6f526c126e49a715376ddd63fc5d6d2dc1b28884`
- Production was not deployed, restarted, or reconfigured.
- Acceptance ran only against the isolated staging service on loopback port
  `18080`. Staging has its own PostgreSQL database and Redis on port `6380`,
  systemd egress deny rules, no active upstream accounts, and no enabled
  provider/payment credentials.
- No paid provider request was sent.

## Implemented guardrails

| Commit | Guardrail |
| --- | --- |
| `ba62d8645` | Price text requests from the actual bounded output cap instead of a fixed balance floor. |
| `ff9474441` | Make balance-to-quota conversion idempotent. |
| `11024c03b` | Make payment order creation idempotent. |
| `40440fbd7` | Persist usage-billing shortfalls in the immutable balance ledger. |
| `b4161e636` | Atomically consume and rotate refresh tokens. |
| `1e6376330` | Require step-up verification for financial and other sensitive mutations. |
| `57842f497` | Bound public auth/payment/webhook bodies and rate-limit anonymous payment recovery. |
| `47e5e5d15` | Require complete backend/frontend/build/audit quality gates before release artifacts. |
| `de69e0c1f` | Atomically revoke single, user-wide, and family refresh tokens while cleaning both indexes. |
| `c539487e8` | Add a guarded systemd preflight, atomic activation, and automatic rollback script. |
| `6f526c126` | Return deterministic HTTP 413 for oversized auth bodies, including chunked requests. |

## Automated verification

- `go test -p 1 -tags=unit ./... -count=1`: passed twice after the complete
  guardrail set, and again after the staging-discovered 413 correction.
- `go vet ./...`: passed after the complete guardrail set and after the 413
  correction.
- Frontend full gate: lint, typecheck, 240 Vitest files and 1,676 tests passed.
- Frontend production build: passed; only pre-existing dynamic-import and
  chunk-size warnings remained.
- `pnpm audit --prod --audit-level high`: no known vulnerabilities.
- Release and backend CI workflows parsed as valid YAML on Linux.
- Guarded systemd release script passed Linux syntax validation and seven
  self-contained scenarios: unreadable preflight environment, non-executable
  binary, invalid port, shared production/preflight environment, failed
  candidate health, production-health rollback, and successful activation.
- Refresh-token concurrency tests proved one rotation winner and no surviving
  token/index after concurrent single, user-wide, or family revocation.

## Staging deployment evidence

- Candidate SHA-256:
  `c0ce5871109e1cbcfe153567d32d25ab5480e9f1ab2613ff866c935828c525a1`
- Candidate size: `121102496` bytes.
- Active staging commit and binary hash matched the candidate.
- Staging restart changed only the staging PID (`2170858` to `2175037`).
- Production PID remained `2067320`; all three services reported `active / running`
  with `NRestarts=0`.
- Isolation verifier after deployment: zero API-key overlap, zero password-hash
  overlap, zero active accounts, zero provider credentials, zero non-test active
  keys, zero reusable database auth sessions, and zero enabled provider configs.
- Retained rollback bundle:
  `/opt/sub2api/backups/guardrails-staging-before-20260830T204346Z-c539487e8617`.
  The intermediate backup containing the superseded 400-response candidate was
  removed after final acceptance.

## Zero-paid staging acceptance

- Health and embedded frontend: HTTP 200.
- Staging login, authenticated profile, and API-key list: passed.
- Two concurrent refreshes of the same token: HTTP `401/200`, exactly one winner.
- Oversized auth JSON: HTTP 413.
- Oversized public payment recovery JSON: HTTP 413.
- Oversized payment webhook: HTTP 413.
- The staging beta balance was temporarily set to `$5.00`; a capped
  `gpt-5.6-sol /v1/responses` request with `max_output_tokens=16` passed balance
  preflight and reached the expected no-account boundary (HTTP 503), proving it
  was not rejected as insufficient balance.
- Before and after that request: balance stayed `$5.00`, payment orders stayed
  `0`, user usage rows stayed `261`, balance-ledger rows stayed `0`, and active
  upstream accounts stayed `0`.
- The fixture balance was restored to its original `22.67764088` after the test.
- Post-check found zero panic/fatal messages, zero migration/database/Redis
  failures, and no acceptance temporary directories.

## Production rollout condition

The code and isolated staging candidate are accepted. Production rollout remains
intentionally blocked until separately authorized. A production release must use
the guarded preflight/atomic rollback flow and re-run health, billing, payment,
auth, and log checks without issuing a paid model request.
