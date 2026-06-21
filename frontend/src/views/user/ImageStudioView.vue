<template>
  <AppSectionShell
    title="AI 作图"
    subtitle="把用途、画幅、参考素材和画面描述整理成图片生成任务，设置只做方向参考，不锁死模型发挥。"
    eyebrow="创意工作台"
    icon="sparkles"
  >
    <section class="image-hero">
      <div>
        <p class="hero-kicker">图片生成工作台</p>
        <h2>把想法整理成可交付的视觉作品</h2>
        <p>
          先说清楚图片用途、主体和画面气质，系统会把这些信息组织成更完整的生成提示词。
        </p>
        <div class="hero-note">
          用途、比例、风格和参考图是创作方向，不是固定人设；自定义比例和风格会写入提示词，方便后续继续升级。
        </div>
        <div class="hero-actions" aria-label="作图辅助入口">
          <RouterLink to="/app/chat" class="hero-helper-link">
            先用对话整理想法
          </RouterLink>
          <span>作图是主流程，对话可以帮你把随口需求整理成更清楚的画面描述。</span>
        </div>
      </div>
      <div class="hero-side">
        <div class="hero-route-card">
          <span>对话辅助保留</span>
          <strong>先聊清楚，再去作图</strong>
          <small>适合整理卖点、标题、画面气质和提示词，不把站点做成纯作图。</small>
        </div>
        <div class="hero-flow" aria-label="创作流程">
          <span>选用途</span>
          <span>定画幅</span>
          <span>补描述</span>
          <span>生成作品</span>
        </div>
        <div class="hero-balance">
          <b>{{ imageCredits }}</b>
          <span>约可生成</span>
        </div>
        <div class="hero-model-card" aria-label="可用图片模型">
          <span>图片模型</span>
          <strong>{{ activeImageModelLabel }}</strong>
          <select
            v-if="imageModelOptions.length"
            v-model="selectedImageModelId"
            class="hero-model-select"
            aria-label="选择图片模型"
          >
            <option v-for="model in imageModelOptions" :key="model.id" :value="model.id">
              {{ model.name }}
            </option>
          </select>
          <small>{{ imageModelHint }}</small>
          <div v-if="imageModelPreview.length" class="hero-model-list">
            <span v-for="model in imageModelPreview" :key="model.id">{{ model.name }}</span>
            <span v-if="hiddenImageModelCount > 0">+{{ hiddenImageModelCount }}</span>
          </div>
        </div>
      </div>
    </section>

    <section class="image-workbench" aria-label="AI 作图工作台">
      <aside class="creation-console" aria-label="创作控制台">
        <div class="console-scroll">
          <section class="console-block">
            <div class="block-heading">
              <span class="step-dot">1</span>
              <div>
                <h3>选择创作目标</h3>
                <p>先明确图片用在哪里，后面的选项只做创意方向参考。</p>
              </div>
            </div>
            <div class="goal-grid">
              <button
                v-for="item in creationGoals"
                :key="item.id"
                type="button"
                class="choice-card"
                :class="{ selected: goal === item.id }"
                @click="selectGoal(item.id)"
              >
                <span>{{ item.label }}</span>
                <small>{{ item.hint }}</small>
              </button>
            </div>
            <label v-if="goal === 'custom'" class="inline-field">
              <span>自定义用途</span>
              <input
                v-model.trim="customGoal"
                class="control-input"
                placeholder="例如：公众号首图、课程封面、门店活动海报"
              />
            </label>
          </section>

          <section class="console-block">
            <div class="block-heading">
              <span class="step-dot">2</span>
              <div>
                <h3>选择画幅比例</h3>
                <p>可选常用电商比例，也可以填自定义比例；当前模型会按最接近尺寸生成。</p>
              </div>
            </div>
            <div class="format-grid">
              <button
                v-for="preset in canvasPresets"
                :key="preset.id"
                type="button"
                class="format-card"
                :class="{ selected: canvasId === preset.id }"
                @click="canvasId = preset.id"
              >
                <b>{{ preset.ratio }}</b>
                <span>{{ preset.label }}</span>
              </button>
              <button
                type="button"
                class="format-card"
                :class="{ selected: canvasId === 'custom' }"
                @click="canvasId = 'custom'"
              >
                <b>自定义</b>
                <span>自定义画幅</span>
              </button>
            </div>

            <div v-if="canvasId === 'custom'" class="sub-panel">
              <div class="ratio-input-row">
                <label class="inline-field">
                  <span>宽</span>
                  <input v-model.trim="customRatioWidth" class="control-input" inputmode="decimal" placeholder="例如：2 或 1080" />
                </label>
                <span class="ratio-separator">:</span>
                <label class="inline-field">
                  <span>高</span>
                  <input v-model.trim="customRatioHeight" class="control-input" inputmode="decimal" placeholder="例如：3 或 1350" />
                </label>
              </div>
              <label class="inline-field">
                <span>画幅用途说明</span>
                <input v-model.trim="customCanvasPurpose" class="control-input" placeholder="例如：朋友圈长图、店铺头图、课程封面" />
              </label>
              <p :class="{ invalid: !isCustomRatioValid }">
                自定义比例会写入提示词；真实输出尺寸按当前图片模型支持的最接近画幅生成。
              </p>
            </div>

            <div class="count-panel">
              <div>
                <span>生成张数</span>
                <small>多张结果会在右侧以缩略图切换，方便对比和下载。</small>
              </div>
              <div class="count-list" role="group" aria-label="生成张数">
                <button
                  v-for="count in imageCountOptions"
                  :key="count"
                  type="button"
                  class="count-chip"
                  :class="{ selected: imageCount === count }"
                  :aria-pressed="imageCount === count"
                  @click="imageCount = count"
                >
                  {{ count }} 张
                </button>
              </div>
            </div>
          </section>

          <section class="console-block">
            <div class="block-heading">
              <span class="step-dot">3</span>
              <div>
                <h3>上传参考素材</h3>
                <p>先保留单张商品图或风格图，后续多图参考单独升级。</p>
              </div>
            </div>
            <label
              class="asset-drop"
              :class="{ filled: Boolean(selectedFile), dragging: referenceDragging }"
              @dragenter.prevent="handleReferenceDragEnter"
              @dragover.prevent="handleReferenceDragEnter"
              @dragleave.prevent="handleReferenceDragLeave"
              @drop.prevent="handleReferenceDrop"
            >
              <template v-if="!selectedFile">
                <Icon name="upload" size="lg" />
                <span>上传商品图或风格参考图</span>
                <small>JPG / PNG / WEBP，上传后自动进入改图模式</small>
              </template>
              <template v-else>
                <span v-if="previewUrl && !previewImageFailed" class="asset-thumb">
                  <img
                    :src="previewUrl"
                    alt="参考素材预览"
                    @load="previewImageFailed = false"
                    @error="handlePreviewImageError"
                  />
                </span>
                <span v-else class="asset-thumb is-error" aria-live="polite">
                  <Icon name="exclamationTriangle" size="sm" />
                  <small>预览失败</small>
                </span>
                <span class="asset-copy">
                  <strong class="asset-title" :title="selectedFile.name">{{ selectedFile.name }}</strong>
                  <small>参考图 1 · {{ referenceMeta }}</small>
                  <em>点击更换参考素材</em>
                </span>
              </template>
              <input class="hidden" type="file" accept="image/png,image/jpeg,image/webp" @change="handleFileChange" />
            </label>
            <p v-if="referencePreviewError" class="reference-upload-note error">{{ referencePreviewError }}</p>
            <button v-if="selectedFile" type="button" class="secondary-button mt-2 w-full" @click="clearReference">
              移除参考图
            </button>
          </section>

          <section class="console-block">
            <div class="block-heading">
              <span class="step-dot">4</span>
              <div>
                <h3>描述主体与画面</h3>
                <p>用普通话说需求，页面会整理成更完整的生成提示词。</p>
              </div>
            </div>
            <label class="inline-field">
              <span>商品 / 主体名称</span>
              <input v-model.trim="productName" class="control-input" placeholder="例如：无线耳机、护肤精华、咖啡杯、运动水杯" />
            </label>
            <label class="inline-field">
              <span>自有品牌名（可选）</span>
              <input v-model.trim="brandName" class="control-input" placeholder="如需品牌露出，请填写自有品牌名；未提供时默认无品牌原创图。" />
            </label>
            <label class="inline-field">
              <span>画面描述</span>
              <textarea
                v-model.trim="prompt"
                class="control-textarea"
                placeholder="例如：帮我做一张适合小红书卖护肤品的封面图，画面高级、清透、留出标题区域。"
              />
            </label>
          </section>

          <section class="console-block">
            <div class="block-heading">
              <span class="step-dot">5</span>
              <div>
                <h3>选择视觉方向</h3>
                <p>风格是方向，不是死规则；不确定时用默认即可。</p>
              </div>
            </div>
            <label class="inline-field">
              <span>场景或氛围</span>
              <input v-model.trim="scene" class="control-input" placeholder="例如：晨光办公桌、极简白棚、户外露营、浴室台面" />
            </label>
            <div class="style-area">
              <span>风格偏好</span>
              <div class="style-list">
                <button
                  v-for="styleOption in styleOptions"
                  :key="styleOption"
                  type="button"
                  class="style-chip"
                  :class="{ selected: style === styleOption }"
                  @click="style = styleOption"
                >
                  {{ styleOption }}
                </button>
              </div>
            </div>
            <label v-if="style === customStyleOption" class="inline-field">
              <span>自定义风格描述</span>
              <input v-model.trim="customStyle" class="control-input" placeholder="例如：复古杂志感、奢侈品广告风、科技蓝银质感" />
            </label>

            <details class="advanced-panel">
              <summary>高级补充</summary>
              <div class="advanced-body">
                <label class="inline-field">
                  <span>补充关键词 / 负面要求</span>
                  <input v-model.trim="keywords" class="control-input" placeholder="例如：高清细节、无水印、不要乱码文字、留白构图" />
                </label>
              </div>
            </details>
          </section>
        </div>

        <div class="console-action">
          <div class="generation-meta">
            <span>{{ selectedFile ? '改图模式' : '文生图模式' }}</span>
            <span>{{ displayCanvasRatio }}</span>
            <span>{{ imageCount }} 张</span>
          </div>
          <p class="safety-note">
            AI 生成图片仅供参考，商用前请确认没有第三方品牌、Logo、版权素材或不真实商品信息。
          </p>
          <button type="button" class="generate-button" :disabled="isGenerateDisabled" @click="generate">
            <Icon :name="generating ? 'refresh' : 'sparkles'" size="sm" :class="{ 'animate-spin': generating }" />
            {{ generateLabel }}
          </button>
          <p v-if="errorMessage" class="error-note">
            {{ errorMessage }}
          </p>
        </div>
      </aside>

      <section ref="previewStageRef" class="canvas-panel" aria-label="作品画布">
        <header class="canvas-summary">
          <div>
            <p>当前任务</p>
            <h3>{{ displayGoalLabel }}</h3>
          </div>
          <div class="canvas-tags">
            <span>{{ displayCanvasRatio }}</span>
            <span>{{ displayStyleLabel }}</span>
            <span>{{ selectedFile ? '改图' : '生成' }}</span>
            <span>{{ canvasStateLabel }}</span>
          </div>
        </header>

        <div class="canvas-stage">
          <div class="canvas-sheet" :style="canvasSheetStyle">
            <img v-if="activeResult" :src="activeResult.src" :alt="`result-${activeResultIndex + 1}`" />
            <div v-else class="canvas-empty" :class="{ failed: errorMessage }">
              <span class="canvas-empty-badge">{{ errorMessage ? '需要处理' : '等待创作' }}</span>
              <Icon :name="generating ? 'refresh' : errorMessage ? 'exclamationCircle' : 'sparkles'" size="lg" :class="{ 'animate-spin': generating }" />
              <h3>{{ emptyStateTitle }}</h3>
              <p>{{ emptyStateDescription }}</p>
              <div class="canvas-empty-guide" aria-label="创作提示">
                <span>普通话说需求</span>
                <span>对话辅助润色</span>
                <span>生成后对比下载</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="results.length > 1" class="thumbnail-strip" aria-label="生成缩略图">
          <button
            v-for="(item, index) in results"
            :key="item.id"
            type="button"
            class="thumbnail-button"
            :class="{ selected: activeResultIndex === index }"
            @click="activeResultIndex = index"
          >
            <img :src="item.src" :alt="`thumbnail-${index + 1}`" />
          </button>
        </div>

        <footer class="canvas-foot">
          <div>
            <span>用途</span>
            <b>{{ displayGoalLabel }}</b>
          </div>
          <div>
            <span>比例</span>
            <b>{{ displayCanvasRatio }}</b>
          </div>
          <div>
            <span>状态</span>
            <b>{{ canvasStateLabel }}</b>
          </div>
          <div>
            <span>模型</span>
            <b>{{ activeImageModelLabel }}</b>
          </div>
        </footer>

        <div class="result-actions">
          <button type="button" class="secondary-button" @click="copyPrompt">复制提示词</button>
          <button type="button" class="secondary-button" :disabled="!activeResult" @click="downloadActiveResult">
            下载图片
          </button>
          <button type="button" class="secondary-button" :disabled="!activeResult || generating" @click="generate">
            重新生成
          </button>
        </div>

        <details class="prompt-preview">
          <summary>查看本次整理后的提示词</summary>
          <pre>{{ fullPrompt }}</pre>
        </details>
      </section>
    </section>

    <section class="recent-works">
      <div class="recent-heading">
        <div>
          <h2>最近作品</h2>
          <p>生成成功后的图片会同步到这里，方便之后回来预览和下载。</p>
        </div>
        <button type="button" class="secondary-button" :disabled="recentWorksLoading" @click="loadRecentWorks">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': recentWorksLoading }" />
          刷新
        </button>
      </div>

      <div v-if="recentWorksLoading && !recentWorks.length" class="recent-empty">
        正在加载最近作品...
      </div>
      <div v-else-if="!recentWorks.length" class="recent-empty">
        <Icon name="grid" size="lg" />
        <p>还没有历史作品。完成一次图片生成后，这里会展示最近结果。</p>
      </div>
      <div v-else class="recent-grid">
        <article v-for="(work, index) in recentWorks" :key="work.id" class="recent-card">
          <button
            type="button"
            class="recent-thumb recent-thumb-button"
            :aria-label="`预览图片作品 ${work.id}`"
            @click="openWorkPreview(work, index)"
          >
            <img :src="workImageSrc(work)" :alt="`work-${work.id}`" />
            <span class="recent-thumb-hint">点击预览</span>
          </button>
          <div class="recent-card-body">
            <div>
              <span>图片作品</span>
              <small>{{ formatWorkTime(work.created_at) }}</small>
            </div>
            <div class="recent-card-actions">
              <button type="button" class="secondary-button" @click="openWorkPreview(work, index)">预览</button>
              <button type="button" class="secondary-button" @click="downloadWork(work, index)">下载</button>
            </div>
          </div>
        </article>
      </div>
    </section>

    <div
      v-if="previewWork"
      class="recent-preview-backdrop"
      role="presentation"
      @click.self="closeWorkPreview"
    >
      <section
        class="recent-preview-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="recent-preview-title"
      >
        <header class="recent-preview-header">
          <div>
            <p>作品预览</p>
            <h2 id="recent-preview-title">图片作品</h2>
          </div>
          <button type="button" class="recent-preview-close" aria-label="关闭预览" @click="closeWorkPreview">
            ×
          </button>
        </header>
        <div class="recent-preview-stage">
          <img :src="previewWork.src" :alt="`preview-work-${previewWork.work.id}`" />
        </div>
        <footer class="recent-preview-footer">
          <div>
            <span>生成时间</span>
            <b>{{ formatWorkTime(previewWork.work.created_at) || '未知' }}</b>
          </div>
          <div>
            <span>模型</span>
            <b>{{ previewWork.work.model || '后台配置' }}</b>
          </div>
          <div class="recent-preview-actions">
            <button type="button" class="secondary-button" @click="downloadPreviewWork">下载图片</button>
            <button type="button" class="secondary-button" @click="closeWorkPreview">关闭</button>
          </div>
        </footer>
      </section>
    </div>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import axios from 'axios'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import { apiClient } from '@/api/client'
