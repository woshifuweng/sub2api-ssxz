# ROADMAP

Last updated: 2026-06-29

## Product Goal

Build SSXZ AI into a lightweight AI creation workspace for about 200-300 private users, while preserving the Sub2API backend strengths: login, balance, usage, payment, API keys, admin operations, provider/account configuration, and security boundaries.

## Current Phase As Of 2026-06-29

The project can start P1 planning and small P1 PRs. P0/P0-Beta structural convergence has enough evidence to move forward:

- ordinary-user workspace routes are established around `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/app/profile`
- chat P0 fixes were validated on staging
- image failure/no-charge paths have regression coverage
- production has been smoke-tested after the 2026-06-21, 2026-06-27, and 2026-06-28 explicit-approval deployments
- PR #184 production validation confirmed ordinary users do not see non-real image-capable models
- PR #186 was deployed to staging only and confirmed `/app/image` uses the server-side OpenAI-compatible image allowlist. The current staging allowlist includes Gemini-named aliases, which is a model-display/product-clarity decision rather than native Gemini provider execution evidence.
- PR #188 was deployed to production after explicit approval and clarified OpenAI-compatible image alias labels without enabling production image generation.
- PR #190 was deployed to staging only and restored the full frontend lint baseline for the user workspace shell surface. Production remained on the prior #188 binary.
- PR #192 was deployed to staging only and made `/app/image` recent-history load failures visible while strengthening download href/filename test coverage. Production remained on the prior #188 binary.
- PR #194 was deployed to staging only and confirmed staging image-capable catalog exposure still comes from real channel data while non-real image-capable models stay hidden.
- PR #195 was merged to main and completed the first `/app/keys` P1 copy/safety polish slice. It was later included in the 2026-06-28 staging-only deployment batch.
- PR #197, PR #198, and PR #199 were merged to main and deployed to staging only as batch `261e3785b`. The batch covered `/app/usage` explanation states, `/app/profile` TOTP failure clarity, and `/app/keys` masked-key configuration guards. Staging route smoke passed; production was not deployed and no real provider was called.
- PR #201 was deployed to staging only and verified the `/app/usage` ordinary/admin data boundary: ordinary usage rows no longer expose internal user/account/provider fields, ordinary users get HTTP 403 on admin usage, admin usage still keeps operational fields, and `/app/usage` refreshes user balance through the auth store. Production was not deployed and no real provider was called.
- PR #203 was deployed to staging only and aligned Google/Gemini-compatible API-key auth restrictions with the standard API-key middleware for IP restrictions, expired keys, and quota-exhausted keys. Production was not deployed and no real provider was called.
- PR #208 was deployed to staging and then production after explicit approval. It prevents final balance billing from deducting ordinary usage below zero, and insufficient-balance usage-billing attempts roll back dedup, API-key quota, and rate-usage side effects. No real provider was called.
- PR #214 was merged to main, deployed to staging, and then deployed to production after explicit approval through main `10f95cd25`. It adds estimated-cost pre-provider billing eligibility checks for bounded generic token requests on Claude/Anthropic-compatible gateway paths and Gemini v1beta. No real provider was called during deployment or smoke.
- PR #217 was merged to main, deployed to staging, and then deployed to production through main `db9d736be` after CI and staging smoke passed. It adds estimated-cost pre-provider billing eligibility checks for bounded OpenAI Responses WebSocket requests before upstream account selection and before later client turns are written upstream. No real provider was called during deployment or smoke.
- PR #219 was merged to main, deployed to staging, and then deployed to production through main `57eccdd3` after CI and staging smoke passed. It adds a conservative pre-provider safety budget for token requests that omit a positive output cap on generic gateway and OpenAI-compatible gateway paths. No real provider was called during deployment or smoke.
- PR #228 was merged to main and deployed to staging only through main `87504f2096b0`. It enables the chat workspace frontend backend gate in production-style builds and confirmed on staging that ordinary-user `/app/chat` loads the real workspace instead of the frontend "backend not enabled" blocker. Production was not deployed and no real provider was called.

This does not mean production image generation or token-request spend-cap coverage is fully accepted. Controlled real-provider production image-generation acceptance remains separate and must verify creation, storage, usage/billing, history, and download before claiming the production image chain is complete. Token-request coverage now includes bounded checks plus a no-cap safety-budget gate; full reservation, pre-charge, or streaming spend-cap design remains separate hardening.

