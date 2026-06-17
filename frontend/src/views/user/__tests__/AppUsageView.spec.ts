import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { usageAPI, authStore } = vi.hoisted(() => ({
  usageAPI: {
    getStatsByDateRange: vi.fn(),
    query: vi.fn(),
    getDashboardTrend: vi.fn()
  },
  authStore: {
    user: {
      balance: 8.53
    }
  }
}))

vi.mock('@/api', () => ({
  usageAPI
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: `
      <main data-testid="app-section-shell">
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
        <slot />
      </main>
    `
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import AppUsageView from '../AppUsageView.vue'

function mountView() {
  return mount(AppUsageView, {
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

describe('AppUsageView', () => {
  beforeEach(() => {
    usageAPI.getStatsByDateRange.mockReset()
    usageAPI.query.mockReset()
    usageAPI.getDashboardTrend.mockReset()
    authStore.user.balance = 8.53
  })

  it('renders usage data inside the new workbench page instead of the old usage UI', async () => {
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 3,
      total_input_tokens: 120,
      total_output_tokens: 80,
      total_cache_tokens: 0,
      total_tokens: 200,
      total_cost: 1.2345,
      total_actual_cost: 1.2345,
      average_duration_ms: 1200
    })
    usageAPI.query.mockResolvedValue({
      items: [
        {
          id: 7,
          request_id: 'req-image',
          api_key_id: 1,
          model: 'gpt-image-2',
          inbound_endpoint: '/v1/images/generations',
          input_tokens: 0,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          image_count: 2,
          image_size: '1024x1024',
          actual_cost: 0.88,
          created_at: '2026-06-18T08:00:00Z'
        }
      ],
      total: 1,
      pages: 1
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [
        {
          date: '2026-06-18',
          requests: 3,
          input_tokens: 120,
          output_tokens: 80,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          total_tokens: 200,
          cost: 1.2345,
          actual_cost: 1.2345
        }
      ],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    const text = wrapper.text()
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(text).toContain('用量信息')
    expect(text).toContain('账户余额')
    expect(text).toContain('$8.53')
    expect(text).toContain('本月消耗')
    expect(text).toContain('$1.2345')
    expect(text).toContain('每月用量')
    expect(text).toContain('真实数据')
    expect(text).toContain('用量明细')
    expect(text).toContain('图片生成')
    expect(text).toContain('gpt-image-2')
    expect(text).toContain('2 张 · 1024x1024')
    expect(usageAPI.query).toHaveBeenCalledWith(expect.objectContaining({
      page: 1,
      page_size: 8
    }))
  })

  it('shows empty states without inventing usage rows when the existing APIs have no data', async () => {
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0
    })
    usageAPI.query.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0
    })
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('暂无月度用量数据')
    expect(wrapper.text()).toContain('暂无用量明细')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('keeps the workbench page in place when usage details cannot load', async () => {
    usageAPI.getStatsByDateRange.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0
    })
    usageAPI.query.mockRejectedValue(new Error('unavailable'))
    usageAPI.getDashboardTrend.mockResolvedValue({
      trend: [],
      start_date: '2026-01-01',
      end_date: '2026-06-18',
      granularity: 'day'
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('用量明细暂时无法加载')
    expect(wrapper.text()).toContain('不会自动跳到旧版页面')
    expect(wrapper.find('table').exists()).toBe(false)
  })
})
