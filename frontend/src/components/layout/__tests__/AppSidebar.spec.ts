import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, appState, authState, adminSettingsState, onboardingState, resellerState, keysListMock } = vi.hoisted(() => ({
  routeState: {
    path: '/app/dashboard'
  },
  appState: {
    sidebarCollapsed: false,
    mobileOpen: false,
    backendModeEnabled: false,
    siteName: 'SSXZ AI Gateway',
    siteLogo: '',
    siteVersion: 'v0.test',
    publicSettingsLoaded: true,
    cachedPublicSettings: {
      payment_enabled: true,
      channel_monitor_enabled: true,
      affiliate_enabled: false,
      custom_menu_items: []
    },
    toggleSidebar: vi.fn(),
    setMobileOpen: vi.fn(),
    showWarning: vi.fn(),
    showError: vi.fn()
  },
  authState: {
    isAdmin: false,
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
  },
  resellerState: {
    isAgent: false,
    isManager: false,
    fetchRole: vi.fn(),
    reset: vi.fn()
  },
  keysListMock: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'nav.usage': 'Usage',
      'nav.dashboard': 'Dashboard',
      'nav.chat': 'Chat',
      'nav.modelTest': 'Model Test',
      'nav.image': 'Image',
      'nav.apiKeys': 'API Keys',
      'nav.imageWorkbench': 'Image Workbench',
      'nav.imageWorkbenchNoKey': 'Create and enable an API key first',
      'nav.imageWorkbenchKeyError': 'Unable to load an API key. Try again later.',
      'nav.models': 'Models',
      'nav.billing': 'Billing',
      'nav.redeem': 'Redeem',
      'nav.orders': 'Orders',
      'nav.billingRecords': 'Billing Records',
      'nav.channelStatus': 'Channel Status',
      'nav.groupOverview': 'Overview',
      'nav.groupUse': 'Use',
      'nav.groupInformation': 'Information',
      'nav.groupBilling': 'Billing',
      'nav.groupAccount': 'Account',
      'nav.groupSystem': 'System',
      'nav.docs': 'Docs',
      'nav.account': 'Account',
      'nav.affiliate': 'Referral Rewards',
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
      'nav.expand': 'Expand',
      'nav.collapse': 'Collapse',
      'appShell.adminConsole': 'Admin Console',
      'appShell.serviceConsole': 'Service Console'
    })[key] ?? key
  })
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appState,
  useAuthStore: () => authState,
  useAdminSettingsStore: () => adminSettingsState,
  useOnboardingStore: () => onboardingState,
  useResellerStore: () => resellerState
}))

