# Security Launch Checklist

This checklist is for the AI relay / Unified Chat Workspace product before each merge, staging release, and production launch.

## 1. Release gate

| Item | Required status |
|---|---|
| Working tree | Clean before release candidate |
| PR scope | Matches issue/task scope |
| CI | All required checks pass |
| Backend tests | Relevant package tests pass |
| Frontend typecheck | Passes when frontend changed |
| Secrets scan | No provider/payment/SMS/OAuth secrets committed |
| Migration review | No accidental production migration |
| Security tests | Not skipped or weakened |

Go/no-go decision:

- **No-go** if provider keys can reach the browser.
- **No-go** if user data ownership is not enforced server-side.
- **No-go** if billing can be controlled by frontend input.
- **No-go** if image/file assets are publicly accessible by default.
- **No-go** if arbitrary provider URLs can be requested.

## 2. Authentication and account security

Check:

- [ ] All workspace/API endpoints require authentication unless explicitly public.
- [ ] Session cookies/tokens are stored according to project policy.
- [ ] User identity is derived from trusted server-side auth context, not from frontend-provided `user_id`.
- [ ] Login/register errors do not leak whether sensitive credentials exist unless intentionally designed.
- [ ] Passwords are never stored or logged in plaintext.
- [ ] OAuth secrets are server-only.
- [ ] Demo bypasses are not active in production.

If SMS login/register exists:

- [ ] Phone-number-level send limit.
- [ ] IP-level send limit.
- [ ] Device/session-level send limit.
- [ ] Cooldown between sends.
- [ ] Daily send cap.
- [ ] Abuse alert for SMS cost spikes.
- [ ] Verification code is generated and checked server-side.
- [ ] Verification code is not embedded in frontend code, logs, or responses.

## 3. API relay and provider security

Check:

- [ ] Provider API keys exist only on the backend.
- [ ] Browser network traffic never exposes upstream provider credentials.
- [ ] Provider base URLs come from server-side allowlist.
- [ ] Model names come from server-side allowlist or capability registry.
- [ ] User cannot pass arbitrary `base_url` or unapproved provider host.
- [ ] Requests to localhost, loopback, private IP ranges, metadata addresses, and internal hostnames are blocked.
- [ ] Provider errors are sanitized before returning to users.
- [ ] Logs redact authorization headers, tokens, and keys.
- [ ] Provider request timeout, retry, and circuit-breaker behavior are defined.
- [ ] Provider usage is recorded for billing/audit.

Key SSRF blocklist targets:

