import { beforeEach, describe, expect, it, vi } from 'vitest'

const putMock = vi.fn()
const postMock = vi.fn()
const deleteMock = vi.fn()

vi.mock('../client', () => ({
  apiClient: {
    put: putMock,
    post: postMock,
    delete: deleteMock
  }
}))

describe('keys API', () => {
  beforeEach(() => {
    putMock.mockReset()
    postMock.mockReset()
    deleteMock.mockReset()
  })

  it('includes allowed_models in create payload', async () => {
    postMock.mockResolvedValue({ data: { id: 1 } })
    const mod = await import('../keys')

    await mod.create(
      'test-key',
      1,
      [1],
      ['gpt-5.4', 'claude-sonnet-4-5'],
      'sk-custom',
      ['127.0.0.1'],
      [],
      10,
      30,
      { rate_limit_1d: 5 }
    )

    expect(postMock).toHaveBeenCalledWith('/keys', expect.objectContaining({
      name: 'test-key',
      group_id: 1,
      group_ids: [1],
      allowed_models: ['gpt-5.4', 'claude-sonnet-4-5'],
      custom_key: 'sk-custom'
    }))
  })

  it('sends update payloads to the key detail endpoint', async () => {
    putMock.mockResolvedValue({ data: { id: 42, status: 'inactive' } })
    const mod = await import('../keys')

    const result = await mod.update(42, {
      status: 'inactive',
      reset_quota: true,
      reset_rate_limit_usage: true
    })

    expect(putMock).toHaveBeenCalledWith('/keys/42', {
      status: 'inactive',
      reset_quota: true,
      reset_rate_limit_usage: true
    })
    expect(result).toEqual({ id: 42, status: 'inactive' })
  })

  it('deletes a key by id', async () => {
    deleteMock.mockResolvedValue({ data: { message: 'deleted' } })
    const mod = await import('../keys')

    const result = await mod.deleteKey(42)

    expect(deleteMock).toHaveBeenCalledWith('/keys/42')
    expect(result).toEqual({ message: 'deleted' })
  })

  it('toggles status through the update endpoint', async () => {
    putMock.mockResolvedValue({ data: { id: 42, status: 'active' } })
    const mod = await import('../keys')

    await mod.toggleStatus(42, 'active')

    expect(putMock).toHaveBeenCalledWith('/keys/42', { status: 'active' })
  })
})
