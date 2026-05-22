<template>
  <div class="mx-auto flex max-w-7xl flex-col gap-5">
    <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div class="inline-flex items-center gap-2 rounded-md bg-primary-50 px-3 py-1 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
            <Icon name="sparkles" size="sm" />
            <span>SSXZ AI 绘图工作台</span>
          </div>
          <h1 class="mt-3 text-2xl font-bold text-gray-900 dark:text-white">商品作图、海报和灵感图，一页完成</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-dark-300">
            上传商品图会自动进入改图模式；不上传图片则按文生图处理。模型、接口、提示词和扣费都由后台处理，客户只需要填写需求。
          </p>
        </div>
        <div class="grid grid-cols-3 gap-2 text-center sm:min-w-[360px]">
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900/50">
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ imageCredits }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">约可生成</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900/50">
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ count }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">本次张数</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900/50">
            <div class="text-lg font-semibold text-gray-900 dark:text-white">gpt-image-2</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">图片模型</div>
          </div>
        </div>
      </div>
    </section>

    <div class="grid gap-5 xl:grid-cols-[minmax(0,430px)_1fr]">
      <section class="space-y-5">
        <div class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">创作设置</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">当前按张扣费，预计消耗 {{ count }} 张</p>
            </div>
            <span class="rounded-md bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
              {{ selectedFile ? '改图模式' : '文生图模式' }}
            </span>
          </div>

          <div class="space-y-4">
            <div>
              <label class="input-label">参考图片</label>
              <label
                class="flex min-h-36 cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-center transition hover:border-primary-400 hover:bg-primary-50/50 dark:border-dark-500 dark:bg-dark-700/60 dark:hover:border-primary-500"
              >
                <Icon name="upload" size="lg" class="text-gray-400" />
                <span class="text-sm font-medium text-gray-700 dark:text-dark-100">
                  {{ selectedFile ? selectedFile.name : '上传商品图、人物图或参考图' }}
                </span>
                <span class="text-xs text-gray-400">JPG / PNG / WEBP，上传后自动进入改图模式</span>
                <input class="hidden" type="file" accept="image/png,image/jpeg,image/webp" @change="handleFileChange" />
              </label>
              <div v-if="previewUrl" class="mt-3 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
                <img :src="previewUrl" alt="preview" class="h-48 w-full bg-gray-50 object-contain dark:bg-dark-900" />
              </div>
            </div>

            <div>
              <label class="input-label">创作方向</label>
              <div class="grid grid-cols-3 gap-2">
                <button
                  v-for="category in categories"
                  :key="category.id"
                  type="button"
                  class="rounded-lg border px-3 py-2 text-center text-sm transition"
                  :class="activeCategory === category.id ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200'"
                  @click="activeCategory = category.id"
                >
                  {{ category.label }}
                </button>
              </div>
            </div>

            <div>
              <label class="input-label">灵感预设</label>
              <div class="grid gap-2">
                <button
                  v-for="preset in filteredPresets"
                  :key="preset.id"
                  type="button"
                  class="rounded-lg border p-3 text-left transition"
                  :class="selectedPresetId === preset.id ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/30' : 'border-gray-200 bg-white hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800'"
                  @click="applyPreset(preset)"
                >
                  <div class="flex items-center justify-between gap-3">
                    <div class="font-medium text-gray-900 dark:text-white">{{ preset.label }}</div>
                    <span class="rounded-md bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-dark-300">{{ preset.badge }}</span>
                  </div>
                  <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">{{ preset.description }}</p>
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="input-label">图片比例</label>
                <select v-model="size" class="input">
                  <option value="1024x1024">1:1 商品主图</option>
                  <option value="1024x1536">2:3 竖版海报</option>
                  <option value="1536x1024">3:2 横版场景</option>
                </select>
              </div>
              <div>
                <label class="input-label">生成张数</label>
                <select v-model.number="count" class="input">
                  <option :value="1">1 张</option>
                  <option :value="2">2 张</option>
                  <option :value="4">4 张</option>
                </select>
              </div>
            </div>

            <div>
              <label class="input-label">商品或主题名称</label>
              <input v-model.trim="productName" class="input" type="text" placeholder="例如：蓝牙耳机、保温杯、护肤套装、露营灯" />
            </div>

            <div>
              <label class="input-label">核心卖点</label>
              <textarea
                v-model.trim="sellingPoints"
                class="input min-h-24 resize-none"
                placeholder="例如：降噪、长续航、轻便、礼盒包装、适合通勤和运动"
              />
            </div>

            <div>
              <label class="input-label">补充要求</label>
              <textarea
                v-model.trim="creativeBrief"
                class="input min-h-24 resize-none"
                placeholder="例如：不要文字，不要夸张变形，画面要适合淘宝主图；或者写清楚想要的颜色、场景、节日氛围"
              />
            </div>

            <button
              type="button"
              class="btn btn-primary w-full justify-center"
              :disabled="generating"
              @click="generate"
            >
              <Icon v-if="!generating" name="sparkles" size="sm" />
              <Icon v-else name="refresh" size="sm" class="animate-spin" />
              <span>{{ generating ? '正在生成...' : `消耗约 ${count} 张，立即生成` }}</span>
            </button>

            <p v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm leading-6 text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
              {{ errorMessage }}
            </p>
          </div>
        </div>
      </section>

      <section class="min-h-[640px] rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">生成结果</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">结果会保留在当前页面，可下载、复制提示词或继续重试。</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="copyPrompt">
              <Icon name="copy" size="sm" />
              <span>复制提示词</span>
            </button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="generating" @click="generate">
              <Icon name="refresh" size="sm" />
              <span>重新生成</span>
            </button>
            <button v-if="results.length" type="button" class="btn btn-secondary btn-sm" @click="clearResults">
              <Icon name="trash" size="sm" />
              <span>清空</span>
            </button>
          </div>
        </div>

        <div v-if="!results.length" class="flex min-h-[520px] flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50 px-6 text-center dark:border-dark-600 dark:bg-dark-900/40">
          <Icon name="sparkles" size="xl" class="text-gray-300 dark:text-dark-500" />
          <h3 class="mt-4 text-base font-semibold text-gray-800 dark:text-dark-100">开始一次创作</h3>
          <p class="mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-dark-400">
            选择一个预设，填写商品名和卖点即可。当前如果上游账号没有开通图片接口，系统会明确提示，不会让客户误以为页面坏了。
          </p>
        </div>

        <div v-else class="grid gap-4 md:grid-cols-2">
          <article
            v-for="(item, index) in results"
            :key="item.id"
            class="overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-900"
          >
            <div class="aspect-square bg-white dark:bg-dark-950">
              <img :src="item.src" :alt="`result-${index + 1}`" class="h-full w-full object-contain" />
            </div>
            <div class="flex items-center justify-between gap-2 border-t border-gray-200 px-3 py-2 dark:border-dark-600">
              <span class="text-sm text-gray-500 dark:text-dark-400">图片 {{ index + 1 }}</span>
              <button type="button" class="btn btn-secondary btn-sm" @click="downloadResult(item, index)">
                <Icon name="download" size="sm" />
                <span>下载</span>
              </button>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'