## Progress Reporting Baseline

Use this progress meter in every major status report. It is a product/operations estimate, not a code-line metric.

| Stage | Current estimate | Meaning |
| --- | ---: | --- |
| P0 / P0-Beta convergence | 99% | Core shell, chat, failure/no-charge, fake-model, catalog authenticity, frontend lint baseline, usage DTO boundary, no-overdraft final billing, bounded image cost gating, bounded OpenAI-compatible token cost gating, bounded generic/Gemini token cost gating, bounded OpenAI Responses WebSocket token cost gating, no-cap token-request safety-budget gating, and staging chat-workspace frontend gate alignment are contained enough to continue P1. Remaining P0/P0-Beta risks are controlled production image-generation acceptance, full reservation/pre-charge spend-cap hardening, production rollout of staging-only chat gate fixes when batched, and any regressions found during P1. |
| P1 product/operations | 42% | P1 has started. Completed or staged slices include image-model alias display clarity (#188), user-shell lint baseline cleanup (#190), first image history/download feedback hardening (#192), staging image catalog exposure verification (#194), API Key third-party access copy/safety polish (#195), usage explanation-state clarity (#197), profile TOTP failure clarity (#198), masked API-key configuration guards (#199), `/app/usage` DTO/balance-refresh boundary verification (#201), Google/Gemini-compatible API-key auth restriction parity (#203), production-deployed usage-billing no-overdraft safety (#208), production-deployed generic/Gemini bounded token request cost gating (#214), production-deployed bounded OpenAI Responses WebSocket token cost gating (#217), and production-deployed no-cap token-request safety-budget gating (#219). Major remaining P1 work is controlled production real-generation acceptance, broader usage/balance workflow verification, full API Key lifecycle/security verification, and admin/ops hardening. |
| Distance to P2 | 58% of P1 remains | P2 should not begin as a main focus until the P1 user/business loops above have credible staging or production evidence. |
| P2 visual polish/enhanced experience | 0% | P2 is intentionally not active. UI polish and advanced workflows wait until P1 is materially closed. |

## Decision Rules

Before implementing new product, quality, UX, or operations work, apply the mature solution comparison principle:

1. Check whether Sub2API / SSXZ already has the required capability.
2. Check whether the older SSXZ site has reusable pages, interactions, flows, or copy.
3. Check mature references such as Open WebUI, LibreChat, LobeChat, Dify, New API, All API Hub, shadcn, Magic UI, 21st.dev, relevant SDKs, and relevant Codex skills.
4. Borrow only what fits. Do not replace the SSXZ AI Workbench trunk, migrate wholesale to an external project, mix unrelated UI systems, or expand scope because a reference product has a feature.
5. Classify reuse separately from product priority:
   - R0: directly reuse existing code or a mature component.
   - R1: borrow a module-level interaction, component, or logic pattern.
   - R2: reference only product structure or UX ideas.
   - R3: build custom only after confirming no suitable option fits.
6. External options must pass checks for license, maintenance activity, dependency weight, product fit for 200-300 private users, SSXZ UI consistency, billing/permissions/provider-routing/data-security impact, and compatibility with conversation / message / asset / usage / ledger.

This rule helps delivery speed and quality. It does not change priority labels: P0, P0-Beta, P1, and P2 remain the project priority system.

## P0: Structure Convergence

### Goals

- Make `/app/*` the ordinary-user workspace family.
- Keep ordinary-user pages inside the user workspace shell.
- Stop ordinary-user navigation from jumping into the older admin/backend-looking shell.
- Preserve existing backend, billing, provider routing, payment, and database behavior.
- Keep the 2026-06-20 staging image-generation success, the 2026-06-21 production smoke, and the 2026-06-27 PR #184 production authenticity-guard smoke as evidence, not as full product acceptance.

### P0 Work Items

1. Confirm user workspace route ownership:
   - `/app/chat` is the canonical new-chat and brand-home route
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

### P0 Exit Criteria

P0 can exit only when these are true:

1. Ordinary-user canonical routes are documented and behave consistently:
   - `/app/chat`
   - `/app/image`
   - `/app/usage`
   - `/app/keys`
   - `/app/profile`
2. The user workspace shell never sends ordinary users into the admin/backend shell for those routes.
3. The brand/logo home action returns to `/app/chat`.
4. Existing chat and image paths have smoke evidence after refresh.
5. Image failure paths remain no-charge, and image success paths record usage/billing/history consistently.
6. `/app/usage` reflects real usage/billing records or honest empty states.
7. Production storage/log-path risks are resolved or explicitly accepted before full production image acceptance.

### P0 Non-Goals

- No further production deployment or production config change without explicit approval.
- No database migration.
- No payment or ledger redesign.
- No provider routing redesign.
- No broad Sora-to-image internal rename.
- No large visual redesign.
- No new smart prompt feature.
- No new external provider integration.

## P1: Product Experience And Operations

### Goals

- Start from small, isolated PRs now that P0/P0-Beta route, status, no-charge, and fake-model risks are sufficiently contained.
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
   - first small slice completed by PR #192: recent-history load failures are visible and download href/filename behavior has frontend regression coverage
   - regenerate flow
   - clear loading/error/empty states
   - model selector clarity for OpenAI-compatible image aliases, including whether Gemini-named allowlisted models should be visible as-is, relabeled, or hidden until the user-facing strategy is explicit. PR #188 deployed the first small slice by labeling Gemini-named OpenAI-compatible aliases as compatible image-route aliases.
   - R2 reference: Picell AI / PicsetAI for e-commerce product visual framing, including platform/channel scenarios, product-image purposes, multi-result generation, and prompt organization around product, platform, use case, and style
   - R1 reference: `CookSleep/gpt_image_playground` for upload, drag-and-drop, clipboard paste, reference image, mask editing, streaming preview, gallery, large preview, download, and parameter comparison interactions
   - Do not adopt either reference as a replacement for SSXZ. Keep SSXZ auth, balance, billing, usage, assets, admin operations, and provider routing as the source of truth.
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
   - first small slice completed by PR #195: Base URL copy action, touched i18n copy, one-time full-key explanation, and model availability guidance
   - second safety slice completed by PR #199: masked list values no longer generate ready-to-use CLI/client configuration or CC Switch import flows
   - first backend auth-parity slice completed by PR #203: Google/Gemini-compatible API-key auth now enforces IP restrictions, expired keys, and quota-exhausted keys like the standard API-key middleware
5. Improve user billing pages:
   - balance
   - usage
   - first data-boundary slice completed by PR #201: ordinary-user usage DTO scrubbing, `/app/usage` balance refresh, ordinary/admin usage authorization split, and staging route/API smoke
   - first backend no-overdraft slice completed by PR #208: final balance billing now rejects insufficient-balance usage attempts instead of making user balances negative, and rolls back usage-billing dedup/quota/rate side effects on insufficient balance
   - token-request billing-safety slices deployed through PR #219: image requests, OpenAI-compatible bounded chat requests, generic Claude/Anthropic-compatible gateway requests, Gemini v1beta requests, bounded OpenAI Responses WebSocket requests, and no-cap token requests perform estimated-cost or conservative safety-budget eligibility checks before provider dispatch where the backend can estimate a positive cost
   - remaining spend-cap gap: full reservation, pre-charge, or streaming ledger-hold design remains later hardening
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
   - server-side human-verification validation if captcha, turnstile, sliding challenge, or OAuth login is used
   - external dependency checks against current official provider documentation before choosing a verification provider
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

Production deployment is a separate release gate. Production deployments happened on 2026-06-21, 2026-06-27, 2026-06-28, and 2026-06-29. As of the user's 2026-06-29 instruction, the agent may decide production deployment timing for larger completed stages after CI, merge, staging deployment, staging smoke, and rollback readiness are confirmed; small or low-impact PRs should still be batched instead of deployed one by one. Full production acceptance should use the gates below:

1. Ordinary-user navigation stays inside the intended user workspace shell.
2. `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/app/profile` behave consistently after refresh.
3. The logo/brand home action returns to `/app/chat`.
4. API Key / third-party access is visible and safe for ordinary users.
5. Image generation keeps the validated staging path healthy:
   - generation succeeds
   - cost/usage is recorded correctly
   - image history is visible
   - download works in a normal browser
   - failure does not mischarge users
6. Payment/order flows are either verified or clearly disabled with honest user-facing states.
7. Admin pages remain admin-only and usable for operations.
8. No provider keys, API keys, cookies, tokens, or Authorization values are printed or persisted in unsafe places.
9. Frontend typecheck/build and relevant tests pass.
10. Production deployment is either explicitly approved by the user or selected by the agent under the 2026-06-29 deployment-autonomy instruction after the previous gates pass.
