import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useResellerStore } from '@/stores/reseller'

const { getRole, getAgentDashboard } = vi.hoisted(() => ({
  getRole: vi.fn(),
  getAgentDashboard: vi.fn()
}))

vi.mock('@/api/reseller', () => ({
  default: { getRole, getAgentDashboard }
}))

describe('useResellerStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('caches the role for the same authenticated user', async () => {
    getRole.mockResolvedValue({ role: 'agent' })
    const store = useResellerStore()

    await store.fetchRole(7)
    await store.fetchRole(7)

    expect(getRole).toHaveBeenCalledTimes(1)
    expect(store.isAgent).toBe(true)
    expect(store.isManager).toBe(false)
  })

  it('invalidates the cached role when the authenticated user changes', async () => {
    getRole
      .mockResolvedValueOnce({ role: 'agent_manager' })
      .mockResolvedValueOnce({ role: null })
    const store = useResellerStore()

    await store.fetchRole(7)
    expect(store.isManager).toBe(true)

    await store.fetchRole(8)
    expect(getRole).toHaveBeenCalledTimes(2)
    expect(store.role).toBeNull()
    expect(store.loadedForUserId).toBe(8)
  })

  it('deduplicates concurrent role requests for one user', async () => {
    let resolveRequest: ((value: { role: 'agent' }) => void) | undefined
    getRole.mockImplementation(() => new Promise((resolve) => { resolveRequest = resolve }))
    const store = useResellerStore()

    const first = store.fetchRole(7)
    const second = store.fetchRole(7)
    expect(getRole).toHaveBeenCalledTimes(1)

    resolveRequest?.({ role: 'agent' })
    await Promise.all([first, second])
    expect(store.role).toBe('agent')
  })

  it('fails closed when the role endpoint is unavailable', async () => {
    getRole.mockRejectedValue(new Error('network error'))
    const store = useResellerStore()

    await expect(store.fetchRole(7)).resolves.toBeNull()

    expect(store.role).toBeNull()
    expect(store.isAgent).toBe(false)
    expect(store.loading).toBe(false)
  })

  it('loads dashboard data only for reseller roles', async () => {
    const dashboard = {
      user_id: 7,
      aff_code: 'CODE7',
      aff_quota: 20,
      aff_frozen_quota: 3,
      aff_history_quota: 48,
      recruit_count: 5,
      rebate_rate: 0.05,
      pending_withdraw: 4,
      commission_earned: 52
    }
    getRole.mockResolvedValue({ role: 'agent' })
    getAgentDashboard.mockResolvedValue(dashboard)
    const store = useResellerStore()

    await store.fetchRole(7)
    await store.fetchDashboard()

    expect(store.dashboard).toEqual(dashboard)
    expect(store.loading).toBe(false)
  })
})
