import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import SettingsView from '../SettingsView.vue'

const {
  settingsAPI,
  groupsGetAll,
  showError,
  showSuccess,
  fetchPublicSettings,
  adminSettingsFetch
} = vi.hoisted(() => ({
  settingsAPI: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    testSmtpConnection: vi.fn(),
    sendTestEmail: vi.fn(),
    getAdminApiKey: vi.fn(),
    regenerateAdminApiKey: vi.fn(),
    deleteAdminApiKey: vi.fn(),
    getOverloadCooldownSettings: vi.fn(),
    updateOverloadCooldownSettings: vi.fn(),
    getStreamTimeoutSettings: vi.fn(),
    updateStreamTimeoutSettings: vi.fn(),
    getRectifierSettings: vi.fn(),
    updateRectifierSettings: vi.fn(),
    getBetaPolicySettings: vi.fn(),
    updateBetaPolicySettings: vi.fn(),
    getTLSFingerprintSettings: vi.fn(),
    updateTLSFingerprintSettings: vi.fn(),
    createTLSFingerprintProfile: vi.fn(),
    updateTLSFingerprintProfile: vi.fn(),
    deleteTLSFingerprintProfile: vi.fn()
  },
  groupsGetAll: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  fetchPublicSettings: vi.fn(),
  adminSettingsFetch: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: settingsAPI,
    groups: {
      getAll: groupsGetAll
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    fetchPublicSettings
  })
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: adminSettingsFetch
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('admin SettingsView', () => {
  const baseSettings = () => ({
    registration_enabled: true,
    email_verify_enabled: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    frontend_url: '',
    invitation_code_enabled: false,
    totp_enabled: false,
    totp_encryption_key_configured: false,
    default_balance: 0,
    default_concurrency: 1,
    default_subscriptions: [],
    affiliate_enabled: true,
    affiliate_rebate_rate: 10,
    affiliate_rebate_freeze_hours: 0,
    affiliate_rebate_duration_days: 0,
    affiliate_rebate_per_invitee_cap: 0,
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    purchase_subscription_enabled: false,
    purchase_subscription_url: '',
    sora_client_enabled: false,
    backend_mode_enabled: false,
    custom_menu_items: [],
    smtp_host: '',
    smtp_port: 587,
    smtp_username: '',
    smtp_password_configured: false,
    smtp_from_email: '',
    smtp_from_name: '',
    smtp_use_tls: true,
    turnstile_enabled: false,
    turnstile_site_key: '',
    turnstile_secret_key_configured: false,
    linuxdo_connect_enabled: false,
    linuxdo_connect_client_id: '',
    linuxdo_connect_client_secret_configured: false,
    linuxdo_connect_redirect_url: '',
    enable_model_fallback: false,
    fallback_model_anthropic: '',
    fallback_model_openai: '',
    fallback_model_gemini: '',
    fallback_model_antigravity: '',
    enable_identity_patch: true,
    identity_patch_prompt: '',
    ops_monitoring_enabled: true,
    ops_realtime_monitoring_enabled: true,
    ops_query_mode_default: 'auto',
    ops_metrics_interval_seconds: 60,
    min_claude_code_version: '',
    max_claude_code_version: '',
    allow_ungrouped_key_scheduling: false,
    auto_delete_401_accounts: false,
    auto_delete_429_accounts: false,
    auto_delete_useless_proxies: false
  })

  beforeEach(() => {
    vi.clearAllMocks()
    settingsAPI.getSettings.mockResolvedValue(baseSettings())
    settingsAPI.updateSettings.mockImplementation(async (payload: unknown) => payload)
    settingsAPI.getAdminApiKey.mockResolvedValue({ exists: false, masked_key: '' })
    settingsAPI.getOverloadCooldownSettings.mockResolvedValue({ enabled: true, cooldown_minutes: 10 })
    settingsAPI.getStreamTimeoutSettings.mockResolvedValue({
      enabled: false,
      action: 'temp_unsched',
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10
    })
    settingsAPI.getRectifierSettings.mockResolvedValue({
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true
    })
    settingsAPI.getBetaPolicySettings.mockResolvedValue({ rules: [] })
    settingsAPI.getTLSFingerprintSettings.mockResolvedValue({
      enabled: true,
      items: [
        {
          profile_id: 'alpha',
          name: 'Alpha',
          enabled: true,
          enable_grease: false,
          cipher_suites: [4866, 4867],
          curves: [29, 23],
          point_formats: [],
          updated_at: '2026-03-28T08:00:00Z'
        }
      ]
    })
    groupsGetAll.mockResolvedValue([])
  })

  async function mountView(overrides: Record<string, unknown> = {}) {
    settingsAPI.getSettings.mockResolvedValue({ ...baseSettings(), ...overrides })

    const wrapper = mount(SettingsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
          Select: { template: '<select><slot /></select>' },
          GroupBadge: true,
          GroupOptionItem: true,
          Toggle: { template: '<input type="checkbox" />' },
          ImageUpload: true,
          BackupSettings: true,
          DataManagementSettings: true
        }
      }
    })

    await flushPromises()
    return wrapper
  }

  const RESET_WARNING = '[data-testid="settings-frontend-url-missing-warning"]'
  const FRONTEND_URL_INPUT = '[data-testid="settings-frontend-url-input"]'
  const FRONTEND_URL_FORMAT_HINT = '[data-testid="settings-frontend-url-format-hint"]'
  const CONTACT_INFO_HINT = '[data-testid="settings-contact-info-missing-hint"]'

  it('warns when password reset is enabled but frontend URL is empty', async () => {
    const wrapper = await mountView({
      email_verify_enabled: true,
      password_reset_enabled: true,
      frontend_url: ''
    })

    expect(wrapper.find(RESET_WARNING).exists()).toBe(true)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  it('still warns when email verification is disabled but password reset is enabled', async () => {
    const wrapper = await mountView({
      email_verify_enabled: false,
      password_reset_enabled: true,
      frontend_url: ''
    })

    // Regression: the warning and the input must not be hidden by the email-verify condition
    expect(wrapper.find(RESET_WARNING).exists()).toBe(true)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  it('hides the frontend URL warning once a URL is configured', async () => {
    const wrapper = await mountView({
      email_verify_enabled: false,
      password_reset_enabled: true,
      frontend_url: 'https://example.com'
    })

    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
    expect(wrapper.find(FRONTEND_URL_FORMAT_HINT).exists()).toBe(false)
  })

  it('does not warn about the frontend URL when password reset is disabled', async () => {
    const wrapper = await mountView({
      password_reset_enabled: false,
      frontend_url: ''
    })

    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
  })

  it('flags a malformed frontend URL inline instead of clearing it', async () => {
    const wrapper = await mountView({
      password_reset_enabled: true,
      frontend_url: 'example.com'
    })

    expect(wrapper.find(FRONTEND_URL_FORMAT_HINT).exists()).toBe(true)

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    // The admin's input must survive: silently blanking it re-arms the silent-failure bug
    expect(settingsAPI.updateSettings).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.settings.registration.frontendUrlInvalidError')
    expect((wrapper.find(FRONTEND_URL_INPUT).element as HTMLInputElement).value).toBe('example.com')
  })

  it('hints when contact info is empty and hides the hint once filled', async () => {
    const emptyWrapper = await mountView({ contact_info: '' })
    expect(emptyWrapper.find(CONTACT_INFO_HINT).exists()).toBe(true)

    const filledWrapper = await mountView({ contact_info: 'QQ: 123456789' })
    expect(filledWrapper.find(CONTACT_INFO_HINT).exists()).toBe(false)
  })

  it('loads and renders TLS fingerprint profiles in gateway tab', async () => {
    const wrapper = await mountView()

    expect(settingsAPI.getTLSFingerprintSettings).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Alpha')
    expect(wrapper.text()).toContain('alpha')
    expect(wrapper.text()).toContain('admin.settings.tlsFingerprint.title')
    expect(wrapper.text()).toContain('推广返利')
  })
})
