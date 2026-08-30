import { computed, ref } from 'vue'
import { userChannelsAPI } from '@/api'
import type { UserAvailableChannel } from '@/api/channels'

export type ChatModelOption = {
  id: string
  name: string
  platform: string
}

const availableChannels = ref<UserAvailableChannel[]>([])
const loading = ref(false)
const loaded = ref(false)
const errorMessage = ref('')

function formatModelName(model: string) {
  return model
    .split('-')
    .map((part) => part ? part.charAt(0).toUpperCase() + part.slice(1) : part)
    .join('-')
}

function isTextModel(model: string) {
  const normalized = model.toLowerCase()
  return !normalized.includes('gpt-image')
    && !normalized.includes('dall-e')
    && !normalized.includes('sora')
    && !normalized.includes('video')
}

export function useUserCapabilities() {
  const chatModels = computed<ChatModelOption[]>(() => {
    const models = new Map<string, ChatModelOption>()
    for (const channel of availableChannels.value) {
      for (const section of channel.platforms || []) {
        for (const model of section.supported_models || []) {
          if (!model.name || !isTextModel(model.name) || models.has(model.name)) continue
          models.set(model.name, {
            id: model.name,
            name: formatModelName(model.name),
            platform: model.platform || section.platform
          })
        }
      }
    }
    return [...models.values()].sort((left, right) => left.id.localeCompare(right.id))
  })

  const defaultTextModel = computed(() => chatModels.value[0]?.id || '')
  const hasChat = computed(() => chatModels.value.length > 0)

  async function loadCapabilities() {
    if (loading.value) return
    loading.value = true
    errorMessage.value = ''
    try {
      availableChannels.value = await userChannelsAPI.getAvailable().catch(() => [])
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
    chatModels,
    defaultTextModel,
    errorMessage,
    hasChat,
    loaded,
    loading,
    loadCapabilities
  }
}
