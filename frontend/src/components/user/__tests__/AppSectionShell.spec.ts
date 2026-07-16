import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { readFileSync } from 'node:fs'

const { routeState, mocks, authState, appState } = vi.hoisted(() => ({
  routeState: {
    path: '/app/chat',
    fullPath: '/app/chat'
  },
  mocks: {
    push: vi.fn(),
    logout: vi.fn(),
    showSuccess: vi.fn()
  },
  authState: {
    isAdmin: false
  },
  appState: {
    cachedPublicSettings: {
      affiliate_enabled: false
    }
  }
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    t: (key: string) => ({
      'nav.dashboard': 'Dashboard',
      'nav.chat': 'Chat',
      'nav.modelTest': 'Model Test',
      'nav.image': 'Image',
      'nav.apiKeys': 'API Keys',
      'nav.models': 'Models',
      'nav.usage': 'Usage',
      'nav.billing': 'Billing',
      'nav.orders': 'Orders',
      'nav.redeem': 'Redeem',
      'nav.channelStatus': 'Channel Status',
      'nav.groupOverview': 'Overview',
      'nav.groupUse': 'Use',
      'nav.groupBilling': 'Billing',
      'nav.groupAccount': 'Account',
      'nav.docs': 'Docs',
      'nav.affiliate': 'Referral Rewards',
      'nav.account': 'Account',
      'nav.logout': 'Sign out',
      'appShell.closeNavigation': 'Close navigation',
      'appShell.openNavigation': 'Open navigation',
      'appShell.expandSidebar': 'Expand sidebar',
      'appShell.collapseSidebar': 'Collapse sidebar',
      'appShell.backToDashboard': 'Back to dashboard',
      'appShell.developerConsole': 'Developer Console',
      'appShell.primaryNavigation': 'Primary navigation',
      'appShell.conversationHistory': 'Conversation history',
      'appShell.untitledConversation': 'Untitled conversation',
      'appShell.syncingHistory': 'Syncing history...',
      'appShell.noHistory': 'No conversation history',
      'appShell.balance': 'Balance',
      'appShell.adminConsole': 'Admin Console',
      'appShell.accountFallback': 'Account',
      'appShell.loggedOut': 'Signed out'
    })[key] ?? key
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: mocks.push })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: appState.cachedPublicSettings,
    showSuccess: mocks.showSuccess
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAuthenticated: true,
    isAdmin: authState.isAdmin,
    user: {
      username: 'tester',
      email: 'tester@example.com',
      balance: 8.53
    },
    logout: mocks.logout
  })
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import AppSectionShell from '../AppSectionShell.vue'

const appSectionShellSource = readFileSync('src/components/user/AppSectionShell.vue', 'utf-8')

function mockDesktopMedia(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn()
    }))
  })
}

