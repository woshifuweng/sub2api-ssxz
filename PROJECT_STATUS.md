# PROJECT_STATUS

Last updated: 2026-06-29

## Product Positioning

SSXZ AI is built on the existing Sub2API codebase. The product target is a lightweight AI creation workspace for about 200-300 private users, not a full enterprise AI gateway console.

The user-side product should focus on:

- AI chat
- AI image generation
- image history and download
- balance, usage, recharge, orders, and subscriptions
- API Key / third-party client access for CC Switch, Cherry Studio, Chatbox, and similar clients

The admin side should preserve operation capabilities for the owner and operators:

- users
- balances
- orders and payments
- usage
- image tasks and image history
- API keys
- invite/redeem/promo codes
- channel, account, model, and pricing configuration
- basic logs and abnormal task handling

## Codebase Facts Observed On 2026-06-18

| Area | Location | Notes |
| --- | --- | --- |
| Frontend | `frontend/src` | Vue 3, Vite, Pinia, vue-router. |
| Backend | `backend/internal`, `backend/cmd/server` | Go, Gin, Ent. |
| Database model | `backend/ent/schema` | Ent schemas for users, API keys, accounts, usage, billing/payment-adjacent records, groups, and related resources. |
| Migrations | `backend/migrations` | SQL migrations include usage/billing consistency, group pricing, image/media fields, Sora/image tables, API key rate limits, and routing-related changes. |
| Frontend routes | `frontend/src/router/index.ts` | User workspace, legacy user pages, payment pages, and admin routes are registered in one route file. |
| Backend routes | `backend/internal/server/routes` | User, admin, payment, gateway, Sora/client, and workspace route registration. |

## User-Side Page State

| Page / route family | 2026-06-18 status | Notes |
| --- | --- | --- |
| `/app`, `/app/chat` | Connected, canonical chat entry | Unified workspace/chat path exists. `/app/chat` is the ordinary-user new chat entry and brand-home destination. The UX is not final. |
| `/app/image` | Staging generation path verified, UX partial | Routes to `ImageStudioView`. On 2026-06-20 staging generated one image through `gpt-image-2`, recorded usage, saved history, and served the PNG download. Product UX is still not final. |
| `/app/usage` | Partial product path, P1 clarity and billing-safety slices deployed | New workbench-style usage page exists. PR #197 clarified failure/no-charge explanation states and was included in the 2026-06-28 staging-only batch. PR #201 was deployed to staging only and confirmed ordinary-user usage DTO scrubbing, balance refresh on `/app/usage`, and the ordinary/admin usage boundary. PR #208 was deployed to staging and then production after explicit approval, and prevents final balance billing from deducting ordinary usage below zero. It should remain in the user workspace shell. |
| `/app/keys` | P1 polish and backend auth parity staged | Intended to keep API Key / third-party access in the user workspace shell. PR #195 moved touched third-party access copy into i18n, added Base URL copy affordances, and clarified one-time full-key visibility. PR #199 prevents masked list values from producing ready-to-use CLI/client configuration or CC Switch import flows. Both are included in the 2026-06-28 staging-only batch. PR #203 was deployed to staging only and aligned Google/Gemini-compatible API-key auth restriction handling with the standard API-key middleware. It must not jump to the admin/backend shell. |
| `/app/profile` | Partial product path, P1 TOTP clarity slice staged | Intended to keep account settings in the user workspace shell. PR #198 clarified TOTP status failure messages and was included in the 2026-06-28 staging-only batch. It must not jump to the admin/backend shell. |
| `/keys` | Legacy user path | Functional API Key page using the older shell. Keep as compatibility, not as the main user entry. |
| `/usage` | Legacy user path | Functional usage page using the older shell. Keep as compatibility, not as the main user entry. |
| `/profile` | Legacy user path | Functional profile page using the older shell. Keep as compatibility, not as the main user entry. |
| `/purchase`, `/orders`, `/payment/*` | Existing payment/order paths | Payment and order flows exist, but the user-facing shell is not fully aligned with the new workspace. |
| `/sora` | Legacy image route | Keep for compatibility. Product copy should not present OpenAI Sora as the long-term image-generation strategy. |
| `/available-channels`, `/channel-status` | Technical user pages | Useful for advanced users/admin troubleshooting, but not part of the lightweight user-side main navigation. |

## Admin Page State

