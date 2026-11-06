<template>
  <LobeProviderIcon
    v-if="providerAlias"
    :provider="providerAlias"
    :size="iconPixelSize"
    type="color"
    aria-hidden="true"
    :class="sizeClass"
  />
  <Antigravity.Color
    v-else-if="platform === 'antigravity'"
    :size="iconPixelSize"
    aria-hidden="true"
    :class="sizeClass"
  />
  <LobeModelIcon
    v-else-if="modelAlias"
    :model="modelAlias"
    :size="iconPixelSize"
    type="color"
    aria-hidden="true"
    :class="sizeClass"
  />
  <svg
    v-else-if="platform === 'composite'"
    :class="sizeClass"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2"
    aria-hidden="true"
  >
    <circle cx="6" cy="12" r="3" />
    <circle cx="18" cy="6" r="3" />
    <circle cx="18" cy="18" r="3" />
    <path stroke-linecap="round" stroke-linejoin="round" d="M8.7 10.7 15.3 7.3M8.7 13.3l6.6 3.4" />
  </svg>
  <span
    v-else
    class="inline-flex items-center justify-center rounded-md bg-slate-500/15 font-semibold text-slate-500 dark:text-slate-300"
    :style="{ width: `${iconPixelSize}px`, height: `${iconPixelSize}px`, fontSize: `${Math.round(iconPixelSize * 0.5)}px` }"
    aria-hidden="true"
  >
    {{ fallbackText }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Antigravity, ModelIcon as LobeModelIcon, ProviderIcon as LobeProviderIcon } from '@lobehub/icons-vue'
import type { GroupPlatform } from '@/types'

interface Props {
  platform?: GroupPlatform | string
  size?: 'xs' | 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm'
})

const PROVIDER_ALIASES: Record<string, string> = {
  anthropic: 'anthropic',
  openai: 'openai',
  gemini: 'gemini',
  grok: 'xai',
  kimi: 'moonshot',
  zhipu: 'zhipu',
  deepseek: 'deepseek',
  minimax: 'minimax',
  opencode_go: 'opencodego',
  typesafe: 'typesafe'
}

const MODEL_ALIASES: Record<string, string> = {
  sora: 'sora',
  kiro: 'kiro'
}

const sizeClass = computed(() => {
  const sizes = {
    xs: 'h-3 w-3',
    sm: 'h-3.5 w-3.5',
    md: 'h-4 w-4',
    lg: 'h-5 w-5'
  }
  return `${sizes[props.size]} flex-shrink-0`
})

const iconPixelSize = computed(() => {
  const sizes = { xs: 12, sm: 14, md: 16, lg: 20 }
  return sizes[props.size]
})

const providerAlias = computed(() => {
  const platform = props.platform
  return platform ? PROVIDER_ALIASES[platform] ?? '' : ''
})

const modelAlias = computed(() => {
  const platform = props.platform
  return platform ? MODEL_ALIASES[platform] ?? '' : ''
})

const fallbackText = computed(() => (props.platform || '?').charAt(0).toUpperCase())
</script>
