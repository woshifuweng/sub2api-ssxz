---
name: security-review
description: Review security boundaries for AI relay, unified chat workspace, provider routing, assets, billing, and user data isolation.
---

# Security Review

Use this skill for PRs touching workspace, API relay, provider calls, model selection, file/image upload, billing, ledger, auth, admin, logs, or task execution.

## Required reads

1. AGENTS.md.
2. docs/security/launch-checklist.md.
3. docs/security/unified-chat-workspace-v1-security-strategy.md.

## Security checklist

### Secrets
- No provider keys in frontend.
- No payment secrets, OAuth secrets, JWT secrets, database credentials, API keys, cookies, or tokens in code/logs/errors/snapshots.

### Frontend is not a security boundary
- Disabled UI is not authorization.
- Backend must enforce auth, ownership, model capability, intent, balance, asset ownership, task ownership, admin permissions, and rate limits.

### User ownership
Check conversation/message/asset/task/usage/ledger IDs:
- frontend must not trust user_id
- backend must verify resource ownership
- user A must not access user B resource by ID

### Provider and SSRF
- no arbitrary provider base_url
- model names must come from allowlist / capability registry
- block localhost, 127.0.0.1, ::1, private ranges, metadata endpoints, internal hosts

### Billing
- frontend cannot decide price/cost/billable/refund
- ledger must be backend-calculated and idempotent
- failed provider call must refund or record no-charge

### Assets
- no long-term base64/data URL in message content or asset payload
- uploaded/generated files must become asset records
- private by default
- short-lived signed URLs or authenticated access
- MIME and size allowlist

### Logs/errors
- no provider keys
- no Authorization/token/secret/cookie
- no stack trace/internal URL/SQL in user-visible error

### Prompt/tool safety
- user content is untrusted
- model output cannot directly trigger payments/refunds/delete/permission changes/provider key changes

## Output severity

- blocker: must fix before merge
- high: strongly fix before merge
- medium: can defer if documented
- low: cleanup or UX concern

For each finding:
- severity
- file path
- issue
- risk
- minimal fix
- merge blocking: yes/no
