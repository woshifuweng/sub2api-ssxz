# Commercial Regression Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current commercial release into a reproducible, evidence-backed release that blocks known billing, group-routing, payment, permission, restore, and critical UI regressions before production.

**Architecture:** Preserve the upstream-first Sub2API architecture and the existing SSXZ visual layer. Add narrowly scoped Go/Vitest regression tests around business boundaries, shell gates that compose existing test suites, and a disposable-database restore drill. Production remains read-only during implementation; code is validated locally and in the isolated staging environment before any separate production release decision.

**Tech Stack:** Go 1.27, Gin, Ent, PostgreSQL, Redis, Vue 3, TypeScript, Vitest, Playwright-compatible browser checks, Bash, systemd.

---

## Task 1: Freeze the clean baseline and risk matrix

**Files:**
- Create: `docs/superpowers/plans/2026-08-31-commercial-regression-hardening.md`
- Modify: `docs/security/business-regression-matrix.md`

- [x] Create an isolated branch from production commit `2e0483442dd53b5c2f34d7cef0eb2e35433d98fb`.
- [x] Run the complete backend baseline with bounded concurrency: `go test -p 4 ./...`.
- [x] Run frontend lint: `pnpm --dir frontend run lint:check`.
- [x] Run frontend typecheck: `pnpm --dir frontend run typecheck`.
- [x] Run frontend tests with bounded workers: `pnpm --dir frontend exec vitest run --minWorkers=1 --maxWorkers=4`.
- [x] Add the exact low-balance, group lifecycle, idempotency, finance authorization, restore, responsive UI, and release-gate scenarios to the business regression matrix.
- [x] Commit the plan and matrix update.

## Task 2: Add a real low-balance pricing matrix

**Files:**
- Modify: `backend/internal/service/request_cost_estimate_test.go`
- Modify: `backend/internal/service/billing_cache_service_test.go`
- Modify: `backend/internal/handler/gateway_handler_billing_error_test.go`
- Modify: `backend/internal/handler/openai_gateway_handler_test.go`

- [x] Add failing tests for an omitted output limit receiving the 16,384 cap without replacing an explicit client limit.
- [x] Add failing tests proving a `$5.00` balance with a `0.35x` customer-group multiplier is checked against the calculated request estimate, not a fixed reserve.
- [x] Cover the exact boundaries: zero balance, one cent below estimate, exactly equal to estimate, and balance above estimate.
- [x] Prove an insufficient request is rejected before an upstream call, while an affordable request reaches the forwarding boundary.
- [x] Run the focused service and handler tests, then the complete backend suite.
- [x] Commit the low-balance regression matrix.

## Task 3: Close the API-key and group lifecycle boundary

**Files:**
- Modify: `backend/internal/server/middleware/api_key_auth_test.go`
- Modify: `backend/internal/server/middleware/api_key_policy_compat_test.go`
- Modify: `backend/internal/handler/api_key_group_routing_test.go`
- Modify: `backend/internal/repository/api_key_policy_compat_test.go`

- [x] Add a failing test showing an explicitly cleared `group_ids: []` persists as empty and cannot silently fall back to an old primary group.
- [x] Add a failing test showing a key bound to a disabled or soft-deleted group fails closed with `GROUP_NOT_ALLOWED`.
- [x] Add a multi-group test showing the selected request group is also the usage/billing group.
- [x] Verify cache hydration cannot restore a stale deleted binding.
- [x] Run focused middleware, handler, repository, and integration tests.
- [x] Commit the API-key/group lifecycle protection.

## Task 4: Prove money mutations are replay-safe and properly authorized

**Files:**
- Modify: `backend/internal/service/payment_order_result_test.go`
- Modify: `backend/internal/handler/payment_handler_idempotency_test.go`
- Modify: `backend/internal/repository/reseller_repo_test.go`
- Modify: `backend/internal/server/routes/finance_step_up_route_test.go`
- Modify: `backend/internal/server/middleware/step_up_test.go`

- [x] Prove the same payment idempotency key plus the same business payload replays the stored checkout response without creating another order.
- [x] Prove the same key plus a changed amount or payment method returns an idempotency conflict.
- [x] Prove the same withdrawal key replays one request and a changed amount is rejected.
- [x] Prove every balance, refund, redeem, affiliate, withdrawal, provider, and payment-plan mutation route requires step-up authentication.
- [x] Prove an API-key-authenticated admin cannot satisfy step-up authentication.
- [x] Run focused tests and the complete backend suite.
- [x] Commit the money-mutation regression protection.

