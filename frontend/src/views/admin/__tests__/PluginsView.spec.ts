import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import PluginsView from '../PluginsView.vue'
enableAutoUnmount(afterEach)

const {
  listPlugins,
  uploadPlugin,
  enablePlugin,
  savePluginConfig,
  createUISession,
  stepUpRun,
} = vi.hoisted(() => ({
  listPlugins: vi.fn(),
  uploadPlugin: vi.fn(),
  enablePlugin: vi.fn(),
  savePluginConfig: vi.fn(),
  createUISession: vi.fn(),
  stepUpRun: vi.fn((action: () => Promise<unknown>) => action()),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    plugins: {
      list: listPlugins,
      upload: uploadPlugin,
      enable: enablePlugin,
      disable: vi.fn(),
      remove: vi.fn(),
      getConfig: vi.fn().mockResolvedValue({}),
      saveConfig: savePluginConfig,
      test: vi.fn().mockResolvedValue({ success: true, message: 'ok', latency_ms: 1 }),
      createUISession,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({ run: stepUpRun }),
  isStepUpBlocked: () => false,
  isStepUpCancelled: () => false,
  stepUpBlockReason: () => '',
}))

vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => key }),
}))

const plugin = {
  id: 7,
  plugin_key: 'local.test.transport',
  name: 'Test Transport',
  version: '1.0.0',
  description: '',
  author: 'test',
  manifest: {
    schema_version: 1,
    id: 'local.test.transport',
    name: 'Test Transport',
    version: '1.0.0',
    requires: {
      sub2api: '>=0.1.0',
      plugin_protocol: 1,
      transport_api: 1,
      ui_bridge: 1,
    },
    capabilities: [],
    ui: { entrypoint: 'ui/index.html' },
  },
  binary_sha256: 'a'.repeat(64),
  signature_status: 'trusted' as const,
  state: 'disabled' as const,
  last_error: '',
  installed_at: '2026-08-22T00:00:00Z',
  updated_at: '2026-08-22T00:00:00Z',
  bindings: [
    {
      id: 1,
      plugin_id: 7,
      capability: 'openai.oauth.outbound_transport.v1',
      platform: 'openai',
      account_type: 'oauth',
      enabled: false,
      rollout_percent: 100,
    },
  ],
  compatibility: {
    compatible: true,
    tested: true,
    status: 'compatible' as const,
    message: '',
    current_sub2api_version: '0.1.0',
    required_sub2api_version: '>=0.1.0',
    recommended_sub2api_version: '0.1.0',
    plugin_protocol: 1,
    transport_api: 1,
    ui_bridge: 1,
  },
  runtime_healthy: false,
  runtime_message: '',
}

function mountView() {
  return mount(PluginsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { name: 'BaseDialog', props: ['show'], emits: ['close'], template: '<div v-if="show"><button data-testid="close-config" @click="$emit(\'close\')">Close</button><slot /></div>' },
        Icon: true,
        TotpStepUpDialog: true,
      },
    },
  })
}

