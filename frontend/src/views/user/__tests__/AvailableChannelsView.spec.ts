import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, userChannelsAPI, userGroupsAPI, appStore } = vi.hoisted(() => ({
  routeState: {
    path: '/app/available-channels'
  },
  userChannelsAPI: {
    getAvailable: vi.fn()
  },
  userGroupsAPI: {
    getUserGroupRates: vi.fn()
  },
  appStore: {
    cachedPublicSettings: {
      available_channels_enabled: true
    } as { available_channels_enabled?: boolean } | null,
    showError: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/api/channels', () => ({
  default: userChannelsAPI
}))

vi.mock('@/api/groups', () => ({
  default: userGroupsAPI
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: () => 'safe error'
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot /></main>'
  }
}))

vi.mock('@/components/layout/TablePageLayout.vue', () => ({
  default: {
    name: 'TablePageLayout',
    template: '<section data-testid="table-page"><slot name="filters" /><slot name="table" /></section>'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />'
  }
}))

vi.mock('@/components/channels/AvailableChannelsTable.vue', () => ({
  default: {
    name: 'AvailableChannelsTable',
    props: ['emptyLabel', 'rows', 'searchQuery', 'userGroupRates'],
    template: '<div data-testid="channels-table">{{ emptyLabel }}</div>'
  }
}))

import AvailableChannelsView from '../AvailableChannelsView.vue'

describe('AvailableChannelsView', () => {
  beforeEach(() => {
    routeState.path = '/app/available-channels'
    userChannelsAPI.getAvailable.mockReset()
    userGroupsAPI.getUserGroupRates.mockReset()
    appStore.showError.mockReset()
    appStore.cachedPublicSettings = { available_channels_enabled: true }
    userChannelsAPI.getAvailable.mockResolvedValue([])
    userGroupsAPI.getUserGroupRates.mockResolvedValue({})
  })

  it('renders inside the user workbench shell on /app/available-channels', async () => {
    const wrapper = mount(AvailableChannelsView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('availableChannels.title')
    expect(wrapper.text()).toContain('availableChannels.description')
    expect(wrapper.text()).not.toContain('availableChannels.userGuideTitle')
    expect(userChannelsAPI.getAvailable).toHaveBeenCalledTimes(1)
  })

  it('explains when the models and pricing page is not enabled', async () => {
    appStore.cachedPublicSettings = { available_channels_enabled: false }

    const wrapper = mount(AvailableChannelsView)
    await flushPromises()

    expect(wrapper.find('[data-testid="channels-table"]').text()).toBe(
      'availableChannels.emptyDisabled'
    )
  })

  it('keeps the genuine empty-model message when the page is enabled', async () => {
    const wrapper = mount(AvailableChannelsView)
    await flushPromises()

    expect(wrapper.find('[data-testid="channels-table"]').text()).toBe('availableChannels.empty')
  })

  it('does not claim there are no models while public settings are unavailable', async () => {
    appStore.cachedPublicSettings = null

    const wrapper = mount(AvailableChannelsView)
    await flushPromises()

    expect(wrapper.find('[data-testid="channels-table"]').text()).toBe(
      'availableChannels.emptyDisabled'
    )
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/available-channels'

    const wrapper = mount(AvailableChannelsView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })
})
