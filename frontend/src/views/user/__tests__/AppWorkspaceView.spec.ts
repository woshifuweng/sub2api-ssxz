import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'

const mocks = vi.hoisted(() => ({
  route: { __v_isRef: true, value: { meta: { appSection: 'image' } } },
  chatModels: { __v_isRef: true, value: [
    { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' },
    { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard' }
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
  useRoute: () => mocks.route.value
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

describe('AppWorkspaceView interactions', () => {
  beforeEach(() => {
    mocks.route.value = { meta: { appSection: 'image' } }
    mocks.chatModels.value = [
      { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' },
      { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard' }
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

  it('mounts the shell without requesting chat-workspace endpoints', async () => {
    const wrapper = mount(AppWorkspaceView)
    await flushPromises()

    expect(mocks.loadCapabilities).toHaveBeenCalledTimes(1)
    expect(mocks.apiClient.get).not.toHaveBeenCalled()
    expect(mocks.apiClient.post).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('统一工作台后端正在接入')

    wrapper.unmount()
  })

  it('opens the model menu and closes after selecting a model', async () => {
    const wrapper = mount(AppWorkspaceView)
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

  it('keeps sending disabled and does not call chat-studio or chat-workspace APIs', async () => {
    const wrapper = mount(AppWorkspaceView)
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

  it('keeps uploaded images as local previews without backend payloads', async () => {
    const wrapper = mount(AppWorkspaceView)
    await nextTick()

    await wrapper.get('[data-testid="workspace-add-content"]').trigger('click')
    const input = wrapper.get('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [new File(['abc'], 'sample.png', { type: 'image/png' })],
      configurable: true
    })
    await input.trigger('change')
    await flushPromises()

    expect(wrapper.find('[data-testid="workspace-asset-previews"]').exists()).toBe(true)
    expect(mocks.createObjectURL).toHaveBeenCalled()
    expect(JSON.stringify(mocks.apiClient.post.mock.calls)).not.toContain('data:image')
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('clears local shell state when starting a new chat without backend calls', async () => {
    const wrapper = mount(AppWorkspaceView)
    await nextTick()

    await wrapper.get('textarea').setValue('hello')
    await wrapper.get('.test-new-chat').trigger('click')

    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.findAll('.message-row')).toHaveLength(0)
    expect(mocks.apiClient.post).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})