import soraAPI, { type SoraGeneration } from '@/api/sora'
import { useUserCapabilities } from '@/composables/useUserCapabilities'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type GoalId = 'product' | 'poster' | 'social' | 'detail' | 'scene' | 'custom'
type CanvasId = 'square' | 'social45' | 'detail34' | 'poster23' | 'scene32' | 'cover169' | 'short916' | 'banner219' | 'custom'

type ResultImage = {
  id: string
  src: string
}

type ImageStudioPayload = {
  data?: Array<{ b64_json?: string; url?: string }>
}

const appStore = useAppStore()
const authStore = useAuthStore()
const capabilities = useUserCapabilities()

const creationGoals: Array<{ id: GoalId; label: string; hint: string }> = [
  { id: 'product', label: '商品主图', hint: '突出主体与质感' },
  { id: 'poster', label: '营销海报', hint: '活动与促销传播' },
  { id: 'social', label: '社媒封面', hint: '小红书与内容发布' },
  { id: 'detail', label: '详情页配图', hint: '细节和功能展示' },
  { id: 'scene', label: '场景图', hint: '生活方式与氛围' },
  { id: 'custom', label: '自定义创作', hint: '自定义用途和画幅' }
]

const canvasPresets: Array<{ id: Exclude<CanvasId, 'custom'>; label: string; ratio: string; width: number; height: number }> = [
  { id: 'square', label: '电商主图', ratio: '1:1', width: 1, height: 1 },
  { id: 'social45', label: '社媒竖图', ratio: '4:5', width: 4, height: 5 },
  { id: 'detail34', label: '详情配图', ratio: '3:4', width: 3, height: 4 },
  { id: 'poster23', label: '竖版海报', ratio: '2:3', width: 2, height: 3 },
  { id: 'scene32', label: '场景横图', ratio: '3:2', width: 3, height: 2 },
  { id: 'cover169', label: '横版封面', ratio: '16:9', width: 16, height: 9 },
  { id: 'short916', label: '短视频封面', ratio: '9:16', width: 9, height: 16 },
  { id: 'banner219', label: 'Banner', ratio: '21:9', width: 21, height: 9 }
]

