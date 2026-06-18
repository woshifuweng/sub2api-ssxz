import { describe, expect, it, vi } from 'vitest'
import type { RouteLocationGeneric } from 'vue-router'

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

import router from '../index'

describe('legacy user routes', () => {
  it.each([
    ['/keys', '/app/keys'],
    ['/usage', '/app/usage'],
    ['/profile', '/app/profile'],
    ['/purchase', '/app/purchase'],
    ['/orders', '/app/orders'],
  ])('redirects %s into the workbench shell at %s', (sourcePath, targetPath) => {
    const route = router.getRoutes().find((record) => record.path === sourcePath)
    expect(route?.redirect).toBeTypeOf('function')

    const redirected = (route?.redirect as (to: RouteLocationGeneric) => unknown)({
      query: { from: 'legacy' },
      hash: '#section',
    } as RouteLocationGeneric)

    expect(redirected).toEqual({
      path: targetPath,
      query: { from: 'legacy' },
      hash: '#section',
    })
  })
})
