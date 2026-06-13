import { computed, ref } from 'vue'
import { userChannelsAPI, userGroupsAPI } from '@/api'
import type { Group } from '@/types'
import type { UserAvailableChannel, UserSupportedModel } from '@/api/channels'

export type ChatModelOption = {
  id: string
  name: string
  tier: 'premium' | 'standard'
}

const availableGroups = ref<Group[]>([])
const availableChannels = ref<UserAvailableChannel[]>([])
const loading = ref(false)
const loaded = ref(false)
const errorMessage = ref('')

function uniqueSorted(values: Array<string | undefined | null>) {
  const seen = new Map<string, string>()
  for (const value of values) {
    const name = String(value || '').trim()
    if (!name) continue
    const key = name.toLowerCase()
    if (!seen.has(key)) seen.set(key, name)
  }
  return [...seen.values()].sort((left, right) => left.localeCompare(right))
}

const workspaceSelectableCapabilities = new Set([
  'text_chat',
  'vision',
  'image_generation',
  'image_edit',
  'function_calling',
  'tool'
])

function isImageModelName(model: string) {
  const normalized = model.toLowerCase()
  return normalized.includes('image') || normalized.includes('dall')
}

function isSelectableWorkspaceModel(model: UserSupportedModel) {
  if ((model.capabilities || []).some((capability) => workspaceSelectableCapabilities.has(capability))) {
    return true
  }
  return !isImageModelName(model.name)
}

function formatModelName(model: string) {
  return model
    .split('-')
    .map((part) => part ? part.charAt(0).toUpperCase() + part.slice(1) : part)
    .join('-')
}

export function useUserCapabilities() {
  const supportedModelNames = computed(() => {
    const names = new Set<string>()
    for (const channel of availableChannels.value) {
      for (const platform of channel.platforms || []) {
        for (const model of platform.supported_models || []) {
          if (model.name && isSelectableWorkspaceModel(model)) names.add(model.name)
        }
      }
    }
    return names
  })

  const textModelNames = computed(() =>
    uniqueSorted([...supportedModelNames.value])
  )

  const chatModels = computed(() => textModelNames.value.map((model) => ({
    id: model,
    name: formatModelName(model),
    tier: model.toLowerCase().includes('mini') || model.toLowerCase().includes('flash') ? 'standard' : 'premium'
  }) satisfies ChatModelOption))

  const defaultTextModel = computed(() => textModelNames.value[0] || '')
  const hasChat = computed(() => chatModels.value.length > 0 || availableGroups.value.some((group) => group.platform === 'openai'))

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
    defaultTextModel,
    errorMessage,
    hasChat,
    loaded,
    loading,
    loadCapabilities
  }
}
