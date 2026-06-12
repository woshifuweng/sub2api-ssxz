# Workspace Provider Limited Beta Runbook

This runbook defines the operational process for a limited beta of the Unified Workspace text provider path. It is a safety document only. It does not enable production provider traffic, change billing, change provider routing, or replace the required backend controls.

## Current State

- DeepSeek staging QA passed with one real staging text request.
- The verified staging model was `deepseek-v4-flash`.
- The response was close to `STAGING_OK`.
- Rollback passed: after restoring the kill switch, `/app` returned the unavailable provider placeholder again.
- The temporary Nginx staging QA path was deleted after validation.
- Production remains disabled for workspace text provider calls.
- `WORKSPACE_TEXT_PROVIDER_KILL_SWITCH` must remain `true` by default.
- DeepSeek is the first staging and limited beta provider. It is not a hardcoded special case and must be treated as an OpenAI-compatible provider behind the same gates as any future text provider.

## Safety Boundaries

The limited beta must never become a broad production opening by accident. The following boundaries are mandatory:

- Do not fully open provider access to all users.
- Do not bypass the kill switch.
- Do not bypass the beta allowlist.
- Do not bypass request or cost caps.
- Do not bypass the billing-safe execution contract.
- Do not bypass usage, audit, reconciliation, or monitoring checks.
- Do not expose API keys, tokens, `Authorization`, cookies, secrets, or full prompts.
- Do not trigger image generation, image editing, asset upload, or image task paths.
- Do not modify billing, ledger, or payment implementation as part of beta gate-on.
- Do not modify provider routing as part of beta gate-on.
- Do not modify production service or Nginx as part of beta gate-on.

## Required Env Controls

Set these only in the intended staging or beta runtime environment. Do not put secrets in Git, frontend env, screenshots, logs, chat, or test snapshots.

```env
WORKSPACE_TEXT_PROVIDER_ENABLED=true
WORKSPACE_TEXT_PROVIDER_KILL_SWITCH=true
WORKSPACE_TEXT_PROVIDER_STAGING_ONLY=true
WORKSPACE_TEXT_PROVIDER_ENVIRONMENT=staging
WORKSPACE_TEXT_PROVIDER_TEST_PROVIDER_LABEL=deepseek-staging
WORKSPACE_TEXT_PROVIDER_LOW_COST_MODEL_ALLOWLIST=deepseek-v4-flash
WORKSPACE_TEXT_PROVIDER_MAX_REQUESTS_PER_TEST_RUN=3

WORKSPACE_TEXT_PROVIDER_BILLING_ELIGIBILITY_KNOWN=true
WORKSPACE_TEXT_PROVIDER_BILLING_ELIGIBLE=true
WORKSPACE_TEXT_PROVIDER_BILLING_POLICY=record_usage_on_provider_reported_usage
WORKSPACE_TEXT_PROVIDER_USAGE_POLICY=record_provider_reported
WORKSPACE_TEXT_PROVIDER_FAILURE_POLICY=provider_failure_no_charge

WORKSPACE_TEXT_PROVIDER_BETA_ALLOWLIST_ENABLED=true
WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_USER_IDS=<comma-separated beta user ids>
WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_GROUP_IDS=<comma-separated beta group ids>
WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_PROVIDER_LABELS=deepseek-staging
WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_MODELS=deepseek-v4-flash

WORKSPACE_TEXT_PROVIDER_BETA_DAILY_REQUEST_CAP=10
WORKSPACE_TEXT_PROVIDER_BETA_TEST_RUN_REQUEST_CAP=3
WORKSPACE_TEXT_PROVIDER_BETA_PROVIDER_REQUEST_CAP=10
WORKSPACE_TEXT_PROVIDER_BETA_MODEL_REQUEST_CAP=10

DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-v4-flash
DEEPSEEK_API_KEY=<server-side secret only, never print>
```

Production defaults must remain disabled:

```env
WORKSPACE_TEXT_PROVIDER_ENABLED=false
WORKSPACE_TEXT_PROVIDER_KILL_SWITCH=true
WORKSPACE_TEXT_PROVIDER_STAGING_ONLY=true
```

## Beta Gate-on Procedure

Follow this order exactly for each beta gate-on window:

