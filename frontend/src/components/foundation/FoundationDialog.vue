<template>
  <dialog
    ref="dialogRef"
    class="f0-dialog"
    :aria-labelledby="titleId"
    :aria-describedby="description ? descriptionId : undefined"
    @cancel.prevent="requestClose"
    @close="handleNativeClose"
    @click="handleBackdropClick"
  >
    <header class="f0-dialog-header">
      <div>
        <h2 :id="titleId" class="f0-dialog-title">{{ title }}</h2>
        <p v-if="description" :id="descriptionId" class="f0-dialog-description">
          {{ description }}
        </p>
      </div>
      <button class="f0-dialog-close" type="button" aria-label="关闭对话框" @click="requestClose">
        <X aria-hidden="true" />
      </button>
    </header>
    <div class="f0-dialog-content">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="f0-dialog-footer">
      <slot name="footer" />
    </footer>
  </dialog>
</template>

<script setup lang="ts">
import { X } from '@lucide/vue'
import { nextTick, onMounted, ref, watch } from 'vue'

let dialogCounter = 0
const dialogId = ++dialogCounter
const titleId = `f0-dialog-title-${dialogId}`
const descriptionId = `f0-dialog-description-${dialogId}`
const dialogRef = ref<HTMLDialogElement | null>(null)

const props = defineProps<{
  open: boolean
  title: string
  description?: string
}>()

const emit = defineEmits<{
  (event: 'update:open', value: boolean): void
}>()

const syncOpenState = async (open: boolean) => {
  await nextTick()
  const dialog = dialogRef.value
  if (!dialog) return

  if (open && !dialog.open) {
    dialog.showModal()
  } else if (!open && dialog.open) {
    dialog.close()
  }
}

const requestClose = () => emit('update:open', false)

const handleNativeClose = () => {
  if (props.open) emit('update:open', false)
}

const handleBackdropClick = (event: MouseEvent) => {
  if (event.target === dialogRef.value) requestClose()
}

watch(() => props.open, syncOpenState)
onMounted(() => syncOpenState(props.open))
</script>
