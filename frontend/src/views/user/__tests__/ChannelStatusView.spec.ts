import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, channelMonitorAPI, appStore, autoRefreshState } = vi.hoisted(() => ({
  routeState: {
    path: '/app/channel-status'
  },
  channelMonitorAPI: {
    list: vi.fn(),
    status: vi.fn()
  },
  appStore: {
    cachedPublicSettings: {
      channel_monitor_enabled: true
    },
    showError: vi.fn()
  },
  autoRefreshState: {
    countdown: { value: 30 },
    enabled: { value: false },
    start: vi.fn(),
    stop: vi.fn(),
    setEnabled: vi.fn()
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

vi.mock('@/api/channelMonitor', () => channelMonitorAPI)

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: () => 'safe error'
}))

vi.mock('@/composables/useAutoRefresh', () => ({
  useAutoRefresh: () => autoRefreshState
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

vi.mock('@/components/user/monitor/MonitorHero.vue', () => ({
  default: {
    name: 'MonitorHero',
    props: ['overallStatus'],
    template: '<section data-testid="monitor-hero" :data-overall-status="overallStatus" />'
  }
}))

vi.mock('@/components/user/monitor/MonitorCardGrid.vue', () => ({
  default: {
    name: 'MonitorCardGrid',
    props: ['emptyDescription'],
    template: '<section data-testid="monitor-grid">{{ emptyDescription }}</section>'
  }
}))

vi.mock('@/components/user/MonitorDetailDialog.vue', () => ({
  default: {
    name: 'MonitorDetailDialog',
    template: '<section data-testid="monitor-detail" />'
  }
}))

import ChannelStatusView from '../ChannelStatusView.vue'

describe('ChannelStatusView', () => {
  beforeEach(() => {
    routeState.path = '/app/channel-status'
    channelMonitorAPI.list.mockReset()
    channelMonitorAPI.status.mockReset()
    appStore.showError.mockReset()
    autoRefreshState.start.mockReset()
    autoRefreshState.stop.mockReset()
    autoRefreshState.setEnabled.mockReset()
    appStore.cachedPublicSettings.channel_monitor_enabled = true
    channelMonitorAPI.list.mockResolvedValue({ items: [] })
  })

  it('renders inside the user workbench shell on /app/channel-status', async () => {
    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('channelStatus.title')
    expect(wrapper.text()).toContain('channelStatus.disclaimer')
    expect(channelMonitorAPI.list).toHaveBeenCalledTimes(1)
  })

  it('does not mark an empty monitor list as operational', async () => {
    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.find('[data-testid="monitor-hero"]').attributes('data-overall-status')).toBe('unknown')
    expect(wrapper.text()).toContain('channelStatus.empty.description')
  })

  it('explains when channel monitoring is disabled', async () => {
    appStore.cachedPublicSettings.channel_monitor_enabled = false

    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.text()).toContain('channelStatus.empty.disabledDescription')
    expect(autoRefreshState.setEnabled).not.toHaveBeenCalled()
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/monitor'

    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })
})
