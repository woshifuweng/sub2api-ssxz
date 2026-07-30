import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Toggle from '../Toggle.vue'

describe('Toggle', () => {
  it('exposes a clear state and emits the next value', async () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: false,
        ariaLabel: 'Enable feature'
      }
    })
    const toggle = wrapper.get('[role="switch"]')

    expect(toggle.attributes('aria-checked')).toBe('false')
    expect(toggle.attributes('data-state')).toBe('off')
    expect(toggle.attributes('aria-label')).toBe('Enable feature')

    await toggle.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
  })

  it('renders the enabled state', () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: true
      }
    })

    expect(wrapper.get('[role="switch"]').attributes('data-state')).toBe('on')
  })

  it('does not emit while disabled', async () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: false,
        disabled: true
      }
    })

    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
