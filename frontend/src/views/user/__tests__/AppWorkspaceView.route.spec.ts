import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAppStore } from '@/stores/app'

const mocks = vi.hoisted(() => ({
  route: { meta: { appSection: 'chat' }, path: '/app/chat', query: {} as Record<string, string> },
  routerReplace: vi.fn(async (location: { query?: Record<string, string> }) => {
    mocks.route.query = location.query ?? {}
  }),
  chatModels: { __v_isRef: true, value: [
    { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium', capabilities: ['text_chat'], modelCatalogSource: 'real_channel' }
  ] },
  defaultTextModel: { __v_isRef: true, value: 'gpt-5.5' },
  hasChat: { __v_isRef: true, value: true },
  loadCapabilities: vi.fn(),
  workspace: {
    activeConversation: { __v_isRef: true, value: null },
    activeConversationId: { __v_isRef: true, value: null as number | null },
    backendEnabled: { __v_isRef: true, value: true },
    conversations: { __v_isRef: true, value: [
      {
        id: 42,
        title: 'Persisted chat',
        status: 'active',
        created_at: '2026-06-25T00:00:00Z',
        updated_at: '2026-06-25T00:00:00Z'
      }
    ] },
    errorMessage: { __v_isRef: true, value: '' },
    loadingHistory: { __v_isRef: true, value: false },
    loadingMessages: { __v_isRef: true, value: false },
    messages: { __v_isRef: true, value: [] as Array<Record<string, unknown>> },
    requestPhase: { __v_isRef: true, value: 'idle' },
    sending: { __v_isRef: true, value: false },
    loadHistory: vi.fn(),
    selectConversation: vi.fn(),
    sendTextMessage: vi.fn(),
    startNewChat: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
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

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['historyItems', 'activeConversationId'],
    emits: ['new-chat', 'select-conversation'],
    template: `
      <div>
        <button class="test-new-chat" type="button" @click="$emit('new-chat')">New chat</button>
        <button
          v-for="item in historyItems"
          :key="item.id"
          class="test-history"
          type="button"
          @click="$emit('select-conversation', item.id)"
        >
          {{ item.title }}
        </button>
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

vi.mock('../workspace/WorkspaceComposer.vue', () => ({
  default: {
    name: 'WorkspaceComposer',
    props: [
      'modelValue',
      'selectedModel',
      'models',
      'intent',
      'imageCapabilityAvailable',
      'backendEnabled',
      'sending',
      'assetPreviews',
      'rejectedFiles',
      'webSearchAvailable',
      'webSearchEnabled'
    ],
    emits: [
      'update:modelValue',
      'update:selectedModel',
      'toggle-web-search',
      'files',
      'unsupported-files',
      'remove-asset',
      'submit'
    ],
    template: '<form><textarea :value="modelValue" /></form>'
  }
}))

vi.mock('../workspace/WorkspaceMessageList.vue', () => ({
  default: {
    name: 'WorkspaceMessageList',
    props: ['messages', 'loading'],
    template: `
      <div>
        <article v-for="message in messages" :key="message.id" class="message-row">
          {{ message.content }}
        </article>
      </div>
    `
  }
}))

vi.mock('../workspace/useWorkspaceConversation', () => ({
  WORKSPACE_TEXT_ONLY_MESSAGE: 'Text beta only.',
  useWorkspaceConversation: () => mocks.workspace
}))

import AppWorkspaceView from '../AppWorkspaceView.vue'

function resetWorkspace() {
  mocks.route.meta = { appSection: 'chat' }
  mocks.route.path = '/app/chat'
  mocks.route.query = {}
  mocks.routerReplace.mockClear()
  mocks.loadCapabilities.mockReset()
  mocks.workspace.activeConversation.value = null
  mocks.workspace.activeConversationId.value = null
  mocks.workspace.backendEnabled.value = true
  mocks.workspace.errorMessage.value = ''
  mocks.workspace.loadingHistory.value = false
  mocks.workspace.loadingMessages.value = false
  mocks.workspace.messages.value = []
  mocks.workspace.requestPhase.value = 'idle'
  mocks.workspace.sending.value = false
  mocks.workspace.loadHistory.mockReset()
  mocks.workspace.selectConversation.mockReset()
  mocks.workspace.sendTextMessage.mockReset()
  mocks.workspace.startNewChat.mockReset()
  mocks.workspace.loadHistory.mockResolvedValue(undefined)
  mocks.workspace.selectConversation.mockImplementation(async (id: number) => {
    mocks.workspace.activeConversationId.value = id
    mocks.workspace.messages.value = [
      {
        id: `message-${id}-user`,
        conversationId: id,
        messageType: 'text',
        role: 'user',
        content: 'saved question'
      },
      {
        id: `message-${id}-assistant`,
        conversationId: id,
        messageType: 'text',
        role: 'assistant',
        content: 'saved answer'
      }
    ]
  })
  mocks.workspace.startNewChat.mockImplementation(async () => {
    mocks.workspace.activeConversationId.value = null
    mocks.workspace.messages.value = []
  })
}

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

describe('AppWorkspaceView route conversation state', () => {
  beforeEach(() => {
    resetWorkspace()
  })

  it('restores the selected conversation from the route query on refresh', async () => {
    mocks.route.query = { conversation_id: '42' }

    const wrapper = mountWithAppStore()
    await flushPromises()
    wrapper.vm.$forceUpdate()
    await flushPromises()

    expect(mocks.workspace.loadHistory).toHaveBeenCalledTimes(1)
    expect(mocks.workspace.selectConversation).toHaveBeenCalledWith(42)
    expect(wrapper.findAll('.message-row')).toHaveLength(2)
    expect(wrapper.text()).toContain('saved question')
    expect(wrapper.text()).toContain('saved answer')

    wrapper.unmount()
  })

  it('writes the selected history conversation to the route query', async () => {
    const wrapper = mountWithAppStore()
    await flushPromises()

    await wrapper.get('.test-history').trigger('click')
    await flushPromises()

    expect(mocks.workspace.selectConversation).toHaveBeenCalledWith(42)
    expect(mocks.routerReplace).toHaveBeenLastCalledWith({
      query: {
        conversation_id: '42'
      }
    })
    expect(mocks.route.query.conversation_id).toBe('42')

    wrapper.unmount()
  })

  it('clears only the active conversation query when starting a new chat', async () => {
    mocks.route.query = { conversation_id: '42', source: 'history' }
    const wrapper = mountWithAppStore()
    await flushPromises()

    await wrapper.get('.test-new-chat').trigger('click')
    await flushPromises()
    wrapper.vm.$forceUpdate()
    await flushPromises()

    expect(mocks.workspace.startNewChat).toHaveBeenCalledTimes(1)
    expect(mocks.routerReplace).toHaveBeenLastCalledWith({
      query: {
        source: 'history'
      }
    })
    expect(mocks.workspace.activeConversationId.value).toBeNull()
    expect(wrapper.findAll('.message-row')).toHaveLength(0)

    wrapper.unmount()
  })
})
