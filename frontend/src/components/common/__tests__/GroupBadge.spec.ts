import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import GroupBadge from '../GroupBadge.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('GroupBadge platform branding', () => {
  it('renders the Claude icon for a Claude-branded OpenAI-compatible group', () => {
    const wrapper = mount(GroupBadge, {
      props: {
        name: 'Claude 满血池(CCMAX)',
        platform: 'openai'
      },
      global: {
        stubs: {
          PlatformIcon: {
            props: ['platform'],
            template: '<svg data-testid="platform-icon" :data-platform="platform" />'
          }
        }
      }
    })

    expect(wrapper.get('[data-testid="platform-icon"]').attributes('data-platform')).toBe('anthropic')
  })

  it('keeps GPT branding for a GPT-named OpenAI group', () => {
    const wrapper = mount(GroupBadge, {
      props: {
        name: '满血 GPT号池',
        platform: 'openai'
      },
      global: {
        stubs: {
          PlatformIcon: {
            props: ['platform'],
            template: '<svg data-testid="platform-icon" :data-platform="platform" />'
          }
        }
      }
    })

    expect(wrapper.get('[data-testid="platform-icon"]').attributes('data-platform')).toBe('openai')
  })
})
