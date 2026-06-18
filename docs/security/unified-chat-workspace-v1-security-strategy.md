# User Workspace Security and Launch Strategy

## 1. Document status

Reviewed on 2026-06-18.

This document is a security boundary document. It does not implement the workspace. It defines what must be true before the user workspace, image generation, assets, usage, and billing flows are considered safe to merge and launch.

## 2. Product direction

The product should not be treated as only an API relay, only a chat page, or only an image tool. It should be defined as:

> A lightweight AI creation workspace with API relay capabilities, where chat, image generation, third-party API access, model routing, assets, tasks, usage, ledger, and risk controls are first-class product and backend concepts.

The user-facing workspace may use separate routes such as `/app/chat`, `/app/image`, `/app/usage`, `/app/keys`, and `/app/profile`. These routes should share the authenticated user workspace boundary and must not jump into the admin/backend-looking shell. Image generation may have a dedicated page, but it must still use backend-controlled task, asset, usage, and billing rules.

## 3. Correct next sequence

Recommended sequence:

1. Keep project status, roadmap, backlog, and agent rules aligned before implementation PRs.
2. Converge ordinary-user routes into the intended user workspace shell.
3. Verify text chat, image generation, API Key, usage, profile, payment/order, and admin boundaries independently.
4. Verify asset, task, usage, and billing behavior before production release.
5. Only then enable web browsing, memory, toolbox, document analysis, public sharing, or advanced tools.

Do not combine these phases into one large PR.

## 4. Unified Chat Workspace v1 scope

The user workspace should do:

- `/app/chat` as the user-side chat route
- `/app/image` as the user-side image-generation route
- `/app/usage`, `/app/keys`, and `/app/profile` as user-side utility routes
- consistent authenticated user workspace shell and ownership boundaries
- real conversation creation
- real message creation
- real message list loading
- real sidebar history loading
- refresh recovery for conversations and messages
- model selection passed to backend
- backend validation of model and intent
- clear error UI for failed sends
- disabled web/memory/toolbox buttons with explicit unavailable state

P0 should not do:

- web browsing
- memory
- toolbox/tools
- document analysis
- payment rewrite
- admin rewrite
- database mega-migration
- provider architecture rewrite
- production deployment changes

## 5. Required backend primitives

The long-term architecture should converge on these entities:

```text
Conversation
Message
Asset
Task
Usage
Ledger
```

Recommended intent types:

```text
chat
vision
image_generation
image_edit
file_analysis
```

Recommended task statuses:

```text
queued
running
succeeded
failed
canceled
```

Recommended message statuses:

```text
pending
streaming
completed
failed
```

Core rule:

> Image generation can have a dedicated user-facing page. The backend model should still treat image generation as a task, generated images as assets, and billing/usage as backend-controlled records.

## 6. v1 security acceptance criteria

Add these checks to the v1 implementation plan:

1. Message send endpoint requires authentication.
2. Conversation list returns only current user's conversations.
3. Conversation detail verifies ownership.
4. Message creation verifies conversation ownership.
5. Model name is validated server-side.
6. Intent is validated server-side.
7. Disabled capabilities cannot be triggered through direct API calls.
8. Backend errors are sanitized.
9. Logs do not expose `Authorization`, `api_key`, `token`, `secret`, cookies, provider keys, stack traces, internal URLs, or SQL.
10. `/app/chat` and `/app/image` do not bypass authenticated user workspace boundaries.
11. Refresh does not lose persisted conversations/messages.
12. Failed sends do not create inconsistent or orphaned message state.

## 7. Image upload/generation acceptance criteria

When image features are added later:

1. Uploaded images become `asset` records.
2. Frontend stores `asset_id`, not long-term base64/data URL content.
3. Assets are bound to `user_id` and conversation/message/task context.
4. Assets are private by default.
5. Access uses short-lived signed URLs or authenticated proxy routes.
6. File type and size are validated by backend.
7. Real MIME/content type is checked.
8. Image generation creates a `task` record.
9. Task state is recoverable after refresh.
10. Generated output becomes an asset.
11. Failed generation refunds or records no charge.
12. Retry/cancel behavior is idempotent and auditable.

## 8. Billing and abuse-control acceptance criteria

Before paid/high-cost usage is broadly opened:

1. Balance checks happen on backend.
2. Pricing is calculated on backend.
3. Frontend cannot decide cost or billable status.
4. Ledger writes are idempotent.
5. Failed provider calls do not silently charge users.
6. Duplicate requests do not double-charge.
7. Usage is recorded with provider/model metadata.
8. User-level and IP-level rate limits exist.
9. High-cost model usage has daily caps.
10. Image generation has concurrency limits.
11. Global provider kill switch exists.
12. Cost spike alerts exist.

## 9. Key security risks to keep visible

Highest-risk areas for this product:

- provider key leakage
- user data authorization bypass
- SSRF through arbitrary provider/base URLs
- public image/file hosting abuse
- duplicate/dishonest billing
- free quota abuse
- prompt injection once tools/files/web are enabled
- admin endpoints without audit
- logs leaking prompts, tokens, provider keys, or internal infrastructure
- Codex/AI agent weakening tests or expanding scope to make a demo work

## 10. Documentation-only merge criteria

This documentation branch is ready for a PR only if:

- only documentation files changed
- `AGENTS.md` was added or incrementally updated
- `docs/security/launch-checklist.md` exists
- `docs/security/unified-chat-workspace-v1-security-strategy.md` exists
- `codex-command-add-security-docs.md` is not committed
- `git diff --check` passes
- no frontend, backend, provider, payment, migration, or production configuration file changed