function mountShell(props: Partial<InstanceType<typeof AppSectionShell>['$props']> = {}) {
  return mount(AppSectionShell, {
    props: {
      title: 'Chat',
      subtitle: 'Test a configured model.',
      ...props
    },
    global: {
      stubs: {
        ThemeToggle: true,
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

function navButtons(wrapper: ReturnType<typeof mountShell>) {
  return wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item')
}

describe('AppSectionShell', () => {
  beforeEach(() => {
    routeState.path = '/app/chat'
    routeState.fullPath = '/app/chat'
    authState.isAdmin = false
    appState.cachedPublicSettings.affiliate_enabled = false
    mocks.push.mockReset()
    mocks.logout.mockReset()
    mocks.showSuccess.mockReset()
    mockDesktopMedia(true)
  })

  it('renders the user navigation without duplicating the top documentation link', () => {
    const wrapper = mountShell()

    expect(navButtons(wrapper).map((button) => button.text())).toEqual([
      'Dashboard',
      'Model Test',
      'API Keys',
      'Models',
      'Usage',
      'Channel Status',
      'Billing',
      'Orders',
      'Redeem',
      'Account'
    ])
    expect(wrapper.get('.ssxz-header-docs').attributes('href')).toBe('/docs')
    expect(wrapper.get('.ssxz-header-docs').text()).toBe('Docs')
    expect(wrapper.text()).not.toMatch(/Affiliate|Referral|Image|Beta|Experiment/)
    expect(wrapper.find('.ssxz-secondary-nav').exists()).toBe(false)
  })

  it('adds the affiliate destination only when the public feature flag is enabled', async () => {
    appState.cachedPublicSettings.affiliate_enabled = true
    const wrapper = mountShell()

    expect(navButtons(wrapper).map((button) => button.text())).toContain('Referral Rewards')
    await navButtons(wrapper).find((button) => button.text() === 'Referral Rewards')!.trigger('click')
    expect(mocks.push).toHaveBeenCalledWith('/app/affiliate')
  })

  it('keeps the shared theme control in the top-right header without forcing dark mode', () => {
    const wrapper = mountShell()

    expect(wrapper.findComponent({ name: 'ThemeToggle' }).exists()).toBe(true)
    expect(appSectionShellSource).not.toContain("document.documentElement.classList.add('dark')")
  })

  it('uses dashboard as the brand destination with the shared SSXZ mark', () => {
    const wrapper = mountShell()
    const brand = wrapper.get('.ssxz-brand-link')

    expect(brand.attributes('href')).toBe('/app/dashboard')
    expect(brand.find('[data-testid="brand-logo"]').exists()).toBe(true)
    expect(brand.find('.brand-logo__mark').exists()).toBe(true)
    expect(brand.find('.ssxz-brand-wordmark').exists()).toBe(false)
    expect(brand.find('svg').exists()).toBe(false)
    expect(brand.find('img').exists()).toBe(false)
  })

  it('keeps all user navigation inside the approved app routes', async () => {
    routeState.path = '/app/test-origin'
    routeState.fullPath = '/app/test-origin'
    const wrapper = mountShell()
    const buttons = navButtons(wrapper)
    const expectedRoutes = [
      '/app/dashboard',
      '/app/chat',
      '/app/keys',
      '/app/available-channels',
      '/app/usage',
      '/app/channel-status',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile'
    ]

    for (const [index, button] of buttons.entries()) {
      await button.trigger('click')
      expect(mocks.push).toHaveBeenNthCalledWith(index + 1, expectedRoutes[index])
    }

    expect(mocks.push.mock.calls.map(([destination]) => destination)).toEqual(expectedRoutes)
  })

  it('starts a new chat through the Chat navigation item', async () => {
    routeState.path = '/app/image'
    routeState.fullPath = '/app/image'
    const wrapper = mountShell()

    await navButtons(wrapper)[1].trigger('click')

    expect(wrapper.emitted('new-chat')).toHaveLength(1)
    expect(mocks.push).toHaveBeenCalledWith('/app/chat')
  })

  it('marks Image active without marking Chat active', () => {
    routeState.path = '/app/channel-status'
    routeState.fullPath = '/app/channel-status'
    const wrapper = mountShell()
    const buttons = navButtons(wrapper)

    expect(buttons[1].classes()).not.toContain('is-active')
    expect(buttons[5].classes()).toContain('is-active')
  })

  it('keeps long history titles inside the sidebar hit target', async () => {
    const wrapper = mountShell({
      historyItems: [
        {
          id: 42,
          title: 'STAGING_173_NO_PROVIDER_FAILURE_WITH_A_VERY_LONG_TITLE',
          status: 'active',
          created_at: '2026-06-26T00:00:00Z',
          updated_at: '2026-06-26T00:00:00Z'
        }
      ],
      activeConversationId: null,
      historyLoading: false
    })
    const historyItem = wrapper.get('.ssxz-history-item')

    await historyItem.trigger('click')

    expect(wrapper.emitted('select-conversation')).toEqual([[42]])
    expect(appSectionShellSource).toMatch(/\.ssxz-history-item\s*\{[\s\S]*min-width:\s*0;[\s\S]*max-width:\s*100%;[\s\S]*overflow:\s*hidden;/)
    expect(appSectionShellSource).toMatch(/\.ssxz-history-item \.ssxz-sidebar-text\s*\{[\s\S]*min-width:\s*0;[\s\S]*max-width:\s*100%;/)
  })

  it('opens a real mobile navigation drawer', async () => {
    mockDesktopMedia(false)
    const wrapper = mountShell()

    await wrapper.get('.ssxz-sidebar-toggle-desktop').trigger('click')

    expect(wrapper.classes()).toContain('ssxz-mobile-nav-open')
    expect(wrapper.find('.ssxz-mobile-sidebar-scrim').exists()).toBe(true)
  })

  it('drops the mobile drawer state when the viewport becomes desktop', async () => {
    mockDesktopMedia(false)
    const wrapper = mountShell()

    await wrapper.get('.ssxz-sidebar-toggle-desktop').trigger('click')
    mockDesktopMedia(true)
    window.dispatchEvent(new Event('resize'))
    await nextTick()

    expect(wrapper.classes()).not.toContain('ssxz-mobile-nav-open')
  })

  it('closes the mobile drawer after navigation', async () => {
    mockDesktopMedia(false)
    const wrapper = mountShell()

    await wrapper.get('.ssxz-sidebar-toggle-desktop').trigger('click')
    await navButtons(wrapper)[0].trigger('click')

    expect(mocks.push).toHaveBeenLastCalledWith('/app/dashboard')
    expect(wrapper.classes()).not.toContain('ssxz-mobile-nav-open')
  })
})
