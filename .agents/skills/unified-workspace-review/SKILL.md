---
name: unified-workspace-review
description: Review Unified Chat Workspace frontend/backend boundary, routes, conversation lifecycle, disabled capabilities, model/intent validation, and image task placeholders.
---

# Unified Workspace Review

Use this skill for tasks involving /app, /app/chat, /app/image, AppWorkspaceView, chatWorkspace API wrappers, WorkspaceComposer, message/history UI, asset upload, or image task placeholders.

## Required reads

1. AGENTS.md.
2. docs/security/launch-checklist.md.
3. docs/security/unified-chat-workspace-v1-security-strategy.md.

## Intended product model

- /app is the main workspace.
- /app/chat redirects into /app.
- /app/image redirects into /app?intent=image_generation or equivalent.
- Image generation is a task, not a separate permanent island.
- Images are assets.
- Conversations/messages/assets/tasks must be recoverable after refresh.

## v1 allowed scope

Allowed:
- workspace shell
- route redirects
- input composer UI
- model selector UI
- history/sidebar UI
- disabled/unavailable buttons
- frontend tests

Not allowed unless explicitly scoped:
- real image generation
- real image editing
- real file analysis
- web browsing
- memory
- toolbox
- payment rewrite
- database migration
- provider architecture rewrite
- billing/ledger implementation

## Review points

1. Does /app mount call missing backend endpoints?
2. Does sending a message call missing /chat-workspace endpoints?
3. Does /app/image trigger createImageTask?
4. Does image upload register fake/unsafe assets?
5. Does Data URL/base64 enter message/asset payload?
6. Does Unified Workspace call /chat-studio/complete as a fake v1 backend?
7. Are model and intent final validation left to backend?
8. Are disabled buttons truly disabled at API level?
9. Are routes authenticated?
10. Are tests covering disabled backend behavior?

## Output

Return:
- blocker/high/medium/low findings
- exact files
- minimal fix
- whether PR can merge as frontend shell
- whether PR should be split or abandoned
