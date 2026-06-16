<template>
  <div id="workspace-asset-panel" class="asset-panel" aria-label="上传能力面板">
    <label
      v-if="imageUploadAvailable"
      class="asset-option is-ready"
      title="上传图片"
    >
      <Icon name="upload" size="sm" />
      <span>
        <strong>图片</strong>
        <small>上传参考图</small>
      </span>
      <input
        class="asset-file-input"
        type="file"
        accept="image/*"
        multiple
        aria-label="上传图片"
        @change="handleFileChange"
      />
    </label>

    <button
      v-else
      type="button"
      class="asset-option is-unavailable"
      data-testid="workspace-upload-image-disabled"
      disabled
      aria-disabled="true"
      title="上传图片暂未接入"
    >
      <Icon name="upload" size="sm" />
      <span>
        <strong>图片</strong>
        <small>暂未接入</small>
      </span>
    </button>

    <button
      v-for="item in futureCapabilities"
      :key="item.label"
      type="button"
      class="asset-option is-unavailable"
      :data-testid="`workspace-asset-${item.key}`"
      disabled
      aria-disabled="true"
      :title="item.description"
    >
      <Icon :name="item.icon" size="sm" />
      <span>
        <strong>{{ item.label }}</strong>
        <small>暂未接入</small>
      </span>
    </button>

    <p class="asset-panel-note">
      当前仅开放文本对话，附件与工具能力会在后续接入时按后端能力逐项开放。
    </p>
    <p v-for="file in rejectedFiles" :key="file.name" class="asset-panel-error">
      {{ file.name }}：{{ file.reason }}
    </p>
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import type { RejectedWorkspaceFile } from './useWorkspaceAssets'

type IconName = InstanceType<typeof Icon>['$props']['name']

withDefaults(defineProps<{
  rejectedFiles: RejectedWorkspaceFile[]
  imageUploadAvailable?: boolean
}>(), {
  imageUploadAvailable: false
})

const emit = defineEmits<{
  (event: 'files', files: File[]): void
}>()

const futureCapabilities: Array<{
  key: 'document' | 'table' | 'code'
  label: string
  description: string
  icon: IconName
}> = [
  {
    key: 'document',
    label: '文档',
    description: '文档分析暂未接入。',
    icon: 'document'
  },
  {
    key: 'table',
    label: '表格',
    description: '表格分析暂未接入。',
    icon: 'chart'
  },
  {
    key: 'code',
    label: '代码',
    description: '代码工具暂未接入。',
    icon: 'terminal'
  }
]

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  emit('files', Array.from(input.files || []))
  input.value = ''
}
</script>