Admin pages live under `frontend/src/views/admin` and are routed under `/admin/*`.

| Admin area | 2026-06-18 status | Notes |
| --- | --- | --- |
| Dashboard / ops | Existing | Stronger than the user-side product shell. Keep admin-only. |
| Users / groups | Existing | Needed for private operation and support. |
| Channel / account / pricing | Existing | Needed for provider/account operations. Do not expose as a main ordinary-user flow. |
| Orders / payment admin | Existing | Payment and order administration exist. Keep separate from user workspace. |
| Usage admin | Existing | Needed for owner/operator auditing. |
| Announcements / IP / redeem / promo / settings | Existing | Useful operational functions. Keep admin-only unless explicitly productized. |

## Module Completion Classification

| Module | Classification | Notes |
| --- | --- | --- |
| Auth/login/register | Mostly complete | Login/register flows and backend handlers exist. Security hardening remains a separate backlog item. |
| User workspace shell | Partial | `/app/*` exists, but not every user entry stays inside this shell. |
| AI chat | Connected | `/app/chat` works through workspace logic; old `ChatStudioView` remains a product reference/asset. |
| Image generation | P0/P0-Beta guard deployed; production real generation acceptance still open | `ImageStudioView`, `ImageStudioHandler`, OpenAI-compatible image gateway, and image/Sora-related storage exist. On 2026-06-20 staging verified generation, usage cost, history, and HTTP image download with `gpt-image-2`. On 2026-06-27 PR #184 was deployed to production after explicit approval and verified that ordinary users do not see non-real image-capable models. PR #188 was also deployed to production after explicit approval and clarified OpenAI-compatible image alias labels without enabling production image generation. Production real image generation remains an acceptance gate because these production validations did not call a real provider. |
| API Key / third-party access | Mostly complete, P1 polish and auth parity staged | Backend and frontend exist. PR #195 improved user-facing Base URL/model guidance, key masking explanation, and one-time full-key display copy. PR #199 added a frontend guard so masked API-key list values are not presented as usable config material. PR #203 aligned Google/Gemini-compatible API-key middleware with ordinary API-key restriction handling for IP restrictions, expired keys, and quota-exhausted keys, and was deployed to staging only. Create/copy/delete/enable/disable behavior and broader API-key lifecycle/security review remain separate verification items. |
| Usage center | Backend complete enough, DTO and no-overdraft boundaries verified | Usage APIs and older page exist; `/app/usage` is the desired new user-shell direction. PR #197 clarified user-facing failure/no-charge states. PR #201 reduced the ordinary-user usage DTO, refreshed user balance on `/app/usage`, and passed staging ordinary/admin boundary checks. PR #208 prevents final balance billing from making the user balance negative and keeps insufficient-balance usage-billing attempts from leaving dedup/quota/rate side effects. PR #214 adds estimated-cost pre-provider checks for bounded generic token requests on Claude/Anthropic-compatible gateway paths and Gemini v1beta, and was deployed to production after explicit approval on 2026-06-29. Broader recharge/order/account-balance workflow verification and WebSocket/unbounded token-request spend-cap design remain open. |
| Recharge/payment | Backend/admin rich, user-shell partial | Payment/order/subscription capabilities exist; user shell alignment remains incomplete. |
| Orders | Existing, user-shell partial | Order pages exist; not yet fully aligned to the new user workspace. |
| Provider routing/channel management | Existing, admin-oriented | Keep for admin/owner operations. Do not let it dominate ordinary-user navigation. |
| Web search | Existing but frozen for main UX | Technical chain exists from prior PRs. Do not surface as ordinary-user main functionality during P0 structure work. |
| Admin operations | Rich but broad | Strong asset from the Sub2API base. Needs product boundary, not deletion. |

## Staging P0 Validation Recorded On 2026-06-20

