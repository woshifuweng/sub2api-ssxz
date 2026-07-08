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
      'nav.ops': 'Runtime Monitor',
      'nav.users': 'Users / Balance',
      'nav.groups': 'Groups / Pricing',
      'nav.subscriptions': 'Subscriptions',
      'nav.accounts': 'Upstream Accounts',
      'nav.announcements': 'Announcements',
      'nav.proxies': 'Proxy IPs',
      'nav.redeemCodes': 'Redeem Codes',
      'nav.promoCodes': 'Promo Codes',
      'nav.affiliates': 'Affiliates',
      'nav.settings': 'Site Settings',
      'nav.channelPricing': 'Channel Pricing',
      'nav.channelMonitor': 'Channel Monitor',
      'nav.paymentSettings': 'Payment Settings',
      'nav.paymentDashboard': 'Revenue Overview',
      'nav.orderManagement': 'Recharge Orders',
      'nav.paymentPlans': 'Plan Settings',
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
      '/app/profile'
    ]))
    expect(destinations).not.toContain('/app/chat')
    expect(destinations).not.toContain('/app/image')
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
      '/app/profile'
    ])
    expect(destinations).not.toEqual(expect.arrayContaining([
      '/available-channels',
      '/monitor',
      '/subscriptions'
    ]))
  })

  it('uses account-oriented labels for recharge and order destinations', () => {
    authState.isAdmin = false

    const wrapper = mountSidebar()
    const text = wrapper.text()

    expect(text).toContain('补充额度')
    expect(text).toContain('账户记录')
    expect(text).not.toContain('充值')
    expect(text).not.toContain('订单')
  })

  it('keeps core commercial entries visible for regular users in simple mode', () => {
    authState.isAdmin = false
    authState.isSimpleMode = true
    appState.cachedPublicSettings.affiliate_enabled = true

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
      '/app/profile'
    ])
    expect(destinations).not.toContain('/app/chat')
    expect(destinations).not.toContain('/app/image')
  })

  it('hides regular user recharge navigation when payment and subscription purchase are disabled', () => {
    authState.isAdmin = false
    appState.cachedPublicSettings.payment_enabled = false
    appState.cachedPublicSettings.purchase_subscription_enabled = false

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).not.toContain('/app/purchase')
    expect(destinations).toContain('/app/orders')
    expect(destinations).toContain('/app/redeem')
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
      '/app/profile'
    ])
  })

  it('keeps the unfinished affiliate route hidden for regular users when enabled', () => {
    authState.isAdmin = false
    appState.cachedPublicSettings.affiliate_enabled = true

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(wrapper.text()).not.toContain('推广中心')
    expect(destinations).toEqual([
      '/app/dashboard',
      '/app/keys',
      '/app/usage',
      '/app/channel-status',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile'
    ])
  })

  it('shows admin affiliate management for admins', () => {
    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toContain('/admin/affiliates')
    expect(wrapper.text()).toContain('Affiliates')
  })

  it('uses owner-facing labels for admin operating entries', () => {
    const wrapper = mountSidebar()
    const text = wrapper.text()

    expect(text).toContain('Users / Balance')
    expect(text).toContain('Groups / Pricing')
    expect(text).toContain('Upstream Accounts')
    expect(text).toContain('Channel Pricing')
    expect(text).toContain('Channel Monitor')
    expect(text).toContain('Revenue Overview')
    expect(text).toContain('Recharge Orders')
    expect(text).toContain('Plan Settings')
  })

  it('keeps payment settings accessible to admins when payment is disabled', () => {
    appState.cachedPublicSettings.payment_enabled = false

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toContain('/admin/orders/settings')
    expect(wrapper.text()).toContain('Payment Settings')
    expect(destinations).not.toContain('/admin/orders/dashboard')
    expect(destinations).not.toContain('/admin/orders')
    expect(destinations).not.toContain('/admin/orders/plans')
  })
})
