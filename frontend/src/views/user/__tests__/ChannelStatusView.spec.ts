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
    props: ['items', 'emptyDescription', 'window', 'loading'],
    emits: ['cardClick'],
    template: `
      <section data-testid="monitor-grid" :data-window="window" :data-loading="String(loading)" :data-count="items.length">
        <p v-if="items.length === 0">{{ emptyDescription }}</p>
        <button
          v-for="item in items"
          :key="item.id"
          type="button"
          data-testid="monitor-card"
          @click="$emit('cardClick', item)"
        >
          <span>{{ item.name }}</span>
          <span>{{ item.group_name }}</span>
          <span>{{ item.primary_model }}</span>
          <span>{{ item.primary_status }}</span>
          <span>{{ item.primary_latency_ms }}</span>
          <span>{{ item.primary_ping_latency_ms }}</span>
          <span>{{ item.availability_7d }}</span>
          <span v-for="extra in item.extra_models" :key="extra.model">{{ extra.model }}:{{ extra.status }}</span>
        </button>
      </section>
    `
  }
}))

vi.mock('@/components/user/MonitorDetailDialog.vue', () => ({
  default: {
    name: 'MonitorDetailDialog',
    props: ['show', 'monitorId', 'title'],
    template: '<section data-testid="monitor-detail" :data-show="String(show)" :data-monitor-id="monitorId || \'\'">{{ title }}</section>'
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
    expect(channelMonitorAPI.list).not.toHaveBeenCalled()
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/monitor'

    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })

  it('renders monitor evidence and opens the detail dialog for investigation', async () => {
    channelMonitorAPI.list.mockResolvedValue({
      items: [
        {
          id: 12,
          name: 'OpenAI primary channel',
          provider: 'openai',
          group_name: 'Pro group',
          primary_model: 'gpt-4o-mini',
          primary_status: 'failed',
          primary_latency_ms: 1842,
          primary_ping_latency_ms: 92,
          availability_7d: 98.75,
          extra_models: [
            {
              model: 'gpt-4.1',
              status: 'operational',
              latency_ms: 990
            }
          ],
          timeline: [
            {
              status: 'failed',
              latency_ms: 1842,
              ping_latency_ms: 92,
              checked_at: '2026-07-02T08:00:00Z'
            }
          ]
        }
      ]
    })

    const wrapper = mount(ChannelStatusView)
    await flushPromises()

    expect(wrapper.find('[data-testid="monitor-hero"]').attributes('data-overall-status')).toBe('degraded')
    expect(wrapper.find('[data-testid="monitor-grid"]').attributes('data-count')).toBe('1')
    expect(wrapper.text()).toContain('OpenAI primary channel')
    expect(wrapper.text()).toContain('Pro group')
    expect(wrapper.text()).toContain('gpt-4o-mini')
    expect(wrapper.text()).toContain('failed')
    expect(wrapper.text()).toContain('1842')
    expect(wrapper.text()).toContain('98.75')
    expect(wrapper.text()).toContain('gpt-4.1:operational')

    await wrapper.find('[data-testid="monitor-card"]').trigger('click')

    const detail = wrapper.find('[data-testid="monitor-detail"]')
    expect(detail.attributes('data-show')).toBe('true')
    expect(detail.attributes('data-monitor-id')).toBe('12')
    expect(detail.text()).toContain('OpenAI primary channel')
  })
})
