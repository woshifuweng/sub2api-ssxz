import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { appStore, authStore, userAPI } = vi.hoisted(() => ({
  appStore: {
    showSuccess: vi.fn(),
    showError: vi.fn()
  },
  authStore: {
    user: {
      id: 8,
      username: '旧昵称',
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
  userAPI: {
    updateProfile: vi.fn(),
    changePassword: vi.fn()
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/api', () => ({
  userAPI
}))

import ProfileEditForm from '../ProfileEditForm.vue'
import ProfilePasswordForm from '../ProfilePasswordForm.vue'

describe('ProfileEditForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authStore.user.username = '旧昵称'
  })

  it('does not submit an empty username', async () => {
    const wrapper = mount(ProfileEditForm, {
      props: { initialUsername: '旧昵称' }
    })

    await wrapper.get('#username').setValue('   ')
    await wrapper.get('form').trigger('submit')

    expect(userAPI.updateProfile).not.toHaveBeenCalled()
    expect(appStore.showError).toHaveBeenCalledWith('profile.usernameRequired')
  })

  it('updates the auth store after a successful profile save', async () => {
    const updatedUser = {
      ...authStore.user,
      username: '新昵称'
    }
    userAPI.updateProfile.mockResolvedValue(updatedUser)

    const wrapper = mount(ProfileEditForm, {
      props: { initialUsername: '旧昵称' }
    })

    await wrapper.get('#username').setValue('新昵称')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(userAPI.updateProfile).toHaveBeenCalledWith({ username: '新昵称' })
    expect(authStore.user).toEqual(updatedUser)
    expect(appStore.showSuccess).toHaveBeenCalledWith('profile.updateSuccess')
  })
})

describe('ProfilePasswordForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not submit when new passwords do not match', async () => {
    const wrapper = mount(ProfilePasswordForm)

    await wrapper.get('#old_password').setValue('old-password')
    await wrapper.get('#new_password').setValue('new-password')
    await wrapper.get('#confirm_password').setValue('different-password')
    await wrapper.get('form').trigger('submit')

    expect(userAPI.changePassword).not.toHaveBeenCalled()
    expect(appStore.showError).toHaveBeenCalledWith('profile.passwordsNotMatch')
  })

  it('does not submit when the new password is too short', async () => {
    const wrapper = mount(ProfilePasswordForm)

    await wrapper.get('#old_password').setValue('old-password')
    await wrapper.get('#new_password').setValue('short')
    await wrapper.get('#confirm_password').setValue('short')
    await wrapper.get('form').trigger('submit')

    expect(userAPI.changePassword).not.toHaveBeenCalled()
    expect(appStore.showError).toHaveBeenCalledWith('profile.passwordTooShort')
  })

  it('submits valid password changes and clears password fields on success', async () => {
    userAPI.changePassword.mockResolvedValue({ message: 'ok' })
    const wrapper = mount(ProfilePasswordForm)

    await wrapper.get('#old_password').setValue('old-password')
    await wrapper.get('#new_password').setValue('new-password')
    await wrapper.get('#confirm_password').setValue('new-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(userAPI.changePassword).toHaveBeenCalledWith('old-password', 'new-password')
    expect((wrapper.get('#old_password').element as HTMLInputElement).value).toBe('')
    expect((wrapper.get('#new_password').element as HTMLInputElement).value).toBe('')
    expect((wrapper.get('#confirm_password').element as HTMLInputElement).value).toBe('')
    expect(appStore.showSuccess).toHaveBeenCalledWith('profile.passwordChangeSuccess')
  })

  it('shows backend password change failures without clearing password fields', async () => {
    userAPI.changePassword.mockRejectedValue({
      response: {
        data: {
          detail: 'current password is incorrect'
        }
      }
    })
    const wrapper = mount(ProfilePasswordForm)

    await wrapper.get('#old_password').setValue('wrong-password')
    await wrapper.get('#new_password').setValue('new-password')
    await wrapper.get('#confirm_password').setValue('new-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(userAPI.changePassword).toHaveBeenCalledWith('wrong-password', 'new-password')
    expect((wrapper.get('#old_password').element as HTMLInputElement).value).toBe('wrong-password')
    expect((wrapper.get('#new_password').element as HTMLInputElement).value).toBe('new-password')
    expect((wrapper.get('#confirm_password').element as HTMLInputElement).value).toBe('new-password')
    expect(appStore.showError).toHaveBeenCalledWith('current password is incorrect')
    expect(appStore.showSuccess).not.toHaveBeenCalled()
  })
})
