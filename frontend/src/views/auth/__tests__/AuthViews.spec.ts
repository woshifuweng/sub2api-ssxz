import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '../LoginView.vue'
import RegisterView from '../RegisterView.vue'

const mocks = vi.hoisted(() => ({
  route: { query: {} as Record<string, string> },
  push: vi.fn(),
  getPublicSettings: vi.fn(),
  validatePromoCode: vi.fn(),
  validateInvitationCode: vi.fn(),
  login: vi.fn(),
  login2FA: vi.fn(),
  register: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.push })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: mocks.login,
    login2FA: mocks.login2FA,
    register: mocks.register
  }),
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
    showWarning: mocks.showWarning
  })
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: mocks.getPublicSettings,
  validatePromoCode: mocks.validatePromoCode,
  validateInvitationCode: mocks.validateInvitationCode,
  isTotp2FARequired: (response: { requires_2fa?: boolean }) => response?.requires_2fa === true
}))

const AuthPortalShellStub = defineComponent({
  name: 'AuthPortalShell',
  template: '<section data-testid="auth-shell"><slot /></section>'
})

const TotpLoginModalStub = defineComponent({
  name: 'TotpLoginModal',
  template: '<div data-testid="totp-modal" />',
  methods: {
    setVerifying() {},
    setError() {}
  }
})

function publicSettings(overrides: Record<string, unknown> = {}) {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    promo_code_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'SSXZ AI',
    linuxdo_oauth_enabled: false,
    backend_mode_enabled: false,
    password_reset_enabled: true,
    registration_email_suffix_whitelist: [],
    ...overrides
  }
}

function mountLogin() {
  return mount(LoginView, {
    attachTo: document.body,
    global: {
      stubs: {
        AuthPortalShell: AuthPortalShellStub,
        LinuxDoOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TurnstileWidget: true,
        TotpLoginModal: TotpLoginModalStub
      }
    }
  })
}

function mountRegister() {
  return mount(RegisterView, {
    attachTo: document.body,
    global: {
      stubs: {
        AuthPortalShell: AuthPortalShellStub,
        LinuxDoOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TurnstileWidget: true
      }
    }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  sessionStorage.clear()
  mocks.route.query = {}
  mocks.getPublicSettings.mockResolvedValue(publicSettings())
})

afterEach(() => {
  document.body.innerHTML = ''
  sessionStorage.clear()
})

describe('LoginView C2 integration', () => {
  it('keeps the real password login and safe return target flow', async () => {
    mocks.route.query = { returnTo: '/app/keys' }
    mocks.login.mockResolvedValue({ access_token: 'test-token' })
    const wrapper = mountLogin()
    await flushPromises()

    await wrapper.get('#email').setValue('customer@example.com')
    await wrapper.get('#password').setValue('strong-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.login).toHaveBeenCalledWith({
      email: 'customer@example.com',
      password: 'strong-password',
      turnstile_token: undefined
    })
    expect(mocks.push).toHaveBeenCalledWith('/app/keys')
  })

  it('still opens the TOTP step without redirecting after the first factor', async () => {
    mocks.login.mockResolvedValue({
      requires_2fa: true,
      temp_token: 'temporary-token',
      user_email_masked: 'c***@example.com'
    })
    const wrapper = mountLogin()
    await flushPromises()

    await wrapper.get('#email').setValue('customer@example.com')
    await wrapper.get('#password').setValue('strong-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-testid="totp-modal"]').exists()).toBe(true)
    expect(mocks.push).not.toHaveBeenCalled()
  })
})

describe('RegisterView C2 integration', () => {
  it('sends the /register?aff= attribution in the direct registration payload', async () => {
    mocks.route.query = { aff: 'AFF-2026' }
    mocks.register.mockResolvedValue({ access_token: 'test-token' })
    const wrapper = mountRegister()
    await flushPromises()

    await wrapper.get('#email').setValue('new-user@example.com')
    await wrapper.get('#password').setValue('strong-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.register).toHaveBeenCalledWith({
      email: 'new-user@example.com',
      password: 'strong-password',
      turnstile_token: undefined,
      promo_code: undefined,
      invitation_code: undefined,
      affiliate_code: 'AFF-2026'
    })
  })

  it('keeps affiliate attribution through the email verification handoff', async () => {
    mocks.route.query = { affiliate: 'AFF-VERIFY' }
    mocks.getPublicSettings.mockResolvedValue(publicSettings({ email_verify_enabled: true }))
    const wrapper = mountRegister()
    await flushPromises()

    await wrapper.get('#email').setValue('verify@example.com')
    await wrapper.get('#password').setValue('strong-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    const pending = JSON.parse(sessionStorage.getItem('register_data') || '{}')
    expect(pending).toMatchObject({
      email: 'verify@example.com',
      affiliate_code: 'AFF-VERIFY'
    })
    expect(mocks.register).not.toHaveBeenCalled()
    expect(mocks.push).toHaveBeenCalledWith('/email-verify')
  })
})