- `localhost`
- `127.0.0.1`
- `::1`
- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`
- `169.254.0.0/16`
- cloud metadata endpoints
- internal service DNS names

## 4. Unified Chat Workspace v1

Scope for v1:

- `/app` is the main workspace.
- `/app/chat` and `/app/image` redirect into the unified workspace.
- Text conversation lifecycle works end to end.
- Real image generation is not required in v1 unless explicitly scoped.
- Web browsing, memory, toolbox, documents, and complex file analysis remain disabled or clearly marked as unavailable.

Security checks:

- [ ] Creating a conversation requires authentication.
- [ ] Listing conversations returns only the current user's conversations.
- [ ] Loading a conversation verifies ownership.
- [ ] Creating a message verifies conversation ownership.
- [ ] Deleting/updating a conversation verifies ownership.
- [ ] Frontend never sends a trusted `user_id` as authority.
- [ ] Model is validated on the backend.
- [ ] Intent is validated on the backend.
- [ ] Disabled tools cannot be triggered through direct API calls.
- [ ] Errors are user-safe and do not expose stack traces or secrets.
- [ ] Refreshing the page restores conversation/message state.
- [ ] Failed message sends show a clear error and do not create inconsistent state.

Acceptance tests:

- [ ] New user can create a conversation.
- [ ] Message persists after refresh.
- [ ] Sidebar history loads from backend.
- [ ] User A cannot access User B's conversation by ID.
- [ ] Invalid model is rejected server-side.
- [ ] Invalid intent is rejected server-side.
- [ ] `/app/chat` no longer opens an isolated legacy experience.
- [ ] `/app/image` no longer opens an isolated legacy experience.

## 5. Image upload and generation

Check before enabling uploads:

- [ ] Uploaded files become `asset` records.
- [ ] Frontend stores asset IDs, not long-lived base64 payloads.
- [ ] Assets are tied to `user_id`.
- [ ] Assets are tied to `conversation_id` and/or `message_id`/`task_id`.
- [ ] Object storage is private by default.
- [ ] Access uses signed URLs or authenticated proxy routes.
- [ ] Signed URLs expire.
- [ ] File size is limited.
- [ ] Supported MIME types are allowlisted.
- [ ] Real file content type is checked.
- [ ] SVG/HTML/executable/archive uploads are disabled unless specifically reviewed.
- [ ] Upload rate limits exist.
- [ ] Abuse monitoring exists for storage/CDN cost spikes.

Recommended initial allowlist:

- `image/jpeg`
- `image/png`
- `image/webp`

Check before enabling image generation:

- [ ] Image generation creates a `task`.
- [ ] Task states are explicit: `queued`, `running`, `succeeded`, `failed`, `canceled`.
- [ ] Output image is stored as an asset.
- [ ] Output asset is attached to assistant message or task.
- [ ] Refreshing the page restores task/result state.
- [ ] Failed task does not double-charge.
- [ ] Failed task writes refund or explicit no-charge ledger entry.
- [ ] Retrying a task is idempotent or creates a clearly linked new task.
- [ ] Generated images are private by default.
- [ ] Public sharing requires review/status controls.

## 6. Billing, balance, usage, and ledger

Check:

- [ ] Balance checks are enforced on the backend.
- [ ] Frontend cannot set price, cost, discount, or ledger amount.
- [ ] Backend calculates cost from model, usage, task type, and pricing table.
- [ ] Usage is measured and persisted.
- [ ] Ledger writes are idempotent.
- [ ] Duplicate request/retry cannot double-charge.
- [ ] Failed provider call refunds or records no charge.
- [ ] Timeout behavior is defined and auditable.
- [ ] Admin can inspect user usage and ledger records.
- [ ] Recharge/payment callback is idempotent.
- [ ] Negative balances are either impossible or explicitly controlled.
- [ ] Suspicious consumption triggers alerting.

Recommended ledger flow for paid tasks:

1. Pre-check balance.
2. Create message/task record.
3. Reserve/pre-charge if needed.
4. Call provider.
5. Record usage.
6. Settle final charge.
7. Refund reservation on failure.
8. Emit audit log.

## 7. Rate limiting and abuse control

Check:

- [ ] IP-level request rate limit.
- [ ] User-level request rate limit.
- [ ] Model-level rate limit.
- [ ] High-cost model daily spend cap.
- [ ] New-user/free-quota cap.
- [ ] Image generation concurrency limit.
- [ ] Task queue backpressure.
- [ ] Provider failure circuit breaker.
- [ ] Global emergency kill switch for costly providers.
- [ ] Alerts for sudden cost spikes.
- [ ] Alerts for failed payment/recharge anomalies.

Abuse scenarios to test:

- [ ] Batch signups consuming free quota.
- [ ] Repeated image generation by script.
- [ ] Multiple tabs sending duplicate paid tasks.
- [ ] Direct API call bypassing disabled frontend controls.
- [ ] User attempting to select unsupported or hidden model.
- [ ] User attempting arbitrary provider URL.

## 8. Admin and audit

Check:

- [ ] Admin endpoints require admin role server-side.
- [ ] Admin actions are audited.
- [ ] Admin views do not expose secrets.
- [ ] User content access by admin is logged.
- [ ] Refund/recharge/manual balance changes are audited.
- [ ] Provider key management is restricted.
- [ ] Dangerous admin actions require confirmation.

## 9. Logs, monitoring, and privacy

Check:

- [ ] Authorization headers are redacted.
- [ ] Cookies/session tokens are redacted.
- [ ] Provider keys are redacted.
- [ ] Payment secrets are redacted.
- [ ] Raw prompts are not logged unless explicitly required and privacy-reviewed.
- [ ] Uploaded file URLs are not permanently logged if sensitive.
- [ ] Error responses do not expose stack traces in production.
- [ ] Monitoring covers request volume, provider latency, failure rate, cost, and task queue backlog.
- [ ] Security-relevant events are auditable.

## 10. Public content and UGC

If public sharing, public galleries, comments, prompt libraries, or user profiles are enabled:

- [ ] Default content status is private or pending.
- [ ] Public content has moderation status: `pending`, `approved`, `rejected`, `removed`.
- [ ] Users can report content.
- [ ] Admin can remove content.
- [ ] Spam controls exist.
- [ ] Public pages do not leak private conversations/assets.
- [ ] Sensitive/illegal content handling process exists.

For the initial workspace launch, public sharing should remain disabled unless review infrastructure is ready.

## 11. CI and test integrity

Check:

- [ ] Security tests are not skipped.
- [ ] Tests are not weakened to pass CI.
- [ ] Test mocks preserve security semantics.
- [ ] Backend unit tests pass.
- [ ] Frontend typecheck passes when relevant.
- [ ] Contract snapshots are intentionally updated.
- [ ] Flaky tests are fixed through deterministic setup/teardown, not broad sleeps or disabled assertions.

## 12. Final go/no-go checklist

Before production release, answer these questions:

1. Can a user access another user's conversation, message, asset, task, usage, or ledger by changing an ID?
2. Can a user make the backend call an arbitrary URL?
3. Can a user make the frontend choose a cheaper/free price for a paid model?
4. Can a failed task charge without refund or audit trail?
5. Can provider keys appear in browser DevTools or logs?
6. Can image/file upload be used as a free public file host?
7. Can disabled features be triggered by direct API calls?
8. Can a script drain free quota or provider credits quickly?
9. Can admin actions happen without audit records?
10. Can CI pass because tests were weakened instead of fixed?

If any answer is "yes" or "unknown", do not release.
