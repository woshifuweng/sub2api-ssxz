import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listAdminAgents, revokeAdminRole, showError, showSuccess } = vi.hoisted(() => ({
  listAdminAgents: vi.fn(),
  revokeAdminRole: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/reseller', () => ({
  default: { listAdminAgents, revokeAdminRole }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/admin/reseller/agents' })
}))

import AdminAgents from '../AdminAgents.vue'

describe('AdminAgents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    revokeAdminRole.mockResolvedValue({ user_id: 7 })
    listAdminAgents.mockResolvedValue({
      items: [{
        user_id: 7,
        email: 'agent@example.com',
        username: 'Agent Seven',
        role: 'agent_manager',
        commission_rate: 0.125,
        aff_code: 'AGENT7',
        recruit_count: 3,
        aff_quota: 12,
        granted_at: '2026-07-30T00:00:00Z',
        granted_by: 1
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
  })

  it('loads the admin agent list and keeps unavailable fields explicit', async () => {
    const wrapper = mount(AdminAgents, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          AdminPageHeader: { template: '<header><slot name="actions" /></header>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: {
            props: ['data'],
            template: `
              <div>
                <div v-for="row in data" :key="row.user_id">
                  <slot name="cell-user" :row="row" />
                  <slot name="cell-role" :row="row" />
                  <slot name="cell-manager" :row="row" />
                  <slot name="cell-total_commission" :row="row" />
                  <slot name="cell-commission_rate" :row="row" :value="row.commission_rate" />
                  <slot name="cell-actions" :row="row" />
                </div>
              </div>
            `
          },
          Pagination: true,
          LiquidButton: { template: '<button><slot /></button>' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          },
          AdminAgentGrantDialog: true,
          Icon: true,
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
        }
      }
    })

    await flushPromises()

    expect(listAdminAgents).toHaveBeenCalledWith(1, 20, '')
    expect(wrapper.text()).toContain('Agent Seven')
    expect(wrapper.text()).toContain('agent@example.com')
    expect(wrapper.text()).toContain('Agent Manager')
    expect(wrapper.text()).toContain('12.50%')
    expect(wrapper.text()).toContain('--')
  })

  it('requires confirmation before revoking a reseller role and refreshes the list', async () => {
    const wrapper = mount(AdminAgents, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          AdminPageHeader: { template: '<header><slot name="actions" /></header>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: {
            props: ['data'],
            template: `
              <div>
                <div v-for="row in data" :key="row.user_id">
                  <slot name="cell-user" :row="row" />
                  <slot name="cell-actions" :row="row" />
                </div>
              </div>
            `
          },
          Pagination: true,
          LiquidButton: { template: '<button><slot /></button>' },
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          },
          AdminAgentGrantDialog: true,
          Icon: true,
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-testid="revoke-agent-7"]').trigger('click')

    expect(revokeAdminRole).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('不会删除用户账号')

    await wrapper.get('[data-testid="confirm-revoke-agent"]').trigger('click')
    await flushPromises()

    expect(revokeAdminRole).toHaveBeenCalledWith(7)
    expect(showSuccess).toHaveBeenCalled()
    expect(listAdminAgents).toHaveBeenCalledTimes(2)
  })
})
