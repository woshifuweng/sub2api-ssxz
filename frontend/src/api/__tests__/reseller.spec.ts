import { beforeEach, describe, expect, it, vi } from 'vitest'

const get = vi.fn()
const post = vi.fn()
const del = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: { get, post, delete: del }
}))

describe('reseller api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 } })
    post.mockResolvedValue({ data: {} })
    del.mockResolvedValue({ data: {} })
  })

  it('uses the scoped agent and manager endpoints', async () => {
    const api = (await import('../reseller')).default

    await api.listRecruits(2, 10)
    await api.requestWithdraw({ amount: 10, method: 'alipay', account_info: { account: 'user@example.com' } })
    await api.listManagedAgents(1, 20, 'user')
    await api.grantManagedAgent(8, 'regional agent')
    await api.revokeManagedAgent(8)
    await api.listManagedWithdrawals(1, 20, 'pending')

    expect(get).toHaveBeenNthCalledWith(1, '/user/reseller/recruits', { params: { page: 2, page_size: 10 } })
    expect(post).toHaveBeenNthCalledWith(1, '/user/reseller/withdraw', {
      amount: 10,
      method: 'alipay',
      account_info: { account: 'user@example.com' }
    })
    expect(get).toHaveBeenNthCalledWith(2, '/user/reseller/manager/agents', {
      params: { page: 1, page_size: 20, search: 'user' }
    })
    expect(post).toHaveBeenNthCalledWith(2, '/user/reseller/manager/agents/8/grant', { notes: 'regional agent' })
    expect(del).toHaveBeenCalledWith('/user/reseller/manager/agents/8/role')
    expect(get).toHaveBeenNthCalledWith(3, '/user/reseller/manager/withdrawals', {
      params: { page: 1, page_size: 20, status: 'pending' }
    })
  })
})
