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

import { apiClient } from '@/api/client'
import { useWorkspaceConversation } from '../useWorkspaceConversation'

describe('useWorkspaceConversation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(apiClient.post).mockResolvedValue({
      data: {
        choices: [{ message: { content: 'assistant answer' } }]
      }
    })
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
    api.listConversations.mockResolvedValue([
      { id: 1, title: 'Saved', status: 'active', created_at: 'a', updated_at: 'b' }
    ])
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
    expect(workspace.messages.value.map((message) => message.role)).toEqual(['user', 'assistant'])
    expect(workspace.messages.value[1].content).toBe('assistant answer')
  })

  it('keeps the user message and records an error card when completion fails', async () => {
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
      {
        id: 1,
        conversation_id: 5,
        message_type: 'text',
        role: 'user',
        content: 'saved',
        created_at: 'a',
        updated_at: 'b'
      }
    ])
    const workspace = useWorkspaceConversation()

    await workspace.selectConversation(5)

    expect(workspace.activeConversationId.value).toBe(5)
    expect(workspace.messages.value[0].content).toBe('saved')
  })
})
