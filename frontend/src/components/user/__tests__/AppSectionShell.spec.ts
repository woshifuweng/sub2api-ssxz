import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  logout: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/app/chat' }),
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
  it('keeps the lightweight chat shell free of developer API entrypoints', () => {
    const wrapper = mountShell()

    expect(wrapper.text()).toContain('用量中心')
    expect(wrapper.text()).toContain('账户设置')
    expect(wrapper.text()).not.toContain('开发者 API')
  })
})
