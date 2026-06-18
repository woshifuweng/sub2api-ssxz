import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const { authStore, authAPI } = vi.hoisted(() => ({
  authStore: {
    user: {
      id: 8,
      username: '测试用户',
      email: 'user@example.test',
      role: 'user',
      balance: 49.4,
      concurrency: 10,
      status: 'active',
      allowed_groups: null,
      created_at: '2026-05-01T00:00:00Z',
      updated_at: '2026-05-01T00:00:00Z'
    }
  },
  authAPI: {
    getPublicSettings: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'profile.accountBalance': '账户余额',
  'profile.accountStatus': '账户状态',
  'profile.statusActive': '正常',
  'profile.statusDisabled': '已停用',
  'profile.memberSince': '注册时间'
}

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: { value: 'zh-CN' },
      t: (key: string) => messages[key] || key
    }
  }),
  useI18n: () => ({
    t: (key: string) => messages[key] || key
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/app/profile' })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/api', () => ({
  authAPI
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    template: '<main data-testid="app-section-shell"><slot /></main>'
  }
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/common/StatCard.vue', () => ({
  default: {
    name: 'StatCard',
    props: ['title', 'value'],
    template: '<article class="stat-card-stub"><span>{{ title }}</span><strong>{{ value }}</strong></article>'
  }
}))

vi.mock('@/components/user/profile/ProfileInfoCard.vue', () => ({
  default: { name: 'ProfileInfoCard', template: '<section />' }
}))

vi.mock('@/components/user/profile/ProfileEditForm.vue', () => ({
  default: { name: 'ProfileEditForm', template: '<section />' }
}))

vi.mock('@/components/user/profile/ProfilePasswordForm.vue', () => ({
  default: { name: 'ProfilePasswordForm', template: '<section />' }
}))

vi.mock('@/components/user/profile/ProfileTotpCard.vue', () => ({
  default: { name: 'ProfileTotpCard', template: '<section />' }
}))

vi.mock('@/components/icons', () => ({
  Icon: { name: 'Icon', template: '<span />' }
}))

import ProfileView from '../ProfileView.vue'

describe('ProfileView', () => {
  it('presents account status instead of the technical concurrency limit', async () => {
    authAPI.getPublicSettings.mockResolvedValue({})

    const wrapper = mount(ProfileView)
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('管理你的登录信息和安全验证')
    expect(text).toContain('基础资料')
    expect(text).toContain('编辑个人资料')
    expect(text).toContain('修改密码')
    expect(text).toContain('账号安全')
    expect(text).toContain('账户状态')
    expect(text).toContain('正常')
    expect(text).not.toContain('并发限制')
  })
})
