import { computed, ref } from 'vue'
import { userChannelsAPI, userGroupsAPI } from '@/api'
import type { Group } from '@/types'
import type { UserAvailableChannel, UserSupportedModel } from '@/api/channels'

export type ChatModelOption = {
  id: string
  name: string
  tier: 'premium' | 'standard'
  capabilities: string[]
  provider?: string
  providerLabel?: string
  platform?: string
  channelName?: string
  modelCatalogSource?: string
  pricingStatus?: string
  usageSupport?: string[]
  fake?: boolean
  testOnly?: boolean
  stagingOnly?: boolean
}

const availableGroups = ref<Group[]>([])
const availableChannels = ref<UserAvailableChannel[]>([])
const loading = ref(false)
const loaded = ref(false)
const errorMessage = ref('')

const workspaceImageSelectableCapabilities = new Set([
  'image_generation',
  'image_edit'
])

function isImageModelName(model: string) {
  const normalized = model.toLowerCase()
  return normalized.includes('image') || normalized.includes('dall')
}

function isSelectableWorkspaceModel(model: UserSupportedModel) {
  const capabilities = model.capabilities || []
  if (capabilities.length > 0) {
    return capabilities.includes('text_chat')
  }
  return !isImageModelName(model.name)
}

function isSelectableImageModel(model: UserSupportedModel) {
  const capabilities = model.capabilities || []
  if (capabilities.some((capability) => workspaceImageSelectableCapabilities.has(capability))) {
    return true
  }
  return model.model_catalog_source === 'real_channel' && isImageModelName(model.name)
}

function formatModelName(model: string) {
  return model
    .split('-')
    .map((part) => part ? part.charAt(0).toUpperCase() + part.slice(1) : part)
    .join('-')
}

function modelTier(model: UserSupportedModel): ChatModelOption['tier'] {
  const name = model.name.toLowerCase()
  if (name.includes('mini') || name.includes('flash')) return 'standard'
  if (model.capabilities?.includes('image_generation')) return 'premium'
  return 'premium'
}

function modelPriority(model: ChatModelOption) {
  if (model.modelCatalogSource === 'real_channel' && model.pricingStatus === 'configured') return 0
  if (model.modelCatalogSource === 'fake_gate' && model.fake && model.testOnly) return 1
  if (model.modelCatalogSource === 'real_channel') return 2
  return 3
}

export function useUserCapabilities() {
  function buildModelOptions(predicate: (model: UserSupportedModel) => boolean) {
    const models = new Map<string, ChatModelOption>()
    for (const channel of availableChannels.value) {
      for (const platform of channel.platforms || []) {
        for (const model of platform.supported_models || []) {
          if (!model.name || !predicate(model)) continue
          const option = {
            id: model.name,
            name: formatModelName(model.name),
            tier: modelTier(model),
            capabilities: model.capabilities || [],
            provider: model.provider,
            providerLabel: model.provider_label,
            platform: model.platform || platform.platform,
            channelName: channel.name,
            modelCatalogSource: model.model_catalog_source,
            pricingStatus: model.pricing_status,
            usageSupport: model.usage_support || [],
            fake: model.fake,
            testOnly: model.test_only,
            stagingOnly: model.staging_only
          } satisfies ChatModelOption
          const existing = models.get(option.id)
          if (!existing || modelPriority(option) < modelPriority(existing)) {
            models.set(option.id, option)
          }
        }
      }
    }
    return [...models.values()].sort((left, right) => left.id.localeCompare(right.id))
  }

  const supportedModelOptions = computed(() => {
    return buildModelOptions(isSelectableWorkspaceModel)
  })

  const chatModels = computed(() => supportedModelOptions.value)
  const imageModels = computed(() => buildModelOptions(isSelectableImageModel))

  const defaultTextModel = computed(() => {
    const textModel = chatModels.value.find((model) => model.capabilities.includes('text_chat'))
    return textModel?.id || chatModels.value[0]?.id || ''
  })
  const defaultImageModel = computed(() => {
    const imageModel = imageModels.value.find((model) => model.capabilities.includes('image_generation'))
    return imageModel?.id || imageModels.value[0]?.id || ''
  })
  const hasChat = computed(() => chatModels.value.length > 0)
  const hasImageGeneration = computed(() => imageModels.value.length > 0)

  async function loadCapabilities() {
    if (loading.value) return
    loading.value = true
    errorMessage.value = ''
    try {
      const [groups, channels] = await Promise.all([
        userGroupsAPI.getAvailable().catch(() => []),
        userChannelsAPI.getAvailable().catch(() => [])
      ])
      availableGroups.value = groups
      availableChannels.value = channels
      loaded.value = true
    } catch (error) {
      console.error('Failed to load user capabilities:', error)
      errorMessage.value = '暂时无法读取账号能力'
    } finally {
      loading.value = false
    }
  }

  return {
    availableChannels,
    availableGroups,
    chatModels,
    defaultImageModel,
    defaultTextModel,
    errorMessage,
    hasChat,
    hasImageGeneration,
    imageModels,
    loaded,
    loading,
    loadCapabilities
  }
}
