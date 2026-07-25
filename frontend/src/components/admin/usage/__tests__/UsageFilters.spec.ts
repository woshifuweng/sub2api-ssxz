import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import UsageFilters from '../UsageFilters.vue'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: vi.fn().mockResolvedValue({ items: [] }),
    },
    dashboard: {
      getModelStats: vi.fn().mockResolvedValue({ models: [] }),
    },
    groups: {
      list: vi.fn().mockResolvedValue({ items: [] }),
    },
    usage: {
      searchApiKeys: vi.fn().mockResolvedValue([]),
      searchUsers: vi.fn().mockResolvedValue([]),
    },
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('admin UsageFilters', () => {
  it('updates request_id filter and emits change', async () => {
    const modelValue: Record<string, unknown> = {}
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue,
        exporting: false,
        startDate: '2026-07-08',
        endDate: '2026-07-08',
      },
      global: {
        stubs: {
          Select: true,
        },
      },
    })

    await wrapper.get('[data-test="usage-request-id-filter"]').setValue(' req_customer_123 ')

    expect(modelValue.request_id).toBe('req_customer_123')
    expect(wrapper.emitted('change')).toHaveLength(1)
  })
})
