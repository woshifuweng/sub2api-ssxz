# SSXZ Existing Capability Inventory

Date: 2026-07-03

This document records what the current SSXZ / Sub2API fork already has, what is intentionally hidden or disabled, and what must be checked before building anything new.

Current product line:

- Main line: private-domain AI API relay operation platform based on Sub2API.
- Secondary line: lightweight SSXZ Workbench entry for model testing and image beta only.
- Do not expand Agent, workflow, Lovart-like canvas, PS/MCP, or complex image design in the current phase.

## Before Building Anything New

Every new request must first answer:

1. Is this already present in the current Sub2API / SSXZ code?
2. Is it hidden by role, feature flag, full-key security, model capability, balance, group, or backend config?
3. Does old Sub2 already provide the page, handler, or API?
4. Is a mature client or open-source pattern only needed as reference, not as replacement?
5. Can the fix reuse an existing page, route, handler, or store?
6. What is the smallest user-visible correction?

## Existing User-Facing Capabilities

| Area | Existing surfaces | Notes / disable conditions | Do not rebuild |
| --- | --- | --- | --- |
| Public entry | `/home`, `/login`, `/register`, email verification, password reset, OAuth callbacks | Registration can be disabled; email verification, promo code, invitation code, and affiliate code are setting-driven | Do not build a second auth flow |
| User dashboard | `/app/dashboard` | Main authenticated landing page after login | Reuse dashboard cards and quick actions |
| API keys | `/app/keys`, `/api/v1/keys` | Full key is only visible at creation; list is masked; inactive/expired/quota/group/balance can block usage | Do not rebuild key management |
| CC Switch / clients | Base URL guide, CC Switch deeplink logic, Cherry Studio / Chatbox / CC Switch text | List import is disabled for masked keys. Best fix is import from create-success full-key dialog | Reuse `importToCcswitch` instead of new import logic |
| Usage records | `/app/usage`, usage detail/stats/dashboard APIs | Has model, endpoint, tokens, cost, first token field, client/source style data foundations | Do not create a separate ledger view from scratch |
| Channel status | `/app/channel-status`, `/api/v1/channel-monitors` | Hidden unless `channel_monitor_enabled`; monitor data depends on backend config | Reuse channel monitor system |
| Available channels | `/app/available-channels`, `/api/v1/channels/available` | User-facing availability list exists | Do not invent a second model availability page |
| Recharge/orders | `/app/purchase`, `/app/orders`, payment APIs | Hidden or redirected when `payment_enabled` is false; payment provider behavior must stay backend-led | Do not touch payment flow casually |
| Redeem codes | `/app/redeem`, admin redeem code APIs | Supports balance, concurrency, subscription, invitation types | Reuse redeem system for codes |
| Affiliate/referral | `/app/affiliate`, `/api/v1/user/aff`, admin affiliate endpoints | Recently closed: registration bind and paid order rebate accrual exist | Do not rebuild inviter/rebate tracking |
| Profile/security | `/app/profile`, password, profile, TOTP endpoints | Existing profile cards and forms | Reuse existing account security UI |
| Chat test entry | `/app/chat` | Downgraded to lightweight model/API test entry; image upload in chat is disabled unless capability and policy allow | Do not make it the main commercial product right now |
| Image beta | `/app/image`, image studio/gateway APIs | Beta/internal image entry; real model/channel/billing capability must be checked | Do not expand complex image design now |
| Sora legacy | `/sora`, Sora gateway/admin routes | Must remain hidden/fail-closed unless explicitly re-enabled; Sora is not current main line | Do not restore Sora experience work |

## Existing Gateway / API Capabilities

The gateway already exposes OpenAI-compatible and adjacent routes:

- `/v1/models`
- `/v1/responses`
- `/v1/chat/completions`
- `/v1/images/generations`
- `/v1/images/edits`
- `/v1/usage`
- `/v1beta/*` Gemini-compatible routes
- Claude Code compatibility endpoints
- Antigravity compatibility endpoints
- Sora compatibility endpoints, currently not product priority

Image generation/edit endpoints are not automatically available for every group/provider. They depend on platform, account, model, pricing, capability, and balance.

