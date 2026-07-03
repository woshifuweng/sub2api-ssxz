import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { readFileSync } from 'node:fs'

const { routeState, mocks, appState } = vi.hoisted(() => ({
  routeState: {
    path: '/app/chat'
  },
  mocks: {
    push: vi.fn(),
    logout: vi.fn(),
    showSuccess: vi.fn()
  },
  appState: {
    cachedPublicSettings: {
      channel_monitor_enabled: true,
      affiliate_enabled: false
    }
  }
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
      title: '聊天',
      subtitle: '辅助写 prompt',
      ...props
    },
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('AppSectionShell', () => {
  beforeEach(() => {
    routeState.path = '/app/chat'
    mocks.push.mockReset()
    mocks.logout.mockReset()
    mocks.showSuccess.mockReset()
    appState.cachedPublicSettings.channel_monitor_enabled = true
    appState.cachedPublicSettings.affiliate_enabled = false
    mockDesktopMedia(true)
  })

  it('keeps API Key available as a third-party client entrypoint', () => {
    const wrapper = mountShell()

    expect(wrapper.text()).toContain('中转运营平台')
    expect(wrapper.text()).toContain('SSXZ AI')
    expect(wrapper.text()).not.toContain('图片工具站')
    expect(wrapper.text()).not.toContain('对话工作台')
    expect(wrapper.text()).toContain('仪表盘')
    expect(wrapper.text()).toContain('API 密钥')
    expect(wrapper.text()).toContain('使用记录')
    expect(wrapper.text()).toContain('通道状态')
    expect(wrapper.text()).toContain('充值')
    expect(wrapper.text()).toContain('订单')
    expect(wrapper.text()).toContain('兑换码')
    expect(wrapper.text()).toContain('个人资料')
    expect(wrapper.text()).toContain('模型测试')
    expect(wrapper.text()).toContain('图片内测')
  })

  it('uses dashboard as the brand home destination', () => {
    const wrapper = mountShell()

    expect(wrapper.get('.ssxz-brand-link').attributes('href')).toBe('/app/dashboard')
  })

  it('keeps long history titles within the sidebar hit target', async () => {
    const wrapper = mountShell({
      historyItems: [
        {
          id: 42,
          title: 'STAGING_173_NO_PROVIDER_FAILURE_ESC_20260626_210940_WITH_A_VERY_LONG_TITLE',
          status: 'active',
          created_at: '2026-06-26T00:00:00Z',
          updated_at: '2026-06-26T00:00:00Z'
        }
      ],
      activeConversationId: null,
      historyLoading: false
    })
    const historyItem = wrapper.get('.ssxz-history-item')
    const historyText = historyItem.get('.ssxz-sidebar-text')

    await historyItem.trigger('click')

    expect(wrapper.emitted('select-conversation')).toEqual([[42]])
    expect(historyText.text()).toContain('STAGING_173_NO_PROVIDER_FAILURE_ESC_20260626_210940_WITH_A_VERY_LONG_TITLE')
    expect(appSectionShellSource).toMatch(/\.ssxz-history-item\s*\{[\s\S]*min-width:\s*0;[\s\S]*max-width:\s*100%;[\s\S]*overflow:\s*hidden;/)
    expect(appSectionShellSource).toMatch(/\.ssxz-history-item \.ssxz-sidebar-text\s*\{[\s\S]*min-width:\s*0;[\s\S]*max-width:\s*100%;/)
  })

  it('switches supported utility menu entries to their own pages instead of rendering inline panels', async () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const buttons = wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item')

    await buttons.find((button) => button.text().includes('使用记录'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/usage')

    await buttons.find((button) => button.text().includes('充值'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/purchase')

    await buttons.find((button) => button.text() === '订单')?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/orders')

    await buttons.find((button) => button.text().includes('兑换码'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/redeem')

    await buttons.find((button) => button.text().includes('API 密钥'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/keys')

    await buttons.find((button) => button.text().includes('个人资料'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/profile')
    expect(wrapper.find('.ssxz-workspace-utility-center').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('打开 API Key / 第三方客户端接入')
  })

  it('keeps the whole workbench sidebar inside user-owned /app routes', async () => {
    const wrapper = mountShell()
    const buttons = [
      ...wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item'),
      ...wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')
    ]
    const expectedRoutes = [
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
    ]

    expect(buttons).toHaveLength(expectedRoutes.length)

    for (const [index, button] of buttons.entries()) {
      routeState.path = '/app/test-origin'
      await button.trigger('click')
      expect(mocks.push).toHaveBeenNthCalledWith(index + 1, expectedRoutes[index])
    }

    const destinations = mocks.push.mock.calls.map(([destination]) => destination)

    expect(destinations).toEqual(expectedRoutes)
    expect(destinations.every((destination) => destination.startsWith('/app/'))).toBe(true)
    expect(destinations).not.toEqual(expect.arrayContaining([
      '/usage',
      '/purchase',
      '/orders',
      '/keys',
      '/profile',
      '/available-channels',
      '/monitor'
    ]))
    expect(wrapper.text()).not.toContain('Available Channels')
    expect(wrapper.text()).not.toContain('Channel Status')
  })

  it('shows the existing affiliate entry only when affiliate is enabled', async () => {
    appState.cachedPublicSettings.affiliate_enabled = true

    const wrapper = mountShell()
    const buttons = [
      ...wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item'),
      ...wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')
    ]

    expect(wrapper.text()).toContain('推广返利')

    const affiliateButton = buttons.find((button) => button.text().includes('推广返利'))
    expect(affiliateButton).toBeTruthy()
    await affiliateButton?.trigger('click')

    expect(mocks.push).toHaveBeenLastCalledWith('/app/affiliate')
  })

  it('hides channel status when monitoring is disabled', async () => {
    appState.cachedPublicSettings.channel_monitor_enabled = false

    const wrapper = mountShell()
    const buttons = [
      ...wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item'),
      ...wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')
    ]
    const expectedRoutes = [
      '/app/dashboard',
      '/app/keys',
      '/app/usage',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/profile',
      '/app/chat',
      '/app/image'
    ]

    expect(wrapper.text()).not.toContain('通道状态')
    expect(buttons).toHaveLength(expectedRoutes.length)

    for (const [index, button] of buttons.entries()) {
      routeState.path = '/app/test-origin'
      await button.trigger('click')
      expect(mocks.push).toHaveBeenNthCalledWith(index + 1, expectedRoutes[index])
    }
  })

  it('keeps the image entry active without highlighting new chat on /app/image', () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const navButtons = wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')

    expect(navButtons[0].text()).toContain('模型测试')
    expect(navButtons[0].classes()).not.toContain('is-active')
    expect(navButtons[1].text()).toContain('图片内测')
    expect(navButtons[1].classes()).toContain('is-active')
  })

  it('starts a new chat through /app/chat instead of the generic /app shell', async () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const navButtons = wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')

    await navButtons[0].trigger('click')

    expect(wrapper.emitted('new-chat')).toHaveLength(1)
    expect(mocks.push).toHaveBeenCalledWith('/app/chat')
    expect(mocks.push).not.toHaveBeenCalledWith('/app')
  })

  it('opens a real mobile navigation drawer instead of only toggling desktop collapse', async () => {
    mockDesktopMedia(false)
    const wrapper = mountShell()

    expect(wrapper.classes()).not.toContain('ssxz-mobile-nav-open')
    await wrapper.get('.ssxz-sidebar-toggle-desktop').trigger('click')

    expect(wrapper.classes()).toContain('ssxz-mobile-nav-open')
    expect(wrapper.find('.ssxz-mobile-sidebar-scrim').exists()).toBe(true)
  })

  it('drops the mobile drawer state when the viewport becomes desktop', async () => {
    mockDesktopMedia(false)
    const wrapper = mountShell()

    await wrapper.get('.ssxz-sidebar-toggle-desktop').trigger('click')
    expect(wrapper.classes()).toContain('ssxz-mobile-nav-open')

    mockDesktopMedia(true)
    window.dispatchEvent(new Event('resize'))
    await nextTick()

    expect(wrapper.classes()).not.toContain('ssxz-mobile-nav-open')
    expect(wrapper.find('.ssxz-mobile-sidebar-scrim').exists()).toBe(false)
  })

  it('closes the mobile drawer when a utility entry changes pages', async () => {
    mockDesktopMedia(false)
    const wrapper = mountShell()

    await wrapper.get('.ssxz-sidebar-toggle-desktop').trigger('click')
    expect(wrapper.classes()).toContain('ssxz-mobile-nav-open')

    const buttons = wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item')
    await buttons[0].trigger('click')

    expect(mocks.push).toHaveBeenLastCalledWith('/app/dashboard')
    expect(wrapper.classes()).not.toContain('ssxz-mobile-nav-open')
    expect(wrapper.find('.ssxz-mobile-sidebar-scrim').exists()).toBe(false)
  })
})
