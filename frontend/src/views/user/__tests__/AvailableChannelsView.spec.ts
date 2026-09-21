import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import type { UserAvailableChannel } from '@/api/channels'
import AvailableChannelsView from '../AvailableChannelsView.vue'

enableAutoUnmount(afterEach)
const mocks = vi.hoisted(() => ({ getAvailable: vi.fn(), getRates: vi.fn(), showError: vi.fn() }))
vi.mock('@/api/channels', () => ({ default: { getAvailable: mocks.getAvailable } }))
vi.mock('@/api/groups', () => ({ default: { getUserGroupRates: mocks.getRates } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { available_channels_enabled: true }, showError: mocks.showError }) }))
vi.mock('vue-router', async original => ({ ...await original<typeof import('vue-router')>(), useRoute: () => ({ path: '/app/available-channels' }) }))
vi.mock('vue-i18n', async original => ({ ...await original<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/user/AppSectionShell.vue', () => ({ default: { template: '<section data-testid="ssxz-shell"><slot /></section>' } }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<section data-testid="legacy-shell"><slot /></section>' } }))

const rows: UserAvailableChannel[] = [{
  name: 'SSXZ custom providers', description: 'Mixed provider catalog',
  platforms: ['anthropic', 'openai', 'gemini', 'deepseek', 'qwen', 'grok', 'custom-provider'].map((platform, id) => ({
    platform,
    groups: [{ id, name: `${platform} standard`, platform, subscription_type: 'standard', rate_multiplier: 1, peak_rate_enabled: false, peak_start: '', peak_end: '', peak_rate_multiplier: 1, is_exclusive: false }],
    supported_models: [{ name: `${platform}-model`, platform, pricing: null }],
  })),
}]

function render() {
  return mount(AvailableChannelsView, { global: { stubs: {
    TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
    AvailableChannelsTable: { name: 'AvailableChannelsTable', props: ['rows', 'loading', 'userGroupRates', 'searchQuery', 'emptyLabel'], template: '<div />' },
    Icon: true,
  } } })
}

describe('canonical SSXZ available channels', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getAvailable.mockResolvedValue(rows)
    mocks.getRates.mockResolvedValue({ 3: 0.75 })
  })

  it('uses the SSXZ shell and passes all provider groups/models and user rates to the catalog', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.find('[data-testid="ssxz-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="legacy-shell"]').exists()).toBe(false)
    const table = wrapper.getComponent({ name: 'AvailableChannelsTable' })
    expect(table.props('rows')).toEqual(rows)
    expect(table.props('userGroupRates')).toEqual({ 3: 0.75 })
    await wrapper.get('input').setValue('deepseek')
    expect(table.props('searchQuery')).toBe('deepseek')
  })

  it('surfaces a failed catalog request without reporting success', async () => {
    mocks.getAvailable.mockRejectedValue(new Error('Catalog unavailable'))
    const wrapper = render()
    await flushPromises()
    expect(mocks.showError).toHaveBeenCalledTimes(1)
    expect(wrapper.getComponent({ name: 'AvailableChannelsTable' }).props('loading')).toBe(false)
  })

  it('does not show default prices when personal rates fail to load', async () => {
    mocks.getRates.mockRejectedValue(new Error('Rates unavailable'))
    const wrapper = render()
    await flushPromises()
    expect(mocks.showError).toHaveBeenCalledTimes(1)
    expect(wrapper.getComponent({ name: 'AvailableChannelsTable' }).props('rows')).toEqual([])
  })
})