1. Confirm production remains disabled and unaffected.
2. Confirm the staging or beta service is active.
3. Confirm `WORKSPACE_TEXT_PROVIDER_KILL_SWITCH=true`.
4. Confirm the beta allowlist includes only the intended test user IDs or group IDs.
5. Confirm request caps are configured and greater than `0`.
6. Confirm billing, usage, and failure policy env values are configured.
7. Confirm provider key presence only by checking that the server-side variable is set and non-empty. Do not print the value.
8. Run a browser gate-off test in `/app`; it must return the unavailable provider placeholder.
9. Confirm browser Network does not show a direct request to `api.deepseek.com` or any provider credential.
10. Set `WORKSPACE_TEXT_PROVIDER_KILL_SWITCH=false` only in the staging or beta environment.
11. Restart only the staging or beta service.
12. Have one beta user send one low-cost text request.
13. Verify response, usage metadata, audit metadata, reconciliation report, and monitoring summary.
14. Immediately restore `WORKSPACE_TEXT_PROVIDER_KILL_SWITCH=true`.
15. Restart only the staging or beta service again.
16. Run a rollback test in `/app`; it must return the unavailable provider placeholder.

Never use this procedure to gate-on production broadly.

## Rollback Procedure

Rollback is the default response to uncertainty.

1. Set `WORKSPACE_TEXT_PROVIDER_KILL_SWITCH=true`.
2. Restart only the staging or beta service.
3. Verify provider call count no longer increases.
4. Verify DeepSeek request count no longer increases.
5. Verify `/app` returns the unavailable provider placeholder.
6. Verify production is unaffected.
7. If the provider path still appears reachable, set `WORKSPACE_TEXT_PROVIDER_ENABLED=false`.
8. Preserve audit, usage, reconciliation, monitoring, and service logs for investigation, but do not print secrets.

## Monitoring / Alert Handling

The monitoring helper converts reconciliation reports and safety signals into alert candidates. Every alert must be handled without printing secrets, full prompts, provider headers, internal URLs, stack traces, or SQL.

| Alert | Severity | Immediate action | Kill switch | Inspect | Do not print |
|---|---:|---|---|---|---|
| `provider_error_rate_exceeded` | warning | Pause beta expansion and inspect recent provider failures. | Turn on if failures affect beta users or are unexplained. | Provider error codes, fallback count, affected model/provider label. | Provider key, full prompts, raw upstream response headers. |
| `provider_consecutive_failures` | warning | Stop new beta requests until consecutive failures are understood. | Turn on if threshold is reached. | Consecutive failed request IDs, provider status, timeout logs. | Key, token, `Authorization`, cookies. |
| `provider_timeout_rate_exceeded` | warning | Reduce traffic and inspect provider latency. | Turn on if users are blocked or retries pile up. | Latency, timeout error code, request cap counters. | Internal stack traces or full base URLs if sensitive. |
| `usage_missing_detected` | warning | Stop beta expansion and reconcile usage metadata. | Turn on if any successful provider call lacks usage. | Message ID, audit metadata, usage fields. | Full prompt, provider key. |
| `audit_missing_detected` | warning | Stop beta expansion and recover audit context. | Turn on if provider calls continue without audit. | Request ID, message ID, audit status, error code. | Raw secrets, cookies, tokens. |
| `provider_called_without_usage` | warning | Treat as billing-risk; reconcile immediately. | Turn on until resolved. | Usage metadata, provider response usage, message status. | Provider credentials or raw prompt. |
| `provider_called_without_audit` | warning | Treat as observability-risk; reconcile immediately. | Turn on until resolved. | Audit status, endpoint label/hash, model metadata. | Secret headers, full prompt. |
| `counter_blocked_but_provider_called` | critical | Stop beta traffic immediately. | Turn on immediately. | Beta counter decision, provider_called flag, request ID. | Key, token, `Authorization`, cookies. |
| `kill_switch_blocked_but_provider_called` | critical | Stop all workspace provider traffic and investigate gate order. | Turn on and keep on. | Kill switch env, service restart time, provider_called metadata. | Secrets or raw prompts. |
| `browser_direct_provider_call_detected` | critical | Stop beta and check frontend/runtime routing. | Turn on immediately. | Browser Network, frontend base URL config, backend proxy path. | Provider key, bearer token, screenshots containing secrets. |
| `key_or_token_leakage_signal` | critical | Rotate affected credential and stop beta. | Turn on immediately. | Where leakage was detected, affected scope, rotation status. | The leaked value itself. |
| `billing_ledger_payment_anomaly_signal` | critical | Stop beta and reconcile usage before further traffic. | Turn on immediately. | Usage/audit records, ledger deltas, payment callbacks if any. | Payment secrets, provider keys. |
| `image_asset_task_unexpected_signal` | warning | Stop the triggering path and verify disabled capabilities. | Turn on if provider path invoked image/asset/task code. | Intent/capability metadata, route logs, task creation. | User-uploaded content, signed URLs, secrets. |

