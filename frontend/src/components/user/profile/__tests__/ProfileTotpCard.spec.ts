import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { totpAPI } = vi.hoisted(() => ({
  totpAPI: {
    getStatus: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'profile.totp.title': 'Two-factor authentication',
  'profile.totp.description': 'Protect your account',
  'profile.totp.featureDisabled': 'Feature unavailable',
  'profile.totp.featureDisabledHint': 'Two-factor authentication is disabled by the operator.',
  'profile.totp.enabled': 'Enabled',
  'profile.totp.enabledAt': 'Enabled at',
  'profile.totp.disable': 'Disable',
  'profile.totp.notEnabled': 'Not enabled',
  'profile.totp.notEnabledHint': 'Enable two-factor authentication to protect your account.',
  'profile.totp.enable': 'Enable',
  'profile.totp.statusLoadFailed': 'Two-factor status is temporarily unavailable',
  'profile.totp.statusLoadFailedHint': 'Unknown status is not shown as not enabled.',
  'profile.totp.retryStatus': 'Retry'
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] || key
  })
}))

vi.mock('@/api', () => ({
  totpAPI
}))

vi.mock('../TotpSetupModal.vue', () => ({
  default: {
    name: 'TotpSetupModal',
    template: '<section data-testid="totp-setup-modal" />'
  }
}))

vi.mock('../TotpDisableDialog.vue', () => ({
  default: {
    name: 'TotpDisableDialog',
    template: '<section data-testid="totp-disable-dialog" />'
  }
}))

import ProfileTotpCard from '../ProfileTotpCard.vue'

describe('ProfileTotpCard', () => {
  beforeEach(() => {
    totpAPI.getStatus.mockReset()
  })

  it('does not show unknown TOTP status as not enabled when status loading fails', async () => {
    totpAPI.getStatus.mockRejectedValueOnce(new Error('status unavailable'))

    const wrapper = mount(ProfileTotpCard)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Two-factor status is temporarily unavailable')
    expect(text).toContain('Unknown status is not shown as not enabled.')
    expect(text).toContain('Retry')
    expect(text).not.toContain('Not enabled')
    expect(text).not.toContain('Enable two-factor authentication to protect your account.')
  })

  it('retries loading status from the explicit failure state', async () => {
    totpAPI.getStatus
      .mockRejectedValueOnce(new Error('status unavailable'))
      .mockResolvedValueOnce({
        feature_enabled: true,
        enabled: false,
        enabled_at: null
      })

    const wrapper = mount(ProfileTotpCard)
    await flushPromises()

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(totpAPI.getStatus).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Not enabled')
    expect(wrapper.text()).not.toContain('Two-factor status is temporarily unavailable')
  })
})