const canvasByGoal: Record<GoalId, Exclude<CanvasId, 'custom'>> = {
  product: 'square',
  poster: 'poster23',
  social: 'social45',
  detail: 'detail34',
  scene: 'scene32',
  custom: 'square'
}

const styleToPrompt: Record<string, string> = {
  '现代极简': 'modern minimal commercial photography, clean composition, restrained palette',
  '高级商业摄影': 'premium commercial photography, refined lighting, realistic texture, high-end composition',
  '清透白底': 'clean white background, soft light, sharp product edges, ecommerce ready',
  '生活方式场景': 'warm lifestyle scene, realistic environment, natural props, soft daylight',
  '自然柔光': 'soft natural light, airy composition, gentle shadows, premium texture',
  '低饱和质感': 'low saturation premium texture, muted color palette, editorial product photography',
  '科技感': 'sleek technology advertising style, cool lighting, glass and metal texture',
  '国潮质感': 'modern Chinese visual style, elegant cultural detail, premium commercial layout',
  '奢华质感': 'luxury advertising style, elegant materials, dramatic yet clean lighting'
}

const customStyleOption = '自定义风格'
const styleOptions = [...Object.keys(styleToPrompt), customStyleOption]
const imageCountOptions = [1, 2, 3]
const CREDIT_UNIT_USD = 0.2
const verifiedDefaultImageModelId = 'gpt-image-2'

const goal = ref<GoalId>('product')
const canvasId = ref<CanvasId>('square')
const customGoal = ref('')
const customRatioWidth = ref('')
const customRatioHeight = ref('')
const customCanvasPurpose = ref('')
const productName = ref('')
const brandName = ref('')
const prompt = ref('')
const scene = ref('')
const style = ref('高级商业摄影')
const customStyle = ref('')
const keywords = ref('')
const imageCount = ref(1)
const selectedImageModelId = ref('')
const selectedFile = ref<File | null>(null)
const previewUrl = ref('')
const previewImageFailed = ref(false)
const referenceDragging = ref(false)
const referencePreviewError = ref('')
const generating = ref(false)
const errorMessage = ref('')
const results = ref<ResultImage[]>([])
const activeResultIndex = ref(0)
const recentWorks = ref<SoraGeneration[]>([])
const recentWorksLoading = ref(false)
const previewStageRef = ref<HTMLElement | null>(null)
const previewWork = ref<{ work: SoraGeneration; index: number; src: string } | null>(null)
let referenceReadSerial = 0

