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
            state: 'error'
          }
        ]
      }
    })

    expect(wrapper.find('.message-image-card').attributes('data-image-state')).toBe('error')
    expect(wrapper.text()).toContain('Image generation failed. Please try again.')
  })
})
