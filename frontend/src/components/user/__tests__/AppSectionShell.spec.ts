import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

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
    expect(wrapper.text()).toContain('API Key / 第三方接入')
    expect(wrapper.text()).toContain('账户设置')
  })

  it('switches supported utility menu entries to their own pages instead of rendering inline panels', async () => {
    routeState.path = '/app/image'
    const wrapper = mountShell()
    const buttons = wrapper.findAll('.ssxz-secondary-nav .ssxz-nav-item')

    await buttons.find((button) => button.text().includes('用量中心'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/usage')

    await buttons.find((button) => button.text().includes('API Key / 第三方接入'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/keys')

    await buttons.find((button) => button.text().includes('账户设置'))?.trigger('click')
    expect(mocks.push).toHaveBeenLastCalledWith('/app/profile')
    expect(wrapper.find('.ssxz-workspace-utility-center').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('打开 API Key / 第三方客户端接入')
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
})
