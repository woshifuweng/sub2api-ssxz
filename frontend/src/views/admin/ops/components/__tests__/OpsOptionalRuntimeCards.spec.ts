import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OpsGatewaySchedulerCard from '../OpsGatewaySchedulerCard.vue'
import OpsOpenAIWSRuntimeCard from '../OpsOpenAIWSRuntimeCard.vue'

const { gateway, websocket } = vi.hoisted(() => ({ gateway: vi.fn(), websocket: vi.fn() }))
vi.mock('@/api/admin/ops', () => ({ opsAPI: {
  getGatewaySchedulerRuntime: gateway, getOpenAIWSRuntime: websocket
} }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe.each([
  ['gateway', OpsGatewaySchedulerCard, gateway],
  ['websocket', OpsOpenAIWSRuntimeCard, websocket]
] as const)('%s optional runtime', (_name, component, request) => {
  const wrappers: ReturnType<typeof mount>[] = []
  const render = () => {
    const wrapper = mount(component, { props: { refreshToken: 0, active: true } })
    wrappers.push(wrapper)
    return wrapper
  }
  beforeEach(() => { gateway.mockReset(); websocket.mockReset() })
  afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

  it.each([{ status: 404 }, { response: { status: 404 } }])('stops polling after unsupported response %j', async error => {
    request.mockRejectedValue(error)
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('admin.ops.runtimeUnsupported')
    expect(wrapper.text()).not.toContain('0 scheduling')
    expect(wrapper.text()).not.toContain('No OpenAI WS or gnet')
    expect(wrapper.findComponent({ name: 'EmptyState' }).exists()).toBe(false)
    await wrapper.setProps({ refreshToken: 1, platformFilter: 'openai' })
    await wrapper.setProps({ active: false })
    await wrapper.setProps({ active: true, refreshToken: 2 })
    await flushPromises()
    expect(request).toHaveBeenCalledTimes(1)
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('allows manual and automatic retry after 500 without rendering empty health', async () => {
    request.mockRejectedValue({ status: 500, message: 'temporary failure' })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('common.loadFailed')
    expect(wrapper.text()).not.toContain('0 scheduling')
    expect(wrapper.text()).not.toContain('No OpenAI WS or gnet')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(request).toHaveBeenCalledTimes(2)
    request.mockResolvedValue({ items: [], active_scheduling_accounts: 3, acquire_total: 1 })
    await wrapper.setProps({ refreshToken: 1 })
    await flushPromises()
    expect(request).toHaveBeenCalledTimes(3)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('ignores stale failures after a newer request succeeds', async () => {
    let rejectOld!: (reason: unknown) => void
    request.mockImplementationOnce(() => new Promise((_resolve, reject) => { rejectOld = reject }))
    request.mockResolvedValue({ items: [], acquire_total: 1 })
    const wrapper = render()
    await wrapper.setProps({ platformFilter: 'openai' })
    await flushPromises()
    rejectOld({ status: 404 })
    await flushPromises()
    expect(wrapper.text()).not.toContain('admin.ops.runtimeUnsupported')
    await wrapper.setProps({ refreshToken: 1 })
    await flushPromises()
    expect(request).toHaveBeenCalledTimes(3)
  })

  it('does not let an older success replace the unsupported state', async () => {
    let resolveOld!: (result: unknown) => void
    request.mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    request.mockRejectedValue({ status: 404 })
    const wrapper = render()
    await wrapper.setProps({ platformFilter: 'openai' })
    await flushPromises()
    resolveOld({ items: [], acquire_total: 0 })
    await flushPromises()
    expect(wrapper.text()).toContain('admin.ops.runtimeUnsupported')
    await wrapper.setProps({ refreshToken: 1 })
    expect(request).toHaveBeenCalledTimes(2)
  })

  it('keeps zero activity from a successful response distinct from an error', async () => {
    request.mockResolvedValue({ items: [], active_scheduling_accounts: 0, acquire_total: 0 })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('admin.ops.runtimeUnsupported')
    expect(wrapper.findComponent({ name: 'EmptyState' }).exists()).toBe(true)
  })

  it('does not request data while inactive, including through a disabled retry', async () => {
    request.mockRejectedValue({ status: 500 })
    const wrapper = mount(component, { props: { active: false } })
    wrappers.push(wrapper)
    await flushPromises()
    expect(request).not.toHaveBeenCalled()
    await wrapper.setProps({ active: true })
    await flushPromises()
    await wrapper.setProps({ active: false })
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    await wrapper.get('button').trigger('click')
    expect(request).toHaveBeenCalledTimes(1)
  })
})
