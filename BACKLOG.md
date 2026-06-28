# BACKLOG

Last updated: 2026-06-28

## Validated On Staging

- 2026-06-20: `/api/v1/image-studio/generate` completed one real staging image-generation request with `gpt-image-2`.
- 2026-06-20: staging image-generation billing/usage recorded one costed usage row and reduced the test account balance by `0.008`.
- 2026-06-20: staging image history stored one completed local image record.
- 2026-06-20: staging image media URL returned HTTP 200 with `image/png`.
- 2026-06-20: staging `/app/image` single reference image preview recovered after PR #142 allowed `blob:` in the default CSP `img-src` policy.
- 2026-06-20: direct route checks showed `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, `/app/chat`, `/app/purchase`, and `/app/orders` rendering in the user workspace shell rather than the admin/backend shell.
- 2026-06-20: invalid `/api/v1/image-studio/generate` multipart input returned HTTP 400, did not change balance, and did not create a new usage record.
- 2026-06-20: code-path audit confirmed image upstream errors return before `RecordUsage`, and image history only persists for captured non-truncated 2xx responses.
- 2026-06-20: service-level image gateway regression tests cover upstream 4xx, upstream 5xx failover, transport timeout/error, and partial-success response write failure returning no successful result.
- 2026-06-20: handler-level image gateway regression tests cover upstream failure without usage, billing, or balance-deduction calls, and failed/truncated image captures without image history creation.
- 2026-06-20: DB-backed handler regression test covers failed/truncated image gateway captures not creating persisted `sora_generations` rows, with a successful-capture control case.
- 2026-06-20: DB-backed handler regression test covers image upstream 4xx and transport timeout not creating persisted usage rows, not creating usage-billing dedup rows, and not changing user balance.
- 2026-06-21: latest main at commit `e40205e09` was deployed to production after explicit user approval. Basic production smoke returned HTTP 200 for `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public`; `/v1/models` returned expected HTTP 401 without an API key.
- 2026-06-21: production deployment did not include Nginx changes or database migrations.
- 2026-06-21: workbench logo/brand home action was corrected to `/app/chat` with `AppSectionShell` test coverage.
- 2026-06-27: PR #184 was merged to main at `d5be5a624` and deployed to production after explicit user approval.
- 2026-06-27: production public smoke returned HTTP 200 for `https://api.ssxzapi.com/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public`.
- 2026-06-27: production `sub2api.service` was active/running with binary SHA-256 `cc2a32fc401dd45d606d22404f91526eb5190067069fef4e61bac9bd976105fa`.
- 2026-06-27: production read-only auth smoke confirmed ordinary users receive HTTP 403 for `/admin/users`, while admin users receive HTTP 200.
- 2026-06-27: production ordinary account had no selectable image models and non-real image-capable model count was `0`, confirming the PR #184 real-channel filter guard. No real provider was called during this validation.
- 2026-06-27: PR #186 was merged to main at `0089a688a` and deployed to staging only. Staging `sub2api-staging.service` ran binary SHA-256 `79c0dc575b6337551d903c2e823c6931af755ea0bc4f7ef7eb024388bcec5e76`; production remained on SHA-256 `cc2a32fc401dd45d606d22404f91526eb5190067069fef4e61bac9bd976105fa`.
- 2026-06-27: after #186 staging deployment, route smoke returned HTTP 200 for `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public` on both staging port `18080` and production port `8080`.
- 2026-06-27: staging `/api/v1/channels/available` exposed `gpt-image-2` plus Gemini-named image aliases as real image models because they are explicitly included in `WORKSPACE_IMAGE_REAL_PROVIDER_ALLOWED_MODELS` under `provider=openai-compatible-images` and provider label `workspace-openai-compatible-image-staging`. This is a staging allowlist/model-label strategy issue, not evidence of native Gemini provider routing in `/app/image`.
- 2026-06-27: PR #188 was merged to main at `1c912f210`, deployed to staging, then deployed to production after explicit user approval.
- 2026-06-27: #188 production and staging binaries both ran SHA-256 `7fb45509c5fb6d74a5cc8ab88530f78f4e34dd5a88b177fe31c178b3f034afa0`; production backup was `/opt/sub2api/backups/production-before-pr188-1c912f210-20260627-210008/sub2api`.
- 2026-06-27: #188 public production smoke returned HTTP 200 for `https://api.ssxzapi.com/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public`.
- 2026-06-27: #188 staging UI displayed Gemini-named image aliases as `OpenAI 兼容图片通道别名`; production ordinary account still had no selectable image models because production catalog did not expose `real_channel` image capabilities. No real provider was called.

