import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import WorkspaceComposer from '../WorkspaceComposer.vue'

const models = [
  { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' as const },
  { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard' as const }
]

function mountComposer(overrides = {}) {
  return mount(WorkspaceComposer, {
    props: {
      modelValue: '',
      selectedModel: 'gpt-5.5',
      models,
      intent: 'image',
      backendEnabled: true,
      sending: false,
      assetPreviews: [],
      rejectedFiles: [],
      ...overrides
    }
  })
}

describe('WorkspaceComposer', () => {
  it('opens the model picker and emits selected model changes', async () => {
    const wrapper = mountComposer()

    await wrapper.get('[data-testid="workspace-model-trigger"]').trigger('click')
    expect(wrapper.find('[data-testid="workspace-model-menu"]').exists()).toBe(true)

    await wrapper.findAll('[data-testid="workspace-model-option"]')[1].trigger('click')
    expect(wrapper.emitted('update:selectedModel')?.[0]).toEqual(['gpt-5.4-mini'])
  })

  it('opens upload capabilities without disabled future buttons', async () => {
    const wrapper = mountComposer({ modelValue: 'ready' })

    await wrapper.get('[data-testid="workspace-add-content"]').trigger('click')

    expect(wrapper.find('#workspace-asset-panel').exists()).toBe(true)
    expect(wrapper.findAll('.asset-option[disabled]')).toHaveLength(0)
  })

  it('emits submit when content is ready', async () => {
    const wrapper = mountComposer({ modelValue: 'hello' })

    await wrapper.get('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('keeps submit disabled when the workspace backend is unavailable', async () => {
    const wrapper = mountComposer({ backendEnabled: false, modelValue: 'hello' })

    await wrapper.get('form').trigger('submit.prevent')

    expect(wrapper.get('[data-testid="workspace-send"]').attributes('disabled')).toBeDefined()
    expect(wrapper.emitted('submit')).toBeUndefined()
  })
})
