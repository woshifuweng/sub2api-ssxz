import { afterEach, describe, expect, it, vi } from 'vitest'

async function importChatWorkspaceWithGate(value?: string) {
  vi.resetModules()
  vi.stubEnv('VITE_CHAT_WORKSPACE_BACKEND_ENABLED', value ?? '')
  return import('../chatWorkspace')
}

describe('chatWorkspace API gate', () => {
  afterEach(() => {
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
})
