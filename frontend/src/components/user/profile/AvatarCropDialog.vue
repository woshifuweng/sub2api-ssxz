<template>
  <BaseDialog :show="show" :title="t('profile.avatar.dialogTitle')" width="normal" @close="emit('close')">
    <div class="avatar-crop-dialog">
      <div class="avatar-crop-stage" aria-live="polite">
        <img
          v-if="previewUrl"
          :src="previewUrl"
          :style="previewStyle"
          alt=""
          class="avatar-crop-preview"
        />
        <div v-else class="avatar-crop-empty">{{ t('profile.avatar.chooseHint') }}</div>
        <div class="avatar-crop-window" aria-hidden="true" />
      </div>

      <div class="avatar-crop-controls">
        <label class="input-label" for="avatar-file">{{ t('profile.avatar.fileLabel') }}</label>
        <input
          id="avatar-file"
          ref="fileInput"
          type="file"
          accept="image/jpeg,image/png,image/webp"
          class="input"
          @change="handleFileChange"
        />
        <label v-if="previewUrl" class="input-label" for="avatar-zoom">
          {{ t('profile.avatar.zoomLabel') }}
        </label>
        <input
          v-if="previewUrl"
          id="avatar-zoom"
          v-model.number="zoom"
          type="range"
          min="1"
          max="3"
          step="0.05"
          class="avatar-crop-range"
        />
        <p class="input-hint">{{ t('profile.avatar.cropHint') }}</p>
        <p v-if="errorMessage" class="input-error-text" role="alert">{{ errorMessage }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="!previewUrl || saving" @click="saveAvatar">
          {{ saving ? t('profile.avatar.saving') : t('profile.avatar.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

withDefaults(defineProps<{
  show: boolean
  saving?: boolean
}>(), {
  saving: false
})

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'save', dataUrl: string): void
}>()

const { t } = useI18n()
const fileInput = ref<HTMLInputElement | null>(null)
const previewUrl = ref('')
const sourceImage = ref<HTMLImageElement | null>(null)
const zoom = ref(1)
const errorMessage = ref('')

const previewStyle = computed(() => ({
  width: `${zoom.value * 100}%`,
  height: `${zoom.value * 100}%`
}))

function resetPreview() {
  if (previewUrl.value.startsWith('blob:')) URL.revokeObjectURL(previewUrl.value)
  previewUrl.value = ''
  sourceImage.value = null
  zoom.value = 1
  errorMessage.value = ''
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  errorMessage.value = ''
  if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
    errorMessage.value = t('profile.avatar.invalidType')
    input.value = ''
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    errorMessage.value = t('profile.avatar.fileTooLarge')
    input.value = ''
    return
  }

  resetPreview()
  const url = URL.createObjectURL(file)
  previewUrl.value = url
  const image = new Image()
  image.onload = () => { sourceImage.value = image }
  image.onerror = () => {
    errorMessage.value = t('profile.avatar.invalidType')
    resetPreview()
  }
  image.src = url
}

function createCroppedDataUrl(): string | null {
  const image = sourceImage.value
  if (!image) return null

  const side = Math.min(image.naturalWidth, image.naturalHeight) / zoom.value
  const sourceX = (image.naturalWidth - side) / 2
  const sourceY = (image.naturalHeight - side) / 2
  const canvas = document.createElement('canvas')
  const outputSide = 256
  canvas.width = outputSide
  canvas.height = outputSide
  const context = canvas.getContext('2d')
  if (!context) return null
  context.drawImage(image, sourceX, sourceY, side, side, 0, 0, outputSide, outputSide)

  for (const quality of [0.86, 0.72, 0.58]) {
    const dataUrl = canvas.toDataURL('image/webp', quality)
    if (dataUrl.length * 0.75 <= 100 * 1024) return dataUrl
  }
  return null
}

function saveAvatar() {
  const dataUrl = createCroppedDataUrl()
  if (!dataUrl) {
    errorMessage.value = t('profile.avatar.outputTooLarge')
    return
  }
  emit('save', dataUrl)
}

onUnmounted(resetPreview)
</script>

<style scoped>
.avatar-crop-dialog { display: grid; gap: 1.25rem; }
.avatar-crop-stage {
  position: relative;
  display: grid;
  width: min(100%, 20rem);
  aspect-ratio: 1;
  margin-inline: auto;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: 1rem;
  background: var(--ssxz-surface-sunken);
}
.avatar-crop-preview { max-width: none; object-fit: cover; }
.avatar-crop-window {
  position: absolute;
  width: 72%;
  aspect-ratio: 1;
  border: 2px solid var(--ssxz-action);
  border-radius: 999px;
  box-shadow: 0 0 0 999px rgb(0 0 0 / 26%);
  pointer-events: none;
}
.avatar-crop-empty { padding: 1rem; color: var(--ssxz-text-secondary); text-align: center; }
.avatar-crop-controls { display: grid; gap: 0.55rem; }
.avatar-crop-range { width: 100%; accent-color: var(--ssxz-action); }
</style>
