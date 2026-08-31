import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileView from '@/views/user/ProfileView.vue'

const {
  fetchPublicSettingsMock,
  refreshUserMock,
  updateProfileMock,
  showSuccessMock,
  showErrorMock,
  authState
} = vi.hoisted(() => ({
  fetchPublicSettingsMock: vi.fn(),
  refreshUserMock: vi.fn(),
  updateProfileMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  authState: {
    user: null as Record<string, unknown> | null,
    refreshUser: vi.fn()
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    fetchPublicSettings: fetchPublicSettingsMock,
    showSuccess: showSuccessMock,
    showError: showErrorMock,
  })
}))

vi.mock('@/api', () => ({
  authAPI: {
    getPublicSettings: fetchPublicSettingsMock,
  },
  userAPI: {
    updateProfile: updateProfileMock,
  },
}))

vi.mock('@/utils/format', () => ({
  formatDate: () => 'April 2026'
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/profile' }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('ProfileView', () => {
  beforeEach(() => {
    refreshUserMock.mockReset()
    fetchPublicSettingsMock.mockReset()
    updateProfileMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    refreshUserMock.mockResolvedValue(undefined)
    authState.refreshUser = refreshUserMock
    authState.user = {
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      role: 'user',
      balance: 10,
      concurrency: 2,
      status: 'active',
      allowed_groups: null,
      balance_notify_enabled: true,
      balance_notify_threshold: null,
      balance_notify_extra_emails: [],
      avatar_url: 'https://cdn.example.com/avatar-before.png',
      created_at: '2026-04-20T00:00:00Z',
      updated_at: '2026-04-20T00:00:00Z'
    }
    fetchPublicSettingsMock.mockResolvedValue({
      contact_info: '',
      balance_low_notify_enabled: false,
      balance_low_notify_threshold: 0,
      linuxdo_oauth_enabled: true,
      wechat_oauth_enabled: true,
      wechat_oauth_open_enabled: true,
      wechat_oauth_mp_enabled: false,
      oidc_oauth_enabled: true,
      oidc_oauth_provider_name: 'OIDC'
    })
  })

  it('renders the profile workbench without the retired stat-card surface', async () => {
    const wrapper = mount(ProfileView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          StatCard: { template: '<div class="stat-card" />' },
          Avatar: true,
          LiquidButton: true,
          ProfileEditForm: { template: '<div data-testid="profile-edit-form" />' },
          ProfileBalanceNotifyCard: { template: '<div data-testid="profile-balance-notify-card" />' },
          ProfilePasswordForm: { template: '<div data-testid="profile-password-form" />' },
          ProfileTotpCard: { template: '<div data-testid="profile-totp-card" />' },
          ProfilePasskeyCard: true,
          AvatarCropDialog: true,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.findAll('.stat-card')).toHaveLength(0)
    const workbench = wrapper.get('.profile-workbench')
    expect(workbench.find('.profile-hero-card').exists()).toBe(true)
    expect(workbench.html()).toContain('profile-edit-form')
    expect(workbench.html()).toContain('profile-password-form')
    expect(workbench.html()).toContain('profile-totp-card')
  })

  it('saves avatars through the supported profile endpoint and updates auth state', async () => {
    const updatedUser = {
      ...authState.user,
      avatar_url: 'data:image/png;base64,updated-avatar',
    }
    updateProfileMock.mockResolvedValue(updatedUser)

    const wrapper = mount(ProfileView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Avatar: {
            props: ['src'],
            template: '<div class="avatar-stub" :data-src="src" />',
          },
          LiquidButton: { template: '<button><slot /></button>' },
          ProfileEditForm: true,
          ProfileBalanceNotifyCard: true,
          ProfilePasswordForm: true,
          ProfileTotpCard: true,
          ProfilePasskeyCard: true,
          AvatarCropDialog: {
            emits: ['save'],
            template: '<button data-testid="save-avatar" @click="$emit(\'save\', \'data:image/png;base64,updated-avatar\')" />',
          },
          Icon: true,
        },
      },
    })

    await flushPromises()
    expect(wrapper.get('.avatar-stub').attributes('data-src')).toBe(
      'https://cdn.example.com/avatar-before.png',
    )

    await wrapper.get('[data-testid="save-avatar"]').trigger('click')
    await flushPromises()

    expect(updateProfileMock).toHaveBeenCalledWith({
      avatar_url: 'data:image/png;base64,updated-avatar',
    })
    expect(authState.user).toEqual(updatedUser)
    expect(showSuccessMock).toHaveBeenCalledWith('profile.avatar.saved')
  })
})
