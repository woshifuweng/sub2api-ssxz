# Unified Chat Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `/app`, `/app/chat`, and `/app/image` use one durable conversation workspace with saved history, persisted messages, image asset registration, and honest capability states.

**Architecture:** Keep `AppWorkspaceView.vue` as the route-level orchestrator, split focused workspace components and composables under `frontend/src/views/user/workspace/`, and wire them to `chatWorkspaceAPI`. Backend work adds tests around the existing chat-workspace service and route feature gate before the frontend depends on it.

**Tech Stack:** Vue 3, Vue Router, Vitest, Axios `apiClient`, Go service tests, Gin route tests, existing `chatWorkspaceAPI`, existing `/chat-studio/complete` temporary completion endpoint.

---

## File Structure

- Modify: `frontend/src/views/user/AppWorkspaceView.vue`
  - Keep route-level intent handling and shell composition.
  - Remove local-only message lifecycle from this file.
- Create: `frontend/src/views/user/workspace/useWorkspaceConversation.ts`
  - Own durable conversation state, optimistic message UI, send lifecycle, history loading, selected conversation loading.
- Create: `frontend/src/views/user/workspace/useWorkspaceAssets.ts`
  - Own image file validation, preview URL cleanup, and `registerAsset` payload creation.
- Create: `frontend/src/views/user/workspace/WorkspaceComposer.vue`
  - Own draft textarea, send button, model picker, asset panel, and submit event.
- Create: `frontend/src/views/user/workspace/WorkspaceModelPicker.vue`
  - Own model dropdown behavior.
- Create: `frontend/src/views/user/workspace/WorkspaceAssetPanel.vue`
  - Own image upload and honest future capability rows.
- Create: `frontend/src/views/user/workspace/WorkspaceMessageList.vue`
  - Own messages, attachments, loading/error cards, and image task cards.
- Create: `frontend/src/views/user/workspace/WorkspaceConversationList.vue`
  - Own saved history rendering for the shell.
- Modify: `frontend/src/components/user/AppSectionShell.vue`
  - Accept real history items and selected conversation id from `AppWorkspaceView`.
  - Emit `select-conversation`.
- Modify: `frontend/src/api/chatWorkspace.ts`
  - Add exported constants for message type and asset kind strings if repeated in frontend code.
- Test: `frontend/src/views/user/__tests__/AppWorkspaceView.spec.ts`
  - Replace local-only tests with durable workspace tests.
- Create: `frontend/src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts`
  - Test composable API lifecycle.
- Create: `frontend/src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts`
  - Test model picker, asset panel, send state, and future capability honesty.
- Create: `backend/internal/service/chat_workspace_test.go`
  - Add service tests with a fake repository.
- Create: `backend/internal/server/routes/chat_workspace_test.go`
  - Add route registration tests for the feature gate.

## Preflight

- [ ] **Step 1: Confirm the working tree before implementation**

Run:

```powershell
git -c safe.directory=C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/sub2api -C C:\Users\24091\Documents\Codex\2026-05-21\sub2-1-2-gpt-5-5\sub2api status --short --branch
```

Expected: the branch is `codex/gnet-sub2api`; there are existing unrelated modified and untracked files. Do not stage those files.

- [ ] **Step 2: Create a focused implementation branch**

Run:

```powershell
git -c safe.directory=C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/sub2api -C C:\Users\24091\Documents\Codex\2026-05-21\sub2-1-2-gpt-5-5\sub2api switch -c codex/unified-chat-workspace-v1
```

Expected: new branch is created from the current approved design state. If the branch already exists, switch to it and continue only after confirming it contains commit `8895c129a`.

---

### Task 1: Backend Chat Workspace Safety Tests

**Files:**
- Create: `backend/internal/service/chat_workspace_test.go`
- Test: `backend/internal/service/chat_workspace_test.go`

- [ ] **Step 1: Write failing service tests**

Create `backend/internal/service/chat_workspace_test.go` with:

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeChatWorkspaceRepo struct {
	conversations map[int64]*ChatConversation
	messages      map[int64][]*ChatMessage
	assets        map[int64]*ChatAsset
	tasks         map[int64]*ChatImageTask
	nextID        int64
}

func newFakeChatWorkspaceRepo() *fakeChatWorkspaceRepo {
	return &fakeChatWorkspaceRepo{
		conversations: map[int64]*ChatConversation{},
		messages:      map[int64][]*ChatMessage{},
		assets:        map[int64]*ChatAsset{},
		tasks:         map[int64]*ChatImageTask{},
		nextID:        1,
	}
}

func (r *fakeChatWorkspaceRepo) next() int64 {
	id := r.nextID
	r.nextID++
	return id
}

func (r *fakeChatWorkspaceRepo) CreateConversation(_ context.Context, conversation *ChatConversation) error {
	conversation.ID = r.next()
	conversation.CreatedAt = time.Now()
	conversation.UpdatedAt = conversation.CreatedAt
	r.conversations[conversation.ID] = cloneConversation(conversation)
	return nil
}

func (r *fakeChatWorkspaceRepo) ListConversations(_ context.Context, userID int64) ([]*ChatConversation, error) {
	var out []*ChatConversation
	for _, item := range r.conversations {
		if item.UserID == userID {
			out = append(out, cloneConversation(item))
		}
	}
	return out, nil
}

func (r *fakeChatWorkspaceRepo) GetConversation(_ context.Context, userID, conversationID int64) (*ChatConversation, error) {
	item := r.conversations[conversationID]
	if item == nil || item.UserID != userID {
		return nil, ErrChatWorkspaceConversationNotFound
	}
	return cloneConversation(item), nil
}

func (r *fakeChatWorkspaceRepo) CreateMessage(_ context.Context, message *ChatMessage) error {
	message.ID = r.next()
	message.CreatedAt = time.Now()
	message.UpdatedAt = message.CreatedAt
	r.messages[message.ConversationID] = append(r.messages[message.ConversationID], cloneMessage(message))
	conversation := r.conversations[message.ConversationID]
	if conversation != nil {
		conversation.LastMessageAt = &message.CreatedAt
		conversation.UpdatedAt = message.CreatedAt
	}
	return nil
}

func (r *fakeChatWorkspaceRepo) ListMessages(_ context.Context, userID, conversationID int64) ([]*ChatMessage, error) {
	if _, err := r.GetConversation(context.Background(), userID, conversationID); err != nil {
		return nil, err
	}
	var out []*ChatMessage
	for _, item := range r.messages[conversationID] {
		out = append(out, cloneMessage(item))
	}
	return out, nil
}