- 2026-06-27: PR #190 was merged to main at `5402cf556` and deployed to staging only. Staging `sub2api-staging.service` ran binary SHA-256 `294cd98b2e5f1e8699733e26ffda5e50efa5856a5c183144512a8163cb0b92ec`; production remained on SHA-256 `7fb45509c5fb6d74a5cc8ab88530f78f4e34dd5a88b177fe31c178b3f034afa0`.
- 2026-06-27: after #190 staging deployment, route smoke returned HTTP 200 for `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` on staging port `18080`; the same public routes returned HTTP 200 on production port `8080`. No real provider was called.
- 2026-06-27: PR #192 was merged to main at `4adc65dba` and deployed to staging only. Staging `sub2api-staging.service` ran binary SHA-256 `7086880a79e22900e4d65ca4f1cc715f6968c77fa22ec92a5c85bced8dfbcd5e`; production remained on SHA-256 `7fb45509c5fb6d74a5cc8ab88530f78f4e34dd5a88b177fe31c178b3f034afa0`.
- 2026-06-27: after #192 staging deployment, route smoke returned HTTP 200 for `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` on staging port `18080`; the same public routes returned HTTP 200 on production port `8080`. #192 was frontend-only image history/download feedback hardening and did not call a real provider.
- 2026-06-27: PR #194 was merged to main at `c208d51a7` and deployed to staging only. Staging binary SHA-256 was `9018e284ee6a80d8fd717ddd49b1457e65fd278025b76377b5fa5b9855d66869`; production was not deployed for #194.
- 2026-06-27: after #194 staging deployment, public route smoke returned HTTP 200 for `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public`. Staging ordinary-user `/api/v1/channels/available` exposed four `image_generation` models including `gpt-image-2`, all from `real_channel`, and non-real image-capable model count was `0`. No real provider was called.
- 2026-06-28: PR #195 was merged to main at `6068b062f` with frontend-only `/app/keys` copy and safety polish. It added Base URL copy guidance, clarified one-time full-key reveal and masked list behavior, moved touched copy into i18n, and added regression tests. #195 was not deployed to staging or production at the time of this backlog update.

## P0 Bugs And Structural Fixes

- User-side menu entries must not jump into the old admin/backend-looking shell.
- `/app/usage`, `/app/keys`, and `/app/profile` should remain in the new user workspace shell after navigation and refresh.
- Purchase, orders, and subscriptions need a decision: either user-workspace shell pages or clearly documented legacy compatibility routes.
- Stale ordinary-user redirects to `/sora` should be replaced with the agreed user entry after route ownership is decided.
- `/sora` should remain a compatibility route, not the main product route.
- Technical pages such as Available Channels and Channel Status should not dominate ordinary-user navigation.
- Image generation has one staging success path and one production smoke deployment, but production real generation/storage/billing/history/download still need acceptance verification.
- Production logs showed `/app` read-only log output and old `SoraStorage` local-path initialization risk. Resolve or explicitly accept storage/log-path configuration before full production image acceptance.
- Image generation upstream failure behavior has DB-backed usage/billing regression coverage for simulated upstream 4xx and transport timeout. Keep this coverage when changing image gateway billing or error handling.
- Placeholder workspace controls must stay clearly disabled until their backend/product behavior exists.

## P1 Entry Queue

- Start P1 in small PRs only. Do not reopen broad P0 structure work unless a regression appears.
- First P1 candidates:
  - continue image history/download/user journey audit after the #192 first UX-hardening slice
  - `/app/image` controlled production real-generation acceptance plan, only after explicit approval to call a real provider
  - follow-up `/app/image` model naming/display strategy only if operators decide to change production image-model policy; the first alias-display slice was completed by PR #188
  - API Key / third-party access behavior/security verification after the #195 first copy/safety polish slice
  - usage/balance explanation improvements based on real data and honest empty states