| Area | Result | Evidence |
| --- | --- | --- |
| Image generation request | Verified on staging only | `/api/v1/image-studio/generate` returned HTTP 200 using `gpt-image-2`. |
| Upstream account | Verified on staging only | Used staging account `OpenAI Image Staging`, account id `14`. No provider key was printed. |
| Billing / usage | Verified on staging only | Balance changed from `49.39622358` to `49.38822358`. Usage record id `1551` stored model `gpt-image-2` and cost `0.008`. |
| Image history | Verified on staging only | New generation id `2`, status `completed`, model `gpt-image-2`, media type `image`, storage type `local`, one media URL. |
| Image download | HTTP path verified | Media URL returned HTTP 200, `image/png`, about `1.2 MB`. The Codex in-app browser could not verify the native download shelf event. |
| Reference image preview | Verified on staging | PR #142 updated the default CSP to allow `blob:` images. On 2026-06-20 staging served `img-src 'self' data: blob: https:` and the user confirmed the `/app/image` reference preview recovered. |
| Image invalid request handling | Verified on staging only | Invalid multipart request returned HTTP 400 with `invalid form data`; balance stayed `49.38822358`; latest usage id stayed `1551`. |
| Image upstream failure guard | Code-path audited and handler/service-regression tested | `/api/v1/image-studio/generate` delegates to the OpenAI Images gateway. The gateway submits `RecordUsage` only after `ForwardImagesContext` returns a successful result, and image history persists only for captured non-truncated 2xx responses. Service tests cover upstream 4xx, upstream 5xx failover, transport timeout/error, and partial-success response write failure returning no successful image result. Handler tests cover upstream failure not recording usage/billing/deduct calls, non-2xx/truncated captures not creating image history, DB-backed `sora_generations` persistence staying clean for failed/truncated captures, and DB-backed usage/billing tables staying clean for upstream 4xx and transport timeout while user balance remains unchanged. |
| User workspace shell | Direct routes verified | `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, `/app/chat`, `/app/purchase`, and `/app/orders` rendered without the admin/backend shell during direct route checks. |
| Production | Not touched during 2026-06-20 staging validation | No production deployment or Nginx change was part of the 2026-06-20 staging validation. A later production release is recorded below. |

## Production Release Recorded On 2026-06-21

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed after explicit user approval | Latest main at commit `e40205e09` was deployed to `sub2api.service`. |
| Binary | Matched staging candidate | Production binary SHA-256 matched staging candidate `352d97c0e4ea5d96d282b61332cad9a6748d79cbccbcee2f12032f7bc39da1bc`. |
| Backup | Created before replacement | `/opt/sub2api/backups/production-before-main-20260621-135335/sub2api`. |
| Production smoke | Basic routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/api/v1/settings/public` returned HTTP 200. `/v1/models` returned HTTP 401 without an API key, which is expected. |
| Scope | No infrastructure change | No Nginx change and no database migration were part of the release. |
| Remaining gate | Production image acceptance not complete | Production logs showed `/app` read-only log output and old `SoraStorage` local-path initialization risk. Resolve or explicitly accept storage/log-path configuration before treating production image generation as fully accepted. |

## Production Release Recorded On 2026-06-27

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed after explicit user approval | PR #184 was merged to main at merge commit `d5be5a624`. Production was deployed from the same candidate already validated on staging. |
| Binary | Running production binary recorded | Production `/opt/sub2api/sub2api` SHA-256 was `cc2a32fc401dd45d606d22404f91526eb5190067069fef4e61bac9bd976105fa`; `sub2api.service` was active/running with PID `130747` after deployment. |
| Backup | Created before replacement | `/opt/sub2api/backups/production-before-pr184-d5be5a624-20260627-161358/sub2api`. |
| Production smoke | Public routes responded | `https://api.ssxzapi.com/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public` returned HTTP 200 after deployment. |
| Ordinary/admin boundary | Read-only auth smoke passed | Ordinary and admin users could load their expected read-only workspace endpoints. Ordinary `/admin/users` returned HTTP 403 and admin `/admin/users` returned HTTP 200. |
| Image model authenticity | Non-real image-capable models hidden | Production ordinary account had no selectable image models, and the non-real image-capable model count was `0`. This confirms the PR #184 guard, not production image generation success. |
| Scope | No sensitive backend scope changed | No database migration, Nginx change, payment/ledger change, provider-routing change, or real-provider call was part of the release. |
| Remaining gate | Production image creation still not fully accepted | #184 did not call a real provider. A later controlled production image-generation acceptance still needs explicit approval before claiming production image creation, storage, billing, history, and download are fully accepted. |

