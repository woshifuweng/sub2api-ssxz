import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appendMessage, createConversation, listConversations, listMessages } = vi.hoisted(() => ({
  appendMessage: vi.fn(),
  createConversation: vi.fn(),
  listConversations: vi.fn(),
  listMessages: vi.fn()
}))

vi.mock('@/api/chatWorkspace', () => ({
  CHAT_MESSAGE_TYPE_IMAGE: 'image',
  CHAT_MESSAGE_TYPE_TEXT: 'text',
  appendMessage,
  chatWorkspaceBackendEnabled: true,
  createConversation,
  listConversations,
  listMessages
}))

import { useWorkspaceConversation } from '../useWorkspaceConversation'

describe('useWorkspaceConversation model payload', () => {
  beforeEach(() => {
    appendMessage.mockReset()
    createConversation.mockResolvedValue({ id: 7, title: '测试', status: 'active' })
    listConversations.mockResolvedValue([])
    listMessages.mockResolvedValue([])
  })

  it.each(['gpt-6-sol', 'gpt-6-luna', 'claude-opus-5-5', 'grok-4.7'])(
    'keeps the selected %s ID in the session request payload',
    async (model) => {
      const workspace = useWorkspaceConversation({ backendEnabled: true })

      await workspace.sendTextMessage({
        text: '测试模型',
        model,
        intent: 'chat',
        attachments: []
      })

      expect(appendMessage).toHaveBeenCalledWith(
        7,
        expect.objectContaining({ model }),
        expect.objectContaining({ idempotencyKey: expect.any(String) })
      )
    }
  )
})
