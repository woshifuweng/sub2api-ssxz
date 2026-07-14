import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { DashboardOperationsSummary } from '@/api/admin/dashboard'
import AdminOperationsSummary from '../AdminOperationsSummary.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const summary: DashboardOperationsSummary = {
  start_date: '2026-07-01T00:00:00Z',
  end_date: '2026-07-15T00:00:00Z',
  new_customers: 7,
  customer_actual_cost: 12.34,
  invitee_recharge_amount: 50,
  rebate_pending: 1.25,
  rebate_available: 2.5,
  rebate_transferred: 0.75,
  active_customers: 4,
  active_api_keys: 6,
  top_customers: [
    {
      user_id: 11,
      email: 'one@example.com',
      username: 'One',
      actual_cost: 8.25,
      requests: 12,
      active_keys: 2
    }
  ]
}

describe('AdminOperationsSummary', () => {
  it('renders real values and emits range and drilldown actions', async () => {
    const wrapper = mount(AdminOperationsSummary, {
      props: { summary, range: '30d' }
    })

    expect(wrapper.get('[data-testid="operations-metric-customers"]').text()).toContain('7')
    expect(wrapper.get('[data-testid="operations-metric-spend"]').text()).toContain('$12.34')
    expect(wrapper.get('[data-testid="operations-metric-active"]').text()).toContain('4 / 6')
    expect(wrapper.get('[data-testid="operations-top-11"]').text()).toContain('One')
    expect(wrapper.get('[data-testid="operations-top-11"]').text()).toContain('$8.25')

    await wrapper.get('[data-testid="operations-range-7d"]').trigger('click')
    expect(wrapper.emitted('update:range')).toEqual([['7d']])

    await wrapper.get('[data-testid="operations-metric-spend"]').trigger('click')
    expect(wrapper.emitted('drilldown')).toContainEqual(['usage'])

    await wrapper.get('[data-testid="operations-top-11"]').trigger('click')
    expect(wrapper.emitted('drilldown')).toContainEqual(['customer', 11])
  })

  it('shows explicit zero values instead of an empty-data failure state', () => {
    const wrapper = mount(AdminOperationsSummary, {
      props: {
        summary: {
          ...summary,
          new_customers: 0,
          customer_actual_cost: 0,
          invitee_recharge_amount: 0,
          active_customers: 0,
          active_api_keys: 0,
          top_customers: []
        },
        range: 'today'
      }
    })

    expect(wrapper.get('[data-testid="operations-metric-customers"]').text()).toContain('0')
    expect(wrapper.get('[data-testid="operations-metric-spend"]').text()).toContain('$0.0000')
    expect(wrapper.text()).toContain('admin.dashboard.operations.zeroCustomers')
  })
})