const selectedGoal = computed(() => creationGoals.find((item) => item.id === goal.value) || creationGoals[0])
const selectedCanvas = computed(() => {
  if (canvasId.value === 'custom') return customCanvas.value
  return canvasPresets.find((item) => item.id === canvasId.value) || canvasPresets[0]
})
const customRatioValue = computed(() => {
  const width = customRatioWidth.value.trim()
  const height = customRatioHeight.value.trim()
  return width && height ? `${width}:${height}` : ''
})
const isCustomRatioValid = computed(() => {
  if (canvasId.value !== 'custom') return true
  return isPositiveRatioPart(customRatioWidth.value) && isPositiveRatioPart(customRatioHeight.value)
})
const customCanvas = computed(() => {
  const parsed = parseRatio(customRatioValue.value)
  return {
    id: 'custom' as const,
    label: customCanvasPurpose.value || '自定义画幅',
    ratio: parsed.label,
    width: parsed.width,
    height: parsed.height
  }
})
const displayGoalLabel = computed(() => goal.value === 'custom' ? (customGoal.value || '自定义创作') : selectedGoal.value.label)
const displayCanvasRatio = computed(() => canvasId.value === 'custom' ? (customRatioValue.value ? `自定义 ${customRatioValue.value}` : '自定义比例') : selectedCanvas.value.ratio)
const displayStyleLabel = computed(() => style.value === customStyleOption ? (customStyle.value || '自定义风格') : style.value)
const requestStyle = computed(() => {
  if (style.value === customStyleOption) return customStyle.value || 'custom user-defined commercial visual style'
  return styleToPrompt[style.value] || style.value
})
const requestSize = computed(() => {
  if (selectedCanvas.value.width === selectedCanvas.value.height) return '1024x1024'
  return selectedCanvas.value.width > selectedCanvas.value.height ? '1536x1024' : '1024x1536'
})
const hasDescription = computed(() => Boolean(productName.value.trim() || prompt.value.trim()))
const isGenerateDisabled = computed(() => generating.value || !hasDescription.value || !isCustomRatioValid.value)
const generateLabel = computed(() => {
  if (generating.value) return '正在生成...'
  if (!hasDescription.value) return '请先描述图片'
  if (!isCustomRatioValid.value) return '请填写有效比例'
  return `消耗约 ${imageCount.value} 张，生成图片`
})
const activeResult = computed(() => results.value[activeResultIndex.value] || null)
const canvasStateLabel = computed(() => {
  if (generating.value) return '生成中'
  if (errorMessage.value) return '失败'
  if (activeResult.value) return '已生成'
  return '待生成'
})
const emptyStateTitle = computed(() => {
  if (generating.value) return '正在生成作品'
  if (errorMessage.value) return '生成失败'
  return '你的作品将在这里呈现'
})
const emptyStateDescription = computed(() => {
  if (generating.value) return '系统正在把左侧需求整理成图片生成任务，请稍等。'
  if (errorMessage.value) return errorMessage.value
  return '选择输出用途、上传参考素材并描述创意需求，生成后的作品可在这里预览和下载。'
})
const canvasSheetStyle = computed(() => ({
  aspectRatio: `${selectedCanvas.value.width} / ${selectedCanvas.value.height}`
}))
const imageCredits = computed(() => {
  const balance = authStore.user?.balance ?? 0
  return Math.max(0, Math.floor(balance / CREDIT_UNIT_USD))
})
const imageModelOptions = computed(() => capabilities.imageModels.value)
const imageModelPreview = computed(() => imageModelOptions.value.slice(0, 3))
const hiddenImageModelCount = computed(() => Math.max(0, imageModelOptions.value.length - imageModelPreview.value.length))
const activeImageModel = computed(() => {
  if (selectedImageModelId.value) {
    const selected = imageModelOptions.value.find((model) => model.id === selectedImageModelId.value)
    if (selected) return selected
  }
  const preferred = resolvePreferredImageModelId(imageModelOptions.value, capabilities.defaultImageModel.value)
  return imageModelOptions.value.find((model) => model.id === preferred) || null
})
const activeImageModelLabel = computed(() => activeImageModel.value?.name || '后台配置')
const imageModelHint = computed(() => {
  if (capabilities.loading.value) return '正在读取账号可用的图片模型。'
  if (activeImageModel.value) return '来自账号可用渠道；当前生成仍以后端配置为准。'
  return '暂未读取到可展示的图片模型，请确认后台账号、分组和价格配置。'
})
const referenceMeta = computed(() => {
  if (!selectedFile.value) return '未上传'
  const ext = selectedFile.value.name.split('.').pop()?.toUpperCase() || fileTypeLabel(selectedFile.value.type)
  return `${ext} · ${formatFileSize(selectedFile.value.size)}`
})
const purposeInstruction = computed(() => {
  switch (goal.value) {
    case 'product':
      return '适合电商商品主图，主体清晰，背景干净，突出材质、轮廓和卖点。'
    case 'poster':
      return '适合营销海报，保留标题和卖点区域，画面有传播感和商业视觉冲击。'
    case 'social':
      return '适合小红书、朋友圈、短视频封面，画面自然、真实、有生活方式氛围。'
    case 'detail':
      return '适合详情页配图，体现产品细节、功能、使用场景或质感模块。'
    case 'scene':
      return '适合场景图，强调环境、空间、氛围和真实使用感。'
    default:
      return '按用户自定义用途生成，不额外套固定用途模板。'
  }
})
const commercialSafetyPrompt = computed(() => {
  const brand = brandName.value.trim()
  const brandLine = brand
    ? `User-owned brand name: ${brand}. Only use this brand if it fits the composition; do not invent other brands.`
    : 'No owned brand was provided. Create an unbranded original commercial image.'
  return [
    brandLine,
    'Do not imitate real-world brand logos, product packaging, model names, certification marks, copyrighted product images, watermarks, QR codes, or stock-photo marks.',
    'Avoid unreadable text. If text is needed, reserve clean blank layout areas instead of generating fake copy.'
  ].join('\n')
})
const fullPrompt = computed(() => [
  `创作用途：${displayGoalLabel.value}`,
  `输出比例：${displayCanvasRatio.value}`,
  productName.value ? `主体名称：${productName.value}` : '',
  brandName.value ? `自有品牌名：${brandName.value}` : '品牌策略：未提供自有品牌时默认无品牌原创图',
  prompt.value ? `画面描述：${prompt.value}` : '',
  scene.value ? `场景氛围：${scene.value}` : '',
  `视觉风格：${displayStyleLabel.value}`,
  keywords.value ? `补充关键词：${keywords.value}` : '',
  `用途增强：${purposeInstruction.value}`,
  `商用安全：${commercialSafetyPrompt.value}`,
  `生成要求：professional commercial photography, realistic lighting, sharp details, balanced composition.`
].filter(Boolean).join('\n'))

watch(goal, (next) => {
  if (canvasId.value !== 'custom') {
    canvasId.value = canvasByGoal[next]
  }
})

watch(results, () => {
  activeResultIndex.value = 0
})

watch(imageModelOptions, (models) => {
  if (!models.length) {
    selectedImageModelId.value = ''
    return
  }
  if (selectedImageModelId.value && models.some((model) => model.id === selectedImageModelId.value)) {
    return
  }
  selectedImageModelId.value = resolvePreferredImageModelId(models, capabilities.defaultImageModel.value)
}, { immediate: true })

onMounted(() => {
  void loadRecentWorks()
  void capabilities.loadCapabilities()
})

onBeforeUnmount(() => {
  releasePreviewUrl()
  removeWorkPreviewKeydown()
})

function selectGoal(next: GoalId) {
  goal.value = next
}

function resolvePreferredImageModelId(models: Array<{ id: string }>, fallbackModelId = '') {
  if (models.some((model) => model.id === verifiedDefaultImageModelId)) return verifiedDefaultImageModelId
  if (fallbackModelId && models.some((model) => model.id === fallbackModelId)) return fallbackModelId
  return models[0]?.id || ''
}

function parseRatio(value: string) {
  const match = value.trim().match(/^(\d+(?:\.\d+)?)\s*[:/]\s*(\d+(?:\.\d+)?)$/)
  if (!match) return { label: '自定义比例', width: 1, height: 1 }
  const width = Number(match[1])
  const height = Number(match[2])
  if (!width || !height) return { label: '自定义比例', width: 1, height: 1 }
  return { label: `${match[1]}:${match[2]}`, width, height }
}

function isPositiveRatioPart(value: string) {
  if (!/^\d+(\.\d)?\d*$/.test(value.trim())) return false
  return Number(value) > 0
}

function fileTypeLabel(mime: string) {
  if (mime.includes('png')) return 'PNG'
  if (mime.includes('jpeg') || mime.includes('jpg')) return 'JPG'
  if (mime.includes('webp')) return 'WEBP'
  return 'IMAGE'
}

