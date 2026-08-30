<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { cn } from '@/lib/utils'

type ColorVariant = 'default' | 'primary' | 'success' | 'error' | 'gold' | 'bronze'

interface Props {
  variant?: ColorVariant
  class?: string
  disabled?: boolean
  block?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
})

const isPressed = ref(false)
const isHovered = ref(false)
const isTouchDevice = ref(false)

onMounted(() => {
  isTouchDevice.value = 'ontouchstart' in window || navigator.maxTouchPoints > 0
})

const colorMap: Record<ColorVariant, { outer: string; inner: string; button: string; textColor: string; textShadow: string }> = {
  default: {
    outer: 'bg-gradient-to-b from-[#000] to-[#A0A0A0]',
    inner: 'bg-gradient-to-b from-[#FAFAFA] via-[#3E3E3E] to-[#E5E5E5]',
    button: 'bg-gradient-to-b from-[#B9B9B9] to-[#969696]',
    textColor: 'text-white',
    textShadow: '[text-shadow:_0_-1px_0_rgb(80_80_80_/_100%)]',
  },
  primary: {
    outer: 'bg-gradient-to-b from-[#000] to-[#A0A0A0]',
    inner: 'bg-gradient-to-b from-primary via-secondary to-muted',
    button: 'bg-gradient-to-b from-primary to-primary/40',
    textColor: 'text-white',
    textShadow: '[text-shadow:_0_-1px_0_rgb(30_58_138_/_100%)]',
  },
  success: {
    outer: 'bg-gradient-to-b from-[#005A43] to-[#7CCB9B]',
    inner: 'bg-gradient-to-b from-[#E5F8F0] via-[#00352F] to-[#D1F0E6]',
    button: 'bg-gradient-to-b from-[#9ADBC8] to-[#3E8F7C]',
    textColor: 'text-[#FFF7F0]',
    textShadow: '[text-shadow:_0_-1px_0_rgb(6_78_59_/_100%)]',
  },
  error: {
    outer: 'bg-gradient-to-b from-[#5A0000] to-[#FFAEB0]',
    inner: 'bg-gradient-to-b from-[#FFDEDE] via-[#680002] to-[#FFE9E9]',
    button: 'bg-gradient-to-b from-[#F08D8F] to-[#A45253]',
    textColor: 'text-[#FFF7F0]',
    textShadow: '[text-shadow:_0_-1px_0_rgb(146_64_14_/_100%)]',
  },
  gold: {
    outer: 'bg-gradient-to-b from-[#917100] to-[#EAD98F]',
    inner: 'bg-gradient-to-b from-[#FFFDDD] via-[#856807] to-[#FFF1B3]',
    button: 'bg-gradient-to-b from-[#FFEBA1] to-[#9B873F]',
    textColor: 'text-[#FFFDE5]',
    textShadow: '[text-shadow:_0_-1px_0_rgb(178_140_2_/_100%)]',
  },
  bronze: {
    outer: 'bg-gradient-to-b from-[#864813] to-[#E9B486]',
    inner: 'bg-gradient-to-b from-[#EDC5A1] via-[#5F2D01] to-[#FFDEC1]',
    button: 'bg-gradient-to-b from-[#FFE3C9] to-[#A36F3D]',
    textColor: 'text-[#FFF7F0]',
    textShadow: '[text-shadow:_0_-1px_0_rgb(124_45_18_/_100%)]',
  },
}

const colors = computed(() => colorMap[props.variant ?? 'default'])
const ease = 'all 250ms cubic-bezier(0.1, 0.4, 0.2, 1)'

const wrapperStyle = computed(() => ({
  transform: isPressed.value ? 'translateY(2.5px) scale(0.99)' : 'translateY(0) scale(1)',
  boxShadow: isPressed.value
    ? '0 1px 2px rgba(0,0,0,0.15)'
    : isHovered.value && !isTouchDevice.value
      ? '0 4px 12px rgba(0,0,0,0.12)'
      : '0 3px 8px rgba(0,0,0,0.08)',
  transition: ease,
  transformOrigin: 'center center',
}))

const innerStyle = computed(() => ({
  transition: ease,
  transformOrigin: 'center center',
  filter: isHovered.value && !isPressed.value && !isTouchDevice.value ? 'brightness(1.05)' : 'none',
}))

const buttonStyle = computed(() => ({
  transform: isPressed.value ? 'scale(0.97)' : 'scale(1)',
  transition: ease,
  transformOrigin: 'center center',
  filter: isHovered.value && !isPressed.value && !isTouchDevice.value ? 'brightness(1.02)' : 'none',
}))
</script>

<template>
  <div
    :class="cn('relative transform-gpu rounded-md p-[1.25px] will-change-transform', props.block ? 'flex w-full' : 'inline-flex', colors.outer)"
    :style="wrapperStyle"
  >
    <div
      :class="cn('absolute inset-[1px] transform-gpu rounded-lg will-change-transform', colors.inner)"
      :style="innerStyle"
    />
    <button
      :class="cn(
        'relative z-10 m-[1px] rounded-md inline-flex h-11 transform-gpu cursor-pointer items-center justify-center overflow-hidden px-6 py-2 text-sm leading-none font-semibold will-change-transform outline-none',
        props.block ? 'w-full' : '',
        colors.button,
        colors.textColor,
        colors.textShadow,
        props.class,
      )"
      :style="buttonStyle"
      :disabled="props.disabled"
      v-bind="$attrs"
      @mousedown="isPressed = true"
      @mouseup="isPressed = false"
      @mouseleave="() => { isPressed = false; isHovered = false }"
      @mouseenter="() => { if (!isTouchDevice) isHovered = true }"
      @touchstart.passive="isPressed = true"
      @touchend.passive="isPressed = false"
      @touchcancel.passive="isPressed = false"
    >
      <!-- Shine overlay on press -->
      <div
        :class="['pointer-events-none absolute inset-0 z-20 overflow-hidden rounded-md transition-opacity duration-300', isPressed ? 'opacity-20' : 'opacity-0']"
      >
        <div class="absolute inset-0 bg-gradient-to-r from-transparent via-neutral-100 to-transparent" />
      </div>
      <slot />
      <!-- Hover top-edge glow -->
      <div
        v-if="isHovered && !isPressed && !isTouchDevice"
        class="pointer-events-none absolute inset-0 rounded-lg bg-gradient-to-t from-transparent to-white/5"
      />
    </button>
  </div>
</template>
