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
        <div
          class="avatar-upload-zone"
          role="group"
          :aria-describedby="'avatar-upload-hint'"
          @dragover.prevent
          @drop.prevent="handleDrop"
        >
          <div class="avatar-upload-icon" aria-hidden="true">
            <Icon name="upload" size="lg" />
          </div>
          <div class="avatar-upload-copy">
            <strong>{{ selectedFileName ? t('profile.avatar.selectedFile', { name: selectedFileName }) : t('profile.avatar.uploadTitle') }}</strong>
            <p id="avatar-upload-hint">{{ t('profile.avatar.uploadHint') }}</p>
          </div>
          <input
            id="avatar-file"
            ref="fileInput"
            type="file"
            accept="image/jpeg,image/png,image/webp"
            class="avatar-file-input"
            @change="handleFileChange"
          />
          <button type="button" class="btn btn-secondary avatar-upload-button" @click="openFilePicker">
            <Icon name="upload" size="sm" />
            {{ t('profile.avatar.uploadButton') }}
          </button>
        </div>
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
import Icon from '@/components/icons/Icon.vue'

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
const selectedFileName = ref('')

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
  selectedFileName.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

function openFilePicker() {
  fileInput.value?.click()
}

function processFile(file?: File) {
  if (!file) return

  errorMessage.value = ''
  if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
    errorMessage.value = t('profile.avatar.invalidType')
    selectedFileName.value = ''
    if (fileInput.value) fileInput.value.value = ''
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    errorMessage.value = t('profile.avatar.fileTooLarge')
    selectedFileName.value = ''
    if (fileInput.value) fileInput.value.value = ''
    return
  }

  resetPreview()
  selectedFileName.value = file.name
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

function handleFileChange(event: Event) {
  processFile((event.target as HTMLInputElement).files?.[0])
}

function handleDrop(event: DragEvent) {
  processFile(event.dataTransfer?.files?.[0])
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
.avatar-upload-zone {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.8rem;
  padding: 0.9rem;
  border: 1px dashed var(--ssxz-border-strong);
  border-radius: 1rem;
  background: var(--ssxz-surface-muted);
  transition: border-color 160ms ease, background-color 160ms ease;
}
.avatar-upload-zone:focus-within,
.avatar-upload-zone:hover {
  border-color: var(--ssxz-action);
  background: var(--ssxz-surface-raised);
}
.avatar-upload-icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.8rem;
  color: var(--ssxz-text-secondary);
}
.avatar-upload-copy { min-width: 0; }
.avatar-upload-copy strong {
  display: block;
  overflow: hidden;
  color: var(--ssxz-text-primary);
  font-size: 0.85rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.avatar-upload-copy p {
  margin: 0.2rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.78rem;
  line-height: 1.45;
}
.avatar-file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
.avatar-upload-button {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  white-space: nowrap;
}
@media (max-width: 36rem) {
  .avatar-upload-zone { grid-template-columns: auto minmax(0, 1fr); }
  .avatar-upload-button { grid-column: 1 / -1; justify-content: center; }
}
</style>
