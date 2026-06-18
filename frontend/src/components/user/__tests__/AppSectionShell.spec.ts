import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

const { routeState, mocks } = vi.hoisted(() => ({
  routeState: {
    path: '/app/chat'
  },
  mocks: {
    push: vi.fn(),
    logout: vi.fn(),
    showSuccess: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: mocks.push })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
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

function mountShell() {
  return mount(AppSectionShell, {
    props: {
      title: '聊天',
      subtitle: '辅助写 prompt'
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
    mockDesktopMedia(true)
  })

  it('keeps API Key available as a third-party client entrypoint', () => {
    const wrapper = mountShell()

    expect(wrapper.text()).toContain('AI 创作工作台')
    expect(wrapper.text()).toContain('SSXZ AI 工作台')
    expect(wrapper.text()).not.toContain('图片工具站')
    expect(wrapper.text()).not.toContain('对话工作台')
    expect(wrapper.text()).toContain('新对话')
    expect(wrapper.text()).toContain('AI 作图')
    expect(wrapper.text()).toContain('用量中心')
    expect(wrapper.text()).toContain('充值')
    expect(wrapper.text()).toContain('订单记录')
    expect(wrapper.text()).toContain('兑换码')
    expect(wrapper.text()).toContain('API Key / 第三方接入')
    expect(wrapper.text()).toContain('账户设置')
  })

  it('uses the image workbench as the brand home destination', () => {
    const wrapper = mountShell()

    expect(wrapper.get('.ssxz-brand-link').attributes('href')).toBe('/app/image')
  })

  it('switches supported utility menu entries to their own pages instead of rendering inline panels', async () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const buttons = wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')

    await buttons.find((button) => button.text().includes('用量中心'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/usage')

    await buttons.find((button) => button.text().includes('充值'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/purchase')

    await buttons.find((button) => button.text().includes('订单记录'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/orders')

    await buttons.find((button) => button.text().includes('兑换码'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/redeem')

    await buttons.find((button) => button.text().includes('API Key / 第三方接入'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/keys')

    await buttons.find((button) => button.text().includes('账户设置'))?.trigger('click')
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
      '/app/chat',
      '/app/image',
      '/app/usage',
      '/app/purchase',
      '/app/orders',
      '/app/redeem',
      '/app/keys',
      '/app/profile'
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
      '/channel-status',
      '/monitor'
    ]))
    expect(wrapper.text()).not.toContain('Available Channels')
    expect(wrapper.text()).not.toContain('Channel Status')
  })

  it('keeps the image entry active without highlighting new chat on /app/image', () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const navButtons = wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item')

    expect(navButtons[0].text()).toContain('新对话')
    expect(navButtons[0].classes()).not.toContain('is-active')
    expect(navButtons[1].text()).toContain('AI 作图')
    expect(navButtons[1].classes()).toContain('is-active')
  })

  it('starts a new chat through /app/chat instead of the generic /app shell', async () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const navButtons = wrapper.findAll('.ssxz-primary-nav .ssxz-nav-item')

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

    const buttons = wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')
    await buttons[0].trigger('click')

    expect(mocks.push).toHaveBeenLastCalledWith('/app/usage')
    expect(wrapper.classes()).not.toContain('ssxz-mobile-nav-open')
    expect(wrapper.find('.ssxz-mobile-sidebar-scrim').exists()).toBe(false)
  })
})
