import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    name: 'Doughnut',
    template: '<div data-testid="model-chart" />'
  }
}))

vi.mock('@/components/common/DateRangePicker.vue', () => ({
  default: {
    name: 'DateRangePicker',
    template: '<button type="button">date range</button>'
  }
}))

vi.mock('@/components/common/Select.vue', () => ({
  default: {
    name: 'Select',
    template: '<button type="button">granularity</button>'
  }
}))

vi.mock('@/components/common/ModelIcon.vue', () => ({
  default: {
    name: 'ModelIcon',
    template: '<span data-testid="model-icon" />'
  }
}))

vi.mock('@/components/charts/TokenUsageTrend.vue', () => ({
  default: {
    name: 'TokenUsageTrend',
    props: ['trendData'],
    template: '<section data-testid="token-trend">{{ trendData.length }}</section>'
  }
}))

import UserDashboardCharts from '../UserDashboardCharts.vue'

const baseProps = {
  loading: false,
  startDate: '2026-07-12',
  endDate: '2026-07-18',
  granularity: 'day',
  trend: [],
  models: [],
  theme: 'dark' as const
}

describe('UserDashboardCharts', () => {
  it('uses a lightweight empty state instead of an empty chart wall', () => {
    const wrapper = mount(UserDashboardCharts, { props: baseProps })

    expect(wrapper.get('.dashboard-model-panel').classes()).toContain('dashboard-chart-panel--empty')
    expect(wrapper.text()).toContain('暂无模型用量')
    expect(wrapper.text()).toContain('完成一次模型调用后')
    expect(wrapper.find('[data-testid="model-chart"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="token-trend"]').text()).toBe('0')
  })

  it('keeps the complete model usage table when real data exists', () => {
    const wrapper = mount(UserDashboardCharts, {
      props: {
        ...baseProps,
        models: [
          {
            model: 'gpt-5.5',
            requests: 12,
            input_tokens: 1200,
            output_tokens: 480,
            cache_creation_tokens: 0,
            cache_read_tokens: 80,
            total_tokens: 1760,
            cost: 1.25,
            actual_cost: 1.1
          }
        ]
      }
    })

    expect(wrapper.get('[data-testid="model-chart"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('gpt-5.5')
    expect(wrapper.text()).toContain('12')
    expect(wrapper.text()).toContain('1.8K')
    expect(wrapper.text()).toContain('$1.10')
    expect(wrapper.text()).toContain('$1.25')
    expect(wrapper.text()).not.toContain('暂无模型用量')
  })
})
