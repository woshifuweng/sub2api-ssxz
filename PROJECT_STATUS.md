# PROJECT_STATUS

Last updated: 2026-06-18

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
| `/app/image` | Connected at entry level | Routes to `ImageStudioView`. Real generation, cost, history, and download need end-to-end verification before production use. |
| `/app/usage` | Partial product path | New workbench-style usage page exists. It should remain in the user workspace shell. |
| `/app/keys` | Partial product path | Intended to keep API Key / third-party access in the user workspace shell. |
| `/app/profile` | Partial product path | Intended to keep account settings in the user workspace shell. |
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
| Image generation | Partial | `ImageStudioView`, `ImageStudioHandler`, OpenAI-compatible image gateway, and image/Sora-related storage exist. Full production path is not verified. |
| API Key / third-party access | Mostly complete, UX partial | Backend and frontend exist. User-facing copy and shell alignment are still being corrected. |
| Usage center | Backend complete enough, frontend partial | Usage APIs and older page exist; `/app/usage` is the desired new user-shell direction. |
| Recharge/payment | Backend/admin rich, user-shell partial | Payment/order/subscription capabilities exist; user shell alignment remains incomplete. |
| Orders | Existing, user-shell partial | Order pages exist; not yet fully aligned to the new user workspace. |
| Provider routing/channel management | Existing, admin-oriented | Keep for admin/owner operations. Do not let it dominate ordinary-user navigation. |
| Web search | Existing but frozen for main UX | Technical chain exists from prior PRs. Do not surface as ordinary-user main functionality during P0 structure work. |
| Admin operations | Rich but broad | Strong asset from the Sub2API base. Needs product boundary, not deletion. |

## Only UI / Only Backend / Not Connected / Placeholder Areas

| Area | Classification | Notes |
| --- | --- | --- |
| Workspace memory/toolbox/capability placeholders | Placeholder UI | Some disabled or "not connected" controls exist in workspace components. They must not imply working backend behavior. |
| Image generation real upstream | Needs verification | Code paths exist, but upstream account/model/permission/cost loop must be verified before production. |
| Image history/download in the new user journey | Needs verification | Storage/history/download pieces exist, but the full `/app/image` loop needs end-to-end validation. |
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

## Risks And Limits

- Do not deploy production until the user workspace information architecture is stable and the core creation/billing flows are verified.
- Do not treat UI visibility as authorization; backend ownership, billing, and admin checks remain mandatory.
- Do not rename or remove Sora-related internals in a broad sweep. Product copy can become neutral while legacy code remains stable.
- Do not assume any external model/provider lifecycle without an external dependency check against official sources checked on the decision date.
- Do not remove API Key access. It is part of the user product for mature private users.
- Do not weaken admin capabilities while simplifying ordinary-user UX.
