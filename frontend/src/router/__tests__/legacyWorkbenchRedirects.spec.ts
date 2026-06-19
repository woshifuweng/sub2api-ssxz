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
    ['/app/usage', 'AppUsage', 'usage'],
    ['/app/purchase', 'AppPurchase', 'purchase'],
    ['/app/orders', 'AppOrders', 'orders'],
    ['/app/redeem', 'AppRedeem', 'redeem'],
    ['/app/affiliate', 'AppAffiliate', 'affiliate'],
    ['/app/available-channels', 'AppAvailableChannels', 'available-channels'],
    ['/app/channel-status', 'AppChannelStatus', 'channel-status'],
    ['/app/keys', 'AppKeys', 'keys'],
    ['/app/profile', 'AppProfile', 'profile'],
  ])('keeps %s owned by the user workbench route %s', (path, name, appSection) => {
    const route = router.getRoutes().find((record) => record.path === path)

    expect(route).toBeDefined()
    expect(route?.name).toBe(name)
    expect(route?.redirect).toBeUndefined()
    expect(route?.meta.requiresAuth).toBe(true)
    expect(route?.meta.requiresAdmin).toBe(false)
    expect(route?.meta.appSection).toBe(appSection)
  })

  it.each([
    ['/app/chat', 'AI Chat'],
    ['/app/image', 'Image Generation'],
    ['/app/usage', 'Usage Information'],
    ['/app/purchase', 'Recharge / Subscription'],
    ['/app/orders', 'Order Records'],
    ['/app/redeem', 'Redeem Code'],
    ['/app/affiliate', 'Affiliate'],
    ['/app/available-channels', 'Available Channels'],
    ['/app/channel-status', 'Channel Status'],
    ['/app/keys', 'API Key / Third-party Access'],
    ['/app/profile', 'Account Settings'],
  ])('uses SSXZ AI as the user workbench document title site name for %s', (path, title) => {
    const route = router.resolve(path)

    expect(route.meta.title).toBe(title)
    expect(route.meta.titleSiteName).toBe('SSXZ AI')
  })

  it.each([
    ['/app', '/app/image'],
    ['/dashboard', '/app/image'],
    ['/ai-chat', '/app/chat'],
    ['/image-studio', '/app/image'],
    ['/keys', '/app/keys'],
    ['/usage', '/app/usage'],
    ['/profile', '/app/profile'],
    ['/purchase', '/app/purchase'],
    ['/orders', '/app/orders'],
    ['/subscriptions', '/app/purchase'],
    ['/redeem', '/app/redeem'],
    ['/affiliate', '/app/affiliate'],
    ['/available-channels', '/app/available-channels'],
    ['/monitor', '/app/channel-status'],
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
