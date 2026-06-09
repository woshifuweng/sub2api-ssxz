# Unified Chat Workspace v1 Security and Launch Strategy

## 1. Current project status

Current baseline from latest `main`:

- Branch used for this documentation change: `docs/security-launch-docs`
- Base branch: `main`
- `main` HEAD at branch creation: `dd3b9ac67 Merge pull request #4 from woshifuweng/codex/ci-security-baseline-issue-3`
- PR #4 commit included in `main`: `8bd896484 fix(ci): repair security baseline checks`
- PR #2 remains untouched by this documentation branch.
- No merge, deploy, Unified Chat Workspace frontend work, database migration, real payment, real billing, provider production logic, or production configuration change is included in this branch.

This document is a launch boundary document. It does not implement the workspace. It defines what must be true before Unified Chat Workspace v1 is considered safe to merge and launch.

## 2. Product direction

The product should not be treated as only an API relay or only a chat page. It should be defined as:

> A unified AI workspace with API relay capabilities, where conversation, model routing, assets, tasks, usage, ledger, and risk controls are first-class backend concepts.

The user-facing entry should be `/app`, similar to ChatGPT/Gemini/Claude-style unified input. Image generation, image understanding, file upload, and future tools should flow through the same conversation context instead of separate isolated pages.

## 3. Correct next sequence

Recommended sequence:

1. Keep the PR #4 CI/security baseline merged independently.
2. Add or update `AGENTS.md` and `docs/security/launch-checklist.md` as a documentation-only change.
3. Build Unified Chat Workspace v1 text-only closed loop.
4. Add asset upload and image task flow.
5. Add billing/ledger hardening, rate limits, and abuse monitoring.
6. Only then enable web browsing, memory, toolbox, document analysis, public sharing, or advanced tools.

Do not combine these phases into one large PR.

## 4. Unified Chat Workspace v1 scope

v1 should do:

- `/app` as the main workspace route
- `/app/chat` redirecting into `/app`
- `/app/image` redirecting into `/app?intent=image_generation` or equivalent
- real conversation creation
- real message creation
- real message list loading
- real sidebar history loading
- refresh recovery for conversations and messages
- model selection passed to backend
- backend validation of model and intent
- clear error UI for failed sends
- disabled web/memory/toolbox buttons with explicit unavailable state

v1 should not do:

- real image generation
- real image editing
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

> Image generation is an async task. Images are assets. Assets attach to messages or tasks. A separate image page is not the long-term product model.

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
10. `/app/chat` and `/app/image` do not bypass unified auth/session/conversation logic.
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
