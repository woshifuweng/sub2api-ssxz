import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { authStore, authAPI, routeState } = vi.hoisted(() => ({
  authStore: {
    user: {
      id: 8,
      username: 'Test User',
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
  },
  routeState: {
    path: '/app/profile'
  }
}))

const messages: Record<string, string> = {
  'profile.accountBalance': 'Account balance',
  'profile.accountStatus': 'Account status',
  'profile.statusActive': 'Active',
  'profile.statusDisabled': 'Disabled',
  'profile.memberSince': 'Member since',
  'profile.workbench.title': 'Account settings',
  'profile.workbench.subtitle': 'Review account information and update profile, password, and security verification settings.',
  'profile.workbench.eyebrow': 'My account',
  'profile.workbench.introAriaLabel': 'Account settings explanation',
  'profile.workbench.introKicker': 'Account and security',
  'profile.workbench.introTitle': 'Manage your login information and security verification',
  'profile.workbench.introDescription': 'This page only handles your profile, password, and two-factor verification.',
  'profile.workbench.basicInfoKicker': 'Basic info',
  'profile.workbench.accountInfoTitle': 'Account information',
  'profile.workbench.displayNameKicker': 'Display name',
  'profile.workbench.editProfileTitle': 'Edit profile',
  'profile.workbench.loginProtectionKicker': 'Login protection',
  'profile.workbench.changePasswordTitle': 'Change password',
  'profile.workbench.twoFactorKicker': 'Two-factor verification',
  'profile.workbench.securityTitle': 'Account security'
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
  useRoute: () => routeState
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
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: `
      <main data-testid="app-section-shell">
        <span>{{ eyebrow }}</span>
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
        <slot />
      </main>
    `
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
  beforeEach(() => {
    routeState.path = '/app/profile'
    authAPI.getPublicSettings.mockResolvedValue({})
    vi.clearAllMocks()
  })

  it('keeps account settings inside the user workbench shell on /app/profile', async () => {
    const wrapper = mount(ProfileView)
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('My account')
    expect(text).toContain('Account settings')
    expect(text).toContain('Manage your login information and security verification')
    expect(text).toContain('Basic info')
    expect(text).toContain('Edit profile')
    expect(text).toContain('Change password')
    expect(text).toContain('Account security')
    expect(text).toContain('Account status')
    expect(text).toContain('Active')
    expect(text).not.toContain('Concurrency Limit')
  })

  it('keeps the legacy profile surface on /profile for compatibility', async () => {
    routeState.path = '/profile'

    const wrapper = mount(ProfileView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })

  it('silently omits optional support contact when public settings cannot load', async () => {
    authAPI.getPublicSettings.mockRejectedValue(new Error('settings unavailable'))
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => undefined)

    const wrapper = mount(ProfileView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('common.contactSupport')
    expect(consoleErrorSpy).not.toHaveBeenCalled()
    consoleErrorSpy.mockRestore()
  })
})
