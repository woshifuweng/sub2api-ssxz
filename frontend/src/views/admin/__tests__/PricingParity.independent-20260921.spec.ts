import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import type { Channel, ChannelModelPricing } from '@/api/admin/channels'
import type { AdminGroup } from '@/types'
import ChannelsView from '../ChannelsView.vue'
import GroupsView from '../GroupsView.vue'

enableAutoUnmount(afterEach)
const mocks = vi.hoisted(() => ({
  channelsList: vi.fn(), channelsUpdate: vi.fn(), groupsList: vi.fn(), groupsAll: vi.fn(),
  groupsUpdate: vi.fn(), groupsCreate: vi.fn(), showError: vi.fn(),
}))
vi.mock('@/api/admin', () => ({ adminAPI: {
  channels: { list: mocks.channelsList, update: mocks.channelsUpdate },
  groups: {
    list: mocks.groupsList, getAll: mocks.groupsAll, update: mocks.groupsUpdate, create: mocks.groupsCreate,
    getModelsListCandidates: vi.fn().mockResolvedValue([]), getModelAllowlistCandidates: vi.fn().mockResolvedValue([]),
    getUsageSummary: vi.fn().mockResolvedValue([]), getCapacitySummary: vi.fn().mockResolvedValue([]),
    getLiveCapability: vi.fn().mockResolvedValue({ supported: false }),
  },
  settings: { getWebSearchEmulationConfig: vi.fn().mockResolvedValue({ enabled: false }) },
  accounts: { list: vi.fn().mockResolvedValue({ items: [] }), getById: vi.fn() },
} }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError, showSuccess: vi.fn() }) }))
vi.mock('@/stores/onboarding', () => ({ useOnboardingStore: () => ({ isCurrentStep: () => false, nextStep: vi.fn() }) }))
vi.mock('vue-i18n', async original => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))

const SelectStub = { name: 'Select', props: ['modelValue', 'options'], emits: ['update:modelValue'], template: '<div />' }
const ToggleStub = { name: 'Toggle', props: ['modelValue'], emits: ['update:modelValue'], template: '<button type="button" role="switch" :aria-checked="modelValue" @click="$emit(\'update:modelValue\', !modelValue)" />' }
const stubs = {
  AppLayout: { template: '<main><slot /></main>' },
  TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
  DataTable: { props: ['data'], template: '<div><div v-for="row in data" :key="row.id"><slot name="cell-actions" :row="row" /></div></div>' },
  BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  Select: SelectStub, Toggle: ToggleStub, Pagination: true, ConfirmDialog: true, EmptyState: true,
  PlatformIcon: true, Icon: true, GroupCapacityBadge: true, GroupRateMultipliersModal: true,
  GroupRPMOverridesModal: true,
  ReasoningEffortPolicyFields: { methods: { validate: () => true, resetValidation: () => undefined }, template: '<div />' },
  CodexManifestAccountsField: true,
  PricingEntryCard: true, VueDraggable: true,
}

function group(overrides: Partial<AdminGroup> = {}): AdminGroup {
  return {
    id: 42, name: 'Review group', platform: 'openai', description: '', status: 'active',
    rate_multiplier: 1, is_exclusive: false, subscription_type: 'standard',
    model_pricing: [], supported_model_scopes: [], allow_messages_dispatch: false,
    codex_models_manifest_config: { enabled: false, account_ids: [], fallback_to_scheduler: false },
    daily_limit_usd: null, weekly_limit_usd: null, monthly_limit_usd: null,
    image_price_1k: null, image_price_2k: null, image_price_4k: null,
    video_price_480p: null, video_price_720p: null, video_price_1080p: null,
    fallback_group_id: null, fallback_group_id_on_invalid_request: null,
    created_at: '2026-09-21T00:00:00Z', updated_at: '2026-09-21T00:00:00Z',
    ...overrides,
  } as AdminGroup
}

function pricing(overrides: Partial<ChannelModelPricing> = {}): ChannelModelPricing {
  return {
    platform: 'openai', models: ['review-model'], billing_mode: 'token',
    input_price: 0.000003, output_price: 0.000012, cache_write_price: 0.000004,
    cache_write_1h_price: 0.000006, cache_read_price: 0, image_input_price: 0.000002,
    image_output_price: 0.000015, per_request_price: null,
    fast_multiplier: 2.5, flex_multiplier: 0.5, max_reasoning_effort_multiplier: 1.75,
    time_pricing: { timezone: 'America/New_York', weekdays_only: true, periods: [{ start_time: '09:00:15', end_time: '12:30:45', multiplier: 1.25 }] },
    intervals: [{
      min_tokens: 200000, max_tokens: null, tier_label: 'long context', sort_order: 0,
      input_price: 0.000006, output_price: 0.000024, cache_write_price: 0.000008,
      cache_write_1h_price: 0.000012, cache_read_price: 0, per_request_price: null,
      input_multiplier: 2, output_multiplier: 1.5, cache_write_multiplier: 2.25, cache_read_multiplier: 0.5,
    }], ...overrides,
  }
}

function channel(overrides: Partial<Channel> = {}): Channel {
  return {
    id: 9, name: 'Review channel', description: 'Unchanged', status: 'active',
    billing_model_source: 'channel_mapped', restrict_models: true, features_config: { custom_feature: { enabled: true } },
    group_ids: [42], model_pricing: [pricing()], model_mapping: { openai: { 'review-model': 'upstream-model' } },
    apply_pricing_to_account_stats: false, account_stats_pricing_rules: [],
    created_at: '2026-09-21T00:00:00Z', updated_at: '2026-09-21T00:00:00Z', ...overrides,
  }
}

