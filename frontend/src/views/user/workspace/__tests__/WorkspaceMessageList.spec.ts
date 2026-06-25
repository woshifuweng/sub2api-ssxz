import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import WorkspaceMessageList from '../WorkspaceMessageList.vue'
import {
  WORKSPACE_GENERATING_MESSAGE,
  WORKSPACE_IMAGE_FAILED_MESSAGE
} from '../useWorkspaceConversation'

describe('WorkspaceMessageList', () => {
  it('renders assistant image messages as image cards', () => {
    const wrapper = mount(WorkspaceMessageList, {
      props: {
        loading: false,
        messages: [
          {
            id: 'message-1',
            messageType: 'image',
            role: 'assistant',
            content: 'Generated image is ready.',
            attachments: [
              {
                id: 'asset-1',
                name: 'asset-1',
                url: 'https://cdn.example.test/workspace/image-1.png',
                type: 'image'
              }
            ]
          }
        ]
      }
    })

    expect(wrapper.find('.message-image-card').exists()).toBe(true)
    expect(wrapper.find('img').attributes('src')).toBe(
      'https://cdn.example.test/workspace/image-1.png'
    )
    expect(wrapper.text()).toContain('Generated image is ready.')
  })

  it('renders failed assistant image messages with a clear fallback', () => {
    const wrapper = mount(WorkspaceMessageList, {
      props: {
        loading: false,
        messages: [
          {
            id: 'message-2',
            messageType: 'image',
            role: 'assistant',
            content: '',
            state: 'failed'
          }
        ]
      }
    })

    expect(wrapper.find('.message-image-card').attributes('data-image-state')).toBe('failed')
    expect(wrapper.text()).toContain('失败')
    expect(wrapper.text()).toContain(WORKSPACE_IMAGE_FAILED_MESSAGE)
  })

  it('renders text request progress states in the message stream', () => {
    const wrapper = mount(WorkspaceMessageList, {
      props: {
        loading: false,
        messages: [
          {
            id: 'local-user-1',
            messageType: 'text',
            role: 'user',
            content: 'hello',
            state: 'sending'
          },
          {
            id: 'local-assistant-1',
            messageType: 'text',
            role: 'assistant',
            content: WORKSPACE_GENERATING_MESSAGE,
            state: 'generating'
          }
        ]
      }
    })

    expect(wrapper.find('[data-state="sending"]').exists()).toBe(true)
    expect(wrapper.find('[data-state="generating"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('发送中')
    expect(wrapper.text()).toContain('生成中')
    expect(wrapper.text()).toContain(WORKSPACE_GENERATING_MESSAGE)
  })
})
