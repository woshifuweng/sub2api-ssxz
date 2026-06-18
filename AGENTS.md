# AGENTS.md

This repository is an AI relay and unified chat workspace product. It handles user conversations, model provider calls, image/file assets, usage records, balance/ledger records, and potentially payment/admin operations. Treat all changes as production-sensitive unless the task explicitly says otherwise.

## 0. Required project context

Before planning or editing, read `PROJECT_STATUS.md`, `ROADMAP.md`, and `BACKLOG.md`. They define the 2026-06-18 product direction: this Sub2API repository is the code trunk; the older site is only a product/UX/reference asset; the target is a lightweight AI creation workspace for about 200-300 private users, with admin/owner operations preserved. Do not treat this project as a pure developer API platform, a full enterprise AI gateway rewrite, or a Sora-specific product.

Interpret user requests by product goal, not literal technology words. Mentions of DeepSeek, web search, `image2`, `gpt-image-2`, or Sora are not automatic architecture decisions. First check existing assets, product fit, cost/risk, and external dependency status where applicable.

The user usually describes needs from a product-owner and private-operator perspective, not with precise engineering terminology. Translate unclear requests into maintainable product, engineering, operations, and security plans before acting. If the literal wording conflicts with the product goal or codebase facts, explain the mismatch and propose the safer interpretation.

## 1. Agent execution protocol

Before making code changes, every coding agent must:

1. Inspect the working tree state:
   - `git status`
   - branch and target PR/issue
   - relevant changed files and existing tests
2. Read the required root documents and any relevant package-local README files before editing.
3. Restate the intended scope in a short plan and get user confirmation before code changes, unless the user has explicitly authorized autonomous execution for that exact scope.
4. Keep the diff minimal and limited to the requested scope.
5. Run the relevant tests and typechecks.
6. Report:
   - files changed
   - tests run
   - checks still failing, if any
   - whether the working tree is clean
   - any security-sensitive behavior changed

Do not continue into a new product phase unless the active PR/checks are complete or the user explicitly authorizes the transition.

Every PR should solve one small closed loop. State the goal, likely files, intentionally untouched areas, risk, rollback, and validation plan.

## 2. Scope control

Use minimal diffs. Do not refactor unrelated areas while completing a targeted task.

Unless explicitly requested, do not modify:

- payment flow
- production deployment config
- database migrations
- provider routing architecture
- admin permissions
- authentication/session architecture
- pricing formulas
- ledger semantics
- public sharing/review logic
- large UI redesigns
- Nginx configuration
- production deployment

If a requested change appears to require one of the above areas, stop and explain the dependency before editing.

UI polish must wait until P0 product structure is stable. Do not use cosmetic changes to hide route/shell ownership problems.

## 3. Security hard rules

### 3.1 Provider secrets and credentials

Never put provider API keys, payment secrets, SMS keys, OAuth secrets, JWT secrets, database credentials, or service tokens in:

- frontend code
- browser-visible config
- committed files
- fixtures that may be published
- screenshots
- logs
- user-visible errors
- test snapshots

Provider calls must go through the backend. Browser DevTools network traffic must not reveal upstream provider credentials.

### 3.2 Frontend is not a security boundary

Frontend disabled states, hidden buttons, route guards, or UI validation are only usability aids. The backend must enforce:

- authentication
- authorization
- user ownership
- model capability checks
- balance checks
- intent validation
- asset ownership
- task ownership
- admin permissions
- rate limits

Never rely on the frontend to decide whether a paid or privileged action is allowed.

### 3.3 User data isolation

All reads/writes for the following entities must be scoped to the authenticated user unless the endpoint is explicitly admin-only and audited:

- conversation
- message
- asset
- task
- usage
- ledger
- generated images
- uploaded files
- API keys owned by a user

Any endpoint that accepts an ID must verify that the resource belongs to the authenticated user.

### 3.4 Provider routing and SSRF prevention

The backend must not accept arbitrary provider base URLs from users.

Provider base URLs and model names must come from server-side allowlists. Requests to the following must be blocked unless explicitly allowed in isolated test code:

- `localhost`
- `127.0.0.1`
- `::1`
- private network ranges
- link-local addresses
- cloud metadata addresses
- internal service hostnames
- file URLs or non-HTTP schemes

Do not weaken URL allowlist tests to make CI pass.

### 3.5 Billing and ledger integrity

Billing must be calculated by the backend. The frontend must not decide:

- model price
- final cost
- ledger amount
- whether a request is billable
- refund amount

Paid/high-cost tasks should use a backend-controlled flow:

1. balance pre-check
2. reservation or pre-charge when appropriate
3. provider call
4. success settlement
5. failure refund or explicit non-charge record
6. idempotent ledger write

Ledger operations must be idempotent. Repeated submissions, retries, network failures, or page refreshes must not double-charge a user.

### 3.6 Image and file assets

Uploaded and generated files must become assets. Do not persist large base64 payloads or data URLs in message content.

Assets should be:

- tied to `user_id`
- tied to `conversation_id` and/or `message_id` / `task_id`
- stored privately by default
- served through short-lived signed URLs or authenticated proxy routes
- limited by file size and MIME type
- checked by real content type, not only filename extension

Do not expose permanent public object URLs by default. Public sharing requires an explicit review/status flow.

### 3.7 Logs and privacy

Logs must not include raw secrets or unnecessary sensitive user content.

Redact these fields by default:

- `Authorization`
- `api_key`
- `access_token`
- `refresh_token`
- `secret`
- `password`
- payment credentials
- provider keys
- cookies/session tokens

User-visible errors must not leak stack traces, SQL, internal URLs, provider keys, internal hostnames, or infrastructure details.

### 3.8 Prompt injection and tool safety

User messages, uploaded files, web pages, OCR text, and images are untrusted input.

Model output must not directly trigger high-risk operations such as:

- payment actions
- refunds
- deleting data
- changing permissions
- modifying provider keys
- exposing system prompts
- exposing other users' data
- calling internal tools without backend authorization

Tool calls require backend permission checks and, when appropriate, user confirmation.

### 3.9 No demo bypass in production logic

Do not implement production behavior with demo shortcuts such as:

- hardcoded verification codes
- frontend-only login checks
- bypassed captcha/human verification
- fake balances
- fake paid success states
- disabled authentication in production routes
- mocked provider calls outside tests/dev mode

Demo-only behavior must be clearly marked and excluded from production builds.

### 3.10 Do not cheat tests

Do not delete, weaken, skip, or invert tests to make CI pass. Fix production code or test mocks. If a test is genuinely obsolete, document why and update it with equivalent or stronger coverage.

## 4. Product structure rules

The ordinary-user side and the admin side must stay clearly separated.

### 4.1 Ordinary-user workspace

Ordinary-user workspace routes are `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/app/profile`. These pages should keep the lightweight user workspace shell, sidebar, background, and card language. Do not make ordinary-user menu items jump into the admin/backend-looking shell.

Keep API Key / third-party access visible to ordinary users. It is required for CC Switch, Cherry Studio, Chatbox, and similar clients. Present it as "API Key / third-party access", not as a heavy developer platform.

`/sora` is a compatibility route. Product copy should use neutral language such as image generation or AI image creation. Do not assume OpenAI Sora is the long-term image-generation upstream.

Legacy routes such as `/keys`, `/usage`, `/profile`, `/sora`, and payment/order routes may remain for compatibility. Do not delete routes only to simplify navigation. Hide or redirect carefully, with rollback in mind.

### 4.2 Admin and owner operations

Admin routes live under `/admin/*`. Preserve owner/admin capabilities for users, balances, orders and payments, usage, image tasks/history, API keys, invite/redeem/promo codes, accounts, groups, channels, models, pricing, basic logs, and abnormal task handling.

Do not weaken admin operations while simplifying the ordinary-user UX.

### 4.3 Workspace primitives

The product should continue to respect the backend primitives `conversation`, `message`, `asset`, `task`, `usage`, and `ledger`. Generated images and uploaded files should be assets with ownership checks. Disabled capabilities such as web browsing, memory, and toolbox must not create fake backend behavior.

## 5. API compatibility

Preserve existing public API contracts unless a breaking change is explicitly requested.

Do not casually rename fields such as:

- `access_token` to `token`
- `phone` to `mobile`
- `conversation_id` to `chat_id`
- `asset_id` to `file_id`

If compatibility must change, add migration notes, tests, and backward compatibility where feasible.

## 6. Testing requirements

For backend changes, run the relevant package tests and any security/provider/gateway tests affected by the diff.

For frontend changes, run typecheck and relevant component/unit tests.

For workspace changes, ensure tests cover:

- authenticated access
- user ownership checks
- conversation/message persistence
- refresh recovery
- invalid model rejection
- invalid intent rejection
- error handling without secret leakage
- disabled capabilities remaining disabled

For asset/task changes, ensure tests cover:

- ownership checks
- file type/size validation
- private asset access
- task status transitions
- retry/cancel semantics
- failure refund or no-charge behavior
- idempotency

## 7. Codex workflow aids

This repository includes lightweight Codex-only workflow aids:

- `docs/development/codex-workflow.md`: project-specific guidance for discovery, planning, implementation, validation, and handoff.
- `scripts/codex-preflight.ps1`: a read-only PowerShell context check. It prints branch/status, detected stack, likely verification commands, and nearby sensitive files. It must not modify files.

Use these as helpers, not as permission to bypass the stricter security rules above. Do not install Claude Code hooks, auto-format hooks, auto-upgrade scripts, or background automation unless the user explicitly asks and the behavior is reviewed first.

## 8. Definition of done

A task is not done until:

- requested scope is implemented
- unrelated scope is untouched
- tests/typechecks are run or explicitly explained
- security-sensitive paths are reviewed
- user ownership is enforced on new endpoints
- no secrets are exposed
- working tree status is reported
- remaining risks are documented
