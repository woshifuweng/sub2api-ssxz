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
})