## Staging Release Recorded On 2026-06-27

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #186 was merged to main at merge commit `0089a688a` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 was `79c0dc575b6337551d903c2e823c6931af755ea0bc4f7ef7eb024388bcec5e76`; production `/opt/sub2api/sub2api` remained `cc2a32fc401dd45d606d22404f91526eb5190067069fef4e61bac9bd976105fa`. |
| Production | Not deployed | Production service remained active on the prior PR #184 binary. No production file replacement or service restart was part of #186 validation. |
| Public smoke | Staging and production routes responded | Local server smoke returned HTTP 200 for `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public` on both staging port `18080` and production port `8080`. |
| Image model strategy finding | Staging image models are server allowlisted OpenAI-compatible image models | Staging `WORKSPACE_IMAGE_REAL_PROVIDER_ALLOWED_MODELS` explicitly allowed `gpt-image-1`, `gpt-image-2`, `gemini-2.5-flash-image`, `gemini-3.1-flash-image-preview`, and `gemini-3-pro-image-preview` under provider label `workspace-openai-compatible-image-staging`. The Gemini-named entries are exposed as `provider=openai-compatible-images` and `platform=openai`, not as a native Gemini provider leak. |
| Scope | No real provider call | #186 validation used login, channel catalog, route smoke, and service/hash checks only. It did not call image generation or any upstream provider. |
| Remaining gate | Production image acceptance still open | PR #188 later clarified the user-facing alias label strategy. Production image creation, storage, usage/billing, history, and download remain unaccepted until explicitly tested. |

## Production Release Recorded On 2026-06-27 For PR #188

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed after explicit user approval | PR #188 was merged to main at merge commit `1c912f210` and deployed to `sub2api.service`. |
| Binary | Production and staging aligned | Production `/opt/sub2api/sub2api` and staging `/opt/sub2api/sub2api-staging` both ran SHA-256 `7fb45509c5fb6d74a5cc8ab88530f78f4e34dd5a88b177fe31c178b3f034afa0` after deployment. |
| Backup | Created before replacement | `/opt/sub2api/backups/production-before-pr188-1c912f210-20260627-210008/sub2api`. |
| Production smoke | Public routes responded | `https://api.ssxzapi.com/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/api/v1/settings/public` returned HTTP 200 after deployment. |
| Image model display | Alias clarity deployed without enabling production image generation | Staging displayed Gemini-named OpenAI-compatible image aliases with `OpenAI 兼容图片通道别名`. Production ordinary account still had no selectable image models because production did not expose `real_channel` image capabilities. |
| Scope | Frontend display only | No Nginx change, database migration, payment/ledger change, provider-routing change, or real-provider call was part of the release. |

## Staging Release Recorded On 2026-06-27 For PR #190

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #190 was merged to main at merge commit `5402cf556` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 was `294cd98b2e5f1e8699733e26ffda5e50efa5856a5c183144512a8163cb0b92ec`; production `/opt/sub2api/sub2api` remained `7fb45509c5fb6d74a5cc8ab88530f78f4e34dd5a88b177fe31c178b3f034afa0`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-pr190-5402cf556-20260627-214306-sub2api-staging`. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging port `18080`. |
| Production | Not deployed | Production service remained active on the prior PR #188 binary. The same public routes returned HTTP 200 on production port `8080`, but no production file replacement or service restart was part of #190 validation. |
| Scope | Frontend lint/test cleanup only | #190 touched `AppSectionShell.vue` and its component test only. No Nginx change, database migration, payment/ledger change, provider-routing change, or real-provider call was part of the staging release. |

## Staging Release Recorded On 2026-06-27 For PR #192

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #192 was merged to main at merge commit `4adc65dba` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 was `7086880a79e22900e4d65ca4f1cc715f6968c77fa22ec92a5c85bced8dfbcd5e`; production `/opt/sub2api/sub2api` remained `7fb45509c5fb6d74a5cc8ab88530f78f4e34dd5a88b177fe31c178b3f034afa0`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-pr192-4adc65dba-20260627-220806-sub2api-staging`. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging port `18080`. |
| Production | Not deployed | Production service remained active on the prior PR #188 binary. The same public routes returned HTTP 200 on production port `8080`, but no production file replacement or service restart was part of #192 validation. |
| Scope | Frontend image-history/download feedback only | #192 touched `ImageStudioView.vue` and its component test only. It made recent image-history load failures visible and strengthened download href/filename assertions. No Nginx change, database migration, payment/ledger change, provider-routing change, or real-provider call was part of the staging release. |

