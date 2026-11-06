import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ModelIcon from '@/components/common/ModelIcon.vue'

describe('ModelIcon', () => {
  it.each(['gpt-5', 'gemini-2.5-pro', 'deepseek-v4.1-flash'])('renders a bundled color icon for %s', (model) => {
    const wrapper = mount(ModelIcon, { props: { model, size: '18px' } })

    expect(wrapper.find('svg.model-icon').exists()).toBe(true)
    expect(wrapper.find('svg.model-icon path').exists()).toBe(true)
  })

  it('keeps an unknown model on the library fallback instead of guessing a logo', () => {
    const wrapper = mount(ModelIcon, { props: { model: 'unknown-model' } })

    expect(wrapper.find('svg.model-icon').exists()).toBe(true)
    expect(wrapper.text()).toBe('')
  })
})
