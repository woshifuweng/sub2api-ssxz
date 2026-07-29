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
    ['/app/dashboard', 'AppDashboard', 'dashboard'],
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
    ['/app/dashboard', '仪表盘'],
    ['/app/chat', '模型测试入口'],
    ['/app/image', '图片工作台'],
    ['/app/usage', '使用记录'],
    ['/app/purchase', '充值 / 订阅'],
    ['/app/orders', '我的订单'],
    ['/app/redeem', '兑换码'],
    ['/app/affiliate', '邀请返利'],
    ['/app/available-channels', '模型价格'],
    ['/app/channel-status', '渠道状态'],
    ['/app/keys', 'API Key / 第三方接入'],
    ['/app/profile', '个人资料'],
  ])('uses SSXZ AI as the user workbench document title site name for %s', (path, title) => {
    const route = router.resolve(path)

    expect(route.meta.title).toBe(title)
    expect(route.meta.titleSiteName).toBe('SSXZ AI')
  })

  it.each([
    ['/app', '/app/dashboard'],
    ['/dashboard', '/app/dashboard'],
    ['/ai-chat', '/app/chat'],
    ['/image-studio', '/app/image'],
    ['/sora', '/app/image'],
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
