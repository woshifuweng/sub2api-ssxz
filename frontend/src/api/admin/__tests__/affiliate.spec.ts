import { beforeEach, describe, expect, it, vi } from 'vitest'

const get = vi.fn()
const put = vi.fn()
const post = vi.fn()
const del = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    put,
    post,
    delete: del
  }
}))

describe('admin affiliate api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('uses the existing admin affiliate endpoints', async () => {
    get.mockResolvedValueOnce({
      data: {
        items: [],
        total: 0,
        page: 1,
        page_size: 20,
        pages: 1
      }
    })
    get.mockResolvedValueOnce({ data: [{ id: 7, email: 'user@example.com', username: 'user' }] })
    put.mockResolvedValue({ data: { user_id: 7 } })
    del.mockResolvedValue({ data: { user_id: 7 } })
    post.mockResolvedValue({ data: { affected: 1 } })

    const mod = await import('../affiliate')

    await mod.listUsers(1, 20, 'user@example.com')
    await mod.lookupUsers('user@example.com')
    await mod.updateUserSettings(7, {
      aff_code: 'SSXZ7',
      aff_rebate_rate_percent: 12
    })
    await mod.clearUserSettings(7)
    await mod.batchSetRate([7], 12)

    expect(get).toHaveBeenNthCalledWith(1, '/admin/affiliates/users', {
      params: { page: 1, page_size: 20, search: 'user@example.com' }
    })
    expect(get).toHaveBeenNthCalledWith(2, '/admin/affiliates/users/lookup', {
      params: { q: 'user@example.com' }
    })
    expect(put).toHaveBeenCalledWith('/admin/affiliates/users/7', {
      aff_code: 'SSXZ7',
      aff_rebate_rate_percent: 12
    })
    expect(del).toHaveBeenCalledWith('/admin/affiliates/users/7')
    expect(post).toHaveBeenCalledWith('/admin/affiliates/users/batch-rate', {
      user_ids: [7],
      aff_rebate_rate_percent: 12,
      clear: false
    })
  })

  it('does not call lookup endpoint for blank search', async () => {
    const mod = await import('../affiliate')

    await expect(mod.lookupUsers('   ')).resolves.toEqual([])
    expect(get).not.toHaveBeenCalled()
  })
})
