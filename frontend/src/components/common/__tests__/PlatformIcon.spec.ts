import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import PlatformIcon from '@/components/common/PlatformIcon.vue'

describe('PlatformIcon', () => {
  it.each(['openai', 'deepseek', 'gemini', 'kimi'])('renders the official color mark for %s', (platform) => {
    const wrapper = mount(PlatformIcon, { props: { platform: platform as never, size: 'md' } })

    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.find('svg path').exists()).toBe(true)
  })

  it('keeps composite groups represented by the network mark', () => {
    const wrapper = mount(PlatformIcon, { props: { platform: 'composite', size: 'md' } })

    expect(wrapper.find('svg circle').exists()).toBe(true)
    expect(wrapper.findAll('svg circle')).toHaveLength(3)
  })

  it('uses a neutral fallback for an unsupported platform value', () => {
    const wrapper = mount(PlatformIcon, { props: { platform: 'unsupported' as never, size: 'md' } })

    expect(wrapper.find('span').text()).toBe('U')
  })
})
