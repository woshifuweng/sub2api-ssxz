import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'channelStatus.windowTab.7d') return '7 days'
      if (key === 'monitorCommon.windowAvailabilityLabel') return `Availability ${params?.window}`
      if (key === 'monitorCommon.extraModelsCount') return `+ ${params?.n} more models`
      if (key === 'monitorCommon.dialogLatency') return 'Response latency'
      if (key === 'monitorCommon.endpointPing') return 'Network ping'
      return key
    }
  })
}))

vi.mock('../ProviderIcon.vue', () => ({
  default: {
    name: 'ProviderIcon',
    template: '<span data-testid="provider-icon" />'
  }
}))

vi.mock('../MonitorMetricPair.vue', () => ({
  default: {
    name: 'MonitorMetricPair',
    props: ['primaryLabel', 'primaryValue', 'secondaryLabel', 'secondaryValue'],
    template: '<section data-testid="metric-pair">{{ primaryLabel }} {{ primaryValue }} {{ secondaryLabel }} {{ secondaryValue }}</section>'
  }
}))

vi.mock('../MonitorAvailabilityRow.vue', () => ({
  default: {
    name: 'MonitorAvailabilityRow',
    props: ['windowLabel', 'value', 'samplesLabel'],
    template: '<section data-testid="availability-row">{{ windowLabel }} {{ value }} {{ samplesLabel }}</section>'
  }
}))

vi.mock('../MonitorTimeline.vue', () => ({
  default: {
    name: 'MonitorTimeline',
    template: '<section data-testid="timeline" />'
  }
}))

import MonitorCard from '../MonitorCard.vue'

describe('MonitorCard', () => {
  it('renders the availability window label through i18n', () => {
    const wrapper = mount(MonitorCard, {
      props: {
        window: '7d',
        availabilityValue: 98.75,
        countdownSeconds: 30,
        item: {
          id: 1,
          name: 'OpenAI primary',
          provider: 'openai',
          group_name: 'Pro',
          primary_model: 'gpt-5.5',
          primary_status: 'operational',
          primary_latency_ms: 900,
          primary_ping_latency_ms: 80,
          availability_7d: 98.75,
          extra_models: [{ model: 'gpt-5.4', status: 'operational', latency_ms: 950 }],
          timeline: []
        }
      }
    })

    const label = wrapper.get('[data-testid="availability-row"]').text()
    expect(label).toContain('Availability 7 days')

    const metrics = wrapper.get('[data-testid="metric-pair"]').text()
    expect(metrics).toContain('Response latency 900')
    expect(metrics).toContain('Network ping 80')
  })
})
