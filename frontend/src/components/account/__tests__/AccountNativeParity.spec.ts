import { defineComponent } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  create: vi.fn(), update: vi.fn(), generate: vi.fn(), exchange: vi.fn(), refresh: vi.fn(),
  grokGenerate: vi.fn(), grokExchange: vi.fn(), grokRefresh: vi.fn(),
  importSession: vi.fn(), importPAT: vi.fn(), error: vi.fn(), warning: vi.fn(), success: vi.fn(),
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: api.error, showWarning: api.warning, showSuccess: api.success }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isSimpleMode: true }) }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@/api/admin/accounts', () => ({ getAntigravityDefaultModelMapping: vi.fn().mockResolvedValue([]) }))
vi.mock('@/api/admin', () => ({ adminAPI: {
  accounts: {
    create: api.create, update: api.update, generateAuthUrl: api.generate,
    exchangeCode: api.exchange, refreshOpenAIToken: api.refresh,
    importCodexSession: api.importSession, createOpenAICodexPAT: api.importPAT,
    checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false }),
    probeUpstreamBilling: vi.fn().mockResolvedValue({}),
    syncUpstreamModels: vi.fn().mockResolvedValue({ models: [] }),
  },
  grok: { generateAuthUrl: api.grokGenerate, exchangeCode: api.grokExchange, refreshGrokToken: api.grokRefresh },
  settings: {
    getSettings: vi.fn().mockResolvedValue({ account_quota_notify_enabled: true }),
    getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: true, providers: [{}] }),
  },
  tlsFingerprintProfiles: { list: vi.fn().mockResolvedValue([{ id: 7, name: 'Native profile' }]) },
} }))

import CreateAccountModal from '../CreateAccountModal.vue'
import EditAccountModal from '../EditAccountModal.vue'
import OAuthAuthorizationFlow from '../OAuthAuthorizationFlow.vue'

