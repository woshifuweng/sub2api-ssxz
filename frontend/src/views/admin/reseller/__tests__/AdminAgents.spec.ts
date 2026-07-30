import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  listAdminAgents,
  getAdminAgent,
  disableAdminAgent,
  enableAdminAgent,
  revokeAdminRole,
  grantAdminRole,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  listAdminAgents: vi.fn(),
  getAdminAgent: vi.fn(),
  disableAdminAgent: vi.fn(),
  enableAdminAgent: vi.fn(),
  revokeAdminRole: vi.fn(),
  grantAdminRole: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/reseller', () => ({
  default: {
    listAdminAgents,
    getAdminAgent,
    disableAdminAgent,
    enableAdminAgent,
    revokeAdminRole,
    grantAdminRole
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/admin/reseller/agents' })
}))

import AdminAgents from '../AdminAgents.vue'

const activeAgent = {
  user_id: 7,
  email: 'agent@example.com',
  username: 'Agent Seven',
  role: 'agent_manager' as const,
  status: 'active' as const,
  manager_id: null,
  manager_email: null,
  effective_rebate_rate_percent: 5,
  rebate_mode: 'global' as const,
  aff_code: 'AGENT7',
  recruit_count: 3,
  commission_balance: '12.00',
  commission_total: '28.00',
  notes: 'regional lead',
  granted_at: '2026-07-30T00:00:00Z',
  updated_at: '2026-07-30T01:00:00Z',
  disabled_at: null,
  disabled_by_email: null,
  disabled_reason: '',
  revoked_at: null,
  granted_by: 1
}

function page(items = [activeAgent]) {
  return { items, total: items.length, page: 1, page_size: 20, pages: 1 }
}

function mountView() {
  return mount(AdminAgents, {
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
                <slot name="cell-status" :row="row" />
                <slot name="cell-role" :row="row" />
                <slot name="cell-manager_email" :row="row" :value="row.manager_email" />
                <slot name="cell-rebate" :row="row" />
                <slot name="cell-commission_balance" :row="row" :value="row.commission_balance" />
                <slot name="cell-commission_total" :row="row" :value="row.commission_total" />
                <slot name="cell-actions" :row="row" />
              </div>
            </div>
          `
        },
        Pagination: true,
        LiquidButton: {
          inheritAttrs: false,
          template: '<button v-bind="$attrs"><slot /></button>'
        },
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        },
        AdminAgentGrantDialog: true,
        AdminAgentEditDrawer: true,
        AdminAgentDetailDrawer: true,
        Icon: true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
      }
    }
  })
}

describe('AdminAgents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listAdminAgents.mockResolvedValue(page())
    getAdminAgent.mockResolvedValue({ ...activeAgent, aff_history_quota: 30, pending_redemption_count: 0 })
    disableAdminAgent.mockResolvedValue({ ...activeAgent, status: 'disabled' })
    enableAdminAgent.mockResolvedValue(activeAgent)
    revokeAdminRole.mockResolvedValue({ user_id: 7 })
    grantAdminRole.mockResolvedValue({ user_id: 7, role: 'agent_manager' })
  })

  it('renders real lifecycle, rebate, and commission fields', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listAdminAgents).toHaveBeenCalledWith(1, 20, {
      search: undefined,
      status: '',
      role: '',
      manager_id: undefined
    })
    expect(wrapper.text()).toContain('Agent Seven')
    expect(wrapper.text()).toContain('启用中')
    expect(wrapper.text()).toContain('Agent Manager')
    expect(wrapper.text()).toContain('跟随全局（5%）')
    expect(wrapper.text()).toContain('$12.00')
    expect(wrapper.text()).toContain('$28.00')
  })

  it('requires a reason before disabling an active agent', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="disable-agent-7"]').trigger('click')
    const confirm = wrapper.get('[data-testid="confirm-disable-agent"]')
    expect(confirm.attributes('disabled')).toBeDefined()

    await wrapper.get('textarea[placeholder="请填写停用原因"]').setValue('合作暂停')
    await confirm.trigger('click')
    await flushPromises()

    expect(disableAdminAgent).toHaveBeenCalledWith(7, '合作暂停')
    expect(showSuccess).toHaveBeenCalled()
  })

  it('requires confirmation before the irreversible role revoke', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="revoke-agent-7"]').trigger('click')
    expect(revokeAdminRole).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('这不是临时停用')

    await wrapper.get('[data-testid="confirm-revoke-agent"]').trigger('click')
    await flushPromises()

    expect(revokeAdminRole).toHaveBeenCalledWith(7)
    expect(showSuccess).toHaveBeenCalled()
  })

  it('restores a disabled agent without deleting and recreating the role', async () => {
    listAdminAgents.mockResolvedValue(page([{ ...activeAgent, status: 'disabled' as const }]))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="enable-agent-7"]').trigger('click')
    await flushPromises()

    expect(enableAdminAgent).toHaveBeenCalledWith(7)
    expect(revokeAdminRole).not.toHaveBeenCalled()
  })
})