type StudioTemplate = 'background' | 'white' | 'scene' | 'poster'
type PresetCategory = 'commerce' | 'marketing' | 'creative'

type ImagePreset = {
  id: string
  label: string
  badge: string
  category: PresetCategory
  templateId: StudioTemplate
  style: string
  size: string
  description: string
  promptHint: string
}

type ResultImage = {
  id: string
  src: string
}

const CREDIT_UNIT_USD = 0.2

const categories: Array<{ id: PresetCategory; label: string }> = [
  { id: 'commerce', label: '电商商品' },
  { id: 'marketing', label: '营销海报' },
  { id: 'creative', label: '创意灵感' }
]

const presets: ImagePreset[] = [
  {
    id: 'main-background',
    label: '商品主图换背景',
    badge: '主图',
    category: 'commerce',
    templateId: 'background',
    style: 'clean studio commercial photography, premium product lighting',
    size: '1024x1024',
    description: '保留商品主体，替换为干净、有质感、适合电商平台的背景。',
    promptHint: '保留商品外观和比例，不改变品牌元素，不添加多余文字，画面适合电商主图。'
  },
  {
    id: 'white-background',
    label: '白底图优化',
    badge: '白底',
    category: 'commerce',
    templateId: 'white',
    style: 'pure white background, sharp ecommerce product photography',
    size: '1024x1024',
    description: '适合商品上架、抠图后白底展示和标准产品图。',
    promptHint: '纯白背景，阴影自然，商品边缘清晰，主体居中，不能变形。'
  },
  {
    id: 'home-lifestyle',
    label: '家居生活方式',
    badge: '场景',
    category: 'commerce',
    templateId: 'scene',
    style: 'warm home lifestyle scene, natural commercial photography',
    size: '1536x1024',
    description: '把商品放进真实使用场景，让客户一眼知道适合什么生活方式。',
    promptHint: '真实家居环境，光线温暖自然，突出商品使用感和生活氛围。'
  },
  {
    id: 'premium-gray',
    label: '高级灰商务图',
    badge: '质感',
    category: 'commerce',
    templateId: 'background',
    style: 'premium gray background, luxury product photography, soft shadow',
    size: '1024x1024',
    description: '适合数码、办公、男装、配件等需要高级感的商品。',
    promptHint: '高级灰背景，柔和阴影，低饱和商务质感，画面克制干净。'
  },
  {
    id: 'xiaohongshu-poster',
    label: '小红书种草海报',
    badge: '种草',
    category: 'marketing',
    templateId: 'poster',
    style: 'fresh social media product poster, clean layout, soft colors',
    size: '1024x1536',
    description: '适合小红书、朋友圈和私域种草，画面更轻、更生活化。',
    promptHint: '小红书风格，清新自然，留出适合后期加标题的位置，不要生成乱码文字。'
  },
  {
    id: 'holiday-campaign',
    label: '节日促销图',
    badge: '促销',
    category: 'marketing',
    templateId: 'poster',
    style: 'holiday promotion product photography, festive but premium',
    size: '1024x1536',
    description: '适合活动页、优惠券图、节日上新和店铺促销。',
    promptHint: '有节日氛围但不要廉价，画面热闹有层次，主体清楚，避免文字乱码。'
  },
  {
    id: 'gift-box',
    label: '礼盒赠品展示',
    badge: '礼赠',
    category: 'marketing',
    templateId: 'poster',
    style: 'gift box product display, premium packaging, elegant lighting',
    size: '1024x1024',
    description: '适合礼盒、套装、赠品组合和高客单价商品。',
    promptHint: '突出礼盒包装和套装层次，适合送礼场景，整体精致高级。'
  },
  {
    id: 'tech-texture',
    label: '科技质感',
    badge: '数码',
    category: 'marketing',
    templateId: 'background',
    style: 'modern technology product photography, clean dark surface, cyan accent light',
    size: '1536x1024',
    description: '适合耳机、键盘、充电器、智能硬件等数码产品。',
    promptHint: '现代科技感，深色台面，冷色边缘光，突出结构和材质。'
  },
  {
    id: 'space-cat',
    label: '太空漫游猫',
    badge: '科幻',
    category: 'creative',
    templateId: 'scene',
    style: 'whimsical sci-fi illustration, cinematic lighting, cute character',
    size: '1024x1024',
    description: '轻松、有记忆点，适合头像、贴纸、社媒创意图。',
    promptHint: '一只可爱的猫穿着宇航服，在太空中漂浮，背景有星云和星星。'
  },
  {
    id: 'dragon-cloud',
    label: '金龙祥云',
    badge: '国风',
    category: 'creative',
    templateId: 'scene',
    style: 'chinese fantasy illustration, golden dragon, auspicious clouds',
    size: '1536x1024',
    description: '国风、水墨、节庆和品牌视觉可用。',
    promptHint: '金色中国龙盘旋在云海之上，水墨国风，富有构图层次。'
  },
  {
    id: 'cyber-rain',
    label: '赛博夜雨',
    badge: '赛博',
    category: 'creative',
    templateId: 'scene',
    style: 'cyberpunk rainy night, neon light, cinematic composition',
    size: '1536x1024',
    description: '适合科技、游戏、潮流内容的氛围图。',
    promptHint: '雨夜街道，霓虹反光，电影感构图，深色背景，高对比度。'
  },
  {
    id: 'jiangnan-fresh',
    label: '江南清新',
    badge: '水彩',
    category: 'creative',
    templateId: 'scene',
    style: 'fresh watercolor illustration, jiangnan town, soft mist',
    size: '1536x1024',
    description: '适合温柔、清新、文旅和生活方式内容。',
    promptHint: '水彩风格江南水乡，小桥流水人家，烟雨朦胧，淡彩晕染。'
  }
]