function formatFileSize(bytes: number) {
  if (!bytes) return '未知大小'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  input.value = ''
  handleReferenceFile(file)
}

function handleReferenceDragEnter() {
  referenceDragging.value = true
}

function handleReferenceDragLeave(event: DragEvent) {
  const currentTarget = event.currentTarget as HTMLElement | null
  const relatedTarget = event.relatedTarget as Node | null
  if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) return
  referenceDragging.value = false
}

function handleReferenceDrop(event: DragEvent) {
  referenceDragging.value = false
  const file = event.dataTransfer?.files?.[0]
  if (!file) return
  handleReferenceFile(file)
}

function handleReferenceFile(file: File) {
  if (!isAllowedReferenceImage(file)) {
    referenceReadSerial += 1
    selectedFile.value = null
    releasePreviewUrl()
    previewImageFailed.value = false
    appStore.showError('请上传 JPG / PNG / WEBP 图片。')
    referencePreviewError.value = '请上传 JPG / PNG / WEBP 图片。'
    return
  }

  const serial = ++referenceReadSerial
  referencePreviewError.value = ''
  previewImageFailed.value = false
  releasePreviewUrl()

  try {
    const nextPreviewUrl = createReferencePreviewUrl(file)
    if (serial !== referenceReadSerial) {
      revokeReferencePreviewUrl(nextPreviewUrl)
      return
    }
    selectedFile.value = file
    previewUrl.value = nextPreviewUrl
  } catch {
    if (serial !== referenceReadSerial) return
    selectedFile.value = null
    previewUrl.value = ''
    previewImageFailed.value = true
    referencePreviewError.value = '参考图预览失败，请重新上传 JPG / PNG / WEBP 图片。'
    appStore.showError(referencePreviewError.value)
  }
}

function clearReference() {
  referenceReadSerial += 1
  selectedFile.value = null
  releasePreviewUrl()
  referencePreviewError.value = ''
  previewImageFailed.value = false
}

function releasePreviewUrl() {
  const currentPreviewUrl = previewUrl.value
  previewUrl.value = ''
  revokeReferencePreviewUrl(currentPreviewUrl)
}

function isAllowedReferenceImage(file: File) {
  return ['image/png', 'image/jpeg', 'image/webp'].includes(file.type)
}

function createReferencePreviewUrl(file: File) {
  if (typeof URL === 'undefined' || typeof URL.createObjectURL !== 'function') {
    throw new Error('reference preview is unavailable')
  }
  return URL.createObjectURL(file)
}

function revokeReferencePreviewUrl(url: string) {
  if (!url || !url.startsWith('blob:')) return
  if (typeof URL === 'undefined' || typeof URL.revokeObjectURL !== 'function') return
  URL.revokeObjectURL(url)
}

function handlePreviewImageError() {
  referenceReadSerial += 1
  selectedFile.value = null
  releasePreviewUrl()
  previewImageFailed.value = false
  referencePreviewError.value = '参考图预览加载失败，请重新上传 JPG / PNG / WEBP 图片。'
  appStore.showError(referencePreviewError.value)
}

async function generate() {
  if (isGenerateDisabled.value) {
    appStore.showInfo(!hasDescription.value ? '请先填写主体或画面描述。' : '请填写有效的自定义比例。')
    return
  }

  generating.value = true
  errorMessage.value = ''
  requestAnimationFrame(scrollPreviewIntoView)

  try {
    const response = await requestImageStudio()
    const nextResults = extractImages(response)
    if (!nextResults.length) {
      throw new Error('图片接口没有返回可展示的结果')
    }
    results.value = nextResults
    await authStore.refreshUser()
    await loadRecentWorks()
    appStore.showSuccess('图片生成完成')
    requestAnimationFrame(scrollPreviewIntoView)
  } catch (error) {
    console.error(error)
    errorMessage.value = normalizeUnknownError(error)
    appStore.showError(errorMessage.value)
  } finally {
    generating.value = false
  }
}

async function requestImageStudio() {
  const form = new FormData()
  form.append('template_id', mapGoalToTemplate(goal.value))
  form.append('product_name', productName.value || displayGoalLabel.value)
  form.append('selling_points', fullPrompt.value)
  form.append('style', requestStyle.value)
  form.append('size', requestSize.value)
  form.append('count', String(imageCount.value))
  if (activeImageModel.value?.id) {
    form.append('model', activeImageModel.value.id)
  }
  if (selectedFile.value) {
    form.append('image', selectedFile.value)
  }

  const response = await apiClient.post<ImageStudioPayload>('/image-studio/generate', form, {
    timeout: 120000
  })
  return response.data
}

function mapGoalToTemplate(value: GoalId) {
  if (value === 'product') return 'white'
  if (value === 'poster' || value === 'social') return 'poster'
  if (value === 'detail' || value === 'scene') return 'scene'
  return 'background'
}

function normalizeUnknownError(error: unknown) {
  if (axios.isAxiosError(error)) {
    const payload = error.response?.data as any
    const raw = payload?.error?.message || payload?.message || payload?.detail || error.message
    return normalizeImageError(String(raw || '生成失败，请稍后重试'), error.response?.status)
  }
  const payload = (error as { response?: { data?: any } })?.response?.data
  const status = (error as { response?: { status?: number } })?.response?.status
  const maybeMessage = payload?.error?.message || payload?.message || payload?.detail || (error as { message?: string })?.message
  return normalizeImageError(maybeMessage || '生成失败，请稍后重试', status)
}

function normalizeImageError(message: string, status?: number) {
  if (/does not support OpenAI Images API|images api|image/i.test(message)) {
    return '当前账号暂不支持图片生成/改图接口。请联系管理员开通支持图片生成的模型或上游账号后再使用。'
  }
  if (/please create an active OpenAI API key/i.test(message)) {
    return '当前没有可用于作图的 API Key。请先在后台创建支持图片生成的可用 Key，或联系管理员分配图片分组。'
  }
  if (
    (typeof status === 'number' && status >= 500)
    || /Request failed with status code 5\d\d/i.test(message)
    || /Network Error|timeout/i.test(message)
  ) {
    return '图片生成服务暂不可用，请稍后重试或联系管理员。'
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

async function loadRecentWorks() {
  recentWorksLoading.value = true
  try {
    const response = await soraAPI.listGenerations({
      status: 'completed',
      media_type: 'image',
      page: 1,
      page_size: 8
    })
    const rows = Array.isArray(response.data) ? response.data : []
    recentWorks.value = rows.filter((item) => workImageSrc(item) !== '')
  } catch (error) {
    console.error('Failed to load image works:', error)
  } finally {
    recentWorksLoading.value = false
  }
}

function workImageSrc(work: SoraGeneration) {
  if (work.media_url) return work.media_url
  return work.media_urls?.find((url) => typeof url === 'string' && url.trim() !== '') || ''
}

function downloadActiveResult() {
  if (!activeResult.value) return
  downloadResult(activeResult.value, activeResultIndex.value)
}

function downloadResult(item: ResultImage, index: number) {
  const link = document.createElement('a')
  link.href = item.src
  link.download = `image-studio-${index + 1}.png`
  document.body.appendChild(link)
  link.click()
  link.remove()
}

function downloadWork(work: SoraGeneration, index: number) {
  const src = workImageSrc(work)
  if (!src) return
  const link = document.createElement('a')
  link.href = src
  link.download = `image-work-${work.id || index + 1}.png`
  document.body.appendChild(link)
  link.click()
  link.remove()
}

function openWorkPreview(work: SoraGeneration, index: number) {
  const src = workImageSrc(work)
  if (!src) return
  removeWorkPreviewKeydown()
  previewWork.value = { work, index, src }
  addWorkPreviewKeydown()
}

function closeWorkPreview() {
  previewWork.value = null
  removeWorkPreviewKeydown()
}

function downloadPreviewWork() {
  if (!previewWork.value) return
  downloadWork(previewWork.value.work, previewWork.value.index)
}

function handleWorkPreviewKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    closeWorkPreview()
  }
}

