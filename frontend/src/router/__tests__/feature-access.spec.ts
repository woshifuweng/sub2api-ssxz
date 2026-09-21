import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import type { RouteRecordRaw } from 'vue-router'
import type { PublicSettings } from '@/types'

type NavigationGuard = (
  to: Record<string, any>,
  from: Record<string, any>,
  next: ReturnType<typeof vi.fn>
) => Promise<void>

const routerHarness = vi.hoisted(() => ({
  guard: null as NavigationGuard | null,
  routes: [] as RouteRecordRaw[],
}))

const authStore = vi.hoisted(() => ({
  checkAuth: vi.fn(),
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
  hasPendingAuthSession: false,
}))

const appStore = vi.hoisted(() => ({
  siteName: 'Sub2API',
  backendModeEnabled: false,
  publicSettingsLoaded: false,
  cachedPublicSettings: null as null | (Partial<PublicSettings> & { plugin_management_enabled?: boolean }),
  fetchPublicSettings: vi.fn(),
}))

vi.mock('vue-router', () => ({
  createWebHistory: vi.fn(() => ({})),
  createRouter: vi.fn((options: { routes: RouteRecordRaw[] }) => {
    routerHarness.routes = options.routes
    return {
      beforeEach: vi.fn((guard: NavigationGuard) => {
        routerHarness.guard = guard
      }),
      afterEach: vi.fn(),
      onError: vi.fn(),
    }
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/stores/reseller', () => ({
  useResellerStore: () => ({
    fetchMyRole: vi.fn(),
    isReseller: false,
    isAgent: false,
  }),
}))

vi.mock('@/stores/adminCompliance', () => ({
  useAdminComplianceStore: () => ({
    initialized: true,
    fetchStatus: vi.fn(),
    requireAcknowledgement: vi.fn(),
  }),
}))

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

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

function runGuard(meta: Record<string, unknown>, path: string) {
  if (!routerHarness.guard) {
    throw new Error('router guard was not registered')
  }

  const next = vi.fn()
  const navigation = routerHarness.guard(
    {
      path,
      fullPath: path,
      name: routerHarness.routes.find((route) => route.path === path)?.name ?? 'FeatureRoute',
      params: {},
      meta: { requiresAuth: true, ...meta },
    },
    {},
    next
  )
  return { navigation, next }
}

beforeAll(async () => {
  await import('@/router')
})

describe('feature route guard', () => {
  beforeEach(() => {
    authStore.isAuthenticated = true
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    appStore.fetchPublicSettings.mockReset()
    appStore.backendModeEnabled = false
  })

  it('waits for the first public-settings request before deciding payment access', async () => {
    const deferred = createDeferred<{ payment_enabled: boolean }>()
    appStore.fetchPublicSettings.mockImplementation(async () => {
      const settings = await deferred.promise
      appStore.cachedPublicSettings = settings
      appStore.publicSettingsLoaded = true
      return settings
    })

    const { navigation, next } = runGuard({ requiresPayment: true }, '/purchase')

    await vi.waitFor(() => expect(appStore.fetchPublicSettings).toHaveBeenCalledTimes(1))
    expect(next).not.toHaveBeenCalled()

    deferred.resolve({ payment_enabled: true })
    await navigation
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['payment', { requiresPayment: true }, '/purchase'],
    ['risk control', { requiresRiskControl: true }, '/admin/risk-control'],
    ['subscription', { requiresSubscription: true }, '/subscriptions'],
  ])('does not treat a failed %s settings load as explicitly disabled', async (_name, meta, path) => {
    authStore.isAdmin = meta.requiresRiskControl === true
    appStore.fetchPublicSettings.mockResolvedValue(null)

    const { navigation, next } = runGuard(meta, path)
    await navigation

    expect(appStore.publicSettingsLoaded).toBe(false)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['payment', { requiresPayment: true }, { payment_enabled: false }, '/app/dashboard'],
    [
      'risk control',
      { requiresRiskControl: true },
      { risk_control_enabled: false },
      '/admin/settings',
    ],
    ['subscription', { requiresSubscription: true }, { subscription_enabled: false }, '/dashboard'],
  ])('redirects when loaded settings explicitly disable %s', async (_name, meta, settings, target) => {
    authStore.isAdmin = meta.requiresRiskControl === true
    appStore.cachedPublicSettings = settings
    appStore.publicSettingsLoaded = true

    const { navigation, next } = runGuard(meta, '/feature')
    await navigation

    expect(appStore.fetchPublicSettings).not.toHaveBeenCalled()
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith(target)
  })
})

describe('subscription route guard (opt-out flag)', () => {
  beforeEach(() => {
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    appStore.publicSettingsLoaded = true
    appStore.fetchPublicSettings.mockReset()
  })

  it.each([
    ['missing key', {}],
    ['explicit true', { subscription_enabled: true }],
  ])('lets /subscriptions through when the flag is %s', async (_name, settings) => {
    appStore.cachedPublicSettings = settings

    const { navigation, next } = runGuard({ requiresSubscription: true }, '/subscriptions')
    await navigation

    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()
  })

  it('sends admins to the admin dashboard when subscriptions are disabled', async () => {
    authStore.isAdmin = true
    appStore.cachedPublicSettings = { subscription_enabled: false }

    const { navigation, next } = runGuard({ requiresSubscription: true }, '/subscriptions')
    await navigation

    expect(next).toHaveBeenCalledWith('/admin/dashboard')
  })
})

describe('SSXZ upstream route parity', () => {
  beforeEach(() => {
    authStore.isAuthenticated = true
    authStore.isAdmin = false
    authStore.isSimpleMode = false
    authStore.hasPendingAuthSession = false
    appStore.backendModeEnabled = false
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = {}
    appStore.fetchPublicSettings.mockReset()
  })

  it.each([
    ['/admin/plugins', 'PluginsView.vue'],
    ['/admin/audit-logs', 'AuditLogView.vue'],
    ['/admin/affiliates/invites', 'AdminAffiliateInvitesView.vue'],
    ['/admin/affiliates/rebates', 'AdminAffiliateRebatesView.vue'],
    ['/admin/affiliates/transfers', 'AdminAffiliateTransfersView.vue'],
  ])('restores admin-only destination %s', async (path, view) => {
    const record = routerHarness.routes.find((route) => route.path === path)!
    expect(record).toBeDefined()
    expect(String(record.component)).toContain(view)
    expect(record.meta?.requiresAdmin).toBe(true)
    const { navigation, next } = runGuard(record.meta!, path)
    await navigation
    expect(next).toHaveBeenCalledWith('/app/dashboard')
  })

  it.each([
    ['/app/purchase', 'PaymentView.vue'],
    ['/app/subscriptions', 'SubscriptionsView.vue'],
    ['/app/shop', 'AppPurchaseView.vue'],
    ['/app/available-channels', 'AvailableChannelsView.vue'],
    ['/batch-image', 'BatchImageGuideView.vue'],
    ['/payment/airwallex', 'AirwallexPaymentView.vue'],
    ['/auth/dingtalk/callback', 'DingTalkCallbackView.vue'],
    ['/auth/dingtalk/email-completion', 'DingTalkEmailCompletionView.vue'],
    ['/auth/oidc/callback', 'OidcCallbackView.vue'],
  ])('connects %s to its native implementation', (path, view) => {
    const record = routerHarness.routes.find((route) => route.path === path)!
    expect(record).toBeDefined()
    expect(String(record.component)).toContain(view)
  })

  it('preserves upstream aliases and legacy subscription query parameters', () => {
    expect(routerHarness.routes.find((route) => route.path === '/batch-image')?.alias).toContain('/docs/batch-image')
    expect(routerHarness.routes.find((route) => route.path === '/auth/callback')?.alias).toContain('/auth/oauth/callback')
    const redirect = routerHarness.routes.find((route) => route.path === '/subscriptions')!.redirect as Function
    expect(redirect({ query: { plan: '7' }, hash: '#usage' })).toEqual({ path: '/app/subscriptions', query: { plan: '7' }, hash: '#usage' })
  })

  it('resolves aliases with the real router without changing OAuth callback paths', async () => {
    const { createRouter, createMemoryHistory } = await vi.importActual<typeof import('vue-router')>('vue-router')
    const router = createRouter({ history: createMemoryHistory(), routes: routerHarness.routes })
    const callback = router.resolve('/auth/oauth/callback?resume=1#result')
    expect(callback.name).toBe('OAuthCallback')
    expect(callback.path).toBe('/auth/oauth/callback')
    expect(callback.query).toEqual({ resume: '1' })
    expect(callback.hash).toBe('#result')
    const batch = router.resolve('/docs/batch-image')
    expect(batch.name).toBe('BatchImageGuide')
    expect(batch.meta.requiresAuth).toBe(true)
  })

  it.each([
    ['/app/purchase', { payment_enabled: true }],
    ['/app/subscriptions', { subscription_enabled: true }],
    ['/app/available-channels', { available_channels_enabled: true }],
    ['/app/channel-status', { channel_monitor_enabled: true }],
    ['/admin/plugins', { plugin_management_enabled: true }],
    ['/admin/audit-logs', {}],
    ['/admin/affiliates/invites', { affiliate_enabled: true }],
    ['/admin/affiliates/rebates', { affiliate_enabled: true }],
    ['/admin/affiliates/transfers', { affiliate_enabled: true }],
  ])('allows an authorized visitor into enabled %s', async (path, settings) => {
    authStore.isAdmin = path.startsWith('/admin/')
    appStore.cachedPublicSettings = settings
    const record = routerHarness.routes.find(route => route.path === path)!
    const { navigation, next } = runGuard(record.meta!, path)
    await navigation
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['/auth/oauth/callback', 'OAuthCallback'],
    ['/auth/dingtalk/callback', 'DingTalkOAuthCallback'],
    ['/auth/dingtalk/email-completion', 'dingtalk-email-completion'],
    ['/auth/oidc/callback', 'OIDCOAuthCallback'],
    ['/payment/airwallex', 'AirwallexPayment'],
  ])('allows anonymous completion at %s in backend mode', async (path, name) => {
    authStore.isAuthenticated = false
    appStore.backendModeEnabled = true
    const record = routerHarness.routes.find((route) => route.name === name)!
    expect(record?.meta?.requiresAuth).toBe(false)
    const { navigation, next } = runGuard(record.meta!, path)
    await navigation
    expect(next).toHaveBeenCalledWith()
  })

  it.each([
    ['/app/channel-status', { channel_monitor_enabled: false }],
    ['/app/available-channels', { available_channels_enabled: false }],
    ['/app/subscriptions', { subscription_enabled: false }],
    ['/app/purchase', { payment_enabled: false }],
    ['/admin/plugins', { plugin_management_enabled: false }],
  ])('blocks direct navigation to explicitly disabled %s', async (path, settings) => {
    authStore.isAdmin = path.startsWith('/admin/')
    appStore.cachedPublicSettings = settings
    const record = routerHarness.routes.find((route) => route.path === path)!
    expect(record).toBeDefined()
    const { navigation, next } = runGuard(record.meta!, path)
    await navigation
    expect(next).not.toHaveBeenCalledWith()
  })

  it('keeps channel status enabled when its opt-out setting is absent', async () => {
    const { navigation, next } = runGuard({}, '/app/channel-status')
    await navigation
    expect(next).toHaveBeenCalledWith()
  })

  it.each(['/register', '/email-verify'])('permits pending OAuth email completion at %s without opening registration generally', async (path) => {
    authStore.isAuthenticated = false
    appStore.backendModeEnabled = true
    const blocked = runGuard({ requiresAuth: false }, path)
    await blocked.navigation
    expect(blocked.next).toHaveBeenCalledWith('/login')
    authStore.hasPendingAuthSession = true
    const allowed = runGuard({ requiresAuth: false }, path)
    await allowed.navigation
    expect(allowed.next).toHaveBeenCalledWith()
  })

  it.each([
    ['/app/available-channels', { available_channels_enabled: false }],
    ['/app/channel-status', { channel_monitor_enabled: false }],
    ['/app/subscriptions', { subscription_enabled: false }],
    ['/app/purchase', { payment_enabled: false }],
  ])('waits for asynchronously loaded flags before entering %s', async (path, settings) => {
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    const pending = createDeferred<void>()
    appStore.fetchPublicSettings.mockImplementation(async () => {
      await pending.promise
      appStore.cachedPublicSettings = settings
      appStore.publicSettingsLoaded = true
    })
    const record = routerHarness.routes.find(route => route.path === path)!
    const { navigation, next } = runGuard(record.meta!, path)
    expect(next).not.toHaveBeenCalled()
    pending.resolve()
    await navigation
    expect(next).not.toHaveBeenCalledWith()
  })

  it('does not let callback lookalike paths bypass backend mode', async () => {
    authStore.isAuthenticated = false
    appStore.backendModeEnabled = true
    const { navigation, next } = runGuard({ requiresAuth: false }, '/auth/dingtalk/callback-untrusted')
    await navigation
    expect(next).toHaveBeenCalledWith('/login')
  })

  const gatedRoutes = [
    ['/admin/plugins', 'plugin_management_enabled'],
    ['/app/available-channels', 'available_channels_enabled'],
    ['/app/channel-status', 'channel_monitor_enabled'],
    ['/admin/channels/monitor', 'channel_monitor_enabled'],
    ['/monitor', 'channel_monitor_enabled'],
    ['/app/affiliate', 'affiliate_enabled'],
    ['/admin/affiliates', 'affiliate_enabled'],
    ['/admin/affiliates/invites', 'affiliate_enabled'],
    ['/admin/affiliates/rebates', 'affiliate_enabled'],
    ['/admin/affiliates/transfers', 'affiliate_enabled'],
    ['/admin/subscriptions', 'subscription_enabled'],
  ] as const

  it.each(gatedRoutes.flatMap(([path, flag]) =>
    [path, `${path}/`, path.toUpperCase(), `${path.toUpperCase()}/`]
      .map((variant) => ({ path, flag, variant })),
  ))('preserves feature and role gates for $variant', async ({ path, flag, variant }) => {
    const { createRouter, createMemoryHistory } = await vi.importActual<typeof import('vue-router')>('vue-router')
    const router = createRouter({ history: createMemoryHistory(), routes: routerHarness.routes })
    const resolved = router.resolve(`${variant}?source=guard-test#details`)
    const record = routerHarness.routes.find((route) => route.path === path)!
    expect(resolved.name).toBe(record.name)
    expect(resolved.meta.requiresAuth).toBe(true)
    const next = vi.fn()
    authStore.isAdmin = record.meta?.requiresAdmin === true
    appStore.cachedPublicSettings = { [flag]: false }
    await routerHarness.guard!(resolved, {}, next)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith(authStore.isAdmin ? '/admin/dashboard' : '/app/dashboard')

    next.mockClear()
    appStore.cachedPublicSettings = { [flag]: true }
    await routerHarness.guard!(resolved, {}, next)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith()

    if (record.meta?.requiresAdmin) {
      next.mockClear()
      authStore.isAdmin = false
      await routerHarness.guard!(resolved, {}, next)
      expect(next).toHaveBeenCalledOnce()
      expect(next).toHaveBeenCalledWith('/app/dashboard')
    }

    next.mockClear()
    authStore.isAuthenticated = false
    await routerHarness.guard!(resolved, {}, next)
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith(expect.objectContaining({ path: '/login' }))
  })

  it('waits for plugin settings on a variant URL and keeps missing opt-in settings closed', async () => {
    const { createRouter, createMemoryHistory } = await vi.importActual<typeof import('vue-router')>('vue-router')
    const router = createRouter({ history: createMemoryHistory(), routes: routerHarness.routes })
    const resolved = router.resolve('/ADMIN/PLUGINS/')
    authStore.isAdmin = true
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    const pending = createDeferred<void>()
    appStore.fetchPublicSettings.mockImplementation(() => pending.promise)
    const next = vi.fn()
    const navigation = routerHarness.guard!(resolved, {}, next)
    expect(appStore.fetchPublicSettings).toHaveBeenCalledOnce()
    expect(next).not.toHaveBeenCalled()
    pending.resolve()
    await navigation
    expect(next).toHaveBeenCalledOnce()
    expect(next).toHaveBeenCalledWith('/admin/dashboard')
  })

  it.each(['/app/redeem', '/app/subscriptions'])('blocks canonical simple-mode path %s', async (path) => {
    authStore.isSimpleMode = true
    const { navigation, next } = runGuard({}, path)
    await navigation
    expect(next).toHaveBeenCalledWith('/app/dashboard')
  })
})