## Staging Release Recorded On 2026-06-27 For PR #194

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #194 was merged to main at merge commit `c208d51a7` and deployed only to `sub2api-staging.service`. |
| Binary | Staging candidate recorded | Staging `/opt/sub2api/sub2api-staging` SHA-256 was `9018e284ee6a80d8fd717ddd49b1457e65fd278025b76377b5fa5b9855d66869`. |
| Production | Not deployed for #194 | Production remained on the previously deployed `832475454` candidate; no production file replacement or service restart was part of #194 validation. |
| Public smoke | Staging routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging. |
| Image model catalog | Staging catalog exposed real image models | Ordinary-user `/api/v1/channels/available` on staging exposed four `image_generation` models including `gpt-image-2`, all under `provider=openai-compatible-images` with `model_catalog_source=real_channel`; non-real image-capable model count was `0`. |
| Scope | Catalog/capability guard only | #194 touched channel catalog filtering and frontend capability tests. No database migration, Nginx change, payment/ledger change, provider-routing change, production deployment, or real-provider call was part of the release. |

## Main Merge Recorded On 2026-06-28 For PR #195

| Area | Result | Evidence |
| --- | --- | --- |
| Merge | Completed to main | PR #195 was merged at `6068b062f` with title `Polish API key third-party access copy (#195)`. |
| Scope | Frontend-only `/app/keys` polish | The PR touched `KeysView.vue`, related component tests, and zh/en i18n locale tests. |
| Product effect | API Key guidance clearer in code | The top `/app/keys` Base URL guide now has a copy action, touched copy is i18n-backed, and the one-time full-key reveal dialog clarifies masked list values and model availability source. |
| Deployment | Later staged | No staging or production deployment was done for #195 at the time of this status entry. The #195 frontend changes were later included in the 2026-06-28 staging-only batch at `261e3785b`. |
| Sensitive scope | Untouched | No backend, database, billing/ledger, payment, provider routing, Nginx, production deployment, or real-provider call was part of #195. |

## Staging Release Recorded On 2026-06-28 For Main `261e3785b`

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | Main `261e3785b` was deployed only to `sub2api-staging.service`. The batch included PR #195, PR #197, PR #198, and PR #199 frontend/user-facing clarity work. |
| Binary | Staging candidate recorded | Staging `/opt/sub2api/sub2api-staging` SHA-256 is `8c501eaaa88af44c25eaf3928943d4c6cc804c05124ba34b2404e3788410294d`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-main-261e3785b-20260628-060446-sub2api-staging`. |
| Production | Not deployed | Production `/opt/sub2api/sub2api` SHA-256 stayed `b5da079f49da94cfb57873de6d4fada5d170b87be310ab0633ecafc53f2d877b` before and after staging deployment. No production file replacement or service restart was part of this release. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging port `18080`. The same routes also returned HTTP 200 on production port `8080` for split-health comparison. |
| Static resource check | New API Key copy keys present | Staging served locale chunks containing the PR #199 API Key masked-value warning keys such as `fullKeyRequiredForImport` and `fullKeyMissingTitle`. |
| Runtime health | Staging active after restart | `sub2api-staging.service` was active after restart, and recent journal lines had no `panic`, `fatal`, `error`, or `failed` matches. |
| Scope | Frontend/user-facing clarity batch only | No database migration, Nginx change, payment/ledger change, provider-routing change, production deployment, or real-provider call was part of the staging release. |

## Staging Release Recorded On 2026-06-28 For PR #201

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #201 was merged to main at merge commit `0dbbaf03f` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 is `0cf888efbffbcb40bf5469eaeec3667034308fad58fb06e71542c46f9f3b1f17`; production `/opt/sub2api/sub2api` stayed `b5da079f49da94cfb57873de6d4fada5d170b87be310ab0633ecafc53f2d877b`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-main-0dbbaf03f165-20260628-145151-sub2api-staging`. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging port `18080`. The same routes also returned HTTP 200 on production port `8080` for split-health comparison only. |
| Usage DTO boundary | Ordinary/admin split verified | Ordinary-user `/api/v1/usage` returned 20 rows without `user_id`, `account_id`, `group_id`, `subscription_id`, `user_agent`, `upstream_endpoint`, `user`, `api_key`, `group`, or `subscription`. Ordinary-user `/api/v1/admin/usage` returned HTTP 403. Admin `/api/v1/admin/usage` returned HTTP 200 and retained operational fields such as `user_id`, `account_id`, `group_id`, `upstream_endpoint`, `user_agent`, and `account`. |
| Usage page data | Staging route and APIs healthy | Ordinary-user `/api/v1/auth/me`, `/api/v1/usage`, `/api/v1/usage/stats`, `/api/v1/usage/dashboard/trend`, and `/app/usage` returned HTTP 200 on staging. |
| Runtime health | Staging active after restart | `sub2api-staging.service` and `sub2api.service` were active. Recent staging journal checks showed no panic/fatal crash; the known staging log-path warning `mkdir /app: permission denied` remains an operations configuration issue, not a #201 code regression. |
| Scope | Usage DTO and frontend balance refresh only | No database migration, Nginx change, payment/ledger change, provider-routing change, production deployment, or real-provider call was part of the staging release. |

