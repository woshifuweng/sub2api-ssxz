import { beforeEach, describe, expect, it, vi } from 'vitest'

const get = vi.fn()
const post = vi.fn()
const patch = vi.fn()
const del = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: { get, post, patch, delete: del }
}))

describe('reseller api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 } })
    post.mockResolvedValue({ data: {} })
    patch.mockResolvedValue({ data: {} })
    del.mockResolvedValue({ data: {} })
  })

  it('uses the scoped agent and manager endpoints', async () => {
    const api = (await import('../reseller')).default

    await api.listRecruits(2, 10)
    await api.requestBalanceConversion(10)
    await api.cancelWithdrawal(12)
    await api.listManagedAgents(1, 20, 'user')
    await api.setManagedAgentRole(8, 'agent', 'regional agent')
    await api.setManagedAgentRole(8, null)
    await api.listManagedWithdrawals(1, 20, 'pending')
    await api.listAdminAgents(2, 50, 'agent@example.com')
    await api.getAdminAgent(8)
    await api.updateAdminAgent(8, {
      role: 'agent',
      manager_id: null,
      rebate_policy: { mode: 'custom', rate_percent: 5 }
    })
    await api.disableAdminAgent(8, '合作暂停')
    await api.enableAdminAgent(8)
    await api.grantAdminRole(8, { role: 'agent_manager', notes: 'regional lead' })
    await api.revokeAdminRole(8)
    await api.listAdminWithdrawals(3, 10, 'pending')
    await api.reviewWithdrawal(12, { action: 'reject', reason: 'invalid request' })

    expect(get).toHaveBeenNthCalledWith(1, '/user/reseller/recruits', { params: { page: 2, page_size: 10 } })
    expect(post).toHaveBeenNthCalledWith(1, '/user/reseller/withdraw', { amount: 10 })
    expect(post).toHaveBeenNthCalledWith(2, '/user/reseller/withdrawals/12/cancel')
    expect(get).toHaveBeenNthCalledWith(2, '/user/reseller/manager/agents', {
      params: { page: 1, page_size: 20, search: 'user' }
    })
    expect(post).toHaveBeenNthCalledWith(3, '/user/reseller/manager/agents/8/grant', { notes: 'regional agent' })
    expect(del).toHaveBeenCalledWith('/user/reseller/manager/agents/8/role')
    expect(get).toHaveBeenNthCalledWith(3, '/user/reseller/manager/withdrawals', {
      params: { page: 1, page_size: 20, status: 'pending' }
    })
    expect(get).toHaveBeenNthCalledWith(4, '/admin/reseller/agents', {
      params: {
        page: 2,
        page_size: 50,
        search: 'agent@example.com',
        status: undefined,
        role: undefined,
        manager_id: undefined
      }
    })
    expect(get).toHaveBeenNthCalledWith(5, '/admin/reseller/agents/8')
    expect(patch).toHaveBeenCalledWith('/admin/reseller/agents/8', {
      role: 'agent',
      manager_id: null,
      rebate_policy: { mode: 'custom', rate_percent: 5 }
    })
    expect(post).toHaveBeenNthCalledWith(4, '/admin/reseller/agents/8/disable', {
      reason: '合作暂停'
    })
    expect(post).toHaveBeenNthCalledWith(5, '/admin/reseller/agents/8/enable')
    expect(post).toHaveBeenNthCalledWith(6, '/admin/reseller/agents/8/role', {
      role: 'agent_manager',
      notes: 'regional lead'
    })
    expect(del).toHaveBeenCalledWith('/admin/reseller/agents/8/role')
    expect(get).toHaveBeenNthCalledWith(6, '/admin/reseller/withdrawals', {
      params: { page: 3, page_size: 10, status: 'pending' }
    })
    expect(post).toHaveBeenNthCalledWith(7, '/admin/reseller/withdrawals/12/review', {
      action: 'reject',
      reason: 'invalid request'
    })
  })
})