vi.mock('@/api/keys', () => ({
  keysAPI: {
    list: keysListMock
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
    routeState.path = '/app/dashboard'
    appState.sidebarCollapsed = false
    appState.mobileOpen = false
    appState.backendModeEnabled = false
    appState.cachedPublicSettings = {
      payment_enabled: true,
      channel_monitor_enabled: true,
      affiliate_enabled: false,
      custom_menu_items: []
    }
    authState.isAdmin = false
    authState.isSimpleMode = false
    adminSettingsState.opsMonitoringEnabled = true
    adminSettingsState.customMenuItems = []
    adminSettingsState.fetch.mockReset()
    resellerState.isAgent = false
    resellerState.isManager = false
    resellerState.fetchRole.mockReset()
    resellerState.reset.mockReset()
    appState.showWarning.mockReset()
    appState.showError.mockReset()
    keysListMock.mockReset()
    keysListMock.mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  it('hides the affiliate entry while the public feature flag is disabled', () => {
    const wrapper = mountSidebar()

    expect(hrefs(wrapper)).toEqual([
      '/app/dashboard',
      '/app/keys',
      '/app/available-channels',
      '/app/usage',
      '/app/channel-status',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile'
    ])
    for (const label of ['Dashboard', 'API Keys', 'Models', 'Usage', 'Channel Status', 'Orders', 'Redeem', 'Account']) {
      expect(wrapper.text()).toContain(label)
    }
    expect(wrapper.text()).toContain('Image Workbench')
    expect(wrapper.text()).not.toContain('Model Test')
    expect(wrapper.findAll('.sidebar-group-label')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Docs')
    expect(wrapper.text()).not.toMatch(/Affiliate|Referral|Beta|Experiment/)
  })

  it('merges the redeem entry into billing records when payment is disabled', () => {
    appState.cachedPublicSettings.payment_enabled = false

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toContain('/app/orders')
    expect(destinations).not.toContain('/app/redeem')
    expect(wrapper.text()).toContain('Billing Records')
    expect(wrapper.text()).not.toContain('Redeem')
    expect(wrapper.text()).not.toContain('Orders')
  })

  it('shows the user affiliate entry when the public feature flag is enabled', () => {
    appState.cachedPublicSettings.affiliate_enabled = true

    const wrapper = mountSidebar()

    expect(hrefs(wrapper)).toContain('/app/affiliate')
    expect(wrapper.text()).toContain('Referral Rewards')
  })

  it('opens the image workbench with the first active API key', async () => {
    keysListMock.mockResolvedValue({
      items: [{ id: 1, key: 'sk test/+', status: 'active' }],
      total: 1,
      pages: 1
    })
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = mountSidebar()

    await wrapper.get('[data-testid="sidebar-image-workbench"]').trigger('click')

    expect(keysListMock).toHaveBeenCalledWith(1, 100, { status: 'active' })
    expect(openSpy).toHaveBeenCalledWith(
      '/image/?apiKey=sk%20test%2F%2B',
      '_blank',
      'noopener,noreferrer'
    )
    openSpy.mockRestore()
  })

  it('does not open the image workbench without an active API key', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const wrapper = mountSidebar()

    await wrapper.get('[data-testid="sidebar-image-workbench"]').trigger('click')

    expect(openSpy).not.toHaveBeenCalled()
    expect(appState.showWarning).toHaveBeenCalledWith('Create and enable an API key first')
    openSpy.mockRestore()
  })

  it('hides the user channel status entry while channel monitoring is disabled', () => {
    appState.cachedPublicSettings.channel_monitor_enabled = false

    const wrapper = mountSidebar()

    expect(hrefs(wrapper)).not.toContain('/app/channel-status')
    expect(wrapper.text()).not.toContain('Channel Status')
  })

  it('uses the shared SSXZ brand mark instead of the legacy text box', () => {
    const wrapper = mountSidebar()
    const header = wrapper.get('.sidebar-header')

    expect(header.find('[data-testid="brand-logo"]').exists()).toBe(true)
    expect(header.find('.brand-logo__mark').exists()).toBe(true)
    expect(header.find('.ssxz-sidebar-wordmark').exists()).toBe(false)
    expect(header.find('svg').exists()).toBe(false)
    expect(header.find('img').exists()).toBe(false)
  })

  it('keeps the admin console isolated from user workspace routes', () => {
    authState.isAdmin = true

    const wrapper = mountSidebar()
    const destinations = hrefs(wrapper)

    expect(destinations).toContain('/admin/dashboard')
    expect(destinations).toContain('/admin/accounts')
    expect(destinations).toContain('/admin/affiliates')
    expect(destinations.every((path) => path.startsWith('/admin/'))).toBe(true)
    expect(destinations.some((path) => path.startsWith('/app/'))).toBe(false)
  })

  it('does not inject user routes into the simple admin console', () => {
    authState.isAdmin = true
    authState.isSimpleMode = true

    const destinations = hrefs(mountSidebar())

    expect(destinations).toContain('/admin/dashboard')
    expect(destinations).toContain('/admin/settings')
    expect(destinations.every((path) => path.startsWith('/admin/'))).toBe(true)
  })

  it('keeps payment settings but hides payment operations when payment is disabled', () => {
    authState.isAdmin = true
    appState.cachedPublicSettings.payment_enabled = false

    const destinations = hrefs(mountSidebar())

    expect(destinations).toContain('/admin/orders/settings')
    expect(destinations).not.toContain('/admin/orders/dashboard')
    expect(destinations).not.toContain('/admin/orders')
    expect(destinations).not.toContain('/admin/orders/plans')
  })
})