const authStore = useAuthStore()
const selectedFile = ref<File | null>(null)
const previewUrl = ref('')
const activeCategory = ref<PresetCategory>('commerce')
const selectedPresetId = ref('main-background')
const templateId = ref<StudioTemplate>('background')
const size = ref('1024x1024')
const count = ref(1)
const productName = ref('')
const sellingPoints = ref('')
const creativeBrief = ref('')
const style = ref('clean studio commercial photography, premium product lighting')
const generating = ref(false)
const errorMessage = ref('')
const results = ref<ResultImage[]>([])

const imageCredits = computed(() => {
  const balance = authStore.user?.balance ?? 0
  return Math.max(0, Math.floor(balance / CREDIT_UNIT_USD))
})

const selectedPreset = computed(() => presets.find((preset) => preset.id === selectedPresetId.value) || presets[0])

const filteredPresets = computed(() => presets.filter((preset) => preset.category === activeCategory.value))

onBeforeUnmount(() => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})

function applyPreset(preset: ImagePreset) {
  selectedPresetId.value = preset.id
  templateId.value = preset.templateId
  style.value = preset.style
  size.value = preset.size
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  selectedFile.value = file || null
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
  if (file) {
    previewUrl.value = URL.createObjectURL(file)
  }
}

async function generate() {
  generating.value = true
  errorMessage.value = ''

  try {
    const response = await requestImageStudio()
    const nextResults = extractImages(response)
    if (!nextResults.length) {
      throw new Error('图片接口没有返回可展示的结果。')
    }
    results.value = [...nextResults, ...results.value]
    await authStore.refreshUser()
  } catch (error) {
    console.error(error)
    errorMessage.value = error instanceof Error ? error.message : '生成失败，请稍后重试。'
  } finally {
    generating.value = false
  }
}

