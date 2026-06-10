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
    chatWorkspaceBackendEnabled: false,
    listConversations: api.listConversations,
    createConversation: api.createConversation,
    listMessages: api.listMessages,
    appendMessage: api.appendMessage,
    createImageTask: api.createImageTask
  }
})

vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn()
  }
}))

import { apiClient } from '@/api/client'
import {
  WORKSPACE_TEXT_ONLY_MESSAGE,
  WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE,
  useWorkspaceConversation
} from '../useWorkspaceConversation'

describe('useWorkspaceConversation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not request chat-workspace history while the backend is disabled', async () => {
    const workspace = useWorkspaceConversation()

    await workspace.loadHistory()

    expect(workspace.conversations.value).toEqual([])
    expect(api.listConversations).not.toHaveBeenCalled()
  })

  it('does not load selected messages while the backend is disabled', async () => {
    const workspace = useWorkspaceConversation()

    await workspace.selectConversation(5)

    expect(api.listMessages).not.toHaveBeenCalled()
    expect(workspace.activeConversationId.value).toBeNull()
    expect(workspace.errorMessage.value).toBe(WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE)
  })

  it('blocks send without calling chat-workspace or chat-studio APIs', async () => {
    const workspace = useWorkspaceConversation()

    const result = await workspace.sendTextMessage({
      text: 'hello',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(workspace.messages.value).toHaveLength(0)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE)
    expect(api.createConversation).not.toHaveBeenCalled()
    expect(api.appendMessage).not.toHaveBeenCalled()
    expect(api.createImageTask).not.toHaveBeenCalled()
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it('does not create image tasks for image intent while disabled', async () => {
    const workspace = useWorkspaceConversation()

    await workspace.sendTextMessage({
      text: 'make an image',
      model: 'gpt-5.5',
      intent: 'image',
      attachments: []
    })

    expect(api.createImageTask).not.toHaveBeenCalled()
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it('clears local shell state for a new chat without backend calls', async () => {
    const workspace = useWorkspaceConversation()

    await workspace.selectConversation(5)
    await workspace.startNewChat()

    expect(workspace.messages.value).toEqual([])
    expect(workspace.errorMessage.value).toBe('')
    expect(api.listMessages).not.toHaveBeenCalled()
  })

  it('loads chat-workspace history when the backend gate is enabled', async () => {
    api.listConversations.mockResolvedValue([
      {
        id: 11,
        title: 'Saved chat',
        status: 'active',
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:00Z'
      }
    ])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.loadHistory()

    expect(api.listConversations).toHaveBeenCalledTimes(1)
    expect(workspace.conversations.value).toHaveLength(1)
    expect(workspace.conversations.value[0].id).toBe(11)
  })

  it('loads persisted messages for the selected conversation when enabled', async () => {
    api.listMessages.mockResolvedValue([
      {
        id: 21,
        conversation_id: 11,
        message_type: 'text',
        role: 'user',
        content: 'hello',
        model: 'gpt-5.5',
        intent: 'chat',
        status: 'completed',
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:00Z'
      }
    ])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.selectConversation(11)

    expect(api.listMessages).toHaveBeenCalledWith(11)
    expect(workspace.activeConversationId.value).toBe(11)
    expect(workspace.messages.value).toMatchObject([
      {
        persistedId: 21,
        conversationId: 11,
        messageType: 'text',
        role: 'user',
        content: 'hello'
      }
    ])
  })

  it('creates a conversation and appends a text chat message when enabled', async () => {
    api.createConversation.mockResolvedValue({
      id: 31,
      title: 'hello workspace',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockResolvedValue({
      id: 41,
      conversation_id: 31,
      message_type: 'text',
      role: 'user',
      content: 'hello workspace',
      model: 'gpt-5.5',
      intent: 'chat',
      status: 'completed',
      created_at: '2026-06-10T00:00:01Z',
      updated_at: '2026-06-10T00:00:01Z'
    })
    api.listConversations.mockResolvedValue([
      {
        id: 31,
        title: 'hello workspace',
        status: 'active',
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:01Z'
      }
    ])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: ' hello workspace ',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(true)
    expect(api.createConversation).toHaveBeenCalledWith({ title: 'hello workspace' })
    expect(api.appendMessage).toHaveBeenCalledWith(31, {
      message_type: 'text',
      role: 'user',
      content: 'hello workspace',
      model: 'gpt-5.5',
      intent: 'chat'
    })
    expect(workspace.activeConversationId.value).toBe(31)
    expect(workspace.messages.value).toHaveLength(1)
    expect(workspace.messages.value[0].content).toBe('hello workspace')
    expect(api.createImageTask).not.toHaveBeenCalled()
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it('treats the /app home intent as text chat when enabled', async () => {
    api.createConversation.mockResolvedValue({
      id: 32,
      title: 'start from app',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockResolvedValue({
      id: 42,
      conversation_id: 32,
      message_type: 'text',
      role: 'user',
      content: 'start from app',
      model: 'gpt-5.5',
      intent: 'chat',
      status: 'completed',
      created_at: '2026-06-10T00:00:01Z',
      updated_at: '2026-06-10T00:00:01Z'
    })
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'start from app',
      model: 'gpt-5.5',
      intent: 'home',
      attachments: []
    })

    expect(result).toBe(true)
    expect(api.appendMessage).toHaveBeenCalledWith(32, expect.objectContaining({
      content: 'start from app',
      intent: 'chat'
    }))
  })

  it('appends to the active conversation without creating another one', async () => {
    api.listMessages.mockResolvedValue([])
    api.appendMessage.mockResolvedValue({
      id: 51,
      conversation_id: 44,
      message_type: 'text',
      role: 'user',
      content: 'follow up',
      model: 'gpt-5.5',
      intent: 'chat',
      status: 'completed',
      created_at: '2026-06-10T00:00:02Z',
      updated_at: '2026-06-10T00:00:02Z'
    })
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.selectConversation(44)
    const result = await workspace.sendTextMessage({
      text: 'follow up',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(true)
    expect(api.createConversation).not.toHaveBeenCalled()
    expect(api.appendMessage).toHaveBeenCalledWith(44, expect.objectContaining({
      content: 'follow up',
      intent: 'chat'
    }))
  })

  it('does not call backend APIs for image intent even when enabled', async () => {
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'make an image',
      model: 'gpt-5.5',
      intent: 'image',
      attachments: []
    })

    expect(result).toBe(false)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_TEXT_ONLY_MESSAGE)
    expect(api.createConversation).not.toHaveBeenCalled()
    expect(api.appendMessage).not.toHaveBeenCalled()
    expect(api.createImageTask).not.toHaveBeenCalled()
  })

  it('does not send local object URL attachments as message payloads', async () => {
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'describe this',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: [
        {
          id: 'local-1',
          name: 'sample.png',
          url: 'blob:workspace-preview-sample.png',
          type: 'image'
        }
      ]
    })

    expect(result).toBe(false)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_TEXT_ONLY_MESSAGE)
    expect(api.createConversation).not.toHaveBeenCalled()
    expect(api.appendMessage).not.toHaveBeenCalled()
  })
})
