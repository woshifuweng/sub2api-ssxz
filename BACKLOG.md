# BACKLOG

Last updated: 2026-06-18

## P0 Bugs And Structural Fixes

- User-side menu entries must not jump into the old admin/backend-looking shell.
- `/app/usage`, `/app/keys`, and `/app/profile` should remain in the new user workspace shell after navigation and refresh.
- Purchase, orders, and subscriptions need a decision: either user-workspace shell pages or clearly documented legacy compatibility routes.
- Stale ordinary-user redirects to `/sora` should be replaced with the agreed user entry after route ownership is decided.
- `/sora` should remain a compatibility route, not the main product route.
- Technical pages such as Available Channels and Channel Status should not dominate ordinary-user navigation.
- Image generation must be verified as a real end-to-end loop before production.
- Placeholder workspace controls must stay clearly disabled until their backend/product behavior exists.

## Chains That Need Verification

- Login/register to user workspace entry.
- `/app/chat` text chat.
- `/app/image` image generation:
  - permission/group
  - model/account
  - request payload
  - billing/usage
  - history
  - download
  - failure refund/no-charge behavior
- `/app/usage` real balance, monthly trend, and detail data.
- `/app/keys` create/copy/delete/reset behavior and key masking.
- `/app/profile` profile/password/TOTP behavior.
- Recharge/payment/order flow:
  - enabled/disabled states
  - user order list
  - admin order visibility
  - no accidental fake success state
- Admin user/balance/order/usage workflows.

## UI/UX Improvements

- User workspace information architecture.
- Image Studio page:
  - clearer prompt flow
  - custom ratio
  - custom style
  - multiple reference images
  - thumbnail previews
  - results area
  - regenerate
  - image history
  - download
  - loading/error/empty states
- Chat page:
  - prompt assistance positioning
  - examples for common private-user tasks
  - better transition between chat and image creation
- API Key / third-party access:
  - Base URL guidance
  - Cherry Studio / Chatbox / CC Switch usage guidance
  - safe key visibility states
- Usage/balance/recharge pages:
  - simple cards
  - real data first
  - no fake charts
- Mobile layout pass after route structure is stable.

## Product Intelligence Backlog

- Evaluate old chat/image product assets before building new prompt intelligence.
- Review existing DeepSeek, web search, and query-quality work as possible assets, not fixed solutions.
- Evaluate simple prompt templates and rule-based prompt expansion before adding model calls.
- If model-assisted prompt writing is added, keep the output as image-generation prompt help, not generic Q&A.
- If web reference is added for image prompts, make it optional, bounded, and clearly separated from ordinary chat web search.
- Check external model/provider/API lifecycle, pricing, and limits against official sources on the decision date before selecting an upstream.
- Measure cost, latency, failure modes, and billing impact before production rollout.

## Security Improvements

- Login/register/invite-code rate limiting review.
- Server-side captcha or human-verification enforcement if captcha is enabled.
- Unified auth error messages to reduce account enumeration.
- Session/cookie security review:
  - HttpOnly
  - Secure
  - SameSite
  - idle and maximum lifetime
  - logout invalidation
- API Key storage, masking, logging, and reset flow review.
- Admin sensitive operation logs:
  - balance adjustment
  - user disable
  - API key revoke/reset
  - order/payment handling
- Backend ownership checks for user content, generated images, API keys, usage, and orders.
- Secret redaction in logs, errors, snapshots, and screenshots.

## Admin And Operations Improvements

- Clarify owner/super-admin versus ordinary admin capabilities.
- Keep owner/operator access to:
  - users
  - balances
  - orders
  - usage
  - image tasks
  - image history
  - API keys
  - invite/redeem/promo codes
  - model/channel/account/pricing basics
  - abnormal tasks and support records
- Avoid weakening admin pages while simplifying ordinary-user UX.
- Add or refine support-friendly views only after P0 user/admin boundary is stable.

## Deferred Items

- Full visual redesign.
- Prompt intelligence implementation.
- Web-search-powered prompt reference flow.
- Enterprise RBAC.
- Generic model gateway rewrite.
- Provider routing redesign.
- New external provider integration.
- Broad internal rename of Sora-related code.
- Production deployment.