function addWorkPreviewKeydown() {
  window.addEventListener('keydown', handleWorkPreviewKeydown)
}

function removeWorkPreviewKeydown() {
  window.removeEventListener('keydown', handleWorkPreviewKeydown)
}

function formatWorkTime(iso: string) {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString()
}

async function copyPrompt() {
  await navigator.clipboard.writeText(fullPrompt.value)
  appStore.showSuccess('提示词已复制')
}

function scrollPreviewIntoView() {
  previewStageRef.value?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
}
</script>

<style scoped>
.image-hero,
.image-workbench,
.creation-console,
.canvas-panel,
.recent-works {
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface);
  box-shadow: var(--ssxz-shadow);
}

.image-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1rem;
  overflow: hidden;
  padding: 1rem 1.1rem;
  border-radius: 1.25rem;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--ssxz-primary) 8%, var(--ssxz-surface)) 0%, var(--ssxz-surface) 52%),
    var(--ssxz-surface);
}

.image-hero::after {
  content: "";
  position: absolute;
  inset: auto -12% -60% 38%;
  height: 12rem;
  pointer-events: none;
  background:
    linear-gradient(135deg, transparent 0 46%, color-mix(in srgb, var(--ssxz-primary) 12%, transparent) 46% 54%, transparent 54% 100%);
  opacity: 0.55;
  transform: rotate(-6deg);
}

.hero-kicker,
.block-heading p,
.canvas-summary p,
.canvas-foot span,
.safety-note,
.asset-copy small,
.asset-copy em,
.prompt-preview,
.recent-heading p {
  color: var(--ssxz-text-muted);
}

.hero-kicker {
  display: inline-flex;
  width: fit-content;
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-primary) 12%, transparent);
  color: var(--ssxz-action);
  font-size: 0.76rem;
  font-weight: 850;
  padding: 0.28rem 0.6rem;
}

.image-hero h2 {
  margin: 0.18rem 0;
  color: var(--ssxz-text-primary);
  font-size: clamp(1.35rem, 2vw, 1.9rem);
}

.image-hero p {
  margin: 0;
  max-width: 48rem;
  line-height: 1.7;
}

.hero-note {
  margin-top: 0.75rem;
  border-left: 3px solid var(--ssxz-action);
  border-radius: 0.85rem;
  background: color-mix(in srgb, var(--ssxz-action-soft) 72%, transparent);
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  line-height: 1.65;
  padding: 0.62rem 0.75rem;
}

.hero-side,
.hero-flow,
.generation-meta,
.canvas-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.55rem;
  margin-top: 0.78rem;
  color: var(--ssxz-text-muted);
  font-size: 0.82rem;
}

.hero-helper-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-primary);
  font-weight: 800;
  padding: 0.42rem 0.72rem;
}

.hero-helper-link:hover {
  border-color: var(--ssxz-border-strong);
  background: color-mix(in srgb, var(--ssxz-action-soft) 66%, var(--ssxz-surface-subtle));
}

.hero-helper-link:focus-visible,
.count-chip:focus-visible {
  outline: none;
  border-color: var(--ssxz-focus);
  box-shadow: 0 0 0 3px var(--ssxz-focus-ring);
}

.hero-side {
  justify-content: flex-end;
  max-width: 28rem;
}

.hero-route-card,
.hero-model-card {
  width: 100%;
  border: 1px solid color-mix(in srgb, var(--ssxz-action) 36%, var(--ssxz-border));
  border-radius: 1rem;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--ssxz-action-soft) 72%, transparent), transparent 68%),
    var(--ssxz-surface-subtle);
  padding: 0.78rem 0.9rem;
}

.hero-route-card span,
.hero-route-card small,
.hero-model-card > span,
.hero-model-card small {
  display: block;
  color: var(--ssxz-text-muted);
}

.hero-route-card span,
.hero-model-card > span {
  font-size: 0.72rem;
  font-weight: 850;
}

.hero-route-card strong,
.hero-model-card strong {
  display: block;
  margin-top: 0.18rem;
  color: var(--ssxz-text-primary);
  font-size: 0.98rem;
}

.hero-model-select {
  width: 100%;
  margin-top: 0.55rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.75rem;
  background: var(--ssxz-surface);
  color: var(--ssxz-text-primary);
  font: inherit;
  font-size: 0.84rem;
  font-weight: 750;
  padding: 0.48rem 0.58rem;
}

.hero-model-select:focus-visible {
  outline: none;
  border-color: var(--ssxz-focus);
  box-shadow: 0 0 0 3px var(--ssxz-focus-ring);
}

.hero-route-card small,
.hero-model-card small {
  margin-top: 0.32rem;
  line-height: 1.55;
}

.hero-model-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.55rem;
}

.hero-model-list span {
  display: inline-flex;
  max-width: 9rem;
  overflow: hidden;
  align-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
  font-size: 0.72rem;
  font-weight: 750;
  padding: 0.28rem 0.5rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hero-flow span,
.generation-meta span,
.canvas-tags span,
.style-chip {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-secondary);
}

.hero-flow span,
.generation-meta span,
.canvas-tags span {
  padding: 0.38rem 0.65rem;
  font-size: 0.78rem;
  font-weight: 700;
}

.hero-balance {
  display: grid;
  min-width: 5.8rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-subtle);
  padding: 0.65rem 0.8rem;
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
}

.hero-balance b {
  color: var(--ssxz-text-primary);
  font-size: 1.15rem;
}

.image-workbench {
  display: grid;
  grid-template-columns: minmax(360px, 440px) minmax(0, 1fr);
  gap: 1rem;
  min-height: min(78vh, 860px);
  border-radius: 1.35rem;
  background: color-mix(in srgb, var(--ssxz-surface) 82%, var(--ssxz-surface-muted));
  padding: 1rem;
}

.creation-console {
  display: flex;
  min-height: 0;
  max-height: min(78vh, 860px);
  flex-direction: column;
  overflow: hidden;
  border-radius: 1.1rem;
  background: color-mix(in srgb, var(--ssxz-surface-raised) 92%, transparent);
}

.console-scroll {
  flex: 1;
  min-height: 0;
  scrollbar-gutter: stable;
  overflow-y: auto;
  padding: 1rem;
  padding-bottom: 2rem;
}

.console-block + .console-block {
  margin-top: 1.1rem;
  border-top: 1px solid color-mix(in srgb, var(--ssxz-border) 72%, transparent);
  padding-top: 1.1rem;
}

.block-heading {
  display: flex;
  gap: 0.7rem;
  margin-bottom: 0.8rem;
}

.block-heading h3 {
  margin: 0;
  color: var(--ssxz-text-primary);
  font-size: 0.98rem;
}

.block-heading p {
  margin: 0.2rem 0 0;
  font-size: 0.82rem;
}

.step-dot {
  display: grid;
  width: 1.7rem;
  height: 1.7rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--ssxz-action-soft), color-mix(in srgb, var(--ssxz-action) 12%, transparent));
  color: var(--ssxz-action);
  font-size: 0.8rem;
  font-weight: 800;
}

.goal-grid,
.format-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.55rem;
}