## Staging Release Recorded On 2026-06-28 For PR #203

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #203 was merged to main at merge commit `c068d6ad6` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 is `b5b7fb31939db5bf897303bd4d13ed7dbecb7e2d1c9fb5318efce7f4bde5df90`; production `/opt/sub2api/sub2api` stayed `b5da079f49da94cfb57873de6d4fada5d170b87be310ab0633ecafc53f2d877b`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-pr203-c068d6ad6-20260628-151908-sub2api-staging`. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging. |
| Google/Gemini-compatible API key auth | Restriction behavior staged | `/v1beta/models` without a key returned HTTP 401, deprecated `api_key` query usage returned HTTP 400, and invalid `key` query usage returned HTTP 401. The merged middleware change enforces IP restrictions and rejects expired or quota-exhausted API keys on the Google/Gemini-compatible API-key path, matching the ordinary API-key middleware. |
| Runtime health | Staging active after restart | `sub2api-staging.service` was active, `sub2api.service` remained active, and recent staging journal checks showed no panic/fatal/traceback matches. Existing staging configuration warnings for URL allowlist, log path, payment key defaults, and trusted proxies remain operations configuration warnings, not a #203 code regression. |
| Scope | Backend API-key auth parity only | No database migration, Nginx change, payment/ledger change, provider-routing change, production deployment, or real-provider call was part of the staging release. |

## Staging Release Recorded On 2026-06-28 For PR #208

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #208 was merged to main at merge commit `e67a48169` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 is `0e26ec8f4f75a90a4efa33887033447a920b76c052502dc3dda10af05f46647d`; production `/opt/sub2api/sub2api` stayed `b5da079f49da94cfb57873de6d4fada5d170b87be310ab0633ecafc53f2d877b`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-pr208-e67a48169-20260628-205154-sub2api-staging`. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging; `/v1/models` without an API key returned HTTP 401. |
| Balance billing safety | No-overdraft behavior staged | `usage_billing` now requires `users.balance >= balance_cost` before final balance deduction. If balance is insufficient, the usage-billing transaction returns insufficient balance and rolls back dedup, API-key quota, and rate-usage side effects. The generic user balance deduction path also rejects overdraft, while manual balance adjustments remain separate. |
| Runtime health | Staging active after restart | `sub2api-staging.service` was active, `sub2api.service` remained active, and recent staging journal checks showed no panic/fatal/traceback/error matches. |
| Scope | Backend repository billing safety only | No database migration, Nginx change, payment file change, provider-routing change, production deployment, or real-provider call was part of the staging release. |