func (r *fakeChatWorkspaceRepo) GetMessage(_ context.Context, userID, messageID int64) (*ChatMessage, error) {
	for _, items := range r.messages {
		for _, item := range items {
			if item.ID == messageID && item.UserID == userID {
				return cloneMessage(item), nil
			}
		}
	}
	return nil, ErrChatWorkspaceMessageNotFound
}

func (r *fakeChatWorkspaceRepo) CreateAsset(_ context.Context, asset *ChatAsset) error {
	asset.ID = r.next()
	asset.CreatedAt = time.Now()
	asset.UpdatedAt = asset.CreatedAt
	r.assets[asset.ID] = cloneAsset(asset)
	return nil
}

func (r *fakeChatWorkspaceRepo) GetAsset(_ context.Context, userID, assetID int64) (*ChatAsset, error) {
	item := r.assets[assetID]
	if item == nil || item.UserID != userID {
		return nil, ErrChatWorkspaceAssetNotFound
	}
	return cloneAsset(item), nil
}

func (r *fakeChatWorkspaceRepo) CreateImageTaskWithMessage(_ context.Context, task *ChatImageTask, message *ChatMessage) error {
	task.ID = r.next()
	message.ID = r.next()
	task.MessageID = &message.ID
	message.TaskID = &task.ID
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt
	message.CreatedAt = task.CreatedAt
	message.UpdatedAt = task.CreatedAt
	r.tasks[task.ID] = cloneTask(task)
	r.messages[task.ConversationID] = append(r.messages[task.ConversationID], cloneMessage(message))
	return nil
}

func (r *fakeChatWorkspaceRepo) GetImageTask(_ context.Context, userID, taskID int64) (*ChatImageTask, error) {
	item := r.tasks[taskID]
	if item == nil || item.UserID != userID {
		return nil, ErrChatWorkspaceTaskNotFound
	}
	return cloneTask(item), nil
}

func cloneConversation(in *ChatConversation) *ChatConversation {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneMessage(in *ChatMessage) *ChatMessage {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneAsset(in *ChatAsset) *ChatAsset {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneTask(in *ChatImageTask) *ChatImageTask {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func TestChatWorkspaceServiceCreatesConversationAndMessage(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	conversation, err := svc.CreateConversation(context.Background(), 42, "First prompt")
	require.NoError(t, err)
	require.Equal(t, int64(42), conversation.UserID)
	require.Equal(t, ChatConversationStatusActive, conversation.Status)

	message, err := svc.AppendMessage(context.Background(), CreateChatMessageInput{
		UserID:         42,
		ConversationID: conversation.ID,
		MessageType:    ChatMessageTypeText,
		Role:           "user",
		Content:        "hello",
	})
	require.NoError(t, err)
	require.Equal(t, "hello", message.Content)

	messages, err := svc.ListMessages(context.Background(), 42, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "user", messages[0].Role)
}

func TestChatWorkspaceServiceRejectsCrossConversationAsset(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	left, err := svc.CreateConversation(context.Background(), 42, "left")
	require.NoError(t, err)
	right, err := svc.CreateConversation(context.Background(), 42, "right")
	require.NoError(t, err)

	asset, err := svc.RegisterAsset(context.Background(), RegisterChatAssetInput{
		UserID:         42,
		ConversationID: &left.ID,
		AssetKind:      ChatAssetKindImage,
		URL:            "data:image/png;base64,abc",
		OriginalName:   "sample.png",
		ContentType:    "image/png",
		ByteSize:       10,
	})
	require.NoError(t, err)

	_, err = svc.AppendMessage(context.Background(), CreateChatMessageInput{
		UserID:         42,
		ConversationID: right.ID,
		MessageType:    ChatMessageTypeAttachment,
		Role:           "user",
		Content:        "wrong conversation",
		AssetID:        &asset.ID,
	})
	require.ErrorIs(t, err, ErrChatWorkspaceInvalidInput)
}

func TestChatWorkspaceServiceCreatesImageTaskAndMessage(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	conversation, err := svc.CreateConversation(context.Background(), 42, "image")
	require.NoError(t, err)

	task, err := svc.CreateImageTask(context.Background(), CreateChatImageTaskInput{
		UserID:         42,
		ConversationID: conversation.ID,
		IdempotencyKey: "image-key-1",
		Prompt:         "make a clean product image",
		Model:          "gpt-image-1",
		Ratio:          "1:1",
		Purpose:        "product",
		Style:          "clean",
	})
	require.NoError(t, err)
	require.Equal(t, ChatImageTaskStatusPending, task.TaskStatus)
	require.Equal(t, ChatImageBillingStatusNotBilled, task.BillingStatus)
	require.NotNil(t, task.MessageID)

	messages, err := svc.ListMessages(context.Background(), 42, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, ChatMessageTypeImageTask, messages[0].MessageType)
}

func TestChatWorkspaceServiceRejectsInvalidInput(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	_, err := svc.CreateConversation(context.Background(), 0, "bad")
	require.ErrorIs(t, err, ErrChatWorkspaceInvalidInput)

	_, err = svc.RegisterAsset(context.Background(), RegisterChatAssetInput{
		UserID:    42,
		AssetKind: "video",
	})
	require.ErrorIs(t, err, ErrChatWorkspaceInvalidAssetKind)

	errUnexpected := errors.New("unexpected")
	require.NotErrorIs(t, errUnexpected, ErrChatWorkspaceInvalidInput)
}
```

- [ ] **Step 2: Run tests and confirm they pass or expose real regressions**

Run:

```powershell
go test ./internal/service -run ChatWorkspace -count=1
```

Expected: PASS. If this fails due to compile drift in `ChatWorkspaceService`, fix the service before continuing.

- [ ] **Step 3: Commit backend service tests**

Run:

```powershell
git add backend/internal/service/chat_workspace_test.go
git commit -m "test(backend): cover chat workspace service"
```

Expected: one commit containing only `backend/internal/service/chat_workspace_test.go`.

---

### Task 2: Chat Workspace Route Gate Tests

**Files:**
- Create: `backend/internal/server/routes/chat_workspace_test.go`
- Test: `backend/internal/server/routes/chat_workspace_test.go`

- [ ] **Step 1: Write route feature-gate tests**

Create `backend/internal/server/routes/chat_workspace_test.go` with:

```go
package routes

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestExecutableChatWorkspaceRoutesDisabledByDefault(t *testing.T) {
	t.Setenv(config.ChatWorkspaceEnabledEnv, "")

	routes := ExecutableChatWorkspaceRoutes(&handler.Handlers{})
	require.Empty(t, routes)
}

func TestExecutableChatWorkspaceRoutesRequireHandler(t *testing.T) {
	t.Setenv(config.ChatWorkspaceEnabledEnv, "true")

	require.Empty(t, ExecutableChatWorkspaceRoutes(nil))
	require.Empty(t, ExecutableChatWorkspaceRoutes(&handler.Handlers{}))
}

func TestExecutableChatWorkspaceRoutesEnabled(t *testing.T) {
	t.Setenv(config.ChatWorkspaceEnabledEnv, "true")

	h := &handler.Handlers{
		ChatWorkspace: &handler.ChatWorkspaceHandler{},
	}
	routes := ExecutableChatWorkspaceRoutes(h)

	require.Len(t, routes, 9)
	require.Equal(t, "/api/v1/chat-workspace/conversations", routes[0].Path)
	require.Equal(t, "/api/v1/chat-workspace/image-tasks/:id", routes[len(routes)-1].Path)
	for _, route := range routes {
		require.Contains(t, route.Middleware, "jwt_auth")
		require.Contains(t, route.Middleware, "backend_mode_user_guard")
	}
}
```

- [ ] **Step 2: Run route tests**

Run:

```powershell
go test ./internal/server/routes -run ChatWorkspace -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit route tests**

Run:

```powershell
git add backend/internal/server/routes/chat_workspace_test.go
git commit -m "test(backend): cover chat workspace routes"
```

Expected: one commit containing only `backend/internal/server/routes/chat_workspace_test.go`.

---

### Task 3: Workspace Conversation Composable

**Files:**
- Create: `frontend/src/views/user/workspace/useWorkspaceConversation.ts`
- Create: `frontend/src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts`
- Modify: `frontend/src/api/chatWorkspace.ts`

- [ ] **Step 1: Add shared workspace constants**

Modify `frontend/src/api/chatWorkspace.ts` near the type declarations:

```ts
export const CHAT_MESSAGE_TYPE_TEXT = 'text'
export const CHAT_MESSAGE_TYPE_ATTACHMENT = 'attachment'
export const CHAT_MESSAGE_TYPE_IMAGE_TASK = 'image_task'
export const CHAT_MESSAGE_TYPE_ERROR_CARD = 'error_card'

export const CHAT_ASSET_KIND_IMAGE = 'image'
export const CHAT_ASSET_SOURCE_USER_UPLOAD = 'user_upload'
export const CHAT_ASSET_ROLE_ATTACHMENT = 'attachment'
```

- [ ] **Step 2: Write failing composable tests**

Create `frontend/src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts` with:

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  listConversations: vi.fn(),
  createConversation: vi.fn(),
  listMessages: vi.fn(),
  appendMessage: vi.fn(),
  createImageTask: vi.fn()
}))

vi.mock('@/api/chatWorkspace', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/chatWorkspace')>()
  return {
    ...actual,
    listConversations: api.listConversations,
    createConversation: api.createConversation,
    listMessages: api.listMessages,
    appendMessage: api.appendMessage,
    createImageTask: api.createImageTask
  }
})

vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn().mockResolvedValue({
      data: {
        choices: [{ message: { content: 'assistant answer' } }]
      }
    })
  }
}))

import { useWorkspaceConversation } from '../useWorkspaceConversation'

describe('useWorkspaceConversation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.listConversations.mockResolvedValue([])
    api.createConversation.mockResolvedValue({
      id: 10,
      title: 'hello',
      status: 'active',
      created_at: '2026-06-08T00:00:00Z',
      updated_at: '2026-06-08T00:00:00Z'
    })
    api.listMessages.mockResolvedValue([])
    api.appendMessage.mockImplementation(async (_conversationId, payload) => ({
      id: Math.floor(Math.random() * 1000) + 1,
      conversation_id: 10,
      message_type: payload.message_type,
      role: payload.role,
      content: payload.content,
      metadata: payload.metadata,
      created_at: '2026-06-08T00:00:00Z',
      updated_at: '2026-06-08T00:00:00Z'
    }))
  })

  it('loads conversations', async () => {
    api.listConversations.mockResolvedValue([{ id: 1, title: 'Saved', status: 'active', created_at: 'a', updated_at: 'b' }])
    const workspace = useWorkspaceConversation()

    await workspace.loadHistory()

    expect(workspace.conversations.value).toHaveLength(1)
    expect(workspace.conversations.value[0].title).toBe('Saved')
  })

  it('creates a conversation and persists user and assistant messages on send', async () => {
    const workspace = useWorkspaceConversation()

    await workspace.sendTextMessage({
      text: 'hello',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(api.createConversation).toHaveBeenCalledWith({ title: 'hello' })
    expect(api.appendMessage).toHaveBeenCalledTimes(2)
    expect(workspace.messages.value.map((m) => m.role)).toEqual(['user', 'assistant'])
    expect(workspace.messages.value[1].content).toBe('assistant answer')
  })

  it('keeps the user message and records an error card when completion fails', async () => {
    const { apiClient } = await import('@/api/client')
    vi.mocked(apiClient.post).mockRejectedValueOnce({ status: 503, message: 'upstream down' })
    const workspace = useWorkspaceConversation()

    await workspace.sendTextMessage({
      text: 'hello',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(workspace.messages.value).toHaveLength(2)
    expect(workspace.messages.value[0].role).toBe('user')
    expect(workspace.messages.value[1].state).toBe('error')
    expect(api.appendMessage).toHaveBeenCalledTimes(2)
  })

  it('loads selected conversation messages', async () => {
    api.listMessages.mockResolvedValue([
      { id: 1, conversation_id: 5, message_type: 'text', role: 'user', content: 'saved', created_at: 'a', updated_at: 'b' }
    ])
    const workspace = useWorkspaceConversation()

    await workspace.selectConversation(5)

    expect(workspace.activeConversationId.value).toBe(5)
    expect(workspace.messages.value[0].content).toBe('saved')
  })
})
```

- [ ] **Step 3: Run composable test and confirm failure**

Run:

```powershell
cd frontend
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts
```

Expected: FAIL because `useWorkspaceConversation.ts` does not exist.

- [ ] **Step 4: Implement `useWorkspaceConversation.ts`**

Create `frontend/src/views/user/workspace/useWorkspaceConversation.ts`:

```ts
import { computed, ref } from 'vue'
import { apiClient } from '@/api/client'
import {
  CHAT_MESSAGE_TYPE_ERROR_CARD,
  CHAT_MESSAGE_TYPE_TEXT,
  type ChatAsset,
  type ChatConversation,
  type ChatMessage,
  appendMessage,
  createConversation,
  listConversations,
  listMessages
} from '@/api/chatWorkspace'

export type WorkspaceIntent = 'home' | 'chat' | 'image'
export type WorkspaceMessageState = 'loading' | 'error'

export interface WorkspaceAttachment {
  id: string
  name: string
  url: string
  type: 'image'
  asset?: ChatAsset
}

export interface WorkspaceMessage {
  id: string
  persistedId?: number
  conversationId?: number
  messageType: string
  role: 'user' | 'assistant'
  content: string
  state?: WorkspaceMessageState
  attachments?: WorkspaceAttachment[]
  createdAt?: string
}

export interface SendTextMessageInput {
  text: string
  model: string
  intent: WorkspaceIntent
  attachments: WorkspaceAttachment[]
}

interface ChatStudioResponse {
  choices?: Array<{
    message?: {
      content?: string | Array<{ text?: string; content?: string }>
    }
  }>
}

export function useWorkspaceConversation() {
  const conversations = ref<ChatConversation[]>([])
  const activeConversationId = ref<number | null>(null)
  const messages = ref<WorkspaceMessage[]>([])
  const loadingHistory = ref(false)
  const loadingMessages = ref(false)
  const sending = ref(false)
  const errorMessage = ref('')

  const activeConversation = computed(() =>
    conversations.value.find((item) => item.id === activeConversationId.value) || null
  )

  async function loadHistory() {
    loadingHistory.value = true
    errorMessage.value = ''
    try {
      conversations.value = await listConversations()
    } catch (error) {
      errorMessage.value = normalizeErrorMessage(error, '无法读取历史会话')
    } finally {
      loadingHistory.value = false
    }
  }

  async function selectConversation(id: number) {
    activeConversationId.value = id
    loadingMessages.value = true
    errorMessage.value = ''
    try {
      const persisted = await listMessages(id)
      messages.value = persisted.map(mapPersistedMessage)
    } catch (error) {
      errorMessage.value = normalizeErrorMessage(error, '无法读取会话消息')
    } finally {
      loadingMessages.value = false
    }
  }

  async function startNewChat() {
    activeConversationId.value = null
    messages.value = []
    errorMessage.value = ''
  }

  async function ensureConversationForAssets(title: string) {
    return ensureConversation(title || '新对话')
  }

  async function sendTextMessage(input: SendTextMessageInput) {
    const text = input.text.trim()
    if (!text && input.attachments.length === 0) return
    if (sending.value) return

    sending.value = true
    errorMessage.value = ''

    const localUser = createLocalMessage('user', text, input.attachments)
    messages.value.push(localUser)

    try {
      const conversationId = await ensureConversation(text || '图片任务')
      const persistedUser = await appendMessage(conversationId, {
        message_type: CHAT_MESSAGE_TYPE_TEXT,
        role: 'user',
        content: text,
        metadata: {
          intent: input.intent,
          model: input.model,
          attachments: input.attachments.map((attachment) => ({
            name: attachment.name,
            type: attachment.type,
            asset_id: attachment.asset?.id,
            url: attachment.url
          }))
        }
      })
      patchPersisted(localUser.id, persistedUser)

      const loadingAssistant = createLocalMessage('assistant', '正在思考...', [])
      loadingAssistant.state = 'loading'
      messages.value.push(loadingAssistant)

      try {
        const completion = await requestChatCompletion(buildCompletionMessages(), input.model)
        const assistantText = extractAssistantText(completion)
        const persistedAssistant = await appendMessage(conversationId, {
          message_type: CHAT_MESSAGE_TYPE_TEXT,
          role: 'assistant',
          content: assistantText,
          metadata: { intent: input.intent, model: input.model }
        })
        patchPersisted(loadingAssistant.id, persistedAssistant)
      } catch (error) {
        const assistantText = normalizeErrorMessage(error, '当前模型服务暂不可用，请稍后重试。')
        const persistedAssistant = await appendMessage(conversationId, {
          message_type: CHAT_MESSAGE_TYPE_ERROR_CARD,
          role: 'assistant',
          content: assistantText,
          metadata: { intent: input.intent, model: input.model }
        })
        patchPersisted(loadingAssistant.id, persistedAssistant, 'error')
      }

      await loadHistory()
    } finally {
      sending.value = false
    }
  }

  async function ensureConversation(title: string) {
    if (activeConversationId.value) return activeConversationId.value
    const conversation = await createConversation({ title: title.slice(0, 80) })
    conversations.value = [conversation, ...conversations.value.filter((item) => item.id !== conversation.id)]
    activeConversationId.value = conversation.id
    return conversation.id
  }

  function patchPersisted(localId: string, persisted: ChatMessage, state?: WorkspaceMessageState) {
    const index = messages.value.findIndex((message) => message.id === localId)
    if (index === -1) return
    messages.value[index] = {
      ...messages.value[index],
      persistedId: persisted.id,
      conversationId: persisted.conversation_id,
      messageType: persisted.message_type,
      role: persisted.role === 'assistant' ? 'assistant' : 'user',
      content: persisted.content,
      state,
      createdAt: persisted.created_at
    }
  }

  function buildCompletionMessages() {
    return messages.value
      .filter((message) => message.messageType === CHAT_MESSAGE_TYPE_TEXT && !message.state && message.content.trim())
      .map((message) => ({
        role: message.role,
        content: message.content.trim()
      }))
      .slice(-40)
  }

  return {
    activeConversation,
    activeConversationId,
    conversations,
    errorMessage,
    loadingHistory,
    loadingMessages,
    messages,
    sending,
    loadHistory,
    selectConversation,
    ensureConversationForAssets,
    sendTextMessage,
    startNewChat
  }
}

function mapPersistedMessage(message: ChatMessage): WorkspaceMessage {
  return {
    id: `persisted-${message.id}`,
    persistedId: message.id,
    conversationId: message.conversation_id,
    messageType: message.message_type,
    role: message.role === 'assistant' ? 'assistant' : 'user',
    content: message.content,
    createdAt: message.created_at,
    state: message.message_type === CHAT_MESSAGE_TYPE_ERROR_CARD ? 'error' : undefined
  }
}

function createLocalMessage(
  role: 'user' | 'assistant',
  content: string,
  attachments: WorkspaceAttachment[]
): WorkspaceMessage {
  return {
    id: `${role}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    messageType: CHAT_MESSAGE_TYPE_TEXT,
    role,
    content,
    attachments
  }
}

async function requestChatCompletion(payloadMessages: Array<{ role: string; content: string }>, model: string) {
  const { data } = await apiClient.post<ChatStudioResponse>('/chat-studio/complete', {
    model,
    mode: 'general',
    messages: payloadMessages,
    temperature: 0.7
  })
  return data
}

function extractAssistantText(payload: ChatStudioResponse): string {
  const content = payload?.choices?.[0]?.message?.content
  if (typeof content === 'string' && content.trim()) return content.trim()
  if (Array.isArray(content)) {
    const text = content.map((item) => item?.text || item?.content || '').filter(Boolean).join('\n').trim()
    if (text) return text
  }
  return '已收到回复，但当前页面无法展示返回内容。'
}

function normalizeErrorMessage(error: unknown, fallback: string) {
  if (typeof error === 'object' && error !== null) {
    const value = error as { message?: unknown; response?: { data?: { message?: unknown; detail?: unknown } } }
    const message = value.response?.data?.message || value.response?.data?.detail || value.message
    if (typeof message === 'string' && message.trim()) return message
  }
  return fallback
}
```

- [ ] **Step 5: Run composable tests**

Run:

```powershell
cd frontend
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts
```

Expected: PASS.

- [ ] **Step 6: Commit composable work**

Run:

```powershell
git add frontend/src/api/chatWorkspace.ts frontend/src/views/user/workspace/useWorkspaceConversation.ts frontend/src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts
git commit -m "feat(frontend): add durable workspace conversation state"
```

Expected: one commit containing the composable, constants, and tests.

---

### Task 4: Workspace Assets Composable

**Files:**
- Create: `frontend/src/views/user/workspace/useWorkspaceAssets.ts`
- Create: `frontend/src/views/user/workspace/__tests__/useWorkspaceAssets.spec.ts`

- [ ] **Step 1: Write failing asset tests**

Create `frontend/src/views/user/workspace/__tests__/useWorkspaceAssets.spec.ts`:

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  registerAsset: vi.fn()
}))

vi.mock('@/api/chatWorkspace', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/chatWorkspace')>()
  return {
    ...actual,
    registerAsset: api.registerAsset
  }
})

