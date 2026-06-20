# PROJECT_STATUS

Last updated: 2026-06-20

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
| `/app`, `/app/chat` | Connected, not final UX | Unified workspace/chat path exists. It is closer to a chat workspace than a finished lightweight user product. |
| `/app/image` | Staging generation path verified, UX partial | Routes to `ImageStudioView`. On 2026-06-20 staging generated one image through `gpt-image-2`, recorded usage, saved history, and served the PNG download. Product UX is still not final. |
| `/app/usage` | Partial product path, real data visible on staging | New workbench-style usage page exists. It should remain in the user workspace shell. |
| `/app/keys` | Partial product path, staging shell verified | Intended to keep API Key / third-party access in the user workspace shell. It must not jump to the admin/backend shell. |
| `/app/profile` | Partial product path, staging shell verified | Intended to keep account settings in the user workspace shell. It must not jump to the admin/backend shell. |
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
| Image generation | Staging closed-loop verified, production not approved | `ImageStudioView`, `ImageStudioHandler`, OpenAI-compatible image gateway, and image/Sora-related storage exist. On 2026-06-20 staging verified generation, usage cost, history, and HTTP image download with `gpt-image-2`. Production remains gated. |
| API Key / third-party access | Mostly complete, UX partial | Backend and frontend exist. User-facing copy and shell alignment are still being corrected. |
| Usage center | Backend complete enough, frontend partial | Usage APIs and older page exist; `/app/usage` is the desired new user-shell direction. |
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
| Production | Not touched | No production deployment or Nginx change was part of this validation. |

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
| Image generation real upstream | Staging verified, production gated | One `gpt-image-2` OpenAI-compatible path works in staging. Production still requires explicit approval and final release checks. |
| Image history/download in the new user journey | Staging HTTP path verified | Storage/history/download pieces work for the validated staging image. Native browser download UX still needs manual confirmation. |
| Reference image upload preview | Staging verified | Single reference image preview recovered after allowing `blob:` in CSP. Multiple reference images remain a later workflow enhancement. |
| Image failure handling | Regression covered for no-charge failure paths | Invalid form input fails before upstream and does not change balance or usage. Code-path audit shows upstream errors do not reach `RecordUsage` and non-2xx responses do not persist image history. Service-level regression tests cover upstream non-2xx, transport timeout/error, and partial-success response write failure returning no successful result. Handler-level regression tests cover upstream failure without usage/billing/deduct calls, failed/truncated captures without image history, DB-backed image-history persistence, and DB-backed usage/billing persistence staying clean for upstream 4xx and transport timeout. |
| `/app/usage` charts and details | Partial | Should show real data when available and empty states otherwise. It must not invent data. |
| Payment/order user workspace | Not fully connected to new shell | Existing pages use older product structure. |
| API Key/Profile in `/app/*` shell | In progress | Keep as user-workspace pages, not admin-console pages. |

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

- Do not deploy production until the user workspace information architecture is stable and the core creation/billing flows are verified.
- Do not treat UI visibility as authorization; backend ownership, billing, and admin checks remain mandatory.
- Do not rename or remove Sora-related internals in a broad sweep. Product copy can become neutral while legacy code remains stable.
- Do not assume any external model/provider lifecycle without an external dependency check against official sources checked on the decision date.
- Do not remove API Key access. It is part of the user product for mature private users.
- Do not weaken admin capabilities while simplifying ordinary-user UX.
- Do not treat the 2026-06-20 staging image-generation success as production approval. Production release still requires explicit user approval and the release gates in `ROADMAP.md`.
