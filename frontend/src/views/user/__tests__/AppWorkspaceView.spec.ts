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
  }
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
    let messageId = 1
    mocks.route.value = { meta: { appSection: 'image' } }
    mocks.chatModels.value = [
      { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' },
      { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard' }
    ]
    mocks.defaultTextModel.value = 'gpt-5.5'
    mocks.hasChat.value = true
    mocks.loadCapabilities.mockClear()
    mocks.apiClient.get.mockReset()
    mocks.apiClient.post.mockReset()
    mocks.apiClient.get.mockResolvedValue({ data: [] })
    mocks.apiClient.post.mockImplementation((url: string, payload: Record<string, unknown>) => {
      if (url === '/chat-workspace/conversations') {
        return Promise.resolve({
          data: {
            id: 1,
            title: payload.title || 'hello',
            status: 'active',
            created_at: '2026-06-08T00:00:00Z',
            updated_at: '2026-06-08T00:00:00Z'
          }
        })
      }
      if (url.includes('/chat-workspace/conversations/1/messages')) {
        return Promise.resolve({
          data: {
            id: messageId++,
            conversation_id: 1,
            message_type: payload.message_type,
            role: payload.role,
            content: payload.content,
            created_at: '2026-06-08T00:00:00Z',
            updated_at: '2026-06-08T00:00:00Z'
          }
        })
      }
      if (url === '/chat-studio/complete') {
        return Promise.resolve({
          data: {
            choices: [{ message: { content: 'assistant reply' } }]
          }
        })
      }
      return Promise.resolve({ data: {} })
    })
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

  it('persists a draft and clears the workspace when starting a new chat', async () => {
    const wrapper = mount(AppWorkspaceView)
    await nextTick()

    await wrapper.get('textarea').setValue('hello')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()
    expect(wrapper.findAll('.message-row')).toHaveLength(2)
    expect(mocks.apiClient.post).toHaveBeenCalledWith('/chat-workspace/conversations', { title: 'hello' })

    await wrapper.get('.test-new-chat').trigger('click')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.findAll('.message-row')).toHaveLength(0)

    wrapper.unmount()
  })
})