async function requestImageStudio() {
  const form = new FormData()
  form.append('template_id', templateId.value)
  form.append('product_name', productName.value)
  form.append('selling_points', buildSellingPoints())
  form.append('style', style.value)
  form.append('size', size.value)
  form.append('count', String(count.value))
  if (selectedFile.value) {
    form.append('image', selectedFile.value)
  }

  const token = localStorage.getItem('auth_token')
  const response = await fetch('/api/v1/image-studio/generate', {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: form
  })
  return readImageResponse(response)
}

function buildSellingPoints() {
  return [
    sellingPoints.value && `核心卖点：${sellingPoints.value}`,
    creativeBrief.value && `补充要求：${creativeBrief.value}`,
    selectedPreset.value?.promptHint && `预设要求：${selectedPreset.value.promptHint}`
  ]
    .filter(Boolean)
    .join('\n')
}

async function readImageResponse(response: Response) {
  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    const rawMessage = payload?.error?.message || payload?.message || payload?.detail || `生成失败 (${response.status})`
    const message = normalizeImageError(rawMessage)
    throw new Error(message)
  }
  return payload
}

function normalizeImageError(message: string) {
  if (/does not support OpenAI Images API|images api|image/i.test(message)) {
    return '当前上游账号暂不支持图片生成或改图接口。请在 Sub2 后台配置支持 gpt-image-2 / OpenAI Images API 的图片分组后再使用。'
  }
  if (/please create an active OpenAI API key/i.test(message)) {
    return '当前没有可用于作图的 API Key。请先在后台创建包含 gpt-image-2 的可用 Key，或给用户分配图片分组。'
  }
  return message
}

function extractImages(payload: any): ResultImage[] {
  const data = Array.isArray(payload?.data) ? payload.data : []
  return data
    .map((item: any, index: number) => {
      if (item?.b64_json) {
        return {
          id: `${Date.now()}-${index}`,
          src: `data:image/png;base64,${item.b64_json}`
        }
      }
      if (item?.url) {
        return {
          id: `${Date.now()}-${index}`,
          src: item.url
        }
      }
      return null
    })
    .filter((item: ResultImage | null): item is ResultImage => item !== null)
}

function downloadResult(item: ResultImage, index: number) {
  const link = document.createElement('a')
  link.href = item.src
  link.download = `ssxz-image-${index + 1}.png`
  document.body.appendChild(link)
  link.click()
  link.remove()
}

async function copyPrompt() {
  const text = [
    `创作预设：${selectedPreset.value.label}`,
    `创作模式：${selectedFile.value ? '上传改图' : '文生图'}`,
    `商品或主题：${productName.value || '未填写'}`,
    `核心卖点：${sellingPoints.value || '未填写'}`,
    `补充要求：${creativeBrief.value || '未填写'}`,
    `隐藏提示：${selectedPreset.value.promptHint}`,
    `风格：${style.value}`,
    `尺寸：${size.value}`,
    `张数：${count.value}`,
    selectedFile.value ? `参考图：${selectedFile.value.name}` : '参考图：无'
  ].join('\n')
  await navigator.clipboard.writeText(text)
}

function clearResults() {
  results.value = []
}
</script>