.choice-card,
.format-card,
.asset-drop,
.sub-panel,
.advanced-panel,
.canvas-sheet,
.prompt-preview,
.recent-empty,
.recent-card {
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-subtle);
}

.choice-card,
.format-card {
  position: relative;
  min-height: 4.4rem;
  padding: 0.72rem;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.choice-card span,
.format-card b,
.asset-title,
.canvas-summary h3,
.canvas-foot b,
.recent-heading h2,
.recent-card-body span {
  color: var(--ssxz-text-primary);
}

.choice-card span,
.choice-card small,
.format-card b,
.format-card span {
  display: block;
}

.choice-card small,
.format-card span {
  margin-top: 0.18rem;
}

.choice-card small,
.format-card span,
.sub-panel p,
.recent-card-body small {
  color: var(--ssxz-text-muted);
}

.choice-card.selected,
.format-card.selected,
.style-chip.selected {
  border-color: var(--ssxz-border-strong);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--ssxz-action-soft) 80%, var(--ssxz-surface)), var(--ssxz-active));
  box-shadow: 0 0 0 3px var(--ssxz-focus-ring), var(--ssxz-shadow);
}

.choice-card:hover,
.format-card:hover,
.style-chip:hover {
  border-color: var(--ssxz-border-strong);
  transform: translateY(-1px);
}

.choice-card.selected::after,
.format-card.selected::after {
  content: "✓";
  position: absolute;
  top: 0.55rem;
  right: 0.55rem;
  display: grid;
  width: 1.35rem;
  height: 1.35rem;
  place-items: center;
  border-radius: 999px;
  background: var(--ssxz-action);
  color: var(--ssxz-action-text);
  font-size: 0.72rem;
  font-weight: 900;
}

.inline-field {
  display: grid;
  gap: 0.45rem;
  margin-top: 0.8rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 700;
}

.control-input,
.control-textarea {
  width: 100%;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-input);
  color: var(--ssxz-text-primary);
  outline: none;
}

.control-input {
  min-height: 2.65rem;
  padding: 0 0.85rem;
}

.control-textarea {
  min-height: 7rem;
  resize: vertical;
  padding: 0.8rem 0.85rem;
}

.control-input:focus,
.control-textarea:focus {
  border-color: var(--ssxz-focus);
  box-shadow: 0 0 0 3px var(--ssxz-focus-ring);
}

.sub-panel,
.advanced-body {
  margin-top: 0.75rem;
  padding: 0.75rem;
}

.ratio-input-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: end;
  gap: 0.55rem;
}

.ratio-separator {
  padding-bottom: 0.75rem;
  color: var(--ssxz-text-muted);
  font-weight: 800;
}

.sub-panel p.invalid {
  color: var(--ssxz-danger);
}

.count-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.85rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 1rem;
  background: var(--ssxz-surface-subtle);
  padding: 0.75rem;
}

.count-panel span,
.count-panel small {
  display: block;
}

.count-panel span {
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 800;
}

.count-panel small {
  margin-top: 0.2rem;
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
}

.count-list {
  display: flex;
  flex: 0 0 auto;
  gap: 0.42rem;
}

.count-chip {
  min-width: 3.1rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface);
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  font-weight: 800;
  padding: 0.4rem 0.58rem;
}

.count-chip.selected {
  border-color: var(--ssxz-border-strong);
  background: var(--ssxz-active);
  color: var(--ssxz-action);
}

.asset-drop {
  display: flex;
  min-height: 10.5rem;
  overflow: hidden;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  gap: 0.9rem;
  padding: 0.8rem;
  text-align: center;
  border-style: dashed;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--ssxz-action-soft) 46%, var(--ssxz-surface-subtle)), var(--ssxz-surface-subtle));
}

.asset-drop:hover {
  border-color: var(--ssxz-border-strong);
  background: color-mix(in srgb, var(--ssxz-action-soft) 68%, var(--ssxz-surface-subtle));
}

.asset-drop.dragging {
  border-color: var(--ssxz-action);
  background: color-mix(in srgb, var(--ssxz-action-soft) 78%, var(--ssxz-surface-subtle));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--ssxz-action) 18%, transparent);
}

.asset-drop.filled {
  justify-content: flex-start;
  text-align: left;
}

.asset-thumb {
  display: grid;
  width: 140px;
  height: 122px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 0.85rem;
  overflow: hidden;
  background: var(--ssxz-surface-subtle);
}

.asset-thumb > img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.asset-thumb.is-error {
  gap: 0.28rem;
  border: 1px dashed var(--ssxz-border);
  color: var(--ssxz-danger);
  font-size: 0.74rem;
  font-weight: 800;
}

.asset-copy {
  display: grid;
  min-width: 0;
  gap: 0.25rem;
}

.asset-title {
  display: block;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.reference-upload-note {
  margin-top: 0.5rem;
  font-size: 0.78rem;
  line-height: 1.5;
}

.reference-upload-note.error {
  color: var(--ssxz-danger);
}

.style-area {
  display: grid;
  gap: 0.55rem;
  margin-top: 0.8rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 700;
}

.style-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.style-chip {
  padding: 0.44rem 0.68rem;
  font-weight: 700;
}

.advanced-panel {
  margin-top: 0.9rem;
  padding: 0.85rem;
}

.advanced-panel summary,
.prompt-preview summary {
  cursor: pointer;
  color: var(--ssxz-text-secondary);
  font-weight: 700;
}

.console-action {
  flex: 0 0 auto;
  border-top: 1px solid var(--ssxz-border);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--ssxz-surface-elevated) 88%, transparent), var(--ssxz-surface-elevated));
  padding: 0.9rem 1rem 1rem;
}

.safety-note,
.error-note {
  margin: 0.65rem 0;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.9rem;
  background: var(--ssxz-surface-subtle);
  padding: 0.7rem 0.8rem;
  font-size: 0.78rem;
  line-height: 1.55;
}

.error-note {
  border-color: var(--ssxz-danger);
  color: var(--ssxz-danger);
}

.generate-button {
  display: inline-flex;
  width: 100%;
  height: 2.8rem;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border-radius: 0.95rem;
  background: linear-gradient(135deg, var(--ssxz-action), color-mix(in srgb, var(--ssxz-action) 76%, #0f766e));
  color: var(--ssxz-action-text);
  font-weight: 800;
}

.generate-button:not(:disabled):hover {
  transform: translateY(-1px);
  box-shadow: var(--ssxz-shadow-lg);
}

.generate-button:disabled {
  cursor: not-allowed;
  background: var(--ssxz-disabled);
  color: var(--ssxz-text-muted);
}

.canvas-panel {
  display: flex;
  min-height: 0;
  flex-direction: column;
  border-radius: 1.1rem;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--ssxz-surface-raised) 94%, transparent), var(--ssxz-surface));
  padding: 1rem;
}

.canvas-summary,
.canvas-foot,
.recent-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.canvas-summary h3,
.canvas-summary p,
.recent-heading h2,
.recent-heading p {
  margin: 0;
}

.canvas-tags {
  justify-content: flex-end;
}

.canvas-stage {
  display: grid;
  flex: 1;
  min-height: 28rem;
  place-items: center;
  padding: 1.3rem;
}

