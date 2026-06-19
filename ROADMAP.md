# ROADMAP

Last updated: 2026-06-20

## Product Goal

Build SSXZ AI into a lightweight AI creation workspace for about 200-300 private users, while preserving the Sub2API backend strengths: login, balance, usage, payment, API keys, admin operations, provider/account configuration, and security boundaries.

## P0: Structure Convergence

### Goals

- Make `/app/*` the ordinary-user workspace family.
- Keep ordinary-user pages inside the user workspace shell.
- Stop ordinary-user navigation from jumping into the older admin/backend-looking shell.
- Preserve existing backend, billing, provider routing, payment, and database behavior.
- Keep the 2026-06-20 staging image-generation success as evidence, not as production approval.

### P0 Work Items

1. Confirm user workspace route ownership:
   - `/app/chat`
   - `/app/image`
   - `/app/usage`
   - `/app/keys`
   - `/app/profile`
2. Keep legacy routes for compatibility:
   - `/keys`
   - `/usage`
   - `/profile`
   - `/sora`
3. Decide the user-shell plan for:
   - recharge/purchase
   - orders
   - subscriptions
4. Remove stale default user redirects to `/sora` where they conflict with the product direction.
5. Keep API Key / third-party access visible to ordinary users.
6. Keep technical channel/status/provider pages out of ordinary-user primary navigation.
7. Keep image generation release gates explicit:
   - one staging success path was verified on 2026-06-20
   - model/account permission must remain correct
   - billing/usage must remain correct
   - image history must remain visible
   - download must work in a normal browser
   - upstream failure must not fake success or mischarge users
   - service-level, handler-level, DB-backed image-history, and DB-backed usage/billing failure regression tests exist
8. Document P0 decisions before UI polish work resumes.

### P0 Non-Goals

- No production deployment.
- No database migration.
- No payment or ledger redesign.
- No provider routing redesign.
- No broad Sora-to-image internal rename.
- No large visual redesign.
- No new smart prompt feature.
- No new external provider integration.

## P1: Product Experience And Operations

### Goals

- Make the ordinary-user product feel like a lightweight AI creation tool, not a technical relay console.
- Preserve owner/admin ability to operate the private user base.
- Improve image and chat workflows without weakening billing/security.

### P1 Work Items

1. Improve Image Studio as the main image-creation experience:
   - prompt input
   - style and ratio controls
   - custom style and custom ratio
   - multiple reference images
   - thumbnail/results area
   - image history and download
   - regenerate flow
   - clear loading/error/empty states
2. Improve chat as prompt assistance and general conversation:
   - keep `/app/chat`
   - evaluate old `ChatStudioView` assets
   - assess prompt-assistance options without hard-wiring one model/provider
3. Evaluate product intelligence before implementing it:
   - review existing DeepSeek, web search, and prompt enhancement assets
   - decide whether image prompt assistance should use rules, templates, chat models, web reference, or a staged combination
   - verify cost, latency, failure behavior, and user value before implementation
4. Improve API Key / third-party access:
   - user-friendly copy
   - clear Base URL/model guidance
   - key masking and safe one-time full-key display
5. Improve user billing pages:
   - balance
   - usage
   - recharge
   - orders/subscriptions
6. Preserve and clarify admin operation pages:
   - users
   - balances
   - orders/payment
   - usage
   - image tasks/history
   - API keys
   - invite/redeem/promo codes
   - model/channel/account/pricing basics
7. Add security hardening as separate PRs:
   - rate limits
   - server-side captcha validation if captcha is used
   - unified auth error wording
   - API key log redaction
   - admin operation logs

### P1 Non-Goals

- No enterprise RBAC overhaul.
- No full generic model gateway rewrite.
- No replacement of the Sub2API trunk.
- No provider switch without an external dependency check and operator approval.

## P2: Visual Polish And Enhanced Experience

### Goals

- Make the product feel polished, simple, and trustworthy after P0/P1 flows are structurally correct.

### P2 Work Items

- Visual system refinement.
- Mobile layout refinement.
- Image prompt templates.
- Prompt intelligence evaluation and possible implementation.
- Optional web reference/prompt research flow.
- Better onboarding and examples.
- Better admin dashboards and owner summaries.
- More complete analytics and support tooling.

### P2 Non-Goals

- No cosmetic redesign that hides broken navigation structure.
- No template copying without checking fit for the SSXZ AI private-user product.
- No dependency-heavy UI rewrite unless maintenance cost is justified.

## Production Release Gates

Production deployment should wait until all gates below are satisfied:

1. Ordinary-user navigation stays inside the intended user workspace shell.
2. `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/app/profile` behave consistently after refresh.
3. API Key / third-party access is visible and safe for ordinary users.
4. Image generation keeps the validated staging path healthy:
   - generation succeeds
   - cost/usage is recorded correctly
   - image history is visible
   - download works in a normal browser
   - failure does not mischarge users
5. Payment/order flows are either verified or clearly disabled with honest user-facing states.
6. Admin pages remain admin-only and usable for operations.
7. No provider keys, API keys, cookies, tokens, or Authorization values are printed or persisted in unsafe places.
8. Frontend typecheck/build and relevant tests pass.
9. The user explicitly approves production deployment.
