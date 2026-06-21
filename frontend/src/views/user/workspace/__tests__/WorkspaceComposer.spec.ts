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
  { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' as const, capabilities: ['text_chat'], modelCatalogSource: 'real_channel' },
  { id: 'gpt-5.4-mini', name: 'GPT-5.4-Mini', tier: 'standard' as const, capabilities: ['text_chat'], modelCatalogSource: 'real_channel' }
]

function mountComposer(overrides = {}) {
  return mount(WorkspaceComposer, {
    props: {
      modelValue: '',
      selectedModel: 'gpt-5.5',
      models,
      intent: 'image',
      imageCapabilityAvailable: false,
      backendEnabled: true,
      sending: false,
      assetPreviews: [],
      rejectedFiles: [],
      webSearchEnabled: false,
      ...overrides
    }
  })
}

describe('WorkspaceComposer', () => {
  it('opens the model picker and emits selected model changes', async () => {
    const wrapper = mountComposer()

    await wrapper.get('[data-testid="workspace-model-trigger"]').trigger('click')
    expect(wrapper.find('[data-testid="workspace-model-menu"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('real channel')
    expect(wrapper.text()).not.toContain('openai')

    await wrapper.findAll('[data-testid="workspace-model-option"]')[1].trigger('click')
    expect(wrapper.emitted('update:selectedModel')?.[0]).toEqual(['gpt-5.4-mini'])
  })

  it('hides the image entry and keeps unsupported capabilities visibly unavailable in text beta', async () => {
    const wrapper = mountComposer({ modelValue: 'ready' })

    expect(wrapper.find('[data-testid="workspace-add-content"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="workspace-capability-web-search"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="workspace-capability-memory"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="workspace-capability-toolbox"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('暂未接入')
  })

  it('keeps the web search capability hidden when the backend says it is unavailable', () => {
    const wrapper = mountComposer({ modelValue: 'ready', webSearchEnabled: true })

    expect(wrapper.find('[data-testid="workspace-capability-web-search"]').exists()).toBe(false)
    expect(wrapper.emitted('toggle-web-search')).toBeUndefined()
  })

  it('shows and toggles web search when the backend says it is available', async () => {
    const wrapper = mountComposer({
      modelValue: 'ready',
      webSearchAvailable: true,
      webSearchEnabled: false
    })

    const toggle = wrapper.get('[data-testid="workspace-capability-web-search"]')
    expect(toggle.attributes('disabled')).toBeUndefined()

    await toggle.trigger('click')

    expect(wrapper.emitted('toggle-web-search')).toHaveLength(1)
  })

  it('emits submit when content is ready', async () => {
    const wrapper = mountComposer({ intent: 'chat', modelValue: 'hello' })

    await wrapper.get('form').trigger('submit.prevent')

    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('keeps image-route composer text-oriented when image capability is unavailable', () => {
    const wrapper = mountComposer({ intent: 'image', modelValue: 'make an image' })

    expect(wrapper.find('textarea').attributes('placeholder')).toContain('直接输入问题')
  })

  it('keeps submit disabled when the workspace backend is unavailable', async () => {
    const wrapper = mountComposer({ backendEnabled: false, modelValue: 'hello' })

    await wrapper.get('form').trigger('submit.prevent')

    expect(wrapper.get('[data-testid="workspace-send"]').attributes('disabled')).toBeDefined()
    expect(wrapper.emitted('submit')).toBeUndefined()
  })
})