.canvas-sheet {
  display: grid;
  width: min(100%, 760px);
  max-height: 62vh;
  place-items: center;
  overflow: hidden;
  background:
    linear-gradient(45deg, color-mix(in srgb, var(--ssxz-border) 28%, transparent) 25%, transparent 25% 75%, color-mix(in srgb, var(--ssxz-border) 28%, transparent) 75%),
    linear-gradient(45deg, color-mix(in srgb, var(--ssxz-border) 18%, transparent) 25%, transparent 25% 75%, color-mix(in srgb, var(--ssxz-border) 18%, transparent) 75%),
    radial-gradient(circle at 50% 20%, var(--ssxz-glow-subtle), transparent 42%),
    var(--ssxz-canvas);
  background-position: 0 0, 12px 12px, center, center;
  background-size: 24px 24px, 24px 24px, auto, auto;
}

.canvas-sheet img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.canvas-empty {
  display: grid;
  max-width: 24rem;
  place-items: center;
  gap: 0.65rem;
  padding: 2rem;
  text-align: center;
}

.canvas-empty-badge {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
  font-weight: 800;
  padding: 0.26rem 0.58rem;
}

.canvas-empty.failed {
  color: var(--ssxz-danger);
}

.canvas-empty h3 {
  margin: 0;
  color: var(--ssxz-text-primary);
}

.canvas-empty p {
  margin: 0;
  color: var(--ssxz-text-secondary);
  line-height: 1.65;
}

.canvas-empty-guide {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.45rem;
  margin-top: 0.35rem;
}

.canvas-empty-guide span {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ssxz-surface-subtle) 82%, transparent);
  color: var(--ssxz-text-muted);
  font-size: 0.74rem;
  font-weight: 750;
  padding: 0.28rem 0.55rem;
}

.thumbnail-strip {
  display: flex;
  gap: 0.6rem;
  overflow-x: auto;
  padding: 0 1rem 0.85rem;
}

.thumbnail-button {
  width: 4.6rem;
  height: 4.6rem;
  flex: 0 0 auto;
  overflow: hidden;
  border: 2px solid transparent;
  border-radius: 0.9rem;
  background: var(--ssxz-surface-subtle);
}

.thumbnail-button.selected {
  border-color: var(--ssxz-action);
}

.thumbnail-button img,
.recent-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.canvas-foot {
  border-top: 1px solid var(--ssxz-border);
  padding-top: 0.85rem;
}

.canvas-foot div {
  display: grid;
  gap: 0.18rem;
}

.result-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.55rem;
  margin-top: 0.85rem;
}

.secondary-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-primary);
  padding: 0.62rem 0.9rem;
  font-weight: 700;
}

.secondary-button:not(:disabled):hover {
  border-color: var(--ssxz-border-strong);
  background: color-mix(in srgb, var(--ssxz-action-soft) 58%, var(--ssxz-surface-subtle));
}

.secondary-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.prompt-preview {
  margin-top: 1rem;
  padding: 0.85rem;
}

.prompt-preview pre {
  margin: 0.75rem 0 0;
  max-height: 14rem;
  overflow: auto;
  white-space: pre-wrap;
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  line-height: 1.55;
}

.recent-works {
  margin-top: 1rem;
  border-radius: 1.25rem;
  padding: 1rem;
}

.recent-empty {
  display: grid;
  min-height: 7rem;
  place-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 1rem;
  text-align: center;
}

.recent-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.9rem;
  margin-top: 1rem;
}

.recent-card {
  overflow: hidden;
}

.recent-thumb {
  aspect-ratio: 1 / 1;
  background: var(--ssxz-canvas);
}

.recent-thumb-button {
  position: relative;
  display: block;
  width: 100%;
  overflow: hidden;
  border: 0;
  border-radius: 0;
  padding: 0;
  color: inherit;
  cursor: zoom-in;
  font: inherit;
}

.recent-thumb-hint {
  position: absolute;
  right: 0.7rem;
  bottom: 0.7rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  padding: 0.28rem 0.55rem;
  background: rgb(0 0 0 / 0.58);
  color: white;
  font-size: 0.74rem;
  font-weight: 800;
  opacity: 0;
  transform: translateY(0.25rem);
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.recent-thumb-button:hover .recent-thumb-hint,
.recent-thumb-button:focus-visible .recent-thumb-hint {
  opacity: 1;
  transform: translateY(0);
}

.recent-thumb-button:focus-visible {
  outline: 3px solid var(--ssxz-focus);
  outline-offset: -3px;
}

.recent-card-body {
  display: grid;
  gap: 0.65rem;
  padding: 0.75rem;
}

.recent-card-body div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.recent-card-body .recent-card-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
}

.recent-preview-backdrop {
  position: fixed;
  z-index: 80;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 1.5rem;
  background: rgb(0 0 0 / 0.72);
}

.recent-preview-dialog {
  display: grid;
  width: min(92vw, 72rem);
  max-height: min(92vh, 58rem);
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: 1.15rem;
  background: var(--ssxz-surface);
  box-shadow: var(--ssxz-shadow);
}

.recent-preview-header,
.recent-preview-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.9rem 1rem;
}

.recent-preview-header {
  border-bottom: 1px solid var(--ssxz-border);
}

.recent-preview-header p,
.recent-preview-footer span {
  margin: 0;
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
}

.recent-preview-header h2 {
  margin: 0.1rem 0 0;
  color: var(--ssxz-text-primary);
  font-size: 1.1rem;
}

.recent-preview-close {
  display: grid;
  width: 2.35rem;
  height: 2.35rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-surface-subtle);
  color: var(--ssxz-text-primary);
  cursor: pointer;
  font-size: 1.45rem;
  line-height: 1;
}

.recent-preview-close:hover,
.recent-preview-close:focus-visible {
  border-color: var(--ssxz-border-strong);
  background: color-mix(in srgb, var(--ssxz-action-soft) 74%, var(--ssxz-surface-subtle));
}

.recent-preview-stage {
  display: grid;
  min-height: 18rem;
  max-height: 72vh;
  place-items: center;
  overflow: auto;
  background: var(--ssxz-canvas);
  padding: 1rem;
}

.recent-preview-stage img {
  display: block;
  max-width: 100%;
  max-height: 68vh;
  border-radius: 0.85rem;
  object-fit: contain;
}

.recent-preview-footer {
  flex-wrap: wrap;
  border-top: 1px solid var(--ssxz-border);
}

.recent-preview-footer b {
  display: block;
  margin-top: 0.2rem;
  color: var(--ssxz-text-primary);
}

.recent-preview-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  margin-left: auto;
}

.hidden {
  display: none;
}

@media (max-width: 1180px) {
  .image-hero {
    grid-template-columns: 1fr;
  }

  .image-workbench {
    grid-template-columns: 1fr;
  }

  .creation-console {
    max-height: none;
  }

  .recent-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .recent-grid {
    grid-template-columns: 1fr;
  }

  .recent-preview-backdrop {
    align-items: end;
    padding: 0.75rem;
  }

  .recent-preview-header,
  .recent-preview-footer {
    align-items: flex-start;
  }

  .image-hero,
  .image-workbench,
  .canvas-panel,
  .recent-works {
    padding: 0.8rem;
  }

  .goal-grid,
  .format-grid {
    gap: 0.48rem;
  }

  .count-panel {
    align-items: flex-start;
    flex-direction: column;
  }

  .choice-card,
  .format-card {
    min-height: 3.7rem;
    padding: 0.62rem;
  }

  .choice-card small,
  .format-card span {
    font-size: 0.72rem;
  }

  .asset-drop {
    min-height: 8.6rem;
  }

  .canvas-summary,
  .canvas-foot,
  .recent-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .canvas-stage {
    min-height: 20rem;
    padding: 0.8rem 0;
  }
}
</style>
