import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import resellerAPI, { type AgentDashboard, type ResellerRole } from '@/api/reseller'

export const useResellerStore = defineStore('reseller', () => {
  const role = ref<ResellerRole | null>(null)
  const dashboard = ref<AgentDashboard | null>(null)
  const loading = ref(false)
  const loadedForUserId = ref<number | null>(null)
  let pending: Promise<ResellerRole | null> | null = null
  let pendingUserId: number | null = null
  let requestVersion = 0
  let loadingRequests = 0

  const isAgent = computed(() => role.value === 'agent' || role.value === 'agent_manager')
  const isManager = computed(() => role.value === 'agent_manager')

  function beginLoading(): void {
    loadingRequests += 1
    loading.value = true
  }

  function endLoading(): void {
    loadingRequests = Math.max(0, loadingRequests - 1)
    loading.value = loadingRequests > 0
  }

  async function fetchRole(userId: number, force = false): Promise<ResellerRole | null> {
    if (!force && loadedForUserId.value === userId) return role.value
    if (!force && pending && pendingUserId === userId) return pending

    if (loadedForUserId.value !== userId) {
      role.value = null
      loadedForUserId.value = null
    }

    const version = ++requestVersion
    pendingUserId = userId
    beginLoading()
    const request = resellerAPI.getRole()
      .then((result) => {
        const nextRole = result.role === 'agent' || result.role === 'agent_manager' ? result.role : null
        if (version === requestVersion) {
          role.value = nextRole
          loadedForUserId.value = userId
        }
        return nextRole
      })
      .catch(() => {
        if (version === requestVersion) {
          role.value = null
          dashboard.value = null
          loadedForUserId.value = userId
        }
        return null
      })
      .finally(() => {
        if (version === requestVersion) {
          pending = null
          pendingUserId = null
        }
        endLoading()
      })

    pending = request
    return request
  }

  async function fetchDashboard(): Promise<AgentDashboard | null> {
    if (!isAgent.value) {
      dashboard.value = null
      return null
    }

    beginLoading()
    try {
      dashboard.value = await resellerAPI.getAgentDashboard()
      return dashboard.value
    } finally {
      endLoading()
    }
  }

  function reset(): void {
    role.value = null
    dashboard.value = null
    loading.value = false
    loadingRequests = 0
    loadedForUserId.value = null
    pending = null
    pendingUserId = null
    requestVersion++
  }

  return {
    role,
    dashboard,
    loading,
    loadedForUserId,
    isAgent,
    isManager,
    fetchRole,
    fetchDashboard,
    reset
  }
})