const Dialog = defineComponent({ props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' })
const Select = defineComponent({
  props: ['modelValue', 'options'], emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="opt in options" :value="opt.value">{{ opt.label }}</option></select>',
})
const global = { stubs: {
  BaseDialog: Dialog, Select, Icon: true, PlatformIcon: true, ProxySelector: true,
  ProxyAdBanner: true, GroupSelector: true, ModelWhitelistSelector: true, QuotaLimitCard: true,
  ConfirmDialog: true, HelpTooltip: true,
}, mocks: { $t: (key: string) => key } }
const wrappers: VueWrapper[] = []
function create() {
  const wrapper = mount(CreateAccountModal, { props: { show: true, proxies: [], groups: [] }, global })
  wrappers.push(wrapper)
  return wrapper
}
async function click(wrapper: VueWrapper, text: string) {
  const button = wrapper.findAll('button').find(b => b.text().includes(text))
  expect(button, text).toBeDefined()
  await button!.trigger('click')
  await flushPromises()
}
async function toggleSetting(wrapper: VueWrapper, label: string) {
  const row = wrapper.findAll('div.flex.items-center.justify-between').find(item => item.find('label').exists() && item.find('label').text() === label)
  expect(row, label).toBeDefined()
  await row!.get('button').trigger('click')
}
async function next(wrapper: VueWrapper) {
  await wrapper.get('form input[type="text"]').setValue('Parity account')
  await wrapper.get('form').trigger('submit.prevent')
  await flushPromises()
}
async function openFlow(platform: string) {
  const wrapper = create()
  if (platform !== 'anthropic') await click(wrapper, platform)
  await next(wrapper)
  return wrapper
}
async function reopen(wrapper: VueWrapper) {
  await click(wrapper, 'common.cancel')
  await wrapper.setProps({ show: false })
  await flushPromises()
  await wrapper.setProps({ show: true })
  await flushPromises()
}
async function importToken(wrapper: VueWrapper, method: string, token: string) {
  const flow = wrapper.getComponent(OAuthAuthorizationFlow)
  await flow.get(`input[value="${method}"]`).setValue()
  await flow.get('textarea').setValue(token)
  await flow.get('button.btn-primary').trigger('click')
  await flushPromises()
}
function edit(extra: Record<string, unknown>, overrides: Record<string, unknown> = {}) {
  const account = {
    id: 1, name: 'Edit account', platform: 'openai', type: 'oauth', status: 'active',
    credentials: { access_token: 'test-access', refresh_token: 'test-refresh' }, extra,
    group_ids: [], proxy_id: null, concurrency: 1, priority: 1, rate_multiplier: 1,
    ...overrides,
  }
  const wrapper = mount(EditAccountModal, { props: { show: true, account: account as never, proxies: [], groups: [] }, global })
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  const token = { access_token: 'test-access', refresh_token: 'test-refresh', expires_at: 2000000000, email: 'fixture@example.test', team_id: 'team', client_id: 'native-client' }
  api.create.mockResolvedValue({ id: 42 })
  api.update.mockResolvedValue({ id: 1 })
  api.generate.mockResolvedValue({ auth_url: 'https://auth.example.test', session_id: 'openai-session', state: 'openai-state' })
  api.grokGenerate.mockResolvedValue({ auth_url: 'https://grok.example.test', session_id: 'grok-session', state: 'grok-state' })
  api.exchange.mockResolvedValue(token)
  api.refresh.mockResolvedValue(token)
  api.grokExchange.mockResolvedValue(token)
  api.grokRefresh.mockResolvedValue(token)
  api.importSession.mockResolvedValue({ created: 1, updated: 0, skipped: 0, failed: 0 })
  api.importPAT.mockResolvedValue({ id: 42 })
})
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

describe('native account parity through real controls', () => {
  // jsdom checks the responsive class contract and DOM presence, not rendered layout.
  it('retains both complete platform rows with mobile-first two-column classes', () => {
    const wrapper = create()
    const nativeRow = wrapper.get('[data-tour="account-form-platform"]')
    const providerRow = wrapper.findAll('div.grid').find(row => row.findAll('button').some(button => button.text() === 'Kimi'))!
    expect(nativeRow.findAll('button').map(button => button.text())).toEqual(['Anthropic', 'OpenAI', 'Gemini', 'Grok', 'Antigravity'])
    expect(providerRow.findAll('button').map(button => button.text())).toEqual(['Kimi', 'Zhipu GLM', 'DeepSeek', 'MiniMax', 'OpenCode', 'TypeSafe / Jev'])
    for (const row of [nativeRow, providerRow]) {
      expect(row.classes()).toEqual(expect.arrayContaining(['grid', 'grid-cols-2', 'sm:grid-cols-3', 'lg:grid-cols-5']))
      expect(row.classes()).not.toContain('flex')
      expect(row.classes()).not.toContain('grid-cols-5')
    }
    expect(api.create).not.toHaveBeenCalled()
  })

  it.each(['Anthropic', 'OpenAI'])('retains three %s account-type choices with a mobile-first single-column class', async platform => {
    const wrapper = create()
    await click(wrapper, platform)
    const row = wrapper.get('[data-tour="account-form-type"]')
    expect(row.findAll('button')).toHaveLength(3)
    expect(row.classes()).toEqual(expect.arrayContaining(['grid', 'grid-cols-1', 'sm:grid-cols-2', 'lg:grid-cols-3']))
    expect(row.classes()).not.toContain('grid-cols-3')
    expect(api.create).not.toHaveBeenCalled()
  })

  it.each([undefined, 'https://relay.example.test/v1'])('Grok code authorization preserves endpoint choice %s', async baseURL => {
    const wrapper = create()
    await click(wrapper, 'Grok')
    if (baseURL) {
      await wrapper.get('[data-testid="grok-custom-base-url-toggle"]').trigger('click')
      await wrapper.get('[data-testid="grok-custom-base-url-input"]').setValue(baseURL)
    }
    await next(wrapper)
    await click(wrapper, 'admin.accounts.oauth.grok.generateAuthUrl')
    await wrapper.getComponent(OAuthAuthorizationFlow).get('textarea').setValue('fixture-code')
    await click(wrapper, 'admin.accounts.oauth.completeAuth')
    expect(api.create.mock.calls[0][0].credentials.base_url).toBe(baseURL)
  })

  it('keeps the official default for Grok API-key accounts', async () => {
    const wrapper = create()
    await click(wrapper, 'Grok')
    await wrapper.get('[data-testid="grok-account-type-api-key"]').trigger('click')
    await wrapper.get('form input[type="password"]').setValue('fixture-api-key')
    await next(wrapper)
    expect(api.create.mock.calls[0][0]).toMatchObject({ platform: 'grok', type: 'apikey', credentials: { base_url: 'https://api.x.ai/v1' } })
  })

  it.each([
    ['Grok', 'refresh_token'],
    ['OpenAI', 'refresh_token'],
    ['OpenAI', 'mobile_refresh_token'],
  ])('locks %s %s through token validation and the final save', async (platform, method) => {
    let finishSave!: (value: { id: number }) => void
    api.create.mockImplementationOnce(() => new Promise(resolve => { finishSave = resolve }))
    const wrapper = await openFlow(platform)
    await importToken(wrapper, method, 'fixture-token')
    const flow = wrapper.getComponent(OAuthAuthorizationFlow)
    const button = flow.get('button.btn-primary')
    const disabledDuringSave = button.attributes('disabled')
    await button.trigger('click')
    await flushPromises()
    const writes = api.create.mock.calls.length
    const refreshes = (platform === 'Grok' ? api.grokRefresh : api.refresh).mock.calls.length
    const back = wrapper.findAll('button').find(item => item.text() === 'common.back')!
    const backDisabled = back.attributes('disabled')
    finishSave({ id: 42 })
    await flushPromises()
    expect(disabledDuringSave).toBeDefined()
    expect(backDisabled).toBeDefined()
    expect(writes).toBe(1)
    expect(refreshes).toBe(1)
    expect(wrapper.emitted('created')).toHaveLength(1)
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('guards same-tick double import and releases the lock after a failed save', async () => {
    let failSave!: (error: Error) => void
    api.create.mockImplementationOnce(() => new Promise((_resolve, reject) => { failSave = reject }))
    const wrapper = await openFlow('Grok')
    const flow = wrapper.getComponent(OAuthAuthorizationFlow)
    await flow.get('input[value="refresh_token"]').setValue()
    await flow.get('textarea').setValue('fixture-token')
    const button = flow.get('button.btn-primary')
    await Promise.all([button.trigger('click'), button.trigger('click')])
    await flushPromises()
    const writes = api.create.mock.calls.length
    failSave(new Error('fixture save failure'))
    await flushPromises()
    expect(writes).toBe(1)
    expect(wrapper.emitted('close')).toBeUndefined()
    expect(button.attributes('disabled')).toBeUndefined()
    await button.trigger('click')
    await flushPromises()
    expect(api.create).toHaveBeenCalledTimes(2)
    expect(wrapper.emitted('created')).toHaveLength(1)
  })

  it.each(['reopen', 'platform switch'])('clears Grok relay and headers after %s', async transition => {
    const wrapper = create()
    await click(wrapper, 'Grok')
    await wrapper.get('[data-testid="grok-custom-base-url-toggle"]').trigger('click')
    await wrapper.get('[data-testid="grok-custom-base-url-input"]').setValue('https://previous-relay.example.test')
    await toggleSetting(wrapper, 'admin.accounts.headerOverride.title')
    await click(wrapper, 'admin.accounts.headerOverride.addRow')
    await wrapper.get('input[placeholder="admin.accounts.headerOverride.namePlaceholder"]').setValue('X-Previous-Account')
    await wrapper.get('input[placeholder="admin.accounts.headerOverride.valuePlaceholder"]').setValue('previous-value')
    if (transition === 'reopen') await reopen(wrapper)
    else await click(wrapper, 'OpenAI')
    await click(wrapper, 'Grok')
    expect(wrapper.find('[data-testid="grok-custom-base-url-input"]').exists()).toBe(false)
    expect(wrapper.find('input[placeholder="admin.accounts.headerOverride.valuePlaceholder"]').exists()).toBe(false)
    await next(wrapper)
    await importToken(wrapper, 'refresh_token', 'fixture-token')
    expect(api.create.mock.calls[0][0].credentials.base_url).toBeUndefined()
    expect(JSON.stringify(api.create.mock.calls[0][0].credentials)).not.toContain('previous-value')
  })

  it('resets image conversion, Responses mode, capabilities and compact mappings for the next account', async () => {
    const wrapper = create()
    await click(wrapper, 'OpenAI')
    await click(wrapper, 'API Key')
    await wrapper.get('[data-testid="openai-images-url-to-b64-json-toggle"]').trigger('click')
    await wrapper.get('[data-testid="openai-responses-mode-select"]').setValue('force_responses')
    await wrapper.get('[data-testid="openai-endpoint-capability-seedance"]').setValue(true)
    await click(wrapper, 'admin.accounts.addMapping')
    await wrapper.get('input[placeholder="admin.accounts.fromModel"]').setValue('previous-source')
    await wrapper.get('input[placeholder="admin.accounts.toModel"]').setValue('previous-target')
    await reopen(wrapper)
    await click(wrapper, 'OpenAI')
    await click(wrapper, 'API Key')
    await wrapper.get('form input[type="password"]').setValue('fixture-key')
    await next(wrapper)
    const payload = api.create.mock.calls[0][0]
    expect(payload.extra.images_url_to_b64_json).toBeUndefined()
    expect(payload.extra.openai_responses_mode).toBeUndefined()
    expect(payload.credentials.openai_capabilities).toBeUndefined()
    expect(payload.credentials.compact_model_mapping).toBeUndefined()
  })

  it('adding an error code preserves explicitly configured request headers', async () => {
    const wrapper = create()
    await click(wrapper, 'OpenAI')
    await click(wrapper, 'API Key')
    await toggleSetting(wrapper, 'admin.accounts.headerOverride.title')
    await click(wrapper, 'admin.accounts.headerOverride.addRow')
    await wrapper.get('input[placeholder="admin.accounts.headerOverride.namePlaceholder"]').setValue('X-Account-Header')
    await wrapper.get('input[placeholder="admin.accounts.headerOverride.valuePlaceholder"]').setValue('fixture-header')
    await toggleSetting(wrapper, 'admin.accounts.customErrorCodes')
    await wrapper.get('input[placeholder="admin.accounts.enterErrorCode"]').setValue(418)
    await wrapper.get('input[placeholder="admin.accounts.enterErrorCode"]').trigger('keyup.enter')
    expect((wrapper.get('input[placeholder="admin.accounts.headerOverride.valuePlaceholder"]').element as HTMLInputElement).value).toBe('fixture-header')
    await wrapper.get('form input[type="password"]').setValue('fixture-key')
    await next(wrapper)
    expect(JSON.stringify(api.create.mock.calls[0][0].credentials)).toContain('fixture-header')
    expect(api.create.mock.calls[0][0].credentials.custom_error_codes).toContain(418)
  })

  it('resets TLS fingerprint and custom relay when reopening on the same Anthropic platform', async () => {
    const wrapper = create()
    await flushPromises()
    await wrapper.get('[data-testid="tls-fingerprint-toggle"]').trigger('click')
    await wrapper.get('[data-testid="tls-fingerprint-profile"]').setValue('7')
    await wrapper.get('[data-testid="custom-base-url-toggle"]').trigger('click')
    await wrapper.get('[data-testid="custom-base-url-input"]').setValue('https://previous-relay.example.test')
    await reopen(wrapper)
    expect(wrapper.find('[data-testid="tls-fingerprint-profile"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="custom-base-url-input"]').exists()).toBe(false)
    await next(wrapper)
    await click(wrapper, 'admin.accounts.oauth.generateAuthUrl')
    await wrapper.getComponent(OAuthAuthorizationFlow).get('textarea').setValue('fixture-code')
    await click(wrapper, 'admin.accounts.oauth.completeAuth')
    expect(api.create.mock.calls[0][0].extra?.tls_fingerprint_profile_id).toBeUndefined()
    expect(api.create.mock.calls[0][0].extra?.custom_base_url).toBeUndefined()
  })

  it('resets native pool retries, Anthropic auth scheme and web-search override', async () => {
    const wrapper = create()
    await click(wrapper, 'admin.accounts.claudeConsole')
    await toggleSetting(wrapper, 'admin.accounts.poolMode')
    await wrapper.get('[data-testid="pool-retry-status-codes"]').setValue('401, 503')
    await wrapper.findAll('select').find(select => select.find('option[value="authorization_bearer"]').exists())!.setValue('authorization_bearer')
    await wrapper.findAll('select').find(select => select.find('option[value="disabled"]').exists())!.setValue('disabled')
    await reopen(wrapper)
    await click(wrapper, 'admin.accounts.claudeConsole')
    await toggleSetting(wrapper, 'admin.accounts.poolMode')
    expect((wrapper.get('[data-testid="pool-retry-status-codes"]').element as HTMLInputElement).value).toBe('')
    await wrapper.get('form input[type="password"]').setValue('fixture-key')
    await next(wrapper)
    const payload = api.create.mock.calls[0][0]
    expect(payload.credentials.pool_mode_retry_status_codes).toBeUndefined()
    expect(payload.extra?.anthropic_apikey_auth_scheme).toBeUndefined()
    expect(payload.extra?.web_search_emulation).toBeUndefined()
  })

  it('ignores a Vertex credential file that finishes reading after the modal was reopened', async () => {
    const wrapper = create()
    await wrapper.get('[data-testid="vertex-account-type"]').trigger('click')
    let finishRead!: (value: string) => void
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [{ text: () => new Promise<string>(resolve => { finishRead = resolve }) }] })
    await input.trigger('change')
    await reopen(wrapper)
    await wrapper.get('[data-testid="vertex-account-type"]').trigger('click')
    finishRead(JSON.stringify({ project_id: 'previous-project', client_email: 'previous@example.test', private_key: 'fixture-private-key' }))
    await flushPromises()
    expect(wrapper.text()).not.toContain('previous@example.test')
    await next(wrapper)
    expect(api.create).not.toHaveBeenCalled()
  })

  it.each(['codex_session', 'codex_pat'])('retains the submission lock during %s imports and permits retry after failure', async method => {
    const importer = method === 'codex_pat' ? api.importPAT : api.importSession
    let failImport!: (error: Error) => void
    importer.mockImplementationOnce(() => new Promise((_resolve, reject) => { failImport = reject }))
    const wrapper = await openFlow('OpenAI')
    await importToken(wrapper, method, 'fixture-content')
    const button = wrapper.getComponent(OAuthAuthorizationFlow).get('button.btn-primary')
    const disabled = button.attributes('disabled')
    await button.trigger('click')
    const calls = importer.mock.calls.length
    failImport(new Error('fixture failure'))
    await flushPromises()
    expect(disabled).toBeDefined()
    expect(calls).toBe(1)
    expect(button.attributes('disabled')).toBeUndefined()
    await button.trigger('click')
    await flushPromises()
    expect(importer).toHaveBeenCalledTimes(2)
    expect(wrapper.emitted('created')).toHaveLength(1)
  })

  it('uses only Grok auth/session/token APIs and saves native credentials', async () => {
    const wrapper = await openFlow('Grok')
    const flow = wrapper.getComponent(OAuthAuthorizationFlow)
    flow.vm.$emit('generate-url')
    await flushPromises()
    await flow.get('textarea').setValue('grok-code')
    await click(wrapper, 'admin.accounts.oauth.completeAuth')
    expect(api.grokGenerate).toHaveBeenCalledOnce()
    expect(api.grokExchange).toHaveBeenCalledWith({ code: 'grok-code', session_id: 'grok-session', state: 'grok-state' })
    expect(api.generate).not.toHaveBeenCalled()
    expect(api.exchange).not.toHaveBeenCalled()
    expect(api.create.mock.calls[0][0]).toMatchObject({ platform: 'grok', type: 'oauth', credentials: { team_id: 'team', client_id: 'native-client' } })
  })

  it('Grok RT batches preserve partial failure and custom upstream config', async () => {
    api.grokRefresh.mockRejectedValueOnce(new Error('fixture failure'))
    const wrapper = create()
    await click(wrapper, 'Grok')
    await wrapper.get('[data-testid="grok-custom-base-url-toggle"]').trigger('click')
    await wrapper.get('[data-testid="grok-custom-base-url-input"]').setValue('https://relay.example.test')
    await next(wrapper)
    await importToken(wrapper, 'refresh_token', 'bad-fixture\ngood-fixture')
    expect(api.grokRefresh).toHaveBeenCalledTimes(2)
    expect(api.refresh).not.toHaveBeenCalled()
    expect(api.create.mock.calls[0][0].credentials.base_url).toBe('https://relay.example.test')
    expect(api.warning).toHaveBeenCalled()
    expect(wrapper.emitted('created')).toHaveLength(1)
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it('MobileRT uses its native client ID for validation and persistence', async () => {
    const wrapper = await openFlow('OpenAI')
    await importToken(wrapper, 'mobile_refresh_token', 'mobile-fixture')
    expect(api.refresh).toHaveBeenCalledWith('mobile-fixture', null, '/admin/openai/refresh-token', 'app_LlGpXReQgckcGGUo2JrYvtJK')
    expect(api.create.mock.calls[0][0].credentials.client_id).toBe('app_LlGpXReQgckcGGUo2JrYvtJK')
  })

  it('does not offer unsupported Sora/Kiro creation while keeping native Grok', () => {
    const wrapper = create()
    const buttons = wrapper.findAll('button').map(button => button.text())
    expect(buttons.some(label => /\bSora\b|\bKiro\b/.test(label))).toBe(false)
    expect(wrapper.get('[data-testid="grok-platform"]').text()).toBe('Grok')
    expect(api.create).not.toHaveBeenCalled()
  })

  it.each(['codex_session', 'agent_identity', 'codex_pat'])('renders and submits the real %s control', async method => {
    const wrapper = await openFlow('OpenAI')
    await importToken(wrapper, method, method === 'agent_identity' ? '{"auth_mode":"agentidentity"}' : 'fixture-content')
    expect(method === 'codex_pat' ? api.importPAT : api.importSession).toHaveBeenCalledOnce()
    expect(wrapper.emitted('created')).toHaveLength(1)
  })

  it('keeps partial Codex import errors visible instead of closing as success', async () => {
    api.importSession.mockResolvedValue({ created: 1, updated: 0, skipped: 0, failed: 1, errors: [{ index: 2, message: 'fixture failure' }] })
    const wrapper = await openFlow('OpenAI')
    await importToken(wrapper, 'codex_session', 'fixture-content')
    expect(wrapper.text()).toContain('fixture failure')
    expect(api.warning).toHaveBeenCalled()
    expect(api.success).not.toHaveBeenCalled()
    expect(wrapper.emitted('close')).toBeUndefined()
  })

  it.each(['force_responses', 'force_chat_completions'])('persists API-key Responses mode %s with images options', async mode => {
    const wrapper = create()
    await click(wrapper, 'OpenAI')
    await click(wrapper, 'API Key')
    await wrapper.get('[data-testid="openai-responses-mode-select"]').setValue(mode)
    await wrapper.get('[data-testid="openai-images-url-to-b64-json-toggle"]').trigger('click')
    await wrapper.get('form input[type="password"]').setValue('fixture-key')
    await next(wrapper)
    expect(api.create.mock.calls[0][0].extra).toMatchObject({ openai_responses_mode: mode, images_url_to_b64_json: true })
  })

  it('disabling text capability clears Responses mode and preserves embeddings-only', async () => {
    const wrapper = create()
    await click(wrapper, 'OpenAI')
    await click(wrapper, 'API Key')
    await wrapper.get('[data-testid="openai-responses-mode-select"]').setValue('force_responses')
    await wrapper.get('[data-testid="openai-endpoint-capability-chat_completions"]').setValue(false)
    expect(wrapper.get('[data-testid="openai-responses-mode-select"]').attributes('disabled')).toBeDefined()
    await wrapper.get('form input[type="password"]').setValue('fixture-key')
    await next(wrapper)
    expect(api.create.mock.calls[0][0].credentials.openai_capabilities).toEqual(['embeddings'])
    expect(api.create.mock.calls[0][0].extra.openai_responses_mode).toBeUndefined()
  })

  it('saves selected TLS fingerprint and custom relay URL during Anthropic OAuth', async () => {
    const wrapper = create()
    await flushPromises()
    await wrapper.get('[data-testid="tls-fingerprint-toggle"]').trigger('click')
    await wrapper.get('[data-testid="tls-fingerprint-profile"]').setValue('7')
    await wrapper.get('[data-testid="custom-base-url-toggle"]').trigger('click')
    await wrapper.get('[data-testid="custom-base-url-input"]').setValue('https://relay.example.test')
    await next(wrapper)
    const flow = wrapper.getComponent(OAuthAuthorizationFlow)
    flow.vm.$emit('generate-url')
    await flushPromises()
    await flow.get('textarea').setValue('anthropic-code')
    await click(wrapper, 'admin.accounts.oauth.completeAuth')
    expect(api.create.mock.calls[0][0].extra).toMatchObject({ enable_tls_fingerprint: true, tls_fingerprint_profile_id: 7, custom_base_url_enabled: true, custom_base_url: 'https://relay.example.test' })
  })

  it('creates Vertex from an uploaded service-account JSON without displaying its key', async () => {
    const wrapper = create()
    await wrapper.get('[data-testid="vertex-account-type"]').trigger('click')
    const input = wrapper.get('input[type="file"]')
    const json = JSON.stringify({ project_id: 'fixture-project', client_email: 'fixture@example.test', private_key: 'fixture-private-key' })
    Object.defineProperty(input.element, 'files', { value: [{ text: async () => json }] })
    await input.trigger('change')
    await flushPromises()
    expect(wrapper.text()).not.toContain('fixture-private-key')
    await next(wrapper)
    expect(api.create.mock.calls[0][0]).toMatchObject({ type: 'service_account', credentials: { service_account_json: json, project_id: 'fixture-project', tier_id: 'vertex' } })
  })

  it('round-trips native credit thresholds but never echoes runtime state', async () => {
    const wrapper = edit({ auto_reset_credit_enabled: true, auto_reset_credit_5h_threshold: 0.8, auto_reset_credit_7d_threshold: 0.9, codex_auto_reset_credit_state: { stale: true } })
    await flushPromises()
    expect((wrapper.get('[data-testid="auto-reset-credit-5h-threshold"]').element as HTMLInputElement).value).toBe('80')
    await wrapper.get('[data-testid="auto-reset-credit-7d-threshold"]').setValue(75)
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()
    expect(api.update.mock.calls[0][1].extra).toMatchObject({ auto_reset_credit_enabled: true, auto_reset_credit_5h_threshold: 0.8, auto_reset_credit_7d_threshold: 0.75 })
    expect(api.update.mock.calls[0][1].extra.codex_auto_reset_credit_state).toBeUndefined()
  })

  it('rejects invalid credit thresholds and does not expose the option for ChatWeb', async () => {
    const wrapper = edit({ auto_reset_credit_enabled: true })
    await wrapper.get('[data-testid="auto-reset-credit-5h-threshold"]').setValue(0)
    await wrapper.get('form').trigger('submit.prevent')
    expect(api.update).not.toHaveBeenCalled()
    expect(edit({ openai_auth_mode: 'chatweb' }).find('[data-testid="auto-reset-credit-settings"]').exists()).toBe(false)
  })

  it('saves Anthropic Bearer auth and web-search override from their actual selectors', async () => {
    const wrapper = create()
    await click(wrapper, 'admin.accounts.claudeConsole')
    const scheme = wrapper.findAll('select').find(select => select.find('option[value="authorization_bearer"]').exists())!
    await scheme.setValue('authorization_bearer')
    const search = wrapper.findAll('select').find(select => select.find('option[value="disabled"]').exists())!
    await search.setValue('disabled')
    await wrapper.get('form input[type="password"]').setValue('fixture-key')
    await next(wrapper)
    expect(api.create.mock.calls[0][0].extra).toMatchObject({ anthropic_apikey_auth_scheme: 'authorization_bearer', web_search_emulation: 'disabled' })
  })

  it('serializes compact-only mappings for Codex imports', async () => {
    const wrapper = create()
    await click(wrapper, 'OpenAI')
    await click(wrapper, 'admin.accounts.addMapping')
    await wrapper.get('input[placeholder="admin.accounts.fromModel"]').setValue('gpt-source')
    await wrapper.get('input[placeholder="admin.accounts.toModel"]').setValue('gpt-compact')
    await next(wrapper)
    await importToken(wrapper, 'codex_pat', 'fixture-content')
    expect(api.importPAT.mock.calls[0][0].credential_extras.compact_model_mapping).toEqual({ 'gpt-source': 'gpt-compact' })
  })

  it('loads custom relay settings from the backend projected DTO without dropping them', async () => {
    const wrapper = edit({}, { platform: 'anthropic', custom_base_url_enabled: true, custom_base_url: 'https://relay.example.test' })
    await flushPromises()
    expect(wrapper.findAll('input').some(input => (input.element as HTMLInputElement).value === 'https://relay.example.test')).toBe(true)
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()
    expect(api.update.mock.calls[0][1].extra).toMatchObject({ custom_base_url_enabled: true, custom_base_url: 'https://relay.example.test' })
  })

  it('disabling native credit reset sends false rather than retaining stale true', async () => {
    const wrapper = edit({ auto_reset_credit_enabled: true })
    await wrapper.get('[data-testid="auto-reset-credit-enabled"]').trigger('click')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()
    expect(api.update.mock.calls[0][1].extra.auto_reset_credit_enabled).toBe(false)
  })
})
