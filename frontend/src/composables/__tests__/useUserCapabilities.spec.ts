import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useUserCapabilities } from '../useUserCapabilities'

const mocks = vi.hoisted(() => ({
  getGroups: vi.fn(),
  getChannels: vi.fn()
}))

vi.mock('@/api', () => ({
  userGroupsAPI: {
    getAvailable: mocks.getGroups
  },
  userChannelsAPI: {
    getAvailable: mocks.getChannels
  }
}))

describe('useUserCapabilities', () => {
  beforeEach(() => {
    mocks.getGroups.mockReset()
    mocks.getChannels.mockReset()
    mocks.getGroups.mockResolvedValue([])
    mocks.getChannels.mockResolvedValue([])
  })

  it('keeps normal chat models available', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Chat Channel',
        description: '',
        platforms: [
          {
            platform: 'openai',
            groups: [],
            supported_models: [
              {
                name: 'gpt-5.5',
                platform: 'openai',
                pricing: null,
                capabilities: ['text_chat'],
                provider: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).toContain('gpt-5.5')
    expect(capabilities.chatModels.value[0]).toMatchObject({
      provider: 'openai',
      capabilities: ['text_chat'],
      modelCatalogSource: 'real_channel',
      pricingStatus: 'configured'
    })
  })

  it('keeps text-capable multimodal models available in chat picker', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Multimodal Channel',
        description: '',
        platforms: [
          {
            platform: 'openai',
            groups: [],
            supported_models: [
              {
                name: 'gpt-4o',
                platform: 'openai',
                pricing: null,
                capabilities: ['text_chat', 'vision'],
                provider: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).toContain('gpt-4o')
  })

  it('does not show vision-only models in chat picker', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Vision Channel',
        description: '',
        platforms: [
          {
            platform: 'openai',
            groups: [],
            supported_models: [
              {
                name: 'vision-preview',
                platform: 'openai',
                pricing: null,
                capabilities: ['vision'],
                provider: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('vision-preview')
  })

  it('does not show function-only or tool-only models in chat picker', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Tool Channel',
        description: '',
        platforms: [
          {
            platform: 'openai',
            groups: [],
            supported_models: [
              {
                name: 'tool-only-model',
                platform: 'openai',
                pricing: null,
                capabilities: ['tool'],
                provider: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              },
              {
                name: 'function-only-model',
                platform: 'openai',
                pricing: null,
                capabilities: ['function_calling'],
                provider: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('tool-only-model')
    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('function-only-model')
  })

  it('does not show explicitly allowed fake image generation model in chat picker', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Workspace Image Fake',
        description: '',
        platforms: [
          {
            platform: 'workspace-image-fake',
            groups: [],
            supported_models: [
              {
                name: 'workspace-image-fake-model',
                platform: 'workspace-image-fake',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-image-fake',
                model_catalog_source: 'fake_gate',
                fake: true,
                test_only: true
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('workspace-image-fake-model')
  })

  it('does not show backend-authorized image generation-only models in chat picker', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-1',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-openai-compatible-image-staging',
                model_catalog_source: 'real_channel',
                staging_only: true
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('gpt-image-1')
  })

  it('exposes backend-authorized image models through the image catalog', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-2',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                provider_label: 'workspace-openai-compatible-image-staging',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              },
              {
                name: 'gpt-5.5',
                platform: 'openai',
                pricing: null,
                capabilities: ['text_chat'],
                provider: 'openai',
                model_catalog_source: 'real_channel',
                pricing_status: 'configured'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).toContain('gpt-5.5')
    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('gpt-image-2')
    expect(capabilities.imageModels.value.map((model) => model.id)).toEqual(['gpt-image-2'])
    expect(capabilities.defaultImageModel.value).toBe('gpt-image-2')
    expect(capabilities.hasImageGeneration.value).toBe(true)
  })

  it('uses real image-like model names as image catalog fallback', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-1',
                platform: 'image-provider',
                pricing: null,
                model_catalog_source: 'real_channel'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.imageModels.value.map((model) => model.id)).toEqual(['gpt-image-1'])
    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('gpt-image-1')
  })

  it('filters image-like model names without backend capability metadata', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-1',
                platform: 'image-provider',
                pricing: null
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('gpt-image-1')
  })

  it('filters image generation models that are not sourced from real channels or fake gates', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Image Channel',
        description: '',
        platforms: [
          {
            platform: 'image-provider',
            groups: [],
            supported_models: [
              {
                name: 'gpt-image-1',
                platform: 'image-provider',
                pricing: null,
                capabilities: ['image_generation'],
                model_catalog_source: 'env_gate'
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('gpt-image-1')
  })

  it('filters fake image model names without explicit fake metadata', async () => {
    mocks.getChannels.mockResolvedValue([
      {
        name: 'Unexpected Channel',
        description: '',
        platforms: [
          {
            platform: 'workspace-image-fake',
            groups: [],
            supported_models: [
              {
                name: 'workspace-image-fake-model',
                platform: 'workspace-image-fake',
                pricing: null
              }
            ]
          }
        ]
      }
    ])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value.map((model) => model.id)).not.toContain('workspace-image-fake-model')
  })

  it('does not treat groups alone as send-ready chat capability', async () => {
    mocks.getGroups.mockResolvedValue([{ id: 1, name: 'default', platform: 'openai' }])
    mocks.getChannels.mockResolvedValue([])

    const capabilities = useUserCapabilities()
    await capabilities.loadCapabilities()

    expect(capabilities.chatModels.value).toEqual([])
    expect(capabilities.hasChat.value).toBe(false)
  })
})