## Production Release Recorded On 2026-06-28 For Main `b1ed2663a` / PR #208

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed after explicit user approval | Current main `b1ed2663a` was deployed to `sub2api.service`. Runtime code change is PR #208; PR #209 was docs-only. |
| Binary | Running production binary recorded | Production `/opt/sub2api/sub2api` SHA-256 is `1d9902d30fd3348127ce298576cfd3c89ad78a8d87ea8f3112938716b00e711a`; `sub2api.service` was active/running with PID `622474` after restart. |
| Backup | Created before replacement | `/opt/sub2api/backups/production-before-main-b1ed2663a-20260628-210956-sub2api`. |
| Production smoke | Public routes responded | `https://api.ssxzapi.com/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200; `/v1/models` without an API key returned HTTP 401. |
| Staging split-health | Staging remained active | Staging `/opt/sub2api/sub2api-staging` stayed on SHA-256 `0e26ec8f4f75a90a4efa33887033447a920b76c052502dc3dda10af05f46647d`, and staging route smoke remained healthy. |
| Scope | Backend billing safety release | No database migration, Nginx change, payment file change, provider-routing change, or real-provider call was part of the production release. |

## Staging Release Recorded On 2026-06-29 For PR #214

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed on staging only | PR #214 was merged to main at merge commit `f74c8f9be` and deployed only to `sub2api-staging.service`. |
| Binary split | Staging and production are different binaries | Staging `/opt/sub2api/sub2api-staging` SHA-256 is `f6a1da5da1f52b8260e7346dcf47c4da0c94fc47611be00ed686d40e9a4a1cfc`; production `/opt/sub2api/sub2api` stayed `1d9902d30fd3348127ce298576cfd3c89ad78a8d87ea8f3112938716b00e711a`. |
| Backup | Created before replacement | `/opt/sub2api/backups/staging-before-pr214-f74c8f9be-20260629-052733-sub2api-staging`. |
| Staging smoke | Public routes responded | `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200 on staging; `/v1/models` without an API key returned HTTP 401. |
| Token-request billing gate | Bounded token requests are checked before provider calls on more gateway paths | Generic Claude/Anthropic-compatible gateway requests and Gemini v1beta requests now estimate positive bounded token cost before provider dispatch and use cost-aware billing eligibility checks. Unbounded requests continue to use the existing zero-estimate eligibility path. |
| Runtime health | Staging active after restart | `sub2api-staging.service` was active, `sub2api.service` remained active, and recent staging journal checks at warning level had no entries. |
| Scope | Backend token-cost billing safety only | No database migration, Nginx change, payment file change, provider-routing change, production deployment, or real-provider call was part of the staging release. |
| Remaining gate | Spend-cap coverage is not complete | OpenAI WebSocket/realtime and other unbounded token requests still need reservation or spend-cap design before this risk can be closed. |

## Production Release Recorded On 2026-06-29 For Main `10f95cd25` / PR #214

