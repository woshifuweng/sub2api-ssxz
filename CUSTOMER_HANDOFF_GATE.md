# CUSTOMER_HANDOFF_GATE

Last updated: 2026-07-08

## Current Decision

SSXZ is in P1 commercial handoff convergence. The near-term launch shape is a controlled private AI relay beta, not a public paid launch and not a full AI Workbench launch.

Allowed:

- whitelist or invite-only customers
- low-balance / low-quota API Keys
- one customer, one key, one client, one small text request as the first acceptance flow
- API Key plus third-party clients as the main sellable path
- `/app/chat` and `/app/image` only as light test/internal entries

Not allowed to promise:

- unlimited usage
- exact first-token speed
- 100% uptime
- Sora availability
- full image product maturity
- public self-serve paid traffic
- automatic support for every third-party client variant

## Production No-Provider Gate Evidence

2026-07-08 production read-only / no-provider validation:

| Area | Result |
| --- | --- |
| Public pages | `/home`, `/login`, `/register`, `/app/dashboard`, `/app/keys`, `/app/usage`, `/app/channel-status`, `/app/purchase`, `/app/orders`, `/app/redeem`, `/app/profile` returned HTTP 200. |
| Public settings | `/api/v1/settings/public` returned HTTP 200 with API base URL settings present. |
| Ordinary login | Ordinary user login succeeded. |
| Admin login | Admin login succeeded. |
| Ordinary user data | Available groups: 2. Available channels: 1. API Keys: 7. Usage total: 83. Redeem history: 3. |
| Permission boundary | Ordinary user access to `/api/v1/admin/redeem-codes` returned HTTP 403. |
| Admin operations | Admin users, usage, redeem codes, payment orders, groups, and dashboard stats returned HTTP 200. |
| Provider calls | 0. No `/v1/responses`, `/v1/chat/completions`, image generation, or real provider request was called. |
| Writes | 0. No API Key, order, redeem, balance, payment, provider, database, or config change was made. |

This evidence means the customer handoff shell and support surfaces are reachable. It does not prove every real customer client is configured correctly.

## First Customer Handoff Flow

Use this flow for the first 10-20 customers. Keep it short.

1. Create or approve the customer account.
2. Ensure the account has a small test balance or redeem/manual credit.
3. Create one API Key bound to the correct group.
4. Give the customer only:
   - Base URL: `https://api.ssxzapi.com/v1`
   - API Key
   - one recommended model
   - one recommended client
5. Let the customer send one small text request.
6. Confirm:
   - the request succeeds or fails with an understandable reason
   - balance did not go negative
   - usage record is visible
   - admin can find the customer, key, usage, and related evidence
7. Raise quota only after the first request is stable.

## Customer-Facing Guidance

Keep customer copy practical and not over-technical.

Suggested short copy:

> First connect with Base URL `https://api.ssxzapi.com/v1` and the provided API Key. Start with the recommended model and a small test request. Different models vary in speed and capability; complex requests may take longer. If quota is insufficient or a model is unavailable, the request will fail with a reason.

Avoid telling customers:

- exact upstream account details
- provider routing internals
- raw cost formulas
- internal failover strategy
- admin diagnostics paths
- unsupported future features

Do tell customers:

- which model to use first
- whether they have enough quota
- whether a failure is quota, model, client config, or temporary service issue
- whether they should retry later or switch to another model

## Operator Checklist

Before handing a key to a customer:

- Account exists and is enabled.
- Balance/test quota is present.
- API Key is active.
- API Key has a group binding.
- Key quota/rate limits are intentional.
- Customer is told to use `https://api.ssxzapi.com/v1`.
- Customer uses one known client first: CC Switch, Cherry Studio, Chatbox, or another OpenAI-compatible client.
- First test uses a small text request, not image generation or long web/search tasks.

After the first request:

- Check user usage rows.
- Check key usage and last-used state.
- Check balance did not go negative.
- Check admin usage can locate the request by user/key/model/time.
- If payment is involved, check order and balance arrival.
- If redeem/manual credit is involved, check redeem history and balance history.

## Common Failure Classification

| Symptom | Likely meaning | Operator response |
| --- | --- | --- |
| 401 invalid API key | Wrong key, pasted masked key, or client using the wrong provider/auth mode. | Issue or copy a fresh full key from the one-time reveal flow. Confirm the client is using API Key auth. |
| 403 insufficient balance | Customer/key/group does not have enough usable quota for that request. | Add a small test balance or ask the customer to use a cheaper/smaller request. |
| 403 quota/rate limit | Key or group limit reached. | Check key quota and rate windows before raising limits. |
| 503 provider unavailable | Upstream account/provider temporarily unavailable or client path mismatch. | Try the recommended model/client path first; do not promise instant recovery. |
| Very slow response | The model is doing more work, the request is complex, or upstream is slow. | Suggest a faster model or simpler request; avoid promising exact speed. |
| No usage row | The request may not have reached the gateway or was rejected before billing. | Check client Base URL, key, and model first. |
| Usage row but customer confused | The response worked, but the customer does not understand cost/source. | Explain at a high level: usage is recorded by request, model, and account balance. |

## CC Switch Handoff Rule

Do not rebuild CC Switch from scratch inside SSXZ.

The existing SSXZ/Sub2 capability should remain the first path:

- newly created full API Keys may offer CCS import or config material
- masked list keys must stay non-importable
- unknown or missing platform keys must not silently default to Claude
- customer testing should start with one recommended provider/client path

If a customer machine fails, collect evidence first:

- screenshot of client provider settings
- selected API mode
- Base URL
- model name
- HTTP status
- request id if visible
- whether the key is a full key or a masked list key

Only then decide whether a code fix is needed. Do not create a parallel CC Switch configuration system until the existing import path is proven insufficient.

## Release Gate Before Broader Paid Use

Broader paid use is allowed only after a separate release gate confirms:

- at least one real customer completes API Key handoff with a third-party client
- usage and balance evidence is visible to the user and admin
- failed requests do not charge incorrectly
- insufficient balance blocks instead of going negative
- order/redeem/manual-credit flow is supportable
- admin can find user, key, usage, order/redeem, group, and channel evidence quickly
- production deployment state is known and rollback path is available

Until then, keep customers controlled, low quota, and manually supported.
