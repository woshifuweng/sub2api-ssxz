import { computed, onMounted, type ComputedRef } from 'vue'
import { getActivePinia } from 'pinia'
import { useResellerStore } from '@/stores/reseller'
import { useAuthStore } from '@/stores/auth'

/**
 * Provide read-only reseller navigation state without breaking isolated layout
 * tests that intentionally mount the sidebar without installing Pinia.
 */
export function useResellerNavRole(): {
  isAgent: ComputedRef<boolean>
  isManager: ComputedRef<boolean>
} {
  if (!getActivePinia()) {
    return {
      isAgent: computed(() => false),
      isManager: computed(() => false)
    }
  }

  const resellerStore = useResellerStore()
  const authStore = useAuthStore()

  onMounted(() => {
    const userId = authStore.user?.id
    if (!userId) return
    void Promise.resolve(resellerStore.fetchRole(userId)).catch(() => null)
  })

  return {
    isAgent: computed(() => resellerStore.isAgent),
    isManager: computed(() => resellerStore.isManager)
  }
}
