<template>
  <div class="mx-auto flex max-w-7xl flex-col gap-6">
    <section class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div class="inline-flex items-center gap-2 rounded-md bg-primary-50 px-3 py-1 text-sm font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
            <Icon name="sparkles" size="sm" />
            <span>图片生成工作台</span>
          </div>
          <h1 class="mt-3 text-2xl font-bold tracking-normal text-gray-900 dark:text-white">商品图、海报和灵感图，一页完成</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-dark-300">
            上传参考图会进入改图流程；不上传图片则按文字描述生成。模型、接口和扣费仍由后台统一处理，用户只需要把需求说清楚。
          </p>
        </div>
        <div class="grid grid-cols-2 gap-2 text-center sm:min-w-[280px]">
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900/50">
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ imageCredits }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">约可生成</div>
          </div>
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900/50">
            <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ count }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">本次张数</div>
          </div>
        </div>
      </div>
    </section>

    <div class="grid gap-5 xl:grid-cols-[minmax(0,420px)_1fr]">
      <section class="space-y-5">
        <div class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">创作设置</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">当前按张扣费，预计消耗 {{ count }} 张</p>
            </div>
            <span class="rounded-md bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
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
                <img :src="previewUrl" alt="preview" class="h-48 w-full object-contain bg-gray-50 dark:bg-dark-900" />
              </div>
            </div>

            <div>
              <label class="input-label">作图类型</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  v-for="template in templates"
                  :key="template.id"
                  type="button"
                  class="rounded-lg border px-3 py-2 text-left text-sm transition"
                  :class="templateId === template.id ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200'"
                  @click="templateId = template.id"
                >
                  {{ template.label }}
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="input-label">图片比例</label>
                <select v-model="size" class="input">
                  <option value="1024x1024">1:1</option>
                  <option value="1024x1536">2:3</option>
                  <option value="1536x1024">3:2</option>
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
              <label class="input-label">商品名称</label>
              <input v-model.trim="productName" class="input" type="text" placeholder="例如：保温杯、蓝牙耳机、护肤套装" />
            </div>

            <div>
              <label class="input-label">核心卖点</label>
              <textarea
                v-model.trim="sellingPoints"
                class="input min-h-24 resize-none"
                placeholder="例如：轻便、防水、高级质感、适合礼赠"
              />
            </div>

            <div>
              <label class="input-label">风格</label>
              <select v-model="style" class="input">
                <option value="clean studio commercial photography">干净棚拍</option>
                <option value="premium gray background commercial photography">高级灰</option>
                <option value="warm home lifestyle scene">家居场景</option>
                <option value="fresh social media product photography">小红书风</option>
                <option value="holiday promotion product photography">节日促销</option>
              </select>
            </div>

            <button
              type="button"
              class="btn btn-primary w-full justify-center"
              :disabled="generating"
              @click="generate"
            >
              <Icon v-if="!generating" name="sparkles" size="sm" />
              <Icon v-else name="refresh" size="sm" class="animate-spin" />
              <span>{{ generating ? '生成中' : `消耗约 ${count} 张，立即生成` }}</span>
            </button>

            <p v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300">
              {{ errorMessage }}
            </p>
          </div>
        </div>
      </section>

      <section class="min-h-[520px] rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">生成结果</h2>
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

        <div v-if="!results.length" class="flex min-h-[440px] flex-col items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50 px-6 text-center dark:border-dark-600 dark:bg-dark-900/40">
          <Icon name="sparkles" size="xl" class="text-gray-300 dark:text-dark-500" />
          <h3 class="mt-4 text-base font-semibold text-gray-800 dark:text-dark-100">开始一次创作</h3>
          <p class="mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-dark-400">
            选择作图类型和风格，填写商品名与卖点。生成结果会出现在这里，方便预览和下载。
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
              <span class="text-sm text-gray-500 dark:text-dark-400">图 {{ index + 1 }}</span>
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

type StudioTemplate = {
  id: 'background' | 'white' | 'scene' | 'poster'
  label: string
}

type ResultImage = {
  id: string
  src: string
}

const CREDIT_UNIT_USD = 0.2

const templates: StudioTemplate[] = [
  {
    id: 'background',
    label: '主图换背景'
  },
  {
    id: 'white',
    label: '白底优化'
  },
  {
    id: 'scene',
    label: '电商场景图'
  },
  {
    id: 'poster',
    label: '小红书海报'
  }
]

const authStore = useAuthStore()
const selectedFile = ref<File | null>(null)
const previewUrl = ref('')
const templateId = ref<StudioTemplate['id']>('background')
const size = ref('1024x1024')
const count = ref(1)
const productName = ref('')
const sellingPoints = ref('')
const style = ref('clean studio commercial photography')
const generating = ref(false)
const errorMessage = ref('')
const results = ref<ResultImage[]>([])

const imageCredits = computed(() => {
  const balance = authStore.user?.balance ?? 0
  return Math.max(0, Math.floor(balance / CREDIT_UNIT_USD))
})

onBeforeUnmount(() => {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value)
})

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
      throw new Error('图片接口没有返回可展示的结果')
    }
    results.value = [...nextResults, ...results.value]
    await authStore.refreshUser()
  } catch (error) {
    console.error(error)
    errorMessage.value = error instanceof Error ? error.message : '生成失败，请稍后重试'
  } finally {
    generating.value = false
  }
}

async function requestImageStudio() {
  const form = new FormData()
  form.append('template_id', templateId.value)
  form.append('product_name', productName.value)
  form.append('selling_points', sellingPoints.value)
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
    return '当前账号暂不支持图片生成/改图接口。请联系管理员开通支持图片生成的模型或上游账号后再使用。'
  }
  if (/please create an active OpenAI API key/i.test(message)) {
    return '当前没有可用于作图的 API Key。请先在后台创建支持图片生成的可用 Key，或联系管理员分配图片分组。'
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
  link.download = `image-studio-${index + 1}.png`
  document.body.appendChild(link)
  link.click()
  link.remove()
}

async function copyPrompt() {
  const text = [
    `作图类型：${templates.find((item) => item.id === templateId.value)?.label || templateId.value}`,
    `商品名称：${productName.value || '未填写'}`,
    `核心卖点：${sellingPoints.value || '未填写'}`,
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
