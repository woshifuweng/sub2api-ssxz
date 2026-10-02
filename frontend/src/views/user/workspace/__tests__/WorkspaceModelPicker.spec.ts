import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import WorkspaceModelPicker from '../WorkspaceModelPicker.vue'

const models = [
  'gpt-6-sol',
  'gpt-6-luna',
  'claude-opus-5-5',
  'grok-4.7'
].map((id) => ({
  id,
  name: id,
  tier: 'premium' as const,
  capabilities: ['text_chat']
}))

describe('WorkspaceModelPicker', () => {
  it.each(models)('emits the exact selected model ID for %s', async (model) => {
    const wrapper = mount(WorkspaceModelPicker, {
      props: { models, selectedModel: models[0].id },
      global: { stubs: { Icon: true } }
    })

    await wrapper.get('[data-testid="workspace-model-trigger"]').trigger('click')
    await wrapper.findAll('[data-testid="workspace-model-option"]')
      .find((option) => option.text() === model.name)!
      .trigger('click')

    expect(wrapper.emitted('update:selectedModel')).toEqual([[model.id]])
  })
})
