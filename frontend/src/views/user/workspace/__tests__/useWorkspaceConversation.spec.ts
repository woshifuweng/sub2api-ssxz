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
  WORKSPACE_MODEL_UNAVAILABLE_MESSAGE,
  WORKSPACE_PROVIDER_FAILED_MESSAGE,
  WORKSPACE_SEND_FAILED_MESSAGE,
  WORKSPACE_TEXT_ONLY_MESSAGE,
  WORKSPACE_BACKEND_UNAVAILABLE_MESSAGE,
  WORKSPACE_REFRESH_AFTER_SEND_FAILED_MESSAGE,
  useWorkspaceConversation
} from '../useWorkspaceConversation'

const unavailableAssistantContent =
  '当前模型暂不可用，请切换其他模型，或联系管理员检查模型、API Key、分组和上游账号配置。'

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

  it('maps persisted assistant image messages for history restore', async () => {
    api.listMessages.mockResolvedValue([
      {
        id: 22,
        conversation_id: 11,
        message_type: 'image',
        role: 'assistant',
        content: 'Generated image is ready.',
        model: 'gpt-5.5',
        intent: 'image_generation',
        status: 'completed',
        metadata: {
          capability: 'image_generation',
          result_type: 'image',
          status: 'completed',
          enhanced_prompt_present: true,
          assets: [
            {
              id: 'asset-1',
              url: 'https://cdn.example.test/workspace/image-1.png',
              mime_type: 'image/png',
              width: 1024,
              height: 1024,
              provider: 'placeholder-provider',
              model: 'placeholder-model'
            }
          ]
        },
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:00Z'
      }
    ])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.selectConversation(11)

    expect(api.listMessages).toHaveBeenCalledWith(11)
    expect(workspace.messages.value).toMatchObject([
      {
        persistedId: 22,
        conversationId: 11,
        messageType: 'image',
        role: 'assistant',
        content: 'Generated image is ready.',
        attachments: [
          {
            id: 'asset-1',
            name: 'asset-1',
            url: 'https://cdn.example.test/workspace/image-1.png',
            type: 'image'
          }
        ]
      }
    ])
    expect(JSON.stringify(workspace.messages.value)).not.toContain('data:image')
  })

  it('maps failed assistant image messages without treating them as success', async () => {
    api.listMessages.mockResolvedValue([
      {
        id: 23,
        conversation_id: 11,
        message_type: 'image',
        role: 'assistant',
        content: '',
        model: 'gpt-5.5',
        intent: 'image_generation',
        status: 'failed',
        metadata: {
          capability: 'image_generation',
          result_type: 'image',
          status: 'failed',
          error_code: 'provider_unavailable',
          error_message: 'Image generation failed. Please try again.',
          assets: [
            {
              id: 'unsafe',
              url: 'data:image/png;base64,abc',
              mime_type: 'image/png'
            }
          ]
        },
        created_at: '2026-06-10T00:00:00Z',
        updated_at: '2026-06-10T00:00:00Z'
      }
    ])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.selectConversation(11)

    expect(workspace.messages.value).toMatchObject([
      {
        persistedId: 23,
        messageType: 'image',
        state: 'error',
        content: 'Image generation failed. Please try again.'
      }
    ])
    expect(workspace.messages.value[0].attachments).toBeUndefined()
    expect(JSON.stringify(workspace.messages.value)).not.toContain('data:image')
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
    api.listMessages.mockResolvedValue([
      {
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
      },
      {
        id: 42,
        conversation_id: 31,
        message_type: 'text',
        role: 'assistant',
        content: unavailableAssistantContent,
        model: 'gpt-5.5',
        intent: 'chat',
        status: 'completed',
        created_at: '2026-06-10T00:00:02Z',
        updated_at: '2026-06-10T00:00:02Z'
      }
    ])
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
    expect(api.listMessages).toHaveBeenCalledWith(31)
    expect(workspace.activeConversationId.value).toBe(31)
    expect(workspace.messages.value).toMatchObject([
      {
        persistedId: 41,
        role: 'user',
        content: 'hello workspace'
      },
      {
        persistedId: 42,
        role: 'assistant',
        content: unavailableAssistantContent
      }
    ])
    expect(api.createImageTask).not.toHaveBeenCalled()
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it('keeps a safe error state if message refresh fails after send', async () => {
    api.createConversation.mockResolvedValue({
      id: 33,
      title: 'sync fail',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockResolvedValue({
      id: 43,
      conversation_id: 33,
      message_type: 'text',
      role: 'user',
      content: 'sync fail',
      model: 'gpt-5.5',
      intent: 'chat',
      status: 'completed',
      created_at: '2026-06-10T00:00:01Z',
      updated_at: '2026-06-10T00:00:01Z'
    })
    api.listMessages.mockRejectedValue(new Error('network'))
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'sync fail',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(api.appendMessage).toHaveBeenCalled()
    expect(api.listMessages).toHaveBeenCalledWith(33)
    expect(workspace.messages.value).toHaveLength(0)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_REFRESH_AFTER_SEND_FAILED_MESSAGE)
    expect(api.createImageTask).not.toHaveBeenCalled()
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it('shows a model unavailable error when the workspace rejects the selected model', async () => {
    api.createConversation.mockResolvedValue({
      id: 36,
      title: 'use model',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockRejectedValue({
      status: 400,
      code: 'WORKSPACE_MODEL_UNAVAILABLE',
      message: 'Model is not available for workspace chat'
    })
    api.listMessages.mockResolvedValue([])
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'use model',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(api.appendMessage).toHaveBeenCalled()
    expect(api.listMessages).toHaveBeenCalledWith(36)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_MODEL_UNAVAILABLE_MESSAGE)
    expect(workspace.errorMessage.value).not.toBe(WORKSPACE_SEND_FAILED_MESSAGE)
  })

  it('refreshes persisted messages when the AI response fails after the user message is submitted', async () => {
    api.createConversation.mockResolvedValue({
      id: 37,
      title: 'provider fail',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockRejectedValue({
      status: 500,
      code: 'WORKSPACE_SERVICE_UNAVAILABLE',
      message: 'Workspace service unavailable'
    })
    api.listMessages.mockResolvedValue([
      {
        id: 47,
        conversation_id: 37,
        message_type: 'text',
        role: 'user',
        content: 'provider fail',
        model: 'gemini-2.5-pro',
        intent: 'chat',
        status: 'completed',
        created_at: '2026-06-10T00:00:01Z',
        updated_at: '2026-06-10T00:00:01Z'
      }
    ])
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'provider fail',
      model: 'gemini-2.5-pro',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(api.listMessages).toHaveBeenCalledWith(37)
    expect(workspace.messages.value).toMatchObject([
      {
        persistedId: 47,
        role: 'user',
        content: 'provider fail'
      }
    ])
    expect(workspace.errorMessage.value).toBe(WORKSPACE_PROVIDER_FAILED_MESSAGE)
  })

  it('classifies message-only provider placeholder failures as AI response failures', async () => {
    api.createConversation.mockResolvedValue({
      id: 39,
      title: 'provider placeholder',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockRejectedValue({
      status: 500,
      message: 'AI response provider is not connected yet. Text message was saved.'
    })
    api.listMessages.mockResolvedValue([
      {
        id: 49,
        conversation_id: 39,
        message_type: 'text',
        role: 'user',
        content: 'provider placeholder',
        model: 'gpt-5.5',
        intent: 'chat',
        status: 'completed',
        created_at: '2026-06-10T00:00:01Z',
        updated_at: '2026-06-10T00:00:01Z'
      }
    ])
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'provider placeholder',
      model: 'gpt-5.5',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(api.listMessages).toHaveBeenCalledWith(39)
    expect(workspace.messages.value).toMatchObject([
      {
        persistedId: 49,
        role: 'user',
        content: 'provider placeholder'
      }
    ])
    expect(workspace.errorMessage.value).toBe(WORKSPACE_PROVIDER_FAILED_MESSAGE)
  })

  it('classifies message-only model availability failures as model unavailable', async () => {
    api.createConversation.mockResolvedValue({
      id: 40,
      title: 'bad model',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockRejectedValue({
      status: 400,
      message: 'Model is not available for workspace chat'
    })
    api.listMessages.mockResolvedValue([])
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'bad model',
      model: 'image-only-model',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(api.listMessages).toHaveBeenCalledWith(40)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_MODEL_UNAVAILABLE_MESSAGE)
  })

  it('falls back to a generic send error when the failure cannot be classified or refreshed', async () => {
    api.createConversation.mockResolvedValue({
      id: 38,
      title: 'unknown fail',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockRejectedValue({
      status: 500,
      code: 'UNKNOWN',
      message: 'unknown'
    })
    api.listMessages.mockRejectedValue(new Error('refresh failed'))
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    const result = await workspace.sendTextMessage({
      text: 'unknown fail',
      model: 'deepseek-v4-flash',
      intent: 'chat',
      attachments: []
    })

    expect(result).toBe(false)
    expect(api.listMessages).toHaveBeenCalledWith(38)
    expect(workspace.messages.value).toHaveLength(0)
    expect(workspace.errorMessage.value).toBe(WORKSPACE_SEND_FAILED_MESSAGE)
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
    api.listMessages.mockResolvedValue([
      {
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
      },
      {
        id: 43,
        conversation_id: 32,
        message_type: 'text',
        role: 'assistant',
        content: unavailableAssistantContent,
        model: 'gpt-5.5',
        intent: 'chat',
        status: 'completed',
        created_at: '2026-06-10T00:00:02Z',
        updated_at: '2026-06-10T00:00:02Z'
      }
    ])
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
    expect(api.listMessages).toHaveBeenCalledWith(32)
    expect(workspace.messages.value).toHaveLength(2)
  })

  it('passes web_search_requested metadata only when the user explicitly enables联网', async () => {
    api.createConversation.mockResolvedValue({
      id: 34,
      title: 'search the web',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockResolvedValue({
      id: 44,
      conversation_id: 34,
      message_type: 'text',
      role: 'user',
      content: 'today world cup fixtures',
      model: 'deepseek-v4-flash',
      intent: 'chat',
      status: 'completed',
      created_at: '2026-06-10T00:00:01Z',
      updated_at: '2026-06-10T00:00:01Z'
    })
    api.listMessages.mockResolvedValue([])
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.sendTextMessage({
      text: 'today world cup fixtures',
      model: 'deepseek-v4-flash',
      intent: 'chat',
      attachments: [],
      webSearchRequested: true
    })

    expect(api.appendMessage).toHaveBeenCalledWith(34, expect.objectContaining({
      metadata: {
        web_search_requested: true
      }
    }))
  })

  it('keeps normal text sends unchanged when联网 is not requested', async () => {
    api.createConversation.mockResolvedValue({
      id: 35,
      title: 'plain text',
      status: 'active',
      created_at: '2026-06-10T00:00:00Z',
      updated_at: '2026-06-10T00:00:00Z'
    })
    api.appendMessage.mockResolvedValue({
      id: 45,
      conversation_id: 35,
      message_type: 'text',
      role: 'user',
      content: 'plain text',
      model: 'deepseek-v4-flash',
      intent: 'chat',
      status: 'completed',
      created_at: '2026-06-10T00:00:01Z',
      updated_at: '2026-06-10T00:00:01Z'
    })
    api.listMessages.mockResolvedValue([])
    api.listConversations.mockResolvedValue([])
    const workspace = useWorkspaceConversation({ backendEnabled: true })

    await workspace.sendTextMessage({
      text: 'plain text',
      model: 'deepseek-v4-flash',
      intent: 'chat',
      attachments: []
    })

    expect(api.appendMessage).toHaveBeenCalledWith(35, expect.objectContaining({
      intent: 'chat',
      metadata: undefined
    }))
  })

  it('appends to the active conversation without creating another one', async () => {
    api.listMessages
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([
        {
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
        },
        {
          id: 52,
          conversation_id: 44,
          message_type: 'text',
          role: 'assistant',
          content: unavailableAssistantContent,
          model: 'gpt-5.5',
          intent: 'chat',
          status: 'completed',
          created_at: '2026-06-10T00:00:03Z',
          updated_at: '2026-06-10T00:00:03Z'
        }
      ])
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
    expect(api.listMessages).toHaveBeenCalledTimes(2)
    expect(api.listMessages).toHaveBeenLastCalledWith(44)
    expect(workspace.messages.value).toHaveLength(2)
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
