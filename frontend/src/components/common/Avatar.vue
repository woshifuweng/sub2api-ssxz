<template>
  <div
    class="ssxz-avatar"
    :class="{ 'ssxz-avatar--fallback': !imageVisible }"
    :style="avatarStyle"
    role="img"
    :aria-label="ariaLabel"
  >
    <img
      v-if="src && imageVisible"
      :src="src"
      alt=""
      class="ssxz-avatar__image"
      @error="imageVisible = false"
    />
    <span v-else aria-hidden="true">{{ initials }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  src?: string | null
  name?: string | null
  size?: number | string
  label?: string
}>(), {
  src: null,
  name: '',
  size: 72,
  label: 'Avatar'
})

const imageVisible = ref(Boolean(props.src))

watch(() => props.src, (value) => {
  imageVisible.value = Boolean(value)
})

const initials = computed(() => {
  const value = props.name?.trim() || 'U'
  return value.slice(0, 1).toUpperCase()
})

const avatarStyle = computed(() => ({
  '--ssxz-avatar-size': typeof props.size === 'number' ? `${props.size}px` : props.size
}))

const ariaLabel = computed(() => props.name?.trim() || props.label)
</script>

<style scoped>
.ssxz-avatar {
  display: grid;
  width: var(--ssxz-avatar-size);
  height: var(--ssxz-avatar-size);
  flex: 0 0 auto;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--ssxz-border-strong, #aab4c0);
  border-radius: 999px;
  background: var(--ssxz-surface-raised, #edf1f5);
  color: var(--ssxz-text-primary, #1d2530);
  font-size: calc(var(--ssxz-avatar-size) * 0.35);
  font-weight: 800;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--ssxz-border) 68%, transparent), var(--ssxz-shadow-sm);
}

.ssxz-avatar--fallback {
  background: linear-gradient(145deg, #dfe5eb 0%, #aeb9c5 100%);
  color: #344150;
}

:global(.dark) .ssxz-avatar--fallback {
  background: linear-gradient(145deg, #6e7b89 0%, #303b49 100%);
  color: #f4f7fa;
}

.ssxz-avatar__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