- Keep production deployment as a separate approval gate for every PR.

## Phase Progress Snapshot

- P0 / P0-Beta convergence: about 94%. Remaining P0 risk is controlled production image-generation acceptance and any regression found while doing P1.
- P1 product/operations: about 24%. Completed slices: image-model alias display clarity, user-shell lint baseline cleanup, first image history/download feedback hardening, staging catalog exposure verification, and first API Key third-party access copy/safety polish. Remaining large P1 loops: production real-generation acceptance, usage/balance explanation, API Key behavior/security verification, and admin/ops hardening.
- Distance to P2: about 76% of P1 remains. Do not prioritize P2 visual polish until P1 loops have evidence.
- P2: 0%. Keep as later polish/enhancement work.

## Chains That Need Verification

- Login/register to user workspace entry.
- `/app/chat` text chat.
- `/app/image` image generation:
  - permission/group: staging success path verified once
  - model/account: staging success path verified once with `gpt-image-2` and account id `14`
  - request payload: staging success path verified once
  - billing/usage: staging success path verified once
  - history: staging success path verified once
  - download: HTTP media URL verified once; PR #192 added frontend href/filename regression coverage; native browser download event still needs manual confirmation
  - invalid form failure: staging no-charge behavior verified once
  - upstream failure: code path audited; service-level and handler-level regression tests added; DB-backed image-history regression added; DB-backed usage/billing regression added
- `/app/usage` real balance, monthly trend, and detail data.
- `/app/keys` create/copy/delete/reset behavior and key masking.
- `/app/profile` profile/password/TOTP behavior.
- Production acceptance for image generation:
  - storage/log-path configuration
  - one controlled real generation when upstream and price settings are ready
  - no-charge behavior for failures
  - usage/billing consistency
  - history and download
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
  - reference image workflow beyond the verified single-image preview
  - results area
  - regenerate
  - image history
  - download
  - loading/error/empty states
  - R2 product reference: Picell AI / PicsetAI for e-commerce product visual generation, including marketplace/channel context such as Taobao, JD, Amazon, TEMU, Xiaohongshu, social media, and ad placements
  - R2 product reference: product visual purposes should include main product image, detail page image, social image, poster, and ad creative; the target is commercial product visuals, not toy image generation
  - R2 product reference: consider a flow where AI first understands the product, target platform, use case, and style before forming the generation prompt
  - R1 interaction reference: `CookSleep/gpt_image_playground` for multi-image upload, drag-and-drop, clipboard paste, reference/mask editing, streaming generation preview, history gallery, large preview, download, and parameter comparison
  - Constraint: do not directly adopt local IndexedDB-only history, standalone provider configuration, or single-user playground assumptions; SSXZ must keep asset / usage / ledger / auth / admin ownership as the system of record
- Chat page:
  - prompt assistance positioning
  - examples for common private-user tasks
  - better transition between chat and image creation
- API Key / third-party access:
  - Base URL guidance
  - Cherry Studio / Chatbox / CC Switch usage guidance
  - safe key visibility states
  - first copy/safety polish slice merged in #195; still verify create/copy/delete/reset behavior and API key log/security handling separately
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
- Make image model selection and provider strategy configurable instead of assuming one hard-coded model forever.
- Picell AI / PicsetAI and `CookSleep/gpt_image_playground` are backlog references only. They are R1/R2 inputs for future `/app/image` work and must not interrupt P0/P0-Beta gates for real generation, preview, failure feedback, no-charge safety, history/download, model authenticity, or request state closure.

## Security Improvements

- Login/register/invite-code rate limiting review.
- Login/register/password-reset/invite-code human-verification review:
  - verify current auth endpoints before adding a provider
  - check current official docs and pricing for any captcha, turnstile, sliding challenge, or OAuth provider on the decision date
  - treat captcha/sliding verification as an auxiliary friction layer, not the main defense
  - require server-side token verification; never trust a frontend-only `captcha=true` state
  - enforce one-time tokens, expiry, replay protection, and safe failure behavior
  - pair verification with IP, account, invite-code, and IP-plus-account rate limits
  - use unified error wording to avoid account enumeration
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
- Further production deployment or production config changes without explicit user approval.
