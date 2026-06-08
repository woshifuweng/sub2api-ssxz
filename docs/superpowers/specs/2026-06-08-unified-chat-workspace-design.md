# Unified Chat Workspace Design

## Purpose

SSXZ AI should feel like one AI product, not a collection of backend menu pages. The primary user experience is a conversation-centered workspace where chat, saved conversations, image generation, image references, and future file analysis all begin from the same input area.

This design aligns the product with the strongest patterns from ChatGPT, Gemini, and Claude:

- ChatGPT: one official entry point, persistent conversations, image generation, and upload actions in the same product surface.
- Gemini: multimodal upload expectations, including documents, spreadsheets, photos, and videos.
- Claude Artifacts: complex outputs can open in a focused work area while still being driven by conversation.

The first release must prove the core loop, not expose every future capability.

## Current Findings

The live site already points `/app`, `/app/chat`, and `/app/image` at a unified input workspace. Browser verification confirmed the routes share a unified input box, history panel, and model selector.

The local codebase has two competing product paths:

- `frontend/src/views/user/AppWorkspaceView.vue` is the routed unified workspace, but it still sends chat through `/chat-studio/complete` and keeps messages in local component memory.
- `frontend/src/views/user/AppImageView.vue` has more specific image-task scaffolding and calls `createConversation` and `createImageTask`, but the current router does not use it for `/app/image`.

The backend already has shell APIs for conversations, messages, assets, and image tasks:

- `frontend/src/api/chatWorkspace.ts`
- `backend/internal/handler/chat_workspace_handler.go`
- `backend/internal/service/chat_workspace.go`
- `backend/internal/repository/chat_workspace_repo.go`
- `backend/migrations/135_create_chat_workspace_tables.sql`

The gap is not that the product lacks pieces. The gap is that the official user entry point is not wired to the durable workspace model.

## Product Principle

`/app` is the only official AI entry point.

Specialized routes like `/app/chat` and `/app/image` may set intent, placeholders, and defaults, but they must not become separate products. They should land in the same workspace and preserve the same conversation state model.

Developer, billing, account, and usage pages are support surfaces. They may appear in the shell, but they should not compete with the conversation workspace as the main product.

## Scope For V1

V1 should deliver one complete, maintainable loop:

1. User opens `/app`, `/app/chat`, or `/app/image`.
2. The workspace loads the user's saved conversations.
3. User starts a new conversation.
4. User types a prompt and optionally attaches image references.
5. The app creates or reuses a conversation.
6. The app persists user messages, assistant messages, and registered assets.
7. Image intent creates an image task shell and renders the task state in the conversation.
8. Refreshing the page or returning later restores conversation history.
9. Errors are shown inline inside the conversation, with clear recovery actions.

V1 should not claim full document, spreadsheet, code, video, memory, browsing, or toolbox execution unless the backed capability exists.

## Experience Design

The workspace layout remains:

- Left rail: brand, new chat, support shortcuts, history.
- Main area: conversation stream or empty state.
- Bottom composer: model picker, add-content button, optional capability chips, send button.
- Optional right or inline work area: used later for generated images, documents, charts, or other artifact-like outputs.

The composer is the product center. Users should not need to decide between "chat page", "image page", and "file page" as separate tools. Instead, the route and uploaded content infer the intent:

- `/app`: general assistant intent.
- `/app/chat`: chat-first intent.
- `/app/image`: image intent with image-oriented placeholder and image-capable defaults.
- Attached image plus prompt: image understanding or image generation/editing depending on wording and selected mode.

The "Add content" panel should show honest capability states:

- Image: enabled if upload and asset registration are supported.
- Document, spreadsheet, code, video: not disabled dead buttons. They should either be hidden from V1 or be clickable informational rows that explain "coming next" and do not imply the action is available.
- Browsing, memory, toolbox: hidden or informational until wired to real execution.

## Data Flow

The workspace should use `chatWorkspaceAPI` as the durable shell:

1. `listConversations()` loads sidebar history.
2. `createConversation({ title })` starts durable state when the first user action occurs.
3. `listMessages(conversationId)` restores a selected conversation.
4. `appendMessage(conversationId, payload)` records user and assistant text cards.
5. `registerAsset(payload)` records uploaded references or generated outputs.
6. `createImageTask(payload)` records image generation state.
7. `getImageTask(id)` refreshes image task status.

`/chat-studio/complete` can remain as a temporary text-completion backend, but the workspace should wrap it with durable message persistence:

- Persist the user message before calling the completion endpoint.
- Persist an assistant loading/error/result message after the call completes.
- If the text completion endpoint fails, keep the conversation and show an inline error card.

For image intent, V1 can continue creating pending image task shells if real generation is not ready. The UI copy must say that clearly. If real image generation is available, the task service should eventually call the upstream image pipeline and attach generated assets to the conversation.

