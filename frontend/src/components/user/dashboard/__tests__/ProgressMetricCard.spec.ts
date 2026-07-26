import { mount } from '@vue/test-utils'
import { BarChart3 } from '@lucide/vue'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data'],
    template: '<div data-testid="metric-chart-line" :data-values="data.datasets[0].data.join(\',\')" />'
  },
  Bar: {
    name: 'Bar',
    props: ['data'],
    template: '<div data-testid="metric-chart-bar" :data-values="data.datasets[0].data.join(\',\')" />'
  }
}))

import ProgressMetricCard from '../ProgressMetricCard.vue'

describe('ProgressMetricCard detailed mode', () => {
  it('slices day-granularity data by period and recomputes its headline and summary', async () => {
    const wrapper = mount(ProgressMetricCard, {
      props: {
        testId: 'daily-cost',
        label: 'Daily spend',
        icon: BarChart3,
        detailed: true,
        series: [1, 2, 3, 4, 5],
        seriesLabels: ['07/01', '07/02', '07/03', '07/04', '07/05'],
        seriesFormatter: (value: number) => `$${value.toFixed(2)}`,
        periodOptions: [
          { value: 'last-3', label: 'Past 3 days', points: 3 },
          { value: 'all', label: 'All days' }
        ],
        defaultPeriod: 'last-3',
        showStats: true
      }
    })

    expect(wrapper.get('.progress-metric-card__value').text()).toBe('$12.00')
    expect(wrapper.get('[data-testid="metric-chart-line"]').attributes('data-values')).toBe('3,4,5')
    expect(wrapper.get('.progress-metric-card__summary').text()).toContain('$4.00')
    expect(wrapper.get('.progress-metric-card__summary').text()).toContain('$5.00')
    expect(wrapper.get('.progress-metric-card__summary').text()).not.toContain('$3.00')

    await wrapper.get('[data-testid="metric-period-select"]').setValue('all')

    expect(wrapper.get('.progress-metric-card__value').text()).toBe('$15.00')
    expect(wrapper.get('[data-testid="metric-chart-line"]').attributes('data-values')).toBe('1,2,3,4,5')
  })

  it('follows a controlled period prop and hides its own period select and view toggle', async () => {
    const wrapper = mount(ProgressMetricCard, {
      props: {
        testId: 'daily-cost',
        label: 'Daily spend',
        icon: BarChart3,
        detailed: true,
        series: [1, 2, 3, 4, 5],
        seriesLabels: ['07/01', '07/02', '07/03', '07/04', '07/05'],
        periodOptions: [
          { value: 'last-3', label: 'Past 3 days', points: 3 },
          { value: 'all', label: 'All days' }
        ],
        period: 'last-3',
        hideViewToggle: true
      }
    })

    expect(wrapper.find('[data-testid="metric-period-select"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="metric-view-bar"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="metric-chart-line"]').attributes('data-values')).toBe('3,4,5')

    await wrapper.setProps({ period: 'all' })

    expect(wrapper.get('[data-testid="metric-chart-line"]').attributes('data-values')).toBe('1,2,3,4,5')
  })

  it('switches between curve and bar views without changing the selected values', async () => {
    const wrapper = mount(ProgressMetricCard, {
      props: {
        testId: 'daily-requests',
        label: 'Daily requests',
        icon: BarChart3,
        detailed: true,
        series: [2, 5, 8],
        defaultView: 'curve'
      }
    })

    expect(wrapper.find('[data-testid="metric-chart-line"]').exists()).toBe(true)
    await wrapper.get('[data-testid="metric-view-bar"]').trigger('click')
    expect(wrapper.find('[data-testid="metric-chart-line"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="metric-chart-bar"]').attributes('data-values')).toBe('2,5,8')
  })
})