async function openEdit(view: typeof ChannelsView | typeof GroupsView) {
  const wrapper = mount(view, { global: { plugins: [createPinia()], stubs } })
  await flushPromises()
  const edit = wrapper.findAll('button').find(button => button.text().includes('common.edit'))
  expect(edit).toBeDefined()
  await edit!.trigger('click')
  await flushPromises()
  return wrapper
}

async function submit(wrapper: VueWrapper) {
  await wrapper.get('form').trigger('submit')
  await flushPromises()
  expect(mocks.showError).not.toHaveBeenCalled()
}

describe('Independent pricing roundtrip and composite messages review', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    mocks.groupsAll.mockResolvedValue([group()])
    mocks.groupsList.mockResolvedValue({ items: [group()], total: 1, pages: 1 })
    mocks.channelsList.mockResolvedValue({ items: [channel()], total: 1 })
    mocks.channelsUpdate.mockResolvedValue(channel())
    mocks.groupsUpdate.mockResolvedValue(group())
    mocks.groupsCreate.mockResolvedValue(group())
  })

  it('preserves every channel price, multiplier, interval and time field on an unchanged save', async () => {
    const original = channel()
    const wrapper = await openEdit(ChannelsView)
    await submit(wrapper)
    expect(mocks.channelsUpdate).toHaveBeenCalledTimes(1)
    expect(mocks.channelsUpdate.mock.calls[0][1]).toMatchObject({
      model_pricing: original.model_pricing, model_mapping: original.model_mapping,
      features_config: original.features_config, group_ids: [42],
    })
  })

  it('preserves concrete provider pricing and mapping on a composite-bound channel', async () => {
    mocks.groupsAll.mockResolvedValue([group({ platform: 'composite' })])
    const original = channel({
      model_pricing: [pricing(), pricing({ platform: 'minimax', models: ['minimax-review'] }), pricing({ platform: 'opencode_go', models: ['go-review'] })],
      model_mapping: { minimax: { 'minimax-review': 'upstream-minimax' }, opencode_go: { 'go-review': 'upstream-go' } },
    })
    mocks.channelsList.mockResolvedValue({ items: [original], total: 1 })
    const wrapper = await openEdit(ChannelsView)
    await submit(wrapper)
    expect(mocks.channelsUpdate.mock.calls[0][1].model_pricing).toEqual(expect.arrayContaining(original.model_pricing))
    expect(mocks.channelsUpdate.mock.calls[0][1].model_pricing).toHaveLength(3)
    expect(mocks.channelsUpdate.mock.calls[0][1].model_mapping).toEqual(original.model_mapping)
  })

  it('preserves accepted group pricing multipliers on an unrelated save', async () => {
    // The backend rejects group time pricing; token intervals have separate group policy.
    const original = pricing({ time_pricing: null, intervals: [] })
    mocks.groupsList.mockResolvedValue({ items: [group({ model_pricing: [original] })], total: 1, pages: 1 })
    const wrapper = await openEdit(GroupsView)
    await submit(wrapper)
    expect(mocks.groupsUpdate.mock.calls[0][1].model_pricing).toEqual([original])
  })

  it('preserves accepted account-stat pricing multipliers on an unrelated channel save', async () => {
    const original = pricing({ time_pricing: null })
    mocks.channelsList.mockResolvedValue({ items: [channel({ account_stats_pricing_rules: [{ name: 'Cost rule', group_ids: [42], account_ids: [], pricing: [original] }] })], total: 1 })
    const wrapper = await openEdit(ChannelsView)
    await submit(wrapper)
    expect(mocks.channelsUpdate.mock.calls[0][1].account_stats_pricing_rules[0].pricing).toEqual([original])
  })

  it.each([true, false])('preserves composite Messages dispatch=%s and allows toggling it', async enabled => {
    mocks.groupsList.mockResolvedValue({ items: [group({ platform: 'composite', allow_messages_dispatch: enabled })], total: 1, pages: 1 })
    const wrapper = await openEdit(GroupsView)
    const toggle = wrapper.findAllComponents(ToggleStub).find(item => item.element.parentElement?.textContent?.includes('admin.groups.openaiMessages.allowDispatch'))
    expect(toggle).toBeDefined()
    expect(toggle!.props('modelValue')).toBe(enabled)
    await toggle!.get('button').trigger('click')
    await submit(wrapper)
    expect(mocks.groupsUpdate.mock.calls[0][1]).toMatchObject({ platform: 'composite', allow_messages_dispatch: !enabled })
    expect(mocks.groupsUpdate.mock.calls[0][1].messages_dispatch_model_config).toBeUndefined()
  })

  it('creates a composite group with Messages dispatch enabled without OpenAI-only mappings', async () => {
    const wrapper = mount(GroupsView, { global: { plugins: [createPinia()], stubs } })
    await flushPromises()
    await wrapper.get('[data-tour="groups-create-btn"]').trigger('click')
    await wrapper.get('[data-tour="group-form-name"]').setValue('Composite review')
    const platform = wrapper.findAllComponents(SelectStub).find(item => item.attributes('data-tour') === 'group-form-platform')!
    expect(platform).toBeDefined()
    platform.vm.$emit('update:modelValue', 'composite')
    await flushPromises()
    const toggle = wrapper.findAllComponents(ToggleStub).find(item => item.element.parentElement?.textContent?.includes('admin.groups.openaiMessages.allowDispatch'))!
    expect(toggle).toBeDefined()
    await toggle.get('button').trigger('click')
    await submit(wrapper)
    expect(mocks.groupsCreate).toHaveBeenCalledTimes(1)
    expect(mocks.groupsCreate.mock.calls[0][0]).toMatchObject({ platform: 'composite', allow_messages_dispatch: true })
    expect(mocks.groupsCreate.mock.calls[0][0].messages_dispatch_model_config).toBeUndefined()
  })
})
