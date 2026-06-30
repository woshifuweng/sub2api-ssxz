import { afterEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.hoisted(() => vi.fn())

vi.mock('../client', () => ({
  apiClient: {
    get: vi.fn(),
    post: postMock
  }
}))

async function importChatWorkspaceWithGate(value?: string) {
  vi.resetModules()
  vi.stubEnv('VITE_CHAT_WORKSPACE_BACKEND_ENABLED', value ?? '')
  return import('../chatWorkspace')
}

describe('chatWorkspace API gate', () => {
  afterEach(() => {
    postMock.mockReset()
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('fails closed when the build-time backend gate is not enabled', async () => {
    const workspace = await importChatWorkspaceWithGate()

    expect(workspace.chatWorkspaceBackendEnabled).toBe(false)
    await expect(workspace.listConversations()).rejects.toThrow(
      'listConversations is unavailable until the chat workspace backend is enabled'
    )
  })

  it('enables workspace API wrappers only when the build-time gate is true', async () => {
    const workspace = await importChatWorkspaceWithGate('true')

    expect(workspace.chatWorkspaceBackendEnabled).toBe(true)
  })

  it('sends append message idempotency keys as headers', async () => {
    postMock.mockResolvedValue({
      data: {
        id: 1,
        conversation_id: 9,
        message_type: 'text',
        role: 'user',
        content: 'hello',
        model: 'gpt-5.5',
        intent: 'chat',
        status: 'completed',
        created_at: '2026-06-30T00:00:00Z',
        updated_at: '2026-06-30T00:00:00Z'
      }
    })
    const workspace = await importChatWorkspaceWithGate('true')
    const payload = {
      message_type: 'text',
      role: 'user',
      content: 'hello',
      model: 'gpt-5.5',
      intent: 'chat'
    }

    await workspace.appendMessage(9, payload, { idempotencyKey: 'chat-request-1' })

    expect(postMock).toHaveBeenCalledWith(
      '/chat-workspace/conversations/9/messages',
      payload,
      {
        headers: {
          'Idempotency-Key': 'chat-request-1'
        }
      }
    )
  })
})
