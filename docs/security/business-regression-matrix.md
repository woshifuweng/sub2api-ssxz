# Business regression matrix

This matrix is the minimum release evidence for the customized product. “Pass”
means the cited automated gate is green and, where marked, the isolated staging
check has been recorded. No test may invoke a paid text, image, video, search,
payment, email, or object-storage provider.

| Journey / risk | Required proof | Automated gate | Staging proof |
| --- | --- | --- | --- |
| Register, login, 2FA, refresh, logout | Valid flow succeeds; brute-force and replay fail | auth handler/service tests; atomic refresh-token concurrency tests | Dedicated staging users only |
| API-key creation | At least one available group is mandatory; model allowlist round-trips | `TestAPIContracts`; API-key service tests | Create/read a staging-only key |
| Group authorization | A key cannot use an unavailable or mismatched group/model | API-key and gateway authorization tests | No real provider account is schedulable |
| Text protocols | Responses, chat completions, messages, Gemini and supported aliases share routing/billing rules | gateway handler/service contract tests | Fixture/mock only; no upstream request |
| Low and zero balance | Estimated real cost, output cap and customer multiplier decide admission; no fixed `$10` floor | request-cost and billing-eligibility tests | `$5` and below-boundary fixture cases |
| Usage settlement | Actual cost is charged once; negative balance is prevented; shortfall is durable and alerted | usage-billing repository/service tests | Ledger queries against staging fixtures |
| Payment order creation | Same idempotency key cannot create two orders or be reused with changed payload | payment idempotency handler/service tests | Mock provider only |
| Public payment recovery | Signed token preferred; legacy lookup is bounded, rate-limited and observable | payment public security/contract tests | Invalid/fixture tokens only |
| Payment webhook | Signature required, duplicate event safe, body over 1 MiB rejected | payment webhook/service tests | Stored signed fixture only |
| Refund | User request and admin processing are authenticated, audited, step-up protected and idempotent | payment route/service tests | Unpaid fixture order only |
| Reseller conversion/withdrawal | `$5` minimum, durable idempotency, correct ownership, review step-up | reseller route/handler/service/repository tests | Fixture balance only |
| Affiliate transfer | Ownership, available quota and step-up policy enforced | affiliate handler/service and route tests | Fixture quota only |
| Admin balance/config/provider/plan mutations | Admin role plus short-lived TOTP step-up; API-key admin auth cannot bypass | `TestFinanceMutationRoutesRequireStepUp`; step-up middleware tests | TOTP test admin only |
| Conversation workspace | Owner scoping, allowlisted model/intent, text-only disabled capabilities | workspace route/service/frontend tests | Fixture conversations only |
| User usage records | No internal user, key, account or subscription identifiers leak | `TestAPIContracts` | Compare response schema only |
| Frontend regressions | All pages compile and all Vitest suites pass; no critical-only subset | `make test-frontend` | Desktop/tablet/mobile smoke, no writes |
| Staging isolation | Separate DB/Redis/users/credentials, no outbound provider access | `ops/staging/verify-staging-isolation.sh` | Verifier output and process/socket evidence |
| Release/rollback | Candidate starts and health-checks before production stop; failed health restores previous release | production preflight script tests | Candidate port only; production untouched |

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
