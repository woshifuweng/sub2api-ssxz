import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import ImageToolsWorkspace from '../ImageToolsWorkspace.vue'

const api = vi.hoisted(() => ({ groups: vi.fn(), channels: vi.fn(), list: vi.fn(), create: vi.fn(), reveal: vi.fn() }))
const auth = reactive({ user: { id: 7 }, isAuthenticated: true })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getAvailable: api.groups } }))
vi.mock('@/api/channels', () => ({ userChannelsAPI: { getAvailable: api.channels } }))
vi.mock('@/api/keys', () => ({ list: api.list, create: api.create, reveal: api.reveal }))

const group = { id: 10, name: '图片分组', status: 'active', platform: 'openai', allow_image_generation: true }
const imageKey = { id: 33, name: 'SSXZ 图片工具 · 10', user_id: 7, status: 'active', group_id: 10,
  group_ids: [10], allowed_models: ['gpt-image-2'], expires_at: null, quota: 0, quota_used: 0, key: 'test-image-key' }
let cleanup: (() => void) | undefined

function render() {
  const host = document.createElement('div')
  document.body.appendChild(host)
  const wrapper = mount(ImageToolsWorkspace, { attachTo: host, global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } })
  cleanup = () => { wrapper.unmount(); host.remove() }
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  auth.user = { id: 7 }
  auth.isAuthenticated = true
  api.groups.mockResolvedValue([group])
  api.channels.mockResolvedValue([{ platforms: [{ groups: [{ id: 10 }], supported_models: [{ name: 'gpt-image-2' }] }] }])
  api.list.mockResolvedValue({ items: [imageKey], total: 1, page_size: 100 })
  api.create.mockResolvedValue(imageKey)
  api.reveal.mockResolvedValue({ key: imageKey.key })
  vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => new Response(JSON.stringify({ data: [{ id: 'gpt-image-2' }] }), { status: 200 })))
})

afterEach(() => { cleanup?.(); cleanup = undefined; vi.unstubAllGlobals(); vi.restoreAllMocks() })

describe('ImageToolsWorkspace', () => {
  it('uses native reveal for eligible masked keys on subsequent visits', async () => {
    api.list.mockResolvedValue({ items: [{ ...imageKey, key: 'sk-abc...xyz' }], total: 1, page_size: 100 })
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(api.reveal).toHaveBeenCalledWith(33)
    expect(api.create).not.toHaveBeenCalled()
    expect(wrapper.find('iframe').exists()).toBe(true)
  })

  it('keeps tools open on same-user token and balance refresh', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    const original = wrapper.get('iframe').element
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_token', newValue: 'refreshed-token' }))
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_user', newValue: '{"id":7,"balance":3}' }))
    await flushPromises()
    expect(wrapper.get('iframe').element).toBe(original)
  })
  it('does not create credentials or open paid tools until the customer connects', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('图片工作台')
    expect(wrapper.text()).toContain('无限画布')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(api.create).not.toHaveBeenCalled()
    expect(api.list).not.toHaveBeenCalled()
  })

  it('reuses the image-only key and only sends it to the matching frame/origin', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(api.create).not.toHaveBeenCalled()
    const frame = wrapper.get('iframe').element
    expect(frame.getAttribute('src')).not.toContain(imageKey.key)
    const child = frame.contentWindow
    if (!child) throw new Error('Expected iframe window')
    const post = vi.spyOn(child, 'postMessage')
    window.dispatchEvent(new MessageEvent('message', { origin: 'https://attacker.example', source: child, data: { type: 'ssxz:image-tool:ready', version: 1 } }))
    expect(post).not.toHaveBeenCalled()
    window.dispatchEvent(new MessageEvent('message', { origin: 'https://image.ssxzapi.com', source: window, data: { type: 'ssxz:image-tool:ready', version: 1 } }))
    expect(post).not.toHaveBeenCalled()
    window.dispatchEvent(new MessageEvent('message', { origin: 'https://image.ssxzapi.com', source: child, data: { type: 'ssxz:image-tool:ready', version: 1 } }))
    expect(post).toHaveBeenCalledWith(expect.objectContaining({ accountId: '7', profile: expect.objectContaining({ model: 'gpt-image-2', apiKey: imageKey.key }) }), 'https://image.ssxzapi.com')
  })

  it('creates a native model-restricted key only if none exists', async () => {
    api.list.mockResolvedValue({ items: [], total: 0, page_size: 100 })
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(api.create).toHaveBeenCalledTimes(1)
    expect(api.create.mock.calls[0].slice(0, 4)).toEqual(['SSXZ 图片工具 · 10', 10, [10], ['gpt-image-2']])
    expect(api.create.mock.calls[0][10]).toEqual({ idempotencyKey: expect.any(String) })
  })

  it('does not depend on the optional public channel showcase', async () => {
    api.channels.mockResolvedValue([])
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(true)
    expect(fetch).toHaveBeenCalledWith('/v1/models', expect.any(Object))
    expect(api.channels).not.toHaveBeenCalled()
  })

  it('retries uncertain creates with one idempotency key but uses a new key after deletion', async () => {
    api.list.mockResolvedValue({ items: [], total: 0, page_size: 100 })
    api.create.mockRejectedValueOnce(new Error('response lost')).mockResolvedValue(imageKey)
    let wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    const initial = api.create.mock.calls[0][10].idempotencyKey
    expect(api.create.mock.calls[1][10].idempotencyKey).toBe(initial)
    // The customer deletes it in the native key page, then reopens this page.
    cleanup?.()
    wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(api.create.mock.calls[2][10].idempotencyKey).not.toBe(initial)
    expect(wrapper.find('iframe').exists()).toBe(true)
  })

  it('does not bypass a disabled image key by creating another', async () => {
    api.list.mockResolvedValue({ items: [{ ...imageKey, status: 'disabled' }], total: 1, page_size: 100 })
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(api.create).not.toHaveBeenCalled()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('不会绕过已有的限制')
  })

  it('keeps visited tools mounted while switching tabs, without another key', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    const original = wrapper.get('iframe').element
    await wrapper.get('#tab-canvas').trigger('click')
    expect(wrapper.findAll('iframe')).toHaveLength(2)
    await wrapper.get('#tab-workbench').trigger('click')
    expect(wrapper.get('iframe').element).toBe(original)
    expect(api.list).toHaveBeenCalledTimes(1)
  })

  it('unloads frames immediately on logout and when another tab changes identity', async () => {
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(true)
    window.dispatchEvent(new StorageEvent('storage', { key: 'auth_user', newValue: '{"id":8}' }))
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    auth.isAuthenticated = false
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.get('button.primary').attributes('disabled')).toBeDefined()
  })

  it('does not send an old account key if identity changes during connection', async () => {
    let resolveKeys: ((value: unknown) => void) | undefined
    api.list.mockReturnValue(new Promise(resolve => { resolveKeys = resolve }))
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    auth.user = { id: 8 }
    await flushPromises()
    resolveKeys?.({ items: [imageKey], total: 1, page_size: 100 })
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(api.create).not.toHaveBeenCalled()
  })

  it('does not launch an unsupported model or charge for a connection check', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [] }), { status: 200 })))
    const wrapper = render()
    await flushPromises()
    await wrapper.find('.primary').trigger('click')
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('尚未提交图片生成')
    expect(fetch).toHaveBeenCalledWith('/v1/models', expect.any(Object))
  })
})