## Task 5: Verify settlement durability and reconciliation evidence

**Files:**
- Modify: `backend/internal/repository/usage_billing_repo_unit_test.go`
- Modify: `backend/internal/repository/usage_billing_repo_integration_test.go`
- Modify: `backend/internal/service/gateway_usage_billing_shortfall_test.go`
- Modify: `docs/security/business-regression-matrix.md`

- [x] Prove concurrent settlement cannot make a user balance negative.
- [x] Prove the durable deduplication row, charged amount, and shortfall record agree after an overdraft race.
- [x] Prove a retry with the same request ID does not charge twice.
- [x] Prove the operational alert includes request, user, key, expected charge, actual charge, and shortfall identifiers without secrets.
- [x] Run focused unit/integration tests and the complete backend suite.
- [x] Commit settlement and reconciliation hardening.

## Task 6: Make backup restoration a release prerequisite

**Files:**
- Create: `ops/production/verify-backup-restore.sh`
- Create: `ops/production/test-verify-backup-restore.sh`
- Modify: `ops/production/README.md`
- Modify: `ops/production/preflight-systemd-release.sh`
- Modify: `ops/production/test-preflight-systemd-release.sh`

- [x] Write shell tests for rejecting missing, empty, stale, or invalid PostgreSQL custom-format backups.
- [x] Implement a restore verifier that creates one uniquely named disposable database, restores the backup, verifies required migrations and critical table counts, and drops only that verified disposable database on exit.
- [x] Ensure the verifier refuses production/staging database names and unresolved or broad targets.
- [x] Add the restore verifier to the release preflight before service cutover.
- [x] Run shell tests locally.
- [x] On the server, run one read-only-source restore drill from the latest production backup into a disposable database and retain sanitized evidence.
- [x] Commit restore-gate hardening.

## Task 7: Consolidate a bounded commercial release gate

**Files:**
- Create: `ops/production/run-commercial-regression-gate.sh`
- Create: `ops/production/test-commercial-regression-gate.sh`
- Modify: `ops/production/README.md`
- Modify: `docs/security/launch-checklist.md`

- [x] Write shell tests proving every required backend, frontend, migration, isolation, restore, and release-preflight command is invoked and any failure stops the gate.
- [x] Implement a bounded-concurrency gate that produces a timestamped, sanitized evidence directory.
- [x] Include frontend lint, typecheck, Vitest, backend full tests, migration regressions, staging isolation, backup restore, and release preflight tests.
- [x] Document the exact operator command and expected artifacts.
- [ ] Run the gate locally without production writes.
- [ ] Commit the commercial release gate.

## Task 8: Audit critical user/admin flows and repair only P1 defects

**Files:**
- Modify as findings require: `frontend/src/views/**`
- Modify as findings require: `frontend/src/components/**`
- Modify as findings require: `frontend/src/locales/**`
- Modify: `docs/security/business-regression-matrix.md`
- Create: `docs/reports/2026-08-31-critical-ui-audit.md`

- [ ] Capture fresh desktop, tablet, and mobile screenshots for login, dashboard, API keys, usage records, recharge/order/redeem, affiliate/reseller, account, admin dashboard, admin keys, admin groups, and admin usage records.
- [ ] Check horizontal overflow, clipped actions, header/content alignment, empty/loading/error states, untranslated keys, keyboard focus, and destructive-action confirmation.
- [ ] Preserve the SSXZ design language; import upstream behavior or layout ideas only through existing SSXZ components and tokens.
- [ ] Fix only P1 usability, accessibility, or data-integrity defects discovered by the audit.
- [ ] Add or update focused Vitest coverage for every repaired defect.
- [ ] Re-run lint, typecheck, Vitest, and production frontend build.
- [ ] Commit UI fixes and the audit report.

## Task 9: Isolated staging acceptance and final evidence

**Files:**
- Modify: `DEPLOYED.md`
- Create: `docs/reports/2026-08-31-commercial-regression-hardening.md`

- [ ] Build the release candidate once and record its SHA-256.
- [ ] Deploy only to the isolated staging service and verify database/Redis/outbound isolation.
- [ ] Run no-cost health, authentication, API-key, group, usage-record, payment-read, and UI smoke checks.
- [ ] Confirm staging did not restart unexpectedly and produced no new 5xx or panic logs.
- [ ] Re-run the commercial release gate against the exact candidate artifact.
- [ ] Record completed work, unchanged scope, test results, restore evidence, known residual risks, and whether production was updated.
- [ ] Do not update production without a separate, explicit release decision after all evidence is green.
