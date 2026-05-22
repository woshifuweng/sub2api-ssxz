import { computed, ref } from 'vue'
import { userChannelsAPI, userGroupsAPI } from '@/api'
import type { Group } from '@/types'
import type { UserAvailableChannel } from '@/api/channels'

export type CustomerCapabilityKey = 'chat' | 'commerce' | 'image' | 'developer'

export type ChatModelOption = {
  id: string
  name: string
  tier: 'premium' | 'standard'
}

export type CustomerCapability = {
  key: CustomerCapabilityKey
  title: string
  description: string
  enabled: boolean
  reason: string
}

const CHAT_MODELS: ChatModelOption[] = [
  { id: 'gpt-5.5', name: 'GPT-5.5', tier: 'premium' },
  { id: 'gpt-5.4', name: 'GPT-5.4', tier: 'premium' },
  { id: 'gpt-5.2', name: 'GPT-5.2', tier: 'premium' },
  { id: 'gpt-5.4-mini', name: 'GPT-5.4 Mini', tier: 'standard' }
]

const IMAGE_MODEL_NAMES = ['gpt-image-2', 'gpt-image-1', 'image-2']
const IMAGE_GROUP_KEYWORDS = ['image', 'images', 'picture', '绘图', '作图', '图片', '生图', '改图']
const COMMERCE_GROUP_KEYWORDS = ['commerce', 'ecommerce', 'shop', '电商', '商品', '文案']
const DEVELOPER_GROUP_KEYWORDS = ['developer', 'dev', 'code', 'codex', 'claude code', 'api', '开发', '编程']

const availableGroups = ref<Group[]>([])
const availableChannels = ref<UserAvailableChannel[]>([])
const loading = ref(false)
const loaded = ref(false)
const errorMessage = ref('')

function includesAny(value: string, keywords: string[]) {
  const normalized = value.toLowerCase()
  return keywords.some((keyword) => normalized.includes(keyword.toLowerCase()))
}

function groupHasKeyword(group: Group, keywords: string[]) {
  return includesAny(`${group.name} ${group.description || ''}`, keywords)
}

export function useUserCapabilities() {
  const supportedModelNames = computed(() => {
    const names = new Set<string>()
    for (const channel of availableChannels.value) {
      for (const platform of channel.platforms || []) {
        for (const model of platform.supported_models || []) {
          if (model.name) names.add(model.name)
        }
      }
    }
    return names
  })

  const openaiGroups = computed(() => availableGroups.value.filter((group) => group.platform === 'openai'))

  const chatModels = computed(() => {
    if (!supportedModelNames.value.size) {
      return CHAT_MODELS
    }
    const matched = CHAT_MODELS.filter((model) => supportedModelNames.value.has(model.id))
    return matched.length ? matched : CHAT_MODELS
  })

  const hasChat = computed(() => chatModels.value.length > 0 || openaiGroups.value.length > 0)

  const hasCommerce = computed(() => {
    return hasChat.value || availableGroups.value.some((group) => groupHasKeyword(group, COMMERCE_GROUP_KEYWORDS))
  })

  const hasImage = computed(() => {
    const hasImageModel = IMAGE_MODEL_NAMES.some((model) => supportedModelNames.value.has(model))
    const hasImageGroup = availableGroups.value.some((group) => groupHasKeyword(group, IMAGE_GROUP_KEYWORDS))
    return hasImageModel || hasImageGroup
  })

  const hasDeveloper = computed(() => {
    return availableGroups.value.length > 0 || availableGroups.value.some((group) => groupHasKeyword(group, DEVELOPER_GROUP_KEYWORDS))
  })

  const activeGroupLabels = computed(() => {
    if (!availableGroups.value.length) return ['默认分组']
    return availableGroups.value.slice(0, 4).map((group) => group.name)
  })

  const capabilities = computed<CustomerCapability[]>(() => [
    {
      key: 'chat',
      title: 'AI 聊天',
      description: '普通问答、写作、翻译、总结和代码辅助。',
      enabled: hasChat.value,
      reason: hasChat.value ? `可用模型：${chatModels.value.map((model) => model.id).join('、')}` : '未检测到可用聊天分组'
    },
    {
      key: 'commerce',
      title: '电商文案',
      description: '商品标题、卖点、详情页、小红书和直播口播。',
      enabled: hasCommerce.value,
      reason: hasCommerce.value ? '使用聊天模型加后台电商提示词模板' : '需要聊天或电商分组'
    },
    {
      key: 'image',
      title: 'AI 作图',
      description: '文生图、上传改图、商品换背景和营销海报。',
      enabled: hasImage.value,
      reason: hasImage.value ? '已检测到图片模型或图片分组' : '需要支持 gpt-image-2 / OpenAI Images API 的图片分组'
    },
    {
      key: 'developer',
      title: '开发者 API',
      description: 'Cherry Studio、Codex、Claude Code、CC Switch 和 SDK 接入。',
      enabled: hasDeveloper.value,
      reason: hasDeveloper.value ? '可以创建 API Key 接入第三方工具' : '需要至少一个可绑定分组'
    }
  ])

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
      errorMessage.value = '暂时无法读取账号能力，页面会按默认能力展示。'
    } finally {
      loading.value = false
    }
  }

  return {
    availableGroups,
    availableChannels,
    activeGroupLabels,
    capabilities,
    chatModels,
    errorMessage,
    hasChat,
    hasCommerce,
    hasDeveloper,
    hasImage,
    loaded,
    loading,
    loadCapabilities
  }
}