import { useWorkspaceAssets } from '../useWorkspaceAssets'

describe('useWorkspaceAssets', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.registerAsset.mockResolvedValue({
      id: 9,
      asset_kind: 'image',
      source_type: 'user_upload',
      asset_role: 'attachment',
      storage_provider: 'pending',
      storage_key: '',
      url: 'data:image/png;base64,abc',
      original_name: 'sample.png',
      content_type: 'image/png',
      byte_size: 10,
      status: 'registered',
      created_at: 'a',
      updated_at: 'b'
    })
  })

  it('accepts image files and rejects non-images', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })
    const text = new File(['abc'], 'notes.txt', { type: 'text/plain' })

    await assets.addFiles([image, text])

    expect(assets.previews.value).toHaveLength(1)
    expect(assets.rejectedFiles.value[0].name).toBe('notes.txt')
  })

  it('registers pending image assets for a conversation', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })

    await assets.addFiles([image])
    const registered = await assets.registerPendingAssets(10)

    expect(api.registerAsset).toHaveBeenCalledWith(expect.objectContaining({
      conversation_id: 10,
      asset_kind: 'image',
      source_type: 'user_upload',
      asset_role: 'attachment',
      original_name: 'sample.png',
      content_type: 'image/png'
    }))
    expect(registered[0].asset?.id).toBe(9)
  })

  it('removes previews and clears state', async () => {
    const assets = useWorkspaceAssets()
    const image = new File(['abc'], 'sample.png', { type: 'image/png' })

    await assets.addFiles([image])
    const id = assets.previews.value[0].id
    assets.removePreview(id)

    expect(assets.previews.value).toHaveLength(0)
  })
})
```

- [ ] **Step 2: Implement `useWorkspaceAssets.ts`**

Create `frontend/src/views/user/workspace/useWorkspaceAssets.ts`:

```ts
import { ref } from 'vue'
import {
  CHAT_ASSET_KIND_IMAGE,
  CHAT_ASSET_ROLE_ATTACHMENT,
  CHAT_ASSET_SOURCE_USER_UPLOAD,
  registerAsset,
  type ChatAsset
} from '@/api/chatWorkspace'
import type { WorkspaceAttachment } from './useWorkspaceConversation'