## Existing Admin / Owner Capabilities

| Area | Existing backend/admin capability |
| --- | --- |
| Users | list, create, edit, delete, balance adjustment, user API keys, user usage, balance history, group replacement, user attributes |
| Groups | list, sort, create, edit, delete, capacity summary, usage summary, rate multipliers |
| Accounts/providers | create, edit, delete, test, refresh, recover state, clear errors, quota reset, schedulability, model refresh, import/export, bulk actions |
| Channels/pricing | channel management and model pricing screens exist |
| Channel monitor | monitor CRUD, run/history/status, templates |
| Payment/orders | payment config, plans, providers, order list/detail/cancel/retry/refund |
| Redeem/promo | redeem code generation/export/expire/delete, promo code CRUD and usage records |
| Affiliate | admin affiliate user list and rebate-related audit path |
| Ops | concurrency, realtime traffic, gateway scheduler, OpenAI WS runtime, alert rules/events/silences, email notification config, errors, request errors, upstream errors, request details, system logs, dashboard snapshots/trends |
| Proxy | proxy CRUD, test, quality check, stats, import/export, batch delete/create |
| Settings | public settings, registration, email, API base URL, CCS import hiding, Sora switch, admin API key, overload cooldown |

Admin pages and routes are already broad. The near-term work should be trimming, Chinese copy, role safety, and acceptance checks, not rebuilding admin.

## Hidden / Disabled Conditions To Check First

- `hide_ccs_import_button`: hides CC Switch import.
- Masked API key list values: direct import and full-key copy are disabled by design.
- `channel_monitor_enabled`: controls channel status/monitor visibility.
- `payment_enabled`: controls recharge/orders access.
- `sora_client_enabled`: controls Sora client visibility; keep off unless explicitly approved.
- `registration_enabled`: controls public registration.
- `email_verify_enabled`: controls email verification flow.
- `promo_code_enabled`: controls promo code registration field.
- `invitation_code_enabled`: controls invitation-code registration requirement.
- `affiliate_enabled`: controls affiliate presentation.
- `web_search`: controlled by enabled, kill switch, user allowlist, caps, provider availability.
- Chat image/file upload: controlled by current product policy and model capability.
- API gateway calls: can fail because of inactive key, expired key, quota, balance, group, model, provider account, provider route, or upstream status.

## Current Reuse Decisions

- CC Switch import exists. Fix should add import at the create-success full-key step, not enable masked-key list import.
- Affiliate/referral exists. Future work should improve owner reporting, not rebuild inviter tracking.
- Usage and ledger foundations exist. Future work should clarify customer-facing copy and owner-facing diagnostics.
- Channel status exists. Future work should validate first-token and availability display against real monitor data.
- Payment/order/redeem/promo exist. Future work should acceptance-test and simplify, not redesign.
- Admin ops is broad. Future work should decide what owner needs daily and hide noisy/deep ops from normal user paths.

## Customer-Facing Versus Owner-Facing Detail

Customer-facing pages should explain what is actionable:

- balance is insufficient
- model is unavailable
- request is still processing
- service is temporarily busy
- choose a faster model for lower latency
- high-quality/deep requests can take longer

Owner/admin pages can show deeper details:

- provider account
- upstream error
- route/provider status
- cost, ledger, request ID, usage ID
- monitor latency and first token
- alert and system logs

Do not expose unnecessary internal provider/routing/cost mechanics to ordinary users.

## Known Gaps

1. CC Switch import should be available immediately after creating a key while the full key is still visible.
2. Some existing admin/ops features need productized Chinese labels and owner-focused grouping.
3. Channel monitor, first-token, and availability should be verified against real runtime data before being sold as a promise.
4. Payment, order, balance, usage, and affiliate paths need end-to-end acceptance evidence before broad paid traffic.
5. Workbench should remain secondary until the API relay commercial loop is stable.

## Recommended Next Small PR

PR: API Key create-success client import.

Scope:

- Reuse existing CC Switch deeplink/import builder.
- Add "Import to CCS" in the create-success full-key reveal dialog.
- Keep existing list-row import disabled for masked keys.
- Do not change backend, payment, provider routing, DB schema, Nginx, or production config.
