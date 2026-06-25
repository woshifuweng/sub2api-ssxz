import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useAppStore } from '@/stores/app'

const mocks = vi.hoisted(() => ({
  route: { __v_isRef: true, value: { meta: { appSection: 'image' }, path: '/app/chat', query: {} as Record<string, string> } },
  routerReplace: vi.fn(async (location: { query?: Record<string, string> }) => {
    mocks.route.value = {
      ...mocks.route.value,
      query: location.query ?? {}
    }
  }),
  chatModels: { __v_isRef: true, value: [
    { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium', capabilities: ['text_chat'], modelCatalogSource: 'real_channel' },
    { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard', capabilities: ['text_chat'], modelCatalogSource: 'real_channel' }
  ] },
  defaultTextModel: { __v_isRef: true, value: 'gpt-5.5' },
  hasChat: { __v_isRef: true, value: true },
  loadCapabilities: vi.fn(),
  apiClient: {
    get: vi.fn(),
    post: vi.fn()
  },
  createObjectURL: vi.fn((file: File) => `blob:workspace-preview-${file.name}`),
  revokeObjectURL: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route.value,
  useRouter: () => ({
    replace: mocks.routerReplace
  })
}))

vi.mock('@/composables/useUserCapabilities', () => ({
  useUserCapabilities: () => ({
    chatModels: mocks.chatModels,
    defaultTextModel: mocks.defaultTextModel,
    hasChat: mocks.hasChat,
    loadCapabilities: mocks.loadCapabilities
  })
}))

vi.mock('@/api/client', () => ({
  apiClient: mocks.apiClient
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    emits: ['new-chat'],
    template: `
      <div>
        <button class="test-new-chat" type="button" @click="$emit('new-chat')">New chat</button>
        <slot />
      </div>
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

import AppWorkspaceView from '../AppWorkspaceView.vue'
import WorkspaceComposer from '../workspace/WorkspaceComposer.vue'
import { WORKSPACE_TEXT_ONLY_MESSAGE } from '../workspace/useWorkspaceConversation'

function mountWithAppStore() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const appStore = useAppStore()
  appStore.cachedPublicSettings = {
    web_search: {
      available: false
    }
  } as any

  return mount(AppWorkspaceView, {
    global: {
      plugins: [pinia]
    }
  })
}

describe('AppWorkspaceView interactions', () => {
  beforeEach(() => {
    mocks.route.value = { meta: { appSection: 'image' }, path: '/app/chat', query: {} }
    mocks.routerReplace.mockClear()
    mocks.chatModels.value = [
      { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium', capabilities: ['text_chat'], modelCatalogSource: 'real_channel' },
      { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard', capabilities: ['text_chat'], modelCatalogSource: 'real_channel' }
    ]
    mocks.defaultTextModel.value = 'gpt-5.5'
    mocks.hasChat.value = true
    mocks.loadCapabilities.mockReset()
    mocks.apiClient.get.mockReset()
    mocks.apiClient.post.mockReset()
    mocks.createObjectURL.mockClear()
    mocks.revokeObjectURL.mockClear()
    vi.stubGlobal('URL', {
      createObjectURL: mocks.createObjectURL,
      revokeObjectURL: mocks.revokeObjectURL
    })
  })

  it('mounts the shell with honest text-beta framing', async () => {
    const wrapper = mountWithAppStore()
    await flushPromises()

    expect(mocks.loadCapabilities).toHaveBeenCalledTimes(1)
    expect(mocks.apiClient.get).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('GPT-5.5')
    expect(wrapper.text()).toContain('文本对话')
    expect(wrapper.text()).toContain('图片理解')
    expect(wrapper.text()).toContain('多图分析')
    expect(wrapper.text()).toContain('AI 作图页')
    expect(wrapper.text()).not.toContain('上传参考图')
    expect(wrapper.text()).not.toContain('生成、修改的画面')

    wrapper.unmount()
  })

  it('opens the model menu and closes after selecting a model', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()

    const trigger = wrapper.get('.model-trigger')
    expect(wrapper.find('.model-menu').exists()).toBe(false)

    await trigger.trigger('click')
    expect(wrapper.find('.model-menu').exists()).toBe(true)
    expect(wrapper.findAll('.model-option')).toHaveLength(2)

    await wrapper.findAll('.model-option')[1].trigger('click')
    expect(wrapper.find('.model-menu').exists()).toBe(false)
    expect(wrapper.get('.model-trigger').text()).toContain('GPT-5.4-Mini')

    wrapper.unmount()
  })

  it('keeps sending disabled and does not call chat APIs when backend is unavailable', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()

    await wrapper.get('textarea').setValue('hello')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.get('[data-testid="workspace-send"]').attributes('disabled')).toBeDefined()
    expect(wrapper.findAll('.message-row')).toHaveLength(0)
    expect(mocks.apiClient.get).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('does not expose the image upload entry during text beta', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()

    expect(wrapper.find('[data-testid="workspace-add-content"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('暂未接入')
    expect(mocks.createObjectURL).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('does not expose image upload in chat even when catalog models include vision capability', async () => {
    mocks.chatModels.value = [
      { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium', capabilities: ['text_chat', 'vision'], modelCatalogSource: 'real_channel' }
    ]

    const wrapper = mountWithAppStore()
    await nextTick()
    const file = new File(['image'], 'vision-sample.png', { type: 'image/png' })

    expect(wrapper.find('[data-testid="workspace-add-content"]').exists()).toBe(false)
    expect(wrapper.find('textarea').attributes('placeholder')).toContain('直接输入问题')
    wrapper.findComponent(WorkspaceComposer).vm.$emit('files', [file])
    await nextTick()

    expect(wrapper.text()).toContain(WORKSPACE_TEXT_ONLY_MESSAGE)
    expect(wrapper.text()).toContain('图片理解')
    expect(wrapper.text()).toContain('多图分析')
    expect(wrapper.find('[data-testid="workspace-asset-previews"]').exists()).toBe(false)
    expect(mocks.createObjectURL).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('rejects unsupported file attempts in text beta without creating local previews', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()
    const file = new File(['image'], 'sample.png', { type: 'image/png' })

    wrapper.findComponent(WorkspaceComposer).vm.$emit('unsupported-files', [file])
    await nextTick()

    expect(wrapper.text()).toContain(WORKSPACE_TEXT_ONLY_MESSAGE)
    expect(wrapper.find('[data-testid="workspace-asset-previews"]').exists()).toBe(false)
    expect(mocks.createObjectURL).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('rejects dragged image files in text beta without creating local previews', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()
    const file = new File(['image'], 'dragged-sample.png', { type: 'image/png' })
    const dropEvent = new Event('drop', { bubbles: true, cancelable: true })
    Object.defineProperty(dropEvent, 'dataTransfer', {
      value: { files: [file] }
    })

    wrapper.get('form').element.dispatchEvent(dropEvent)
    await nextTick()

    expect(dropEvent.defaultPrevented).toBe(true)
    expect(wrapper.text()).toContain(WORKSPACE_TEXT_ONLY_MESSAGE)
    expect(wrapper.find('[data-testid="workspace-asset-previews"]').exists()).toBe(false)
    expect(mocks.createObjectURL).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('does not register files if the image capability is unavailable', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()
    const file = new File(['image'], 'sample.png', { type: 'image/png' })

    wrapper.findComponent(WorkspaceComposer).vm.$emit('files', [file])
    await nextTick()

    expect(wrapper.text()).toContain(WORKSPACE_TEXT_ONLY_MESSAGE)
    expect(wrapper.find('[data-testid="workspace-asset-previews"]').exists()).toBe(false)
    expect(mocks.createObjectURL).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('clears local shell state when starting a new chat without backend calls', async () => {
    const wrapper = mountWithAppStore()
    await nextTick()

    await wrapper.get('textarea').setValue('hello')
    await wrapper.get('.test-new-chat').trigger('click')

    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.findAll('.message-row')).toHaveLength(0)
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})