export interface WorkspaceAssetPreview {
  id: string
  file: File
  name: string
  url: string
  size: number
  sizeLabel: string
  asset?: ChatAsset
}

export interface RejectedWorkspaceFile {
  name: string
  reason: string
}

const MAX_IMAGES = 4

export function useWorkspaceAssets() {
  const previews = ref<WorkspaceAssetPreview[]>([])
  const rejectedFiles = ref<RejectedWorkspaceFile[]>([])
  const registering = ref(false)

  async function addFiles(files: File[]) {
    rejectedFiles.value = []
    const slots = Math.max(0, MAX_IMAGES - previews.value.length)
    for (const file of files.slice(0, slots)) {
      if (!file.type.startsWith('image/')) {
        rejectedFiles.value.push({ name: file.name, reason: '当前仅支持图片上传' })
        continue
      }
      previews.value.push({
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}-${file.name}`,
        file,
        name: file.name,
        url: await readAsDataUrl(file),
        size: file.size,
        sizeLabel: formatFileSize(file.size)
      })
    }
    for (const file of files.slice(slots)) {
      rejectedFiles.value.push({ name: file.name, reason: `最多添加 ${MAX_IMAGES} 张图片` })
    }
  }

  function removePreview(id: string) {
    previews.value = previews.value.filter((preview) => preview.id !== id)
  }

  function clearPreviews() {
    previews.value = []
    rejectedFiles.value = []
  }

  async function registerPendingAssets(conversationId: number): Promise<WorkspaceAttachment[]> {
    registering.value = true
    try {
      const attachments: WorkspaceAttachment[] = []
      for (const preview of previews.value) {
        const asset = preview.asset || await registerAsset({
          conversation_id: conversationId,
          asset_kind: CHAT_ASSET_KIND_IMAGE,
          source_type: CHAT_ASSET_SOURCE_USER_UPLOAD,
          asset_role: CHAT_ASSET_ROLE_ATTACHMENT,
          storage_provider: 'pending',
          url: preview.url,
          preview_url: preview.url,
          original_name: preview.name,
          content_type: preview.file.type,
          byte_size: preview.file.size
        })
        preview.asset = asset
        attachments.push({
          id: preview.id,
          name: preview.name,
          url: preview.url,
          type: 'image',
          asset
        })
      }
      return attachments
    } finally {
      registering.value = false
    }
  }

  return {
    previews,
    registering,
    rejectedFiles,
    addFiles,
    clearPreviews,
    registerPendingAssets,
    removePreview
  }
}

function readAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
    reader.onerror = () => reject(reader.error || new Error('file read failed'))
    reader.readAsDataURL(file)
  })
}

function formatFileSize(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 KB'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  const precision = value >= 10 || unitIndex === 0 ? 0 : 1
  return `${value.toFixed(precision)} ${units[unitIndex]}`
}
```

- [ ] **Step 3: Run asset tests**

Run:

```powershell
cd frontend
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/workspace/__tests__/useWorkspaceAssets.spec.ts
```

Expected: PASS.

- [ ] **Step 4: Commit asset composable**

Run:

```powershell
git add frontend/src/views/user/workspace/useWorkspaceAssets.ts frontend/src/views/user/workspace/__tests__/useWorkspaceAssets.spec.ts
git commit -m "feat(frontend): add workspace asset registration"
```

Expected: one focused commit.

---

### Task 5: Split Workspace UI Components

**Files:**
- Create: `frontend/src/views/user/workspace/WorkspaceModelPicker.vue`
- Create: `frontend/src/views/user/workspace/WorkspaceAssetPanel.vue`
- Create: `frontend/src/views/user/workspace/WorkspaceComposer.vue`
- Create: `frontend/src/views/user/workspace/WorkspaceMessageList.vue`
- Create: `frontend/src/views/user/workspace/WorkspaceConversationList.vue`
- Create: `frontend/src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts`

- [ ] **Step 1: Write composer behavior tests**

Create `frontend/src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts`:

```ts
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import WorkspaceComposer from '../WorkspaceComposer.vue'

describe('WorkspaceComposer', () => {
  const models = [
    { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' as const },
    { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard' as const }
  ]

  it('opens model picker and selects a model', async () => {
    const wrapper = mount(WorkspaceComposer, {
      props: {
        modelValue: '',
        selectedModel: 'gpt-5.5',
        models,
        intent: 'chat',
        sending: false,
        assetPreviews: [],
        rejectedFiles: []
      }
    })

    await wrapper.get('[data-testid="workspace-model-trigger"]').trigger('click')
    expect(wrapper.find('[data-testid="workspace-model-menu"]').exists()).toBe(true)

    await wrapper.findAll('[data-testid="workspace-model-option"]')[1].trigger('click')
    expect(wrapper.emitted('update:selectedModel')?.[0]).toEqual(['gpt-5.4-mini'])
  })

  it('shows honest future capability states', async () => {
    const wrapper = mount(WorkspaceComposer, {
      props: {
        modelValue: '',
        selectedModel: 'gpt-5.5',
        models,
        intent: 'image',
        sending: false,
        assetPreviews: [],
        rejectedFiles: []
      }
    })

    await wrapper.get('[data-testid="workspace-add-content"]').trigger('click')

    expect(wrapper.text()).toContain('图片')
    expect(wrapper.text()).toContain('即将接入')
    expect(wrapper.findAll('button[disabled]').length).toBe(0)
  })

  it('enables send when draft has text', async () => {
    const wrapper = mount(WorkspaceComposer, {
      props: {
        modelValue: '',
        selectedModel: 'gpt-5.5',
        models,
        intent: 'chat',
        sending: false,
        assetPreviews: [],
        rejectedFiles: []
      }
    })

    await wrapper.get('textarea').setValue('hello')
    expect(wrapper.get('[data-testid="workspace-send"]').attributes('disabled')).toBeUndefined()
  })
})
```

- [ ] **Step 2: Implement `WorkspaceModelPicker.vue`**

Create `frontend/src/views/user/workspace/WorkspaceModelPicker.vue`:

```vue
<template>
  <div ref="rootRef" class="model-selector">
    <button
      type="button"
      class="model-trigger"
      data-testid="workspace-model-trigger"
      :disabled="!models.length"
      :aria-expanded="open"
      aria-controls="workspace-model-menu"
      @click.stop="toggle"
    >
      <span>{{ selectedLabel }}</span>
      <Icon name="chevronDown" size="xs" />
    </button>
    <div v-if="open" id="workspace-model-menu" class="model-menu" data-testid="workspace-model-menu">
      <button
        v-for="model in models"
        :key="model.id"
        type="button"
        class="model-option"
        data-testid="workspace-model-option"
        :class="{ 'is-selected': model.id === selectedModel }"
        @click.stop="select(model.id)"
      >
        <span>{{ model.name || model.id }}</span>
        <small>{{ model.tier === 'standard' ? '轻量' : '高级' }}</small>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { ChatModelOption } from '@/composables/useUserCapabilities'

const props = defineProps<{
  models: ChatModelOption[]
  selectedModel: string
}>()

const emit = defineEmits<{
  (event: 'update:selectedModel', value: string): void
}>()

const open = ref(false)
const rootRef = ref<HTMLElement | null>(null)

const selectedLabel = computed(() => {
  const model = props.models.find((item) => item.id === props.selectedModel)
  return model?.name || props.selectedModel || '暂无可用模型'
})

function toggle() {
  if (!props.models.length) return
  open.value = !open.value
}

function select(modelId: string) {
  emit('update:selectedModel', modelId)
  open.value = false
}

function handlePointerDown(event: PointerEvent) {
  if (!open.value) return
  if (event.target instanceof Node && rootRef.value?.contains(event.target)) return
  open.value = false
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('pointerdown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
})
</script>
```

- [ ] **Step 3: Implement asset panel, composer, message list, and conversation list**

Create the remaining components with these responsibilities:

```ts
// WorkspaceAssetPanel.vue props/events
defineProps<{
  rejectedFiles: Array<{ name: string; reason: string }>
}>()
defineEmits<{
  (event: 'files', files: File[]): void
}>()
```

```ts
// WorkspaceComposer.vue props/events
defineProps<{
  modelValue: string
  selectedModel: string
  models: ChatModelOption[]
  intent: WorkspaceIntent
  sending: boolean
  assetPreviews: WorkspaceAssetPreview[]
  rejectedFiles: RejectedWorkspaceFile[]
}>()
defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'update:selectedModel', value: string): void
  (event: 'files', files: File[]): void
  (event: 'remove-asset', id: string): void
  (event: 'submit'): void
}>()
```

```ts
// WorkspaceMessageList.vue props/events
defineProps<{
  messages: WorkspaceMessage[]
  loading: boolean
}>()
```

```ts
// WorkspaceConversationList.vue props/events
defineProps<{
  conversations: ChatConversation[]
  activeConversationId: number | null
  loading: boolean
}>()
defineEmits<{
  (event: 'select', id: number): void
}>()
```

Use class names already present in `AppWorkspaceView.vue` where possible: `composer-card`, `asset-panel`, `message-list`, `message-row`, `message-bubble`, `model-selector`, `model-menu`, `model-option`.

- [ ] **Step 4: Run composer tests**

Run:

```powershell
cd frontend
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit UI split**

Run:

```powershell
git add frontend/src/views/user/workspace
git commit -m "feat(frontend): split workspace UI components"
```

Expected: one commit containing only workspace UI components and tests.

---

### Task 6: Wire AppWorkspaceView To Durable State

**Files:**
- Modify: `frontend/src/views/user/AppWorkspaceView.vue`
- Modify: `frontend/src/components/user/AppSectionShell.vue`
- Modify: `frontend/src/views/user/__tests__/AppWorkspaceView.spec.ts`

- [ ] **Step 1: Update `AppSectionShell.vue` contract**

Add props and emit:

```ts
import type { ChatConversation } from '@/api/chatWorkspace'

const props = withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
  historyItems?: ChatConversation[]
  activeConversationId?: number | null
  historyLoading?: boolean
}>(), {
  eyebrow: 'SSXZ AI 对话工作台',
  icon: 'sparkles',
  historyItems: () => [],
  activeConversationId: null,
  historyLoading: false
})

const emit = defineEmits<{
  (e: 'new-chat'): void
  (e: 'select-conversation', id: number): void
}>()
```

Replace the hard-coded empty `historyItems` with `props.historyItems`. Render each item as a button, not a router link:

```vue
<button
  v-for="item in historyItems"
  :key="item.id"
  type="button"
  class="ssxz-nav-item"
  :class="{ 'is-active': item.id === activeConversationId }"
  :title="item.title || '未命名会话'"
  @click="$emit('select-conversation', item.id)"
>
  <Icon name="chat" size="sm" />
  <span class="ssxz-sidebar-text">{{ item.title || '未命名会话' }}</span>
</button>
<p v-if="historyLoading" class="ssxz-empty-history ssxz-sidebar-text">正在加载历史会话...</p>
<p v-else-if="historyItems.length === 0" class="ssxz-empty-history ssxz-sidebar-text">暂无历史会话</p>
```

- [ ] **Step 2: Replace local message lifecycle in `AppWorkspaceView.vue`**

Use:

```ts
import WorkspaceComposer from './workspace/WorkspaceComposer.vue'
import WorkspaceMessageList from './workspace/WorkspaceMessageList.vue'
import { useWorkspaceAssets } from './workspace/useWorkspaceAssets'
import { useWorkspaceConversation, type WorkspaceIntent } from './workspace/useWorkspaceConversation'
```

Set intent from route:

```ts
const workspaceIntent = computed<WorkspaceIntent>(() => {
  if (activeSection.value === 'image') return 'image'
  if (activeSection.value === 'chat') return 'chat'
  return 'home'
})
```

Wire state:

```ts
const workspace = useWorkspaceConversation()
const assets = useWorkspaceAssets()
const draft = ref('')
const selectedModelId = ref('')
```

On mount:

```ts
onMounted(() => {
  loadCapabilities()
  workspace.loadHistory()
})
```

Submit:

```ts
async function submitDraft() {
  const model = resolveUsableChatModel()
  if (!model) return
  const conversationId = await workspace.ensureConversationForAssets(draft.value || '图片任务')
  const attachments = await assets.registerPendingAssets(conversationId)
  await workspace.sendTextMessage({
    text: draft.value,
    model,
    intent: workspaceIntent.value,
    attachments
  })
  draft.value = ''
  assets.clearPreviews()
}
```

- [ ] **Step 3: Update `AppWorkspaceView` template**

Replace local composer and message markup with:

```vue
<WorkspaceMessageList
  v-if="workspace.messages.value.length > 0"
  :messages="workspace.messages.value"
  :loading="workspace.loadingMessages.value"
/>

<WorkspaceComposer
  v-model="draft"
  :selected-model="activeChatModel"
  :models="chatModels"
  :intent="workspaceIntent"
  :sending="workspace.sending.value || assets.registering.value"
  :asset-previews="assets.previews.value"
  :rejected-files="assets.rejectedFiles.value"
  @update:selected-model="selectedModelId = $event"
  @files="assets.addFiles"
  @remove-asset="assets.removePreview"
  @submit="submitDraft"
/>
```

Pass history to shell:

```vue
<AppSectionShell
  :title="activeContent.shellTitle"
  :subtitle="activeContent.shellSubtitle"
  :eyebrow="activeContent.eyebrow"
  :icon="activeContent.icon"
  :history-items="workspace.conversations.value"
  :active-conversation-id="workspace.activeConversationId.value"
  :history-loading="workspace.loadingHistory.value"
  @new-chat="startNewChat"
  @select-conversation="workspace.selectConversation"
>
```

- [ ] **Step 4: Update `AppWorkspaceView.spec.ts`**

Replace the current local-only expectations with:

```ts
it('loads history on mount and passes it to the shell', async () => {
  // mock listConversations to return one item
  // mount AppWorkspaceView
  // expect shell stub to render the title
})

it('sends through durable workspace APIs', async () => {
  // type into composer
  // submit
  // expect createConversation and appendMessage called
})

it('keeps image route in the unified workspace', async () => {
  // set route.meta.appSection = 'image'
  // expect image placeholder and same composer test id
})
```

Use exact mocks from `useWorkspaceConversation.spec.ts` so no network is used.

- [ ] **Step 5: Run frontend tests**

Run:

```powershell
cd frontend
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/__tests__/AppWorkspaceView.spec.ts src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts src/views/user/workspace/__tests__/useWorkspaceAssets.spec.ts src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts
```

Expected: PASS.

- [ ] **Step 6: Run typecheck**

Run:

```powershell
cd frontend
node C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/.corepack/v1/pnpm/11.1.3/bin/pnpm.cjs run typecheck
```

Expected: PASS.

- [ ] **Step 7: Commit wiring**

Run:

```powershell
git add frontend/src/views/user/AppWorkspaceView.vue frontend/src/components/user/AppSectionShell.vue frontend/src/views/user/__tests__/AppWorkspaceView.spec.ts
git commit -m "feat(frontend): persist unified workspace conversations"
```

Expected: one commit with route-level wiring and tests.

---

### Task 7: Capability Honesty And Copy Review

**Files:**
- Modify: `frontend/src/views/user/workspace/WorkspaceAssetPanel.vue`
- Modify: `frontend/src/views/user/workspace/WorkspaceComposer.vue`
- Test: `frontend/src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts`

- [ ] **Step 1: Ensure future capabilities are informational, not disabled dead controls**

Use rows with `type="button"` and `aria-disabled="true"` only if the click opens an explanation inside the panel. Preferred V1 copy:

```ts
const futureCapabilities = [
  { label: '文档', description: '即将接入文档分析，当前请先上传图片或输入文字。' },
  { label: '表格', description: '即将接入表格分析，当前不会读取 xlsx/csv 文件。' },
  { label: '代码', description: '即将接入代码文件分析，当前请直接粘贴关键代码。' }
]
```

- [ ] **Step 2: Add tests for the copy**

Extend `WorkspaceComposer.spec.ts`:

```ts
it('does not render disabled future capability buttons', async () => {
  const wrapper = mount(WorkspaceComposer, {
    props: {
      modelValue: '',
      selectedModel: 'gpt-5.5',
      models,
      intent: 'chat',
      sending: false,
      assetPreviews: [],
      rejectedFiles: []
    }
  })

  await wrapper.get('[data-testid="workspace-add-content"]').trigger('click')

  expect(wrapper.findAll('button[disabled]')).toHaveLength(0)
  expect(wrapper.text()).toContain('当前请先上传图片或输入文字')
})
```

- [ ] **Step 3: Run composer tests**

Run:

```powershell
cd frontend
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts
```

Expected: PASS.

- [ ] **Step 4: Commit capability honesty changes**

Run:

```powershell
git add frontend/src/views/user/workspace/WorkspaceAssetPanel.vue frontend/src/views/user/workspace/WorkspaceComposer.vue frontend/src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts
git commit -m "fix(frontend): clarify workspace capability states"
```

Expected: one focused commit.

---

### Task 8: Build And Browser Verification

**Files:**
- No source files unless tests expose defects.

- [ ] **Step 1: Run frontend typecheck and targeted tests**

Run:

```powershell
cd frontend
node C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/.corepack/v1/pnpm/11.1.3/bin/pnpm.cjs run typecheck
node node_modules/.pnpm/vitest@2.1.9_@types+node@20.19.27_jsdom@24.1.3/node_modules/vitest/vitest.mjs run src/views/user/__tests__/AppWorkspaceView.spec.ts src/views/user/workspace/__tests__/useWorkspaceConversation.spec.ts src/views/user/workspace/__tests__/useWorkspaceAssets.spec.ts src/views/user/workspace/__tests__/WorkspaceComposer.spec.ts
```

Expected: all PASS.

- [ ] **Step 2: Run backend tests**

Run:

```powershell
cd backend
go test ./internal/service -run ChatWorkspace -count=1
go test ./internal/server/routes -run ChatWorkspace -count=1
```

Expected: PASS.

- [ ] **Step 3: Build frontend**

Run:

```powershell
cd frontend
node C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/.corepack/v1/pnpm/11.1.3/bin/pnpm.cjs run build
```

Expected: PASS.

- [ ] **Step 4: Build backend with embedded frontend**

Run:

```powershell
cd backend
go build -tags embed -o sub2api-test-build.exe ./cmd/server
```

Expected: PASS and `backend/sub2api-test-build.exe` exists.

- [ ] **Step 5: Run local browser verification**

Start the app using the repository's existing dev or preview workflow. In the browser, verify:

```text
/app loads unified shell.
/app/chat loads the same shell with chat intent.
/app/image loads the same shell with image intent.
Model picker opens and selection updates the label.
Add content opens and image is enabled.
Future capabilities are explanatory, not dead disabled controls.
Sending a text message creates history and survives reload.
New chat clears draft and active message stream.
```

Expected: screenshots captured for `/app`, model menu open, add-content panel open, and restored history.

- [ ] **Step 6: Commit any verification fixes**

If verification found small defects, fix only files already in this plan and commit:

```powershell
git add <exact files fixed>
git commit -m "fix(frontend): stabilize workspace verification flow"
```

Expected: no unrelated files staged.

---

### Task 9: Push Branch And Open PR

**Files:**
- No source files.

- [ ] **Step 1: Confirm staged files are empty**

Run:

```powershell
git -c safe.directory=C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/sub2api -C C:\Users\24091\Documents\Codex\2026-05-21\sub2-1-2-gpt-5-5\sub2api diff --cached --name-only
```

Expected: no output.

- [ ] **Step 2: Confirm PR file scope**

Run:

```powershell
git -c safe.directory=C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/sub2api -C C:\Users\24091\Documents\Codex\2026-05-21\sub2-1-2-gpt-5-5\sub2api diff --name-only origin/main...HEAD
```

Expected: includes only the approved spec/plan plus backend chat-workspace tests and frontend workspace files. If admin, payment, unrelated API key, or unrelated generated ent changes appear in this PR, stop and split the branch.

- [ ] **Step 3: Push branch**

Run:

```powershell
git -c safe.directory=C:/Users/24091/Documents/Codex/2026-05-21/sub2-1-2-gpt-5-5/sub2api -C C:\Users\24091\Documents\Codex\2026-05-21\sub2-1-2-gpt-5-5\sub2api push -u origin codex/unified-chat-workspace-v1
```

Expected: branch pushed.

- [ ] **Step 4: Open PR**

Run:

```powershell
gh pr create --base main --head codex/unified-chat-workspace-v1 --title "feat: persist unified chat workspace" --body "## Summary
- Wires the unified /app workspace to durable chat-workspace state
- Persists conversations, user messages, assistant messages, and image assets
- Clarifies future capability states in the composer
- Adds frontend workspace tests and backend chat-workspace safety tests

## Tests
- frontend typecheck
- workspace Vitest suite
- go test ./internal/service -run ChatWorkspace -count=1
- go test ./internal/server/routes -run ChatWorkspace -count=1
- browser verification for /app, /app/chat, /app/image

## Deployment
- Do not deploy from an old server build directory
- Deploy only from this PR commit or a tagged artifact after review"
```

Expected: PR URL printed.

---

## Self-Review Checklist

- Spec coverage: the plan covers durable conversations, history restore, image asset registration, image intent routing, honest capability states, tests, browser verification, and PR hygiene.
- Scope control: this plan does not implement document parsing, spreadsheet analysis, video analysis, memory, browsing, toolbox execution, or full image generation.
- Risk control: production deploy is not part of this plan. Deployment happens only after PR review and exact commit build.
- Test coverage: frontend and backend tests are introduced before or alongside each behavior.
- Dirty worktree protection: each task commits a focused set of files and checks staged files before PR creation.
