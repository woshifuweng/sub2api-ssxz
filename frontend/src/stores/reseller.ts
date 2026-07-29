import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import resellerAPI, { type ResellerRole } from '@/api/reseller'

export const useResellerStore = defineStore('reseller', () => {
  const role = ref<ResellerRole | null>(null)
  const loadedForUserId = ref<number | null>(null)
  let pending: Promise<ResellerRole | null> | null = null
  let pendingUserId: number | null = null
  let requestVersion = 0

  const isAgent = computed(() => role.value === 'agent' || role.value === 'agent_manager')
  const isManager = computed(() => role.value === 'agent_manager')

  async function fetchRole(userId: number, force = false): Promise<ResellerRole | null> {
    if (!force && loadedForUserId.value === userId) return role.value
    if (!force && pending && pendingUserId === userId) return pending

    if (loadedForUserId.value !== userId) {
      role.value = null
      loadedForUserId.value = null
    }

    const version = ++requestVersion
    pendingUserId = userId
    const request = resellerAPI.getRole()
      .then((result) => {
        const nextRole = result.role === 'agent' || result.role === 'agent_manager' ? result.role : null
        if (version === requestVersion) {
          role.value = nextRole
          loadedForUserId.value = userId
        }
        return nextRole
      })
      .finally(() => {
        if (version === requestVersion) {
          pending = null
          pendingUserId = null
        }
      })

    pending = request
    return request
  }

  function reset(): void {
    role.value = null
    loadedForUserId.value = null
    pending = null
    pendingUserId = null
    requestVersion++
  }

  return {
    role,
    loadedForUserId,
    isAgent,
    isManager,
    fetchRole,
    reset
  }
})
