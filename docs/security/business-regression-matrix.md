# Business regression matrix

This matrix is the minimum release evidence for the customized product. “Pass”
means the cited automated gate is green and, where marked, the isolated staging
check has been recorded. No test may invoke a paid text, image, video, search,
payment, email, or object-storage provider.

| Journey / risk | Required proof | Automated gate | Staging proof |
| --- | --- | --- | --- |
| Register, login, 2FA, refresh, logout | Valid flow succeeds; brute-force and replay fail | auth handler/service tests; atomic refresh-token concurrency tests | Dedicated staging users only |
| API-key creation | At least one available group is mandatory; model allowlist round-trips; an explicit `group_ids: []` clear cannot revive a stale primary group | `TestAPIContracts`; API-key service/repository policy tests | Create/read/update a staging-only key |
| Group authorization | A key cannot use a disabled, soft-deleted, unavailable or mismatched group/model; the selected request group is also the usage/billing group | API-key middleware and gateway authorization/routing tests | No real provider account is schedulable |
| Text protocols | Responses, chat completions, messages, Gemini and supported aliases share routing/billing rules | gateway handler/service contract tests | Fixture/mock only; no upstream request |
| Low and zero balance | Estimated real cost, explicit/implicit output cap and customer multiplier decide admission; zero and one cent below estimate fail, exact/above estimate pass; no fixed `$10` floor | request-cost, billing-eligibility and pre-upstream handler tests | `$5` at `0.35x`, exact-boundary and below-boundary fixture cases |
| Usage settlement | Actual cost is charged once; concurrent settlement cannot make balance negative; request retry does not recharge; durable dedup, charged amount and shortfall agree and alert without secrets | usage-billing repository/service tests | Ledger queries against staging fixtures |
| Payment order creation | Same idempotency key and payload replay one stored checkout response; the key cannot create two orders or be reused with changed amount/method | payment idempotency handler/service tests | Mock provider only |
| Public payment recovery | Signed token preferred; legacy lookup is bounded, rate-limited and observable | payment public security/contract tests | Invalid/fixture tokens only |
| Payment webhook | Signature required, duplicate event safe, body over 1 MiB rejected | payment webhook/service tests | Stored signed fixture only |
| Refund | User request and admin processing are authenticated, audited, step-up protected and idempotent | payment route/service tests | Unpaid fixture order only |
| Reseller conversion/withdrawal | `$5` minimum, same-key replay, changed-amount conflict, reserved-balance locking, correct ownership and review step-up | reseller route/handler/service/repository tests | Fixture balance only |
| Affiliate transfer | Ownership, available quota and step-up policy enforced | affiliate handler/service and route tests | Fixture quota only |
| Admin balance/config/provider/plan mutations | Admin role plus short-lived TOTP step-up; API-key admin auth cannot bypass | `TestFinanceMutationRoutesRequireStepUp`; step-up middleware tests | TOTP test admin only |
| Conversation workspace | Owner scoping, allowlisted model/intent, text-only disabled capabilities | workspace route/service/frontend tests | Fixture conversations only |
| User usage records | No internal user, key, account or subscription identifiers leak | `TestAPIContracts` | Compare response schema only |
| Frontend regressions | All pages compile and all Vitest suites pass; critical pages have no overflow, clipping, untranslated keys, broken empty/error/loading states or inaccessible finance actions | `make test-frontend`; focused responsive/state tests | Fresh desktop/tablet/mobile captures, no paid writes |
| Staging isolation | Separate DB/Redis/users/credentials, no outbound provider access | `ops/staging/verify-staging-isolation.sh` | Verifier output and process/socket evidence |
| Backup restoration | Latest custom-format backup restores into a uniquely named disposable database; required migrations and critical row counts verify; production/staging names are refused | `ops/production/test-verify-backup-restore.sh` | Sanitized restore-drill evidence; source database read-only |
| Release/rollback | Full commercial gate is green; candidate starts and health-checks before production stop; failed health restores previous release | commercial gate and production preflight script tests | Candidate port only; production untouched |

## Evidence record template

- Candidate commit and branch:
- UTC/Beijing test time:
- Backend unit/contract total:
- Frontend test total, lint, typecheck, build:
- Staging isolation verifier result:
- Production PID/start timestamp before and after staging checks:
- Artifact SHA-256 hashes:
- Paid provider requests made: **0**
- Remaining risks and explicit rollout approval:
