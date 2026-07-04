import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, appState, authState, adminSettingsState, onboardingState } = vi.hoisted(() => ({
  routeState: {
    path: '/app/image'
  },
  appState: {
    sidebarCollapsed: false,
    mobileOpen: false,
    backendModeEnabled: false,
    siteName: 'SSXZ AI',
    siteLogo: '',
    siteVersion: 'v0.test',
    publicSettingsLoaded: true,
    cachedPublicSettings: {
      payment_enabled: true,
      purchase_subscription_enabled: true,
      available_channels_enabled: true,
      channel_monitor_enabled: true,
      affiliate_enabled: false,
      sora_client_enabled: true,
      custom_menu_items: []
    },
    toggleSidebar: vi.fn(),
    setMobileOpen: vi.fn()
  },
  authState: {
    isAdmin: true,
    isSimpleMode: false
  },
  adminSettingsState: {
    opsMonitoringEnabled: true,
    customMenuItems: [],
    fetch: vi.fn()
  },
  onboardingState: {
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'nav.sora': 'Image Generation',
      'nav.usage': 'Usage',
      'nav.buySubscription': 'Recharge',
      'nav.profile': 'Profile',
      'nav.apiKeys': 'API Key / Third-party Access',
      'nav.mySubscriptions': 'Subscriptions',
      'nav.redeem': 'Redeem',
      'nav.myAccount': 'My Account',
      'nav.dashboard': 'Dashboard',
      'nav.ops': 'Ops',
      'nav.users': 'Users',
      'nav.groups': 'Groups',
      'nav.subscriptions': 'Admin Subscriptions',
      'nav.accounts': 'Accounts',
      'nav.announcements': 'Announcements',
      'nav.proxies': 'IP Management',
      'nav.redeemCodes': 'Redeem Codes',
      'nav.promoCodes': 'Promo Codes',
      'nav.affiliates': 'Affiliates',
      'nav.settings': 'Settings',
      'nav.lightMode': 'Light Mode',
      'nav.darkMode': 'Dark Mode',
      'nav.expand': 'Expand',
      'nav.collapse': 'Collapse'
    })[key] ?? key
  })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appState,
  useAuthStore: () => authState,
  useAdminSettingsStore: () => adminSettingsState,
  useOnboardingStore: () => onboardingState
}))

vi.mock('@/components/common/VersionBadge.vue', () => ({
  default: {
    name: 'VersionBadge',
    props: ['version'],
    template: '<span class="version-badge">{{ version }}</span>'
  }
}))

import AppSidebar from '../AppSidebar.vue'

function mountSidebar() {
  return mount(AppSidebar, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a class="router-link-stub" :href="to"><slot /></a>'
        }
      }
    }
  })
}

function hrefs(wrapper: ReturnType<typeof mountSidebar>) {
  return wrapper.findAll('a.router-link-stub').map((link) => link.attributes('href'))
}

describe('AppSidebar', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn()
      }))
    })
    routeState.path = '/app/image'
    appState.sidebarCollapsed = false
    appState.mobileOpen = false
    appState.backendModeEnabled = false
    appState.cachedPublicSettings = {
      payment_enabled: true,
      purchase_subscription_enabled: true,
      available_channels_enabled: true,
      channel_monitor_enabled: true,
      affiliate_enabled: false,
      sora_client_enabled: true,
      custom_menu_items: []
    }
    authState.isAdmin = true
    authState.isSimpleMode = false
    adminSettingsState.opsMonitoringEnabled = true
    adminSettingsState.customMenuItems = []
    adminSettingsState.fetch.mockReset()
  })

  it('keeps the admin My Account section focused on user workspace destinations', () => {
    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toEqual(expect.arrayContaining([
      '/app/dashboard',
      '/app/keys',
      '/app/usage',
      '/app/channel-status',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile',
      '/app/chat',
      '/app/image'
    ]))
    expect(destinations.filter((destination) => destination === '/app/image')).toHaveLength(1)
    expect(destinations).not.toEqual(expect.arrayContaining([
      '/available-channels',
      '/monitor',
      '/subscriptions'
    ]))
  })

  it('keeps regular user navigation focused on the operating platform first', () => {
    authState.isAdmin = false

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toEqual([
      '/app/dashboard',
      '/app/keys',
      '/app/usage',
      '/app/channel-status',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile',
      '/app/chat',
      '/app/image'
    ])
    expect(destinations).not.toEqual(expect.arrayContaining([
      '/available-channels',
      '/monitor',
      '/subscriptions'
    ]))
  })

  it('hides the user channel status entry when monitoring is disabled', () => {
    authState.isAdmin = false
    appState.cachedPublicSettings.channel_monitor_enabled = false

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).not.toContain('/app/channel-status')
    expect(wrapper.text()).not.toContain('通道状态')
    expect(destinations).toEqual([
      '/app/dashboard',
      '/app/keys',
      '/app/usage',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile',
      '/app/chat',
      '/app/image'
    ])
  })

  it('shows the existing affiliate route for regular users when enabled', () => {
    authState.isAdmin = false
    appState.cachedPublicSettings.affiliate_enabled = true

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(wrapper.text()).toContain('推广返利')
    expect(destinations).toEqual([
      '/app/dashboard',
      '/app/keys',
      '/app/usage',
      '/app/channel-status',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/affiliate',
      '/app/profile',
      '/app/chat',
      '/app/image'
    ])
  })

  it('shows admin affiliate management for admins', () => {
    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toContain('/admin/affiliates')
    expect(wrapper.text()).toContain('Affiliates')
  })

  it('keeps payment settings accessible to admins when payment is disabled', () => {
    appState.cachedPublicSettings.payment_enabled = false

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toContain('/admin/orders/settings')
    expect(wrapper.text()).toContain('支付配置')
    expect(destinations).not.toContain('/admin/orders/dashboard')
    expect(destinations).not.toContain('/admin/orders')
    expect(destinations).not.toContain('/admin/orders/plans')
  })
})
