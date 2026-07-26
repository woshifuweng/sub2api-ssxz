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
    password_reset_enabled_stored: false,
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
  const RESET_LATENT_WARNING = '[data-testid="settings-frontend-url-latent-warning"]'
  const FRONTEND_URL_INPUT = '[data-testid="settings-frontend-url-input"]'
  const FRONTEND_URL_FORMAT_HINT = '[data-testid="settings-frontend-url-format-hint"]'
  const CONTACT_INFO_HINT = '[data-testid="settings-contact-info-missing-hint"]'

  /**
   * 构造一个「真实 API 可能返回」的密码重置相关响应片段。
   *
   * 后端 parseSettings 把 password_reset_enabled 与 email_verify_enabled 取与后才回传，
   * 所以 { email_verify_enabled: false, password_reset_enabled: true } 这种组合
   * **永远不会**出现在真实响应里。之前的用例直接手写了这个不可能的形状，
   * 缺陷仍在时也会绿。这里由 helper 强制取与，让假 fixture 无法被构造出来。
   */
  function passwordResetApiShape(options: {
    emailVerify: boolean
    /** DB 里 password_reset_enabled 的原始存储值 */
    storedPasswordReset: boolean
    frontendUrl?: string
    /**
     * 后端解析出的重置链接基址。真实后端的规则是「DB 值 → config.yaml 回落 → Origin 兜底」，
     * 所以它可以在 frontend_url 为空时依然非空。默认让它跟随 frontendUrl，
     * 需要构造「DB 空但配置文件已配」时显式传入。
     */
    resetLinkBase?: string
  }) {
    return {
      email_verify_enabled: options.emailVerify,
      // 生效值：后端一定是取与后的结果
      password_reset_enabled: options.emailVerify && options.storedPasswordReset,
      // 观测值：原始存储值，不取与
      password_reset_enabled_stored: options.storedPasswordReset,
      frontend_url: options.frontendUrl ?? '',
      password_reset_link_base: options.resetLinkBase ?? options.frontendUrl ?? ''
    }
  }

  // 状态 A「正在静默失败」：邮箱验证开 + 重置开 + 前端地址空。
  it('warns that password reset is silently failing right now (state A)', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: true, storedPasswordReset: true })
    )

    expect(wrapper.find(RESET_WARNING).exists()).toBe(true)
    // 状态 B 的「尚未生效」文案在这里是错的，绝不能同时出现
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  // 误报回归：DB 的 frontend_url 为空，但后端通过 config.yaml 的 server.frontend_url
  // 回落解析出了链接基址 —— 邮件其实发得出去，绝不能打「客户收不到邮件」的紧急告警。
  // 这条用来钉死「前端不得拿 frontend_url 原始值自行推断」。
  it('does not warn when the backend resolved a reset link base from the config file', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({
        emailVerify: true,
        storedPasswordReset: true,
        frontendUrl: '',
        resetLinkBase: 'https://fallback.example.com'
      })
    )

    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
    // 输入框仍要可见可填，方便管理员把地址显式写进 DB
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  // 同上，但处于状态 B：解析得出链接就同样不该有潜伏告警
  it('does not warn in the latent state when the reset link base resolves', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({
        emailVerify: false,
        storedPasswordReset: true,
        frontendUrl: '',
        resetLinkBase: 'https://fallback.example.com'
      })
    )

    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
  })

  // 管理员正在输入时不该边填边报错：表单已有合法地址即视为可解析，
  // 哪怕后端上一次返回的解析结果还是空的。
  it('clears the warning as soon as a valid URL is typed, before saving', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: true, storedPasswordReset: true })
    )
    expect(wrapper.find(RESET_WARNING).exists()).toBe(true)

    const vm = wrapper.vm as unknown as { form: Record<string, unknown> }
    vm.form.frontend_url = 'https://typed.example.com'
    await flushPromises()

    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
  })

  // 状态 B「潜伏」：DB 里存着 true，但邮箱验证关闭 —— 真实响应里 password_reset_enabled 是 false。
  // 这是坑 A 的回归用例：只要渲染判据回到 form.password_reset_enabled，这条必然变红。
  it('warns about the latent misconfiguration when email verification is off (state B)', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: false, storedPasswordReset: true })
    )

    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(true)
    // 此刻并没有在静默失败，状态 A 的紧急文案属于误报
    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  // 状态 B 的文案原话是「一旦你开启邮箱验证，重置邮件会立即开始静默失败」，
  // 管理员照做就会走进这个过渡态。若 form 持有的是后端取与后的 false 而非原始存储值，
  // 打开邮箱验证的瞬间：潜伏告警消失、状态 A 告警不出现、输入框消失 —— 文案把人引进坑里。
  it('escalates from the latent warning to the live warning when email verification is switched on', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: false, storedPasswordReset: true })
    )

    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(true)
    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)

    // 管理员按文案指引打开「邮箱验证」
    const vm = wrapper.vm as unknown as { form: Record<string, unknown> }
    vm.form.email_verify_enabled = true
    await flushPromises()

    // 必须升级为状态 A，而不是整块塌陷
    expect(wrapper.find(RESET_WARNING).exists()).toBe(true)
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  // 同一过渡态下保存：DB 里的 true 不能被抹成 false
  it('does not clobber the stored password reset flag across the email verification switch', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: false, storedPasswordReset: true })
    )

    const vm = wrapper.vm as unknown as { form: Record<string, unknown> }
    vm.form.email_verify_enabled = true
    vm.form.frontend_url = 'https://example.com'
    await flushPromises()

    await (wrapper.vm as unknown as { saveSettings: () => Promise<void> }).saveSettings()
    await flushPromises()

    expect(settingsAPI.updateSettings).toHaveBeenCalled()
    const payload = settingsAPI.updateSettings.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(payload.password_reset_enabled).toBe(true)
  })

  it('hides the frontend URL warning once a URL is configured (state A)', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({
        emailVerify: true,
        storedPasswordReset: true,
        frontendUrl: 'https://example.com'
      })
    )

    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
    expect(wrapper.find(FRONTEND_URL_FORMAT_HINT).exists()).toBe(false)
  })

  it('hides the frontend URL warning once a URL is configured (state B)', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({
        emailVerify: false,
        storedPasswordReset: true,
        frontendUrl: 'https://example.com'
      })
    )

    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
  })

  it('does not warn about the frontend URL when password reset was never enabled', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: true, storedPasswordReset: false })
    )

    expect(wrapper.find(RESET_WARNING).exists()).toBe(false)
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(false)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(false)
  })

  // 老后端不返回 password_reset_enabled_stored 时必须回退到生效值，而不是当成 false 把整块藏掉。
  it('falls back to the effective value when the backend omits the stored field', async () => {
    const wrapper = await mountView({
      email_verify_enabled: true,
      password_reset_enabled: true,
      password_reset_enabled_stored: undefined,
      frontend_url: ''
    })

    expect(wrapper.find(RESET_WARNING).exists()).toBe(true)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  it('flags a malformed frontend URL inline instead of clearing it', async () => {
    const wrapper = await mountView(
      passwordResetApiShape({
        emailVerify: true,
        storedPasswordReset: true,
        frontendUrl: 'example.com'
      })
    )

    expect(wrapper.find(FRONTEND_URL_FORMAT_HINT).exists()).toBe(true)

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    // The admin's input must survive: silently blanking it re-arms the silent-failure bug
    expect(settingsAPI.updateSettings).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.settings.registration.frontendUrlInvalidError')
    expect((wrapper.find(FRONTEND_URL_INPUT).element as HTMLInputElement).value).toBe('example.com')
  })

  // 保存后失真：Object.assign(form, updated) 会把 password_reset_enabled 刷成取与后的 false。
  // 告警必须跟着落库的原始值走，保存成功不能让状态 B 的提示凭空消失。
  it('keeps the latent warning visible after a successful save (state B)', async () => {
    settingsAPI.updateSettings.mockImplementation(async (payload: any) => ({
      ...baseSettings(),
      ...payload,
      // 真实后端的响应形状：生效值取与，stored 是刚落库的原始值
      password_reset_enabled: Boolean(payload.email_verify_enabled && payload.password_reset_enabled),
      password_reset_enabled_stored: Boolean(payload.password_reset_enabled)
    }))

    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: false, storedPasswordReset: true })
    )
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(true)

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(settingsAPI.updateSettings).toHaveBeenCalledTimes(1)
    expect(showSuccess).toHaveBeenCalled()
    expect(wrapper.find(RESET_LATENT_WARNING).exists()).toBe(true)
    expect(wrapper.find(FRONTEND_URL_INPUT).exists()).toBe(true)
  })

  // 「忘记密码」开关在邮箱验证关闭时根本不渲染，管理员碰不到它。
  // 保存别的字段时不得把 DB 里存着的 true 悄悄改写成 false。
  it('does not clobber the stored password reset flag when its toggle is not rendered', async () => {
    await mountView(passwordResetApiShape({ emailVerify: false, storedPasswordReset: true }))

    const wrapper = await mountView(
      passwordResetApiShape({ emailVerify: false, storedPasswordReset: true })
    )
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(settingsAPI.updateSettings).toHaveBeenCalled()
    const payload = settingsAPI.updateSettings.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(payload.password_reset_enabled).toBe(true)
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
