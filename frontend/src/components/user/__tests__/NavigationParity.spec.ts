import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'
import type { PublicSettings } from '@/types'
import AppSectionShell from '../AppSectionShell.vue'
import AppSidebar from '@/components/layout/AppSidebar.vue'

enableAutoUnmount(afterEach)
const state = vi.hoisted(() => ({ app: {} as any, auth: {} as any, admin: {} as any, batch: {} as any, push: vi.fn(), path: '/app/dashboard' }))
vi.mock('vue-i18n', async (original) => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-router', async (original) => ({ ...await original<typeof import('vue-router')>(), useRoute: () => ({ path: state.path }), useRouter: () => ({ push: state.push }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => state.app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => state.auth }))
vi.mock('@/stores', () => ({
  useAppStore: () => state.app,
  useAuthStore: () => state.auth,
  useAdminSettingsStore: () => state.admin,
  useOnboardingStore: () => ({ isCurrentStep: () => false }),
  useResellerStore: () => ({ isAgent: false, isManager: false, reset: vi.fn(), fetchRole: vi.fn().mockResolvedValue(null) }),
}))
vi.mock('@/composables/useResellerNavRole', () => ({ useResellerNavRole: () => ({ isAgent: ref(false), isManager: ref(false) }) }))
vi.mock('@/composables/useBatchImageAccess', () => ({ useBatchImageAccess: () => state.batch }))

const stubs = {
  BrandLogo: true, ThemeToggle: true, AnnouncementBell: true, LocaleSwitcher: true, Icon: true,
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}

function render(component: typeof AppSectionShell | typeof AppSidebar) {
  return mount(component, { props: component === AppSectionShell ? { title: 'SSXZ', subtitle: '' } : {}, global: { stubs } })
}

describe('SSXZ route and navigation parity', () => {
  beforeEach(() => {
    state.app = reactive({ cachedPublicSettings: {} as Partial<PublicSettings> & { plugin_management_enabled?: boolean }, siteName: 'SSXZ AI', backendModeEnabled: false, sidebarScrollTop: 0 })
    state.auth = reactive({ isAdmin: false, isAuthenticated: true, isSimpleMode: false, user: { id: 1, username: 'tester' } })
    state.admin = reactive({ customMenuItems: [], opsMonitoringEnabled: false, fetch: vi.fn() })
    state.batch = { canUseBatchImage: ref(false), refreshBatchImageAccess: vi.fn().mockResolvedValue(false) }
    state.push.mockReset()
    state.path = '/app/dashboard'
  })

  it.each([AppSectionShell, AppSidebar])('honors the same feature flags without coupling redemption to payment', async (component) => {
    state.app.cachedPublicSettings = { payment_enabled: false, subscription_enabled: false, channel_monitor_enabled: false, available_channels_enabled: false }
    const wrapper = render(component)
    expect(wrapper.text()).not.toContain('nav.channelStatus')
    expect(wrapper.text()).not.toContain('nav.models')
    expect(wrapper.text()).not.toContain('nav.mySubscriptions')
    expect(wrapper.text()).not.toContain('nav.buySubscription')
    expect(wrapper.text()).toContain('nav.shop')
    expect(wrapper.text()).toContain('nav.redeem')
    state.app.cachedPublicSettings = { payment_enabled: true, subscription_enabled: true, channel_monitor_enabled: true, available_channels_enabled: true }
    await flushPromises()
    expect(wrapper.text()).toContain('nav.channelStatus')
    expect(wrapper.text()).toContain('nav.models')
    expect(wrapper.text()).toContain('nav.mySubscriptions')
    expect(wrapper.text()).toContain('nav.buySubscription')
  })

  it.each([AppSectionShell, AppSidebar])('keeps groups of user custom menus sorted and hides admin-only entries', (component) => {
    state.app.cachedPublicSettings.custom_menu_items = [
      { id: 'late', label: 'Late menu', visibility: 'user', sort_order: 9 },
      { id: 'private', label: 'Admin secret menu', visibility: 'admin', sort_order: 0 },
      { id: 'early', label: 'Early menu', visibility: 'user', sort_order: 1 },
    ]
    const wrapper = render(component)
    expect(wrapper.text()).toContain('Early menu')
    expect(wrapper.text()).not.toContain('Admin secret menu')
    expect(wrapper.text().indexOf('Early menu')).toBeLessThan(wrapper.text().indexOf('Late menu'))
  })

  it.each([AppSectionShell, AppSidebar])('hides subscription and redemption in simple mode', (component) => {
    state.auth.isSimpleMode = true
    const wrapper = render(component)
    expect(wrapper.text()).not.toContain('nav.redeem')
    expect(wrapper.text()).not.toContain('nav.mySubscriptions')
  })

  it('keeps groups in the simple admin console and respects plugin opt-in', async () => {
    state.auth.isAdmin = true
    state.auth.isSimpleMode = true
    const wrapper = render(AppSidebar)
    expect(wrapper.text()).toContain('nav.groups')
    expect(wrapper.text()).not.toContain('nav.plugins')
    state.app.cachedPublicSettings.plugin_management_enabled = true
    await flushPromises()
    expect(wrapper.text()).toContain('nav.plugins')
  })

  it('exposes admin audit and affiliate detail navigation when enabled', () => {
    state.auth.isAdmin = true
    state.app.cachedPublicSettings.affiliate_enabled = true
    const wrapper = render(AppSidebar)
    expect(wrapper.find('a[href="/admin/audit-logs"]').exists()).toBe(true)
    for (const path of ['invites', 'rebates', 'transfers']) {
      expect(wrapper.find(`a[href="/admin/affiliates/${path}"]`).exists()).toBe(true)
    }
  })

  it.each([AppSectionShell, AppSidebar])('only exposes batch images for eligible keys', async (component) => {
    const wrapper = render(component)
    expect(wrapper.text()).not.toContain('nav.batchImage')
    expect(state.batch.refreshBatchImageAccess).toHaveBeenCalledWith(true)
    state.batch.canUseBatchImage.value = true
    await flushPromises()
    const entry = wrapper.findAll('button, a').find(item => item.text() === 'nav.batchImage')!
    expect(entry).toBeDefined()
    if (component === AppSidebar) expect(entry.attributes('href')).toBe('/batch-image')
    else {
      await entry.trigger('click')
      expect(state.push).toHaveBeenCalledWith('/batch-image')
    }
  })

  it.each([AppSectionShell, AppSidebar])('labels native purchase for the configured billing mode', async (component) => {
    state.app.cachedPublicSettings = { subscription_enabled: false, payment_enabled: true }
    const wrapper = render(component)
    expect(wrapper.text()).toContain('nav.recharge')
    expect(wrapper.text()).not.toContain('nav.mySubscriptions')
    state.app.cachedPublicSettings = { subscription_enabled: true, payment_enabled: true, payment_balance_disabled: true }
    await flushPromises()
    expect(wrapper.text()).toContain('nav.subscribe')
    expect(wrapper.text()).toContain('nav.mySubscriptions')
  })

  it('hides admin subscription management when disabled', async () => {
    state.auth.isAdmin = true
    state.app.cachedPublicSettings.subscription_enabled = false
    const wrapper = render(AppSidebar)
    expect(wrapper.find('a[href="/admin/subscriptions"]').exists()).toBe(false)
    state.app.cachedPublicSettings.subscription_enabled = true
    await flushPromises()
    expect(wrapper.find('a[href="/admin/subscriptions"]').exists()).toBe(true)
  })

  it('lets keyboard users collapse an active security group and reopen its destinations', async () => {
    state.auth.isAdmin = true
    state.app.cachedPublicSettings.risk_control_enabled = true
    state.path = '/admin/prompt-audit'
    const wrapper = render(AppSidebar)
    const toggle = wrapper.get('button[aria-expanded]')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('a[href="/admin/prompt-audit"]').exists()).toBe(true)
    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('a[href="/admin/prompt-audit"]').exists()).toBe(false)
    await toggle.trigger('click')
    expect(wrapper.find('a[href="/admin/risk-control"]').exists()).toBe(true)
  })
})
