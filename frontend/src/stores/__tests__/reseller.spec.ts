import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getRole } = vi.hoisted(() => ({ getRole: vi.fn() }))

vi.mock('@/api/reseller', () => ({
  default: {
    getRole,
    getAgentDashboard: vi.fn()
  }
}))

import { useResellerStore } from '@/stores/reseller'

describe('useResellerStore role visibility', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getRole.mockReset()
  })

  it('shows agent navigation to agents and managers, but manager navigation only to managers', async () => {
    getRole.mockResolvedValueOnce({ role: 'agent' })
    const store = useResellerStore()

    await store.fetchRole(10)
    expect(store.isAgent).toBe(true)
    expect(store.isManager).toBe(false)

    getRole.mockResolvedValueOnce({ role: 'agent_manager' })
    await store.fetchRole(11)
    expect(store.isAgent).toBe(true)
    expect(store.isManager).toBe(true)
  })

  it('deduplicates concurrent role requests for the same signed-in user', async () => {
    let resolveRole!: (value: { role: 'agent' }) => void
    getRole.mockReturnValueOnce(new Promise((resolve) => { resolveRole = resolve }))
    const store = useResellerStore()

    const first = store.fetchRole(10)
    const second = store.fetchRole(10)
    resolveRole({ role: 'agent' })

    await Promise.all([first, second])
    expect(getRole).toHaveBeenCalledTimes(1)
  })

  it('fails closed when the role endpoint is unavailable', async () => {
    getRole.mockRejectedValueOnce(new Error('unavailable'))
    const store = useResellerStore()

    await expect(store.fetchRole(10)).resolves.toBeNull()
    expect(store.isAgent).toBe(false)
    expect(store.isManager).toBe(false)
  })
})