## Component Boundaries

The current `AppWorkspaceView.vue` is doing too much. V1 should split responsibilities without a broad visual rewrite:

- `AppWorkspaceView.vue`: route-level orchestration, active intent, loading conversations, selected conversation.
- `WorkspaceConversationList.vue`: history list and empty state.
- `WorkspaceMessageList.vue`: rendered messages, attachments, task cards, inline errors.
- `WorkspaceComposer.vue`: draft text, model selector, attachment panel, send lifecycle.
- `WorkspaceAssetPanel.vue`: upload and future capability state rows.
- `WorkspaceModelPicker.vue`: model menu, selection, keyboard/escape/outside-click behavior.
- `useWorkspaceConversation.ts`: durable conversation state, API calls, optimistic updates.
- `useWorkspaceAssets.ts`: file validation, preview URLs, asset registration payloads.

This keeps future document, spreadsheet, and artifact work from turning one view file into a fragile product monolith.

## Error Handling

Errors should be recoverable inside the user's workflow:

- Unauthenticated: route guard sends user to login with return path.
- No available model: composer disabled with a clear upgrade/configuration action.
- Upload rejected: inline message beside the asset, including supported types and size limits.
- Conversation create failed: keep draft intact and offer retry.
- Completion failed: persist user message, append assistant error card, allow retry.
- Image task pending: show pending status honestly; do not imply generation happened.
- Asset preview failed: remove broken preview safely and keep the original message.

The UI must never silently discard a draft, uploaded file, or message after a failed request.

## Testing Strategy

Frontend tests should cover the real product loop, not just visual toggles:

- Model menu opens, selects a model, closes on outside click and Escape.
- New chat clears local draft while preserving saved history.
- First send creates a conversation and appends the user message.
- Successful completion appends and persists assistant text.
- Failed completion leaves user message and shows an assistant error card.
- Image upload validates type, creates preview, registers asset, and can be removed.
- `/app/image` sets image intent but still uses the same workspace component.
- History click loads messages for the selected conversation.

Backend tests should be added for chat-workspace:

- Create/list/get conversations scoped to the current user.
- Append/list messages scoped to the current user and conversation.
- Reject cross-user conversation, message, asset, and task access.
- Register asset validates kind/source/role and defaults.
- Create image task creates task and task message atomically.
- Idempotency key handling for image task creation.
- Routes expose expected authenticated endpoints.

Browser verification should cover:

- Login with a test user.
- `/app`, `/app/chat`, `/app/image` load the same workspace shell.
- Model picker is visually open and selectable.
- Add content opens with honest capability states.
- New chat creates a clean composer.
- A saved conversation appears in history after sending.

## Release Strategy

Do not deploy from an old server build directory.

The release source of truth must be one of:

- A clean local branch pushed to GitHub and built on the server from that exact commit.
- A clean server checkout of the PR branch.
- A tagged release artifact built from the commit hash being deployed.

Before production deploy:

1. Confirm working tree only contains intended files.
2. Run frontend typecheck and targeted workspace tests.
3. Run backend chat-workspace tests.
4. Build frontend.
5. Build backend with embedded frontend.
6. Back up current production binary.
7. Deploy exact build artifact.
8. Browser-test production after deploy.

## PR Strategy

The current local branch is ahead of `origin/main` and has a large dirty working tree. A single PR with every change would be hard to review and risky to maintain.

Split into small PRs:

1. Workspace durability PR: wire `AppWorkspaceView` to chat-workspace conversations, messages, history, and tests.
2. Asset and image task PR: register image references and render image task cards.
3. Capability honesty PR: replace disabled future buttons with honest informational states or hide them.
4. Shell support PR: improve billing/developer/account support panels without disrupting the conversation flow.
5. Deployment hygiene PR: document and script exact commit-based deployment.

The first PR should avoid broad visual redesign and avoid unrelated backend/admin/payment changes.

## Non-Goals For V1

- Full document parsing.
- Spreadsheet analysis.
- Video analysis.
- Real memory implementation.
- Browser/tool execution.
- A full Claude-style artifact editor.
- Rebuilding the admin or billing products.
- Moving every existing legacy page into `/app`.

These can come later once the conversation state model is reliable.

## Open Product Decisions

The recommended defaults are:

- Show image upload as enabled.
- Hide or informationalize document, spreadsheet, code, video, memory, browsing, and toolbox.
- Keep `/app/image` as an intent route, not a separate UI.
- Use inline conversation cards for V1 image task state.
- Add a right-side artifact panel only after there is a real artifact lifecycle.

The key decision for implementation is whether image generation in V1 should remain a pending task shell or call the real image pipeline immediately. If upstream image capability is not reliably configured, keep it pending and say so.