| Area | Result | Evidence |
| --- | --- | --- |
| Deployment | Completed after explicit user approval | Current main `10f95cd25` was deployed to `sub2api.service`. Runtime code change is PR #214; PR #215 was docs-only. |
| Binary | Running production binary recorded | Production `/opt/sub2api/sub2api` SHA-256 is `f36b341a5e850127b1e08b65accb52e1ca7eafbbd373fdf49ce1ab3ecdcb42b1`. |
| Backup | Created before replacement | `/opt/sub2api/backups/production-before-main-10f95cd25-20260629-054440/sub2api`. |
| Production smoke | Public routes responded | `https://api.ssxzapi.com/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, `/app/profile`, and `/api/v1/settings/public` returned HTTP 200; `/v1/models` without an API key returned HTTP 401. |
| Staging split-health | Staging remained active | Staging `/opt/sub2api/sub2api-staging` stayed on SHA-256 `f6a1da5da1f52b8260e7346dcf47c4da0c94fc47611be00ed686d40e9a4a1cfc`, and `sub2api-staging.service` remained active. |
| Runtime health | Production service active | `sub2api.service` was active after restart, and the 5-minute production journal warning check returned no entries. |
| Scope | Backend token-cost billing safety release | No database migration, Nginx change, payment file change, provider-routing change, or real-provider call was part of the production release. |
| Remaining gate | Spend-cap coverage is not complete | This production release does not close OpenAI WebSocket/realtime or other unbounded token-request reservation/spend-cap design. It also does not count as real-provider production image-generation acceptance. |

## Historical Product Decisions Preserved On 2026-06-18

- The older site is not the code trunk and should not be copied wholesale. It is a product reference for user-side AI chat, AI image creation, navigation, prompt flow, result display, balance, and API access.
- The SSXZ AI product should not become image-only. Chat, image generation, balance/usage, payment/order, and API Key / third-party access are all part of the private-user tool site.
- Image generation should not be a plain fixed form only. Later product work should evaluate prompt assistance, templates, custom ratio/style, multi-reference-image input, thumbnails, history, regenerate, and download.
- DeepSeek, web search, `image2`, `gpt-image-2`, and Sora-related code are historical assets or candidate implementation details, not fixed product strategy. Reuse requires effect review, cost/risk review, and external dependency checks where applicable.
- Web search and answer-quality work are existing assets. They should not be accidentally removed, but they also should not be exposed in ordinary-user UX during P0 structure convergence.
- OpenAI Sora must not be assumed as the long-term image-generation upstream. `/sora` and Sora-named internals may remain for compatibility while product copy uses neutral image-generation language.

## Only UI / Only Backend / Not Connected / Placeholder Areas

| Area | Classification | Notes |
| --- | --- | --- |
| Workspace memory/toolbox/capability placeholders | Placeholder UI | Some disabled or "not connected" controls exist in workspace components. They must not imply working backend behavior. |
| Image generation real upstream | Staging verified, production authenticity and alias-display guards deployed | One `gpt-image-2` OpenAI-compatible path works in staging. Production was deployed on 2026-06-21 after explicit approval, PR #184 was deployed on 2026-06-27 after explicit approval, and PR #188 was deployed on 2026-06-27 after explicit approval. PR #186 confirmed staging image selection is driven by the server-side OpenAI-compatible image allowlist, and PR #188 clarified Gemini-named aliases as OpenAI-compatible image-route aliases. Real production image generation, storage, billing, history, and download still require controlled acceptance checks. |
| Image history/download in the new user journey | Staging HTTP path verified; frontend feedback hardening deployed to staging | Storage/history/download pieces work for the validated staging image. PR #192 made recent-history load failures visible and added frontend regression coverage for generated download links and filenames. Native browser download UX still needs manual confirmation. |
| Reference image upload preview | Staging verified | Single reference image preview recovered after allowing `blob:` in CSP. Multiple reference images remain a later workflow enhancement. |
| Image failure handling | Regression covered for no-charge failure paths | Invalid form input fails before upstream and does not change balance or usage. Code-path audit shows upstream errors do not reach `RecordUsage` and non-2xx responses do not persist image history. Service-level regression tests cover upstream non-2xx, transport timeout/error, and partial-success response write failure returning no successful result. Handler-level regression tests cover upstream failure without usage/billing/deduct calls, failed/truncated captures without image history, DB-backed image-history persistence, and DB-backed usage/billing persistence staying clean for upstream 4xx and transport timeout. |
| `/app/usage` charts and details | Partial, staging boundary checked | Should show real data when available and empty states otherwise. It must not invent data. PR #201 staging checks confirmed ordinary-user usage data is scrubbed, stats/trend endpoints are reachable, and admin usage data remains admin-only. |
| Payment/order user workspace | Not fully connected to new shell | Existing pages use older product structure. |
| API Key/Profile in `/app/*` shell | In progress | Keep as user-workspace pages, not admin-console pages. API Key copy and one-time full-key explanation have a first P1 polish slice merged in #195, but behavior/security verification remains open. |

## Largest Confusion Points

1. Two shells coexist: the new user workspace shell and the older Sub2API/admin-style shell.
2. Ordinary-user entries can still jump into pages that look like admin/backend console pages.
3. `/app/image`, `/sora`, and `/image-studio` all relate to image creation, but only `/app/image` should be the user-side main image-generation route.
4. `/keys`, `/usage`, and `/profile` are functional legacy routes, but the desired main paths are `/app/keys`, `/app/usage`, and `/app/profile`.
5. Payment and order flows exist but have not been fully absorbed into the user workspace information architecture.
6. Technical channel/status/provider pages should remain available when needed but should not be ordinary-user primary navigation.
7. Several local PRs have fixed individual symptoms, but the deeper product structure problem is route/shell ownership.
8. As of 2026-06-20, `ImageStudioHandler` uses the fixed model value `gpt-image-2`; future provider/model flexibility should be handled deliberately, not by hard-coding product strategy around one upstream.

## Risks And Limits

- Do not treat a production deployment as full product acceptance. Production creation, storage, billing, history, and download still need explicit acceptance evidence.
- Do not treat UI visibility as authorization; backend ownership, billing, and admin checks remain mandatory.
- Do not rename or remove Sora-related internals in a broad sweep. Product copy can become neutral while legacy code remains stable.
- Do not assume any external model/provider lifecycle without an external dependency check against official sources checked on the decision date.
- Do not remove API Key access. It is part of the user product for mature private users.
- Do not weaken admin capabilities while simplifying ordinary-user UX.
- Future production deployments still require explicit user approval and the release gates in `ROADMAP.md`.