describe('管理员插件页二次验证', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    stepUpRun.mockImplementation((action: () => Promise<unknown>) => action())
    listPlugins.mockResolvedValue([plugin])
    uploadPlugin.mockResolvedValue(plugin)
    enablePlugin.mockResolvedValue(plugin)
    savePluginConfig.mockResolvedValue({ enabled: true })
    createUISession.mockResolvedValue({
      url: '/api/v1/plugin-ui/token/index.html#bridge_token=bridge',
      bridge_token: 'bridge',
      ui_bridge_version: 1,
      expires_at: '2026-08-22T01:00:00Z',
    })
  })

  it('启用插件通过 step-up 控制器执行', async () => {
    const wrapper = mountView()
    await flushPromises()

    const button = wrapper.findAll('button').find((item) => item.text().includes('admin.plugins.enable'))
    expect(button).toBeDefined()
    await button!.trigger('click')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(enablePlugin).toHaveBeenCalledWith(7, 100, false)
  })

  it('上传插件通过 step-up 控制器执行', async () => {
    const wrapper = mountView()
    await flushPromises()
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [new File(['plugin'], 'transport.s2plugin', { type: 'application/zip' })],
    })

    await input.trigger('change')
    await flushPromises()

    expect(stepUpRun).toHaveBeenCalledTimes(1)
    expect(uploadPlugin).toHaveBeenCalledTimes(1)
  })

  it('keeps incompatible plugins unavailable without calling enable', async () => {
    listPlugins.mockResolvedValue([{ ...plugin, compatibility: { ...plugin.compatibility, compatible: false } }])
    const wrapper = mountView()
    await flushPromises()
    const button = wrapper.findAll('button').find(item => item.text().includes('admin.plugins.enable'))!
    expect(button.attributes('disabled')).toBeDefined()
    await button.trigger('click')
    expect(enablePlugin).not.toHaveBeenCalled()
    expect(stepUpRun).not.toHaveBeenCalled()
  })

  it('ignores a superseded configuration session when another plugin is opened', async () => {
    let resolveFirst!: (value: unknown) => void
    createUISession.mockImplementationOnce(() => new Promise(resolve => { resolveFirst = resolve }))
    createUISession.mockResolvedValueOnce({ url: '/api/v1/plugin-ui/second/index.html', bridge_token: 'second', ui_bridge_version: 1, expires_at: '' })
    listPlugins.mockResolvedValue([plugin, { ...plugin, id: 8, name: 'Second Transport' }])
    const wrapper = mountView()
    await flushPromises()
    const configure = wrapper.findAll('button').filter(item => item.text().includes('admin.plugins.configure'))
    await configure[0].trigger('click')
    await wrapper.get('[data-testid="close-config"]').trigger('click')
    await configure[1].trigger('click')
    await flushPromises()
    expect(wrapper.get('iframe').attributes('sandbox')).toBe('allow-scripts')
    expect(wrapper.get('iframe').attributes('src')).toContain('/second/')
    resolveFirst({ url: '/api/v1/plugin-ui/first/index.html', bridge_token: 'first', ui_bridge_version: 1, expires_at: '' })
    await flushPromises()
    expect(wrapper.get('iframe').attributes('src')).toContain('/second/')
  })

  it('removes the bridge listener when leaving the page', async () => {
    const add = vi.spyOn(window, 'addEventListener')
    const remove = vi.spyOn(window, 'removeEventListener')
    const wrapper = mountView()
    await flushPromises()
    const listener = add.mock.calls.find(([name]) => name === 'message')![1]
    wrapper.unmount()
    expect(remove).toHaveBeenCalledWith('message', listener)
    add.mockRestore()
    remove.mockRestore()
  })

  it('does not save after its configuration closes during step-up', async () => {
    let release!: () => void
    stepUpRun.mockImplementationOnce(async action => {
      await new Promise<void>(resolve => { release = resolve })
      return action()
    })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.findAll('button').find(item => item.text().includes('admin.plugins.configure'))!.trigger('click')
    await flushPromises()
    const source = { postMessage: vi.fn() }
    Object.defineProperty(wrapper.get('iframe').element, 'contentWindow', { value: source })
    const event = new MessageEvent('message', {
      origin: 'null',
      data: { source: 'sub2api-plugin-ui', bridge_token: 'bridge', type: 'config.save', request_id: 'save-1', config: { enabled: true } },
    })
    Object.defineProperty(event, 'source', { value: source })
    window.dispatchEvent(event)
    expect(stepUpRun).toHaveBeenCalledTimes(1)
    await wrapper.get('[data-testid="close-config"]').trigger('click')
    release()
    await flushPromises()
    expect(savePluginConfig).not.toHaveBeenCalled()
    expect(source.postMessage).not.toHaveBeenCalled()
  })

  it('rejects configuration messages from another frame or origin', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.findAll('button').find(item => item.text().includes('admin.plugins.configure'))!.trigger('click')
    await flushPromises()
    const source = { postMessage: vi.fn() }
    Object.defineProperty(wrapper.get('iframe').element, 'contentWindow', { value: source })
    for (const [origin, sender] of [['https://untrusted.invalid', source], ['null', {}]] as const) {
      const event = new MessageEvent('message', {
        origin,
        data: { source: 'sub2api-plugin-ui', bridge_token: 'bridge', type: 'config.save', request_id: 'save-1', config: { enabled: true } },
      })
      Object.defineProperty(event, 'source', { value: sender })
      window.dispatchEvent(event)
    }
    await flushPromises()
    expect(stepUpRun).not.toHaveBeenCalled()
    expect(savePluginConfig).not.toHaveBeenCalled()
  })
})
