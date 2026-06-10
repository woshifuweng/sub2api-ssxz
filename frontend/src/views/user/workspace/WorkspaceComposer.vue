<template>
  <form class="composer-card" @submit.prevent="handleSubmit">
    <div v-if="assetPreviews.length" class="attachment-preview-list" data-testid="workspace-asset-previews">
      <article v-for="asset in assetPreviews" :key="asset.id" class="attachment-preview-card">
        <img :src="asset.url" :alt="asset.name" />
        <span>{{ asset.name }}</span>
        <small>{{ asset.sizeLabel }}</small>
        <button
          type="button"
          class="remove-preview"
          :aria-label="`移除 ${asset.name}`"
          @click="emit('remove-asset', asset.id)"
        >
          <Icon name="x" size="xs" />
        </button>
      </article>
    </div>

    <textarea
      :value="modelValue"
      rows="2"
      :placeholder="placeholder"
      @input="handleInput"
      @keydown="handleKeydown"
    />

    <WorkspaceAssetPanel
      v-if="assetPanelOpen"
      :rejected-files="rejectedFiles"
      @files="emit('files', $event)"
    />

    <div class="composer-tool-row">
      <WorkspaceModelPicker
        :models="models"
        :selected-model="selectedModel"
        @update:selected-model="emit('update:selectedModel', $event)"
      />

      <button
        type="button"
        class="toolbar-tool"
        data-testid="workspace-add-content"
        aria-controls="workspace-asset-panel"
        :aria-expanded="assetPanelOpen"
        title="添加内容"
        @click="assetPanelOpen = !assetPanelOpen"
      >
        <Icon name="plus" size="sm" />
        <span>添加</span>
      </button>

      <button
        v-for="tool in informationalTools"
        :key="tool.label"
        type="button"
        class="toolbar-tool is-informational"
        :title="tool.description"
        :aria-label="`${tool.label}，${tool.description}`"
      >
        <Icon :name="tool.icon" size="sm" />
        <span>{{ tool.label }}</span>
      </button>

      <button
        class="send-button"
        type="submit"
        data-testid="workspace-send"
        :disabled="!canSubmit"
        :title="backendEnabled ? '发送' : '统一工作台后端正在接入，暂不可发送'"
        :aria-label="backendEnabled ? '发送' : '统一工作台后端正在接入，暂不可发送'"
      >
        <Icon v-if="sending" name="sync" size="sm" />
        <Icon v-else name="arrowUp" size="sm" />
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { ChatModelOption } from '@/composables/useUserCapabilities'
import WorkspaceAssetPanel from './WorkspaceAssetPanel.vue'
import WorkspaceModelPicker from './WorkspaceModelPicker.vue'
import type { RejectedWorkspaceFile, WorkspaceAssetPreview } from './useWorkspaceAssets'
import type { WorkspaceIntent } from './useWorkspaceConversation'

type IconName = InstanceType<typeof Icon>['$props']['name']

const props = defineProps<{
  modelValue: string
  selectedModel: string
  models: ChatModelOption[]
  intent: WorkspaceIntent
  backendEnabled?: boolean
  sending: boolean
  assetPreviews: WorkspaceAssetPreview[]
  rejectedFiles: RejectedWorkspaceFile[]
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'update:selectedModel', value: string): void
  (event: 'files', files: File[]): void
  (event: 'remove-asset', id: string): void
  (event: 'submit'): void
}>()

const assetPanelOpen = ref(false)

const placeholders: Record<WorkspaceIntent, string> = {
  home: '直接提问、上传图片或开始一个新任务',
  chat: '输入问题，继续这段对话',
  image: '描述你想生成或修改的图片，也可以先上传参考图'
}

const informationalTools: Array<{ label: string; description: string; icon: IconName }> = [
  { label: '联网', description: '联网检索会在后续版本接入，当前不会假装开启。', icon: 'globe' },
  { label: '记忆', description: '长期记忆会在隐私策略完成后接入。', icon: 'brain' },
  { label: '工具箱', description: '文档、表格、代码工具正在收敛到同一个输入区。', icon: 'grid' }
]

const placeholder = computed(() => placeholders[props.intent])
const canSubmit = computed(() =>
  props.backendEnabled === true &&
  !props.sending &&
  props.intent !== 'image' &&
  Boolean(props.selectedModel) &&
  props.assetPreviews.length === 0 &&
  props.modelValue.trim().length > 0
)

function handleInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  handleSubmit()
}

function handleSubmit() {
  if (canSubmit.value) emit('submit')
}
</script>
