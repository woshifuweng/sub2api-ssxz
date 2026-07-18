import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { authStore, appStore, authAPI, userAPI, routeState } = vi.hoisted(() => ({
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
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn()
  },
  authAPI: {
    getPublicSettings: vi.fn()
  },
  userAPI: {
    getAvatar: vi.fn(),
    updateAvatar: vi.fn()
  },
  routeState: {
    path: '/app/profile'
  }
}))

const messages: Record<string, string> = {
  'profile.accountBalance': 'Account balance',
  'profile.accountStatus': 'Account status',
  'profile.administrator': 'Administrator',
  'profile.user': 'User',
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
  'profile.workbench.securityTitle': 'Account security',
  'profile.avatar.uploadTitle': 'Choose an avatar image',
  'profile.avatar.uploadHint': 'JPEG, PNG, or WebP up to 5MB',
  'profile.avatar.change': 'Change avatar'
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

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/api', () => ({
  authAPI,
  userAPI
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

vi.mock('@/components/common/Avatar.vue', () => ({
  default: {
    name: 'Avatar',
    props: ['src', 'name', 'size'],
    template: '<span class="avatar-stub" :data-name="name" :data-size="size" />'
  }
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
    userAPI.getAvatar.mockResolvedValue(null)
    vi.clearAllMocks()
  })

  it('keeps account settings inside the user workbench shell on /app/profile', async () => {
    const wrapper = mount(ProfileView)
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('My account')
    expect(text).toContain('Account settings')
    expect(text).toContain('Test User')
    expect(text).toContain('user@example.test')
    expect(text).toContain('User')
    expect(text).toContain('Basic info')
    expect(text).toContain('Account information')
    expect(text).toContain('Choose an avatar image')
    expect(text).toContain('Change avatar')
    expect(text).toContain('Change password')
    expect(text).toContain('Account security')
    expect(text).toContain('Account balance')
    expect(text).toContain('$49.40')
    expect(text).toContain('Account status')
    expect(text).toContain('Active')
    expect(text).not.toContain('Concurrency Limit')
    expect(wrapper.findAll('.profile-hero-card__stats > div')).toHaveLength(3)
    expect(wrapper.find('.profile-identity-grid').exists()).toBe(true)
    expect(wrapper.findAll('.avatar-stub')).toHaveLength(2)
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
