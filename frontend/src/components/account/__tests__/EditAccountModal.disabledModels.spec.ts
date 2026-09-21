import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'

const { update, showError } = vi.hoisted(() => ({ update: vi.fn(), showError: vi.fn() }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError, showSuccess: vi.fn(), showInfo: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isSimpleMode: true }) }))
vi.mock('@/api/admin', () => ({ adminAPI: {
  accounts: { update, checkMixedChannelRisk: vi.fn().mockResolvedValue({ has_risk: false }) },
  settings: { getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false, providers: [] }), getSettings: vi.fn().mockResolvedValue({}) },
  tlsFingerprintProfiles: { list: vi.fn().mockResolvedValue([]) }
} }))
vi.mock('@/api/admin/accounts', () => ({ getAntigravityDefaultModelMapping: vi.fn() }))
vi.mock('vue-i18n', async () => ({ ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'), useI18n: () => ({ t: (key: string) => key }) }))
import EditAccountModal from '../EditAccountModal.vue'

const account = {
  id: 69, name: 'Fixture', notes: '', platform: 'openai', type: 'apikey',
  credentials: { base_url: 'https://upstream.example/v1', model_mapping: { 'gpt-5.6-sol': 'gpt-5.6-sol' } },
  credentials_status: { has_api_key: true },
  extra: { openai_passthrough: true, openai_disabled_models: ['gpt-5.5'], unrelated_setting: 'retain' },
  proxy_id: null, concurrency: 1, priority: 1, rate_multiplier: 0.12, status: 'active',
  group_ids: [], expires_at: null, auto_pause_on_expired: false
}
const BaseDialogStub = defineComponent({ template: '<div><slot/><slot name="footer"/></div>' })
const mountModal = () => mount(EditAccountModal, { props: { show: true, account, proxies: [], groups: [] },
  global: { stubs: { BaseDialog: BaseDialogStub, Select: true, Icon: true, ProxySelector: true, GroupSelector: true, ModelWhitelistSelector: true } } })

describe('EditAccountModal explicit model quarantine', () => {
  beforeEach(() => { update.mockReset(); update.mockResolvedValue(account); showError.mockReset() })
  it('shows and saves the persistent exclusion without changing passthrough or unrelated extra', async () => {
    const wrapper = mountModal()
    expect(wrapper.get<HTMLTextAreaElement>('#edit-disabled-models').element.value).toBe('gpt-5.5')
    await wrapper.get('#edit-disabled-models').setValue('gpt-5.5\ngpt-5.1')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await vi.waitFor(() => expect(update).toHaveBeenCalled())
    expect(update.mock.calls[0][1].extra).toMatchObject({ openai_passthrough: true, openai_disabled_models: ['gpt-5.5', 'gpt-5.1'], unrelated_setting: 'retain' })
    wrapper.unmount()
  })
  it('persists an explicit empty array when an operator restores models', async () => {
    const wrapper = mountModal()
    await wrapper.get('#edit-disabled-models').setValue('')
    await wrapper.get('form#edit-account-form').trigger('submit.prevent')
    await vi.waitFor(() => expect(update).toHaveBeenCalled())
    expect(update.mock.calls[0][1].extra.openai_disabled_models).toEqual([])
    wrapper.unmount()
  })
})
