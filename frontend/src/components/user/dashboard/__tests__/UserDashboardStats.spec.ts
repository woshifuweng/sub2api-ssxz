import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data'],
    template: '<div data-testid="metric-sparkline" :data-values="data.datasets[0].data.join(\',\')" />'
  },
  Bar: {
    name: 'Bar',
    props: ['data'],
    template: '<div data-testid="metric-bar" :data-values="data.datasets[0].data.join(\',\')" />'
  }
}))

import UserDashboardStats from '../UserDashboardStats.vue'

const stats = {
  total_api_keys: 3,
  active_api_keys: 2,
  total_requests: 12345,
  total_input_tokens: 1000,
  total_output_tokens: 800,
  total_cache_creation_tokens: 0,
  total_cache_read_tokens: 0,
  total_tokens: 1800,
  total_cost: 9.99,
  total_actual_cost: 8.75,
  today_requests: 27,
  today_input_tokens: 120,
  today_output_tokens: 80,
  today_cache_creation_tokens: 0,
  today_cache_read_tokens: 0,
  today_tokens: 200,
  today_cost: 1.5,
  today_actual_cost: 1.25,
  average_duration_ms: 900,
  rpm: 3,
  tpm: 120
}

const trend = [
  {
    date: '2026-07-14',
    requests: 18,
    input_tokens: 80,
    output_tokens: 40,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: 120,
    cost: 0.9,
    actual_cost: 0.75
  },
  {
    date: '2026-07-15',
    requests: 27,
    input_tokens: 120,
    output_tokens: 80,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: 200,
    cost: 1.5,
    actual_cost: 1.25
  }
]

const todayTrend = [
  {
    date: '2026-07-15T08:00:00',
    requests: 4,
    input_tokens: 30,
    output_tokens: 20,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: 50,
    cost: 0.3,
    actual_cost: 0.25
  },
  {
    date: '2026-07-15T09:00:00',
    requests: 9,
    input_tokens: 45,
    output_tokens: 30,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    total_tokens: 75,
    cost: 0.6,
    actual_cost: 0.5
  }
]

function mountStats(isSimple = false) {
  return mount(UserDashboardStats, {
    props: {
      stats,
      balance: 42.35,
      isSimple,
      trend,
      todayTrend
    },
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('UserDashboardStats', () => {
  it('lets KPI content determine card height without hard-coded minimums', () => {
    const gridSource = readFileSync(resolve(process.cwd(), 'src/components/user/dashboard/UserDashboardStats.vue'), 'utf8')
    const cardSource = readFileSync(resolve(process.cwd(), 'src/components/user/dashboard/ProgressMetricCard.vue'), 'utf8')

    expect(gridSource).toMatch(/\.dashboard-metrics\s*\{[\s\S]*align-items:\s*start;/)
    expect(cardSource).not.toMatch(/min-height:\s*8\.75rem/)
    expect(cardSource).not.toMatch(/\.progress-metric-card\s*\{[\s\S]*height:\s*100%;/)
  })

  it('links balance and usage KPIs to their natural destinations', () => {
    const wrapper = mountStats()

    expect(wrapper.get('[data-testid="metric-card-balance"]').attributes('href')).toBe('/app/purchase')
    for (const testId of [
      'metric-card-total-requests',
      'metric-card-today-requests',
      'metric-card-today-cost',
      'metric-card-today-tokens',
      'metric-card-total-tokens'
    ]) {
      expect(wrapper.get(`[data-testid="${testId}"]`).attributes('href')).toBe('/app/usage')
    }
    expect(wrapper.get('[data-testid="metric-card-performance"]').attributes('href')).toBeUndefined()
    expect(wrapper.get('[data-testid="metric-card-average-duration"]').attributes('href')).toBeUndefined()
  })

  it('keeps all eight native metric groups and their real supporting values', () => {
    const wrapper = mountStats()

    expect(wrapper.findAll('[data-testid^="metric-card-"]')).toHaveLength(8)
    expect(wrapper.get('[data-testid="metric-card-balance"]').text()).toContain('$42.35')
    expect(wrapper.get('[data-testid="metric-card-balance"]').text()).toContain('2 / 3')
    expect(wrapper.get('[data-testid="metric-card-balance"]').text()).toContain('Key 可用率')
    expect(wrapper.get('[data-testid="metric-card-balance"]').text()).toContain('67%')
    expect(wrapper.get('[data-testid="metric-card-balance"]').find('[data-testid="metric-sparkline"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="metric-card-total-requests"] .progress-metric-card__value').text()).toBe('12,345')
    expect(wrapper.get('[data-testid="metric-card-today-requests"] .progress-metric-card__value').text()).toBe('27')
    expect(wrapper.get('[data-testid="metric-card-total-requests"] [data-testid="metric-sparkline"]').attributes('data-values')).toBe('18,27')
    expect(wrapper.get('[data-testid="metric-card-today-requests"] [data-testid="metric-sparkline"]').attributes('data-values')).toBe('4,9')
    expect(wrapper.get('[data-testid="metric-card-today-cost"]').text()).toContain('dashboard.todayCost · dashboard.actual')
    expect(wrapper.get('[data-testid="metric-card-today-cost"]').text()).toContain('dashboard.todayCost · dashboard.standard $1.50')
    expect(wrapper.get('[data-testid="metric-card-today-cost"]').text()).toContain('common.total · dashboard.actual $8.75')
    expect(wrapper.get('[data-testid="metric-card-today-cost"]').text()).toContain('common.total · dashboard.standard $9.99')
    expect(wrapper.get('[data-testid="metric-card-today-cost"]').text()).toContain('$1.25')
    expect(wrapper.get('[data-testid="metric-card-today-cost"]').text()).not.toContain('$1.2500')
    expect(wrapper.get('[data-testid="metric-card-today-tokens"]').text()).toContain('200')
    expect(wrapper.get('[data-testid="metric-card-today-tokens"]').text()).toContain('120')
    expect(wrapper.get('[data-testid="metric-card-today-tokens"]').text()).toContain('80')
    expect(wrapper.get('[data-testid="metric-card-total-tokens"]').text()).toContain('1.8K')
    expect(wrapper.get('[data-testid="metric-card-total-tokens"]').text()).toContain('1.0K')
    expect(wrapper.get('[data-testid="metric-card-total-tokens"]').text()).toContain('800')
    expect(wrapper.get('[data-testid="metric-card-performance"]').text()).toContain('3')
    expect(wrapper.get('[data-testid="metric-card-performance"]').text()).toContain('120')
    expect(wrapper.get('[data-testid="metric-card-average-duration"]').text()).toContain('900ms')
  })

  it('preserves the existing simple-mode rule that hides only the balance card', () => {
    const wrapper = mountStats(true)

    expect(wrapper.findAll('[data-testid^="metric-card-"]')).toHaveLength(7)
    expect(wrapper.find('[data-testid="metric-card-balance"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="metric-card-total-requests"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="metric-card-average-duration"]').exists()).toBe(true)
  })
})