## Beta Entry Criteria

Do not start a limited beta unless all items below are true:

- Staging QA passed.
- Rollback passed.
- Temporary Nginx staging path was removed.
- Beta allowlist is configured for a small user/group set.
- Request caps are configured and non-zero.
- Reconciliation report helper is available.
- Monitoring alert helper is available.
- Provider key is stored server-side only.
- Production remains disabled.
- Browser Network has no direct provider calls.
- No key, token, `Authorization`, or cookie leakage was observed.
- Image, asset, and task paths were not touched.

## Beta Exit Criteria

Successful beta exit requires:

- 24 to 72 hours with no key or token leakage.
- Provider success rate meets the beta target.
- Usage, audit, and reconciliation reports remain consistent.
- Request caps are enforced.
- Error and fallback events are explainable.
- User experience is acceptable for the allowed beta cohort.

Failed beta exit is required if any of these occur:

- Any secret leakage signal.
- `provider_called_without_usage`.
- `provider_called_without_audit`.
- `counter_blocked_but_provider_called`.
- `kill_switch_blocked_but_provider_called`.
- Billing, ledger, or payment anomaly.
- Non-beta users reach the provider path.
- Image, asset, or task path is triggered unexpectedly.
- Consecutive provider failures or timeout rate exceeds threshold.

## Temporary Nginx Path Policy

Temporary Nginx staging paths are not a beta access strategy. They are disabled by default and should not be used again unless there is no safer alternative.

If a temporary path is unavoidable:

1. Use a random long path.
2. Proxy only to the staging service.
3. Do not expose production provider access through it.
4. Delete it immediately after validation.
5. Verify the old URL returns `404`.
6. Run `nginx -t` before and after removal.
7. Ensure no `sites-enabled/*.bak` file remains included by Nginx.
8. Document why SSH tunnel or internal access was not sufficient.

## Incident Checklist

### Secret leakage suspected

- Turn on the kill switch.
- Disable provider if needed.
- Rotate the affected secret.
- Preserve evidence without copying the secret value.
- Review browser Network, logs, audit output, and screenshots for exposure.

### Provider error spike

- Turn on the kill switch if error rate or consecutive failures exceed threshold.
- Inspect provider error codes, latency, fallback count, and affected model/provider label.
- Keep request caps in place.

### Usage or audit missing

- Turn on the kill switch for successful provider calls missing usage or audit.
- Reconcile message, audit, usage, and provider metadata by request ID.
- Do not treat usage-missing requests as silently successful.

### Request cap bypass

- Turn on the kill switch.
- Inspect beta counter metadata.
- Confirm user/provider/model/test-run counters are enforced before provider calls.

### Beta allowlist bypass

- Turn on the kill switch.
- Confirm user ID, group ID, provider label, and model allowlist decisions.
- Remove any overly broad allowlist values.

### Kill switch failure

- Set `WORKSPACE_TEXT_PROVIDER_ENABLED=false`.
- Restart the affected service.
- Inspect gate order and provider_called metadata.

### Billing anomaly

- Turn on the kill switch.
- Preserve usage, audit, and ledger context.
- Do not change billing, ledger, or payment logic during incident triage.

### Browser direct provider call

- Turn on the kill switch.
- Stop beta traffic.
- Inspect frontend configuration and API proxy behavior.
- Confirm provider keys are not present in frontend env, bundle, or browser storage.

### Accidental production impact

- Turn on all provider kill switches.
- Confirm production service and Nginx were not changed as part of beta.
- If production was changed, rollback production config separately and document the exact change.

## Do Not Do

- Do not send provider keys through chat, GitHub, logs, screenshots, or test snapshots.
- Do not `cat` a full environment file containing secrets.
- Do not fully open the provider path.
- Do not directly enable the provider path on production.
- Do not bypass the billing-safe execution contract.
- Do not hardcode DeepSeek as the only provider.
- Do not use a temporary Nginx path as a beta access mechanism.
- Do not loosen request caps to hide failures.
- Do not weaken or skip tests to pass CI.
- Do not add image, asset, task, web, memory, or toolbox behavior as part of text provider beta.
