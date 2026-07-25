<template>
  <span
    class="brand-logo"
    :class="[
      `brand-logo--${variant}`,
      theme ? `brand-logo--theme-${theme}` : undefined
    ]"
    :style="brandStyle"
    role="img"
    :aria-label="label"
    data-testid="brand-logo"
  >
    <template v-if="variant === 'animated'">
      <img
        :src="animatedLogoUrl"
        :style="artworkStyle"
        alt=""
        class="brand-logo__artwork brand-logo__artwork--animated"
        width="120"
        height="194"
        aria-hidden="true"
      />
      <img
        :src="staticLogoUrl"
        :style="artworkStyle"
        alt=""
        class="brand-logo__artwork brand-logo__artwork--static"
        width="120"
        height="194"
        aria-hidden="true"
      />
    </template>
    <span v-else class="brand-logo__mark" aria-hidden="true" />
  </span>
</template>

<script setup lang="ts">
import { computed, type CSSProperties } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'mark' | 'animated'
  size?: string
  theme?: 'light' | 'dark'
  label?: string
}>(), {
  variant: 'mark',
  size: '2.875rem',
  theme: undefined,
  label: 'SSXZ 双S猫狗交缠 Logo'
})

const animatedLogoUrl = '/brand/ssxz-cat-dog-line-draw-safari-safe.svg'
const staticLogoUrl = '/brand/ssxz-cat-dog-static.svg'

const brandStyle = computed<CSSProperties>(() => ({
  '--brand-logo-size': props.size
}) as CSSProperties)

const artworkStyle = computed<CSSProperties>(() => (
  props.theme ? { colorScheme: props.theme } : {}
))
</script>

<style scoped>
.brand-logo {
  position: relative;
  display: inline-grid;
  width: var(--brand-logo-size);
  height: var(--brand-logo-size);
  flex: 0 0 auto;
  place-items: center;
  color: #17191d;
  isolation: isolate;
}

.brand-logo--theme-dark,
:global(.dark) .brand-logo:not(.brand-logo--theme-light) {
  color: #f7f8fa;
}

.brand-logo--theme-light {
  color: #17191d;
}

.brand-logo__mark {
  display: block;
  width: 100%;
  height: 100%;
  background: currentColor;
  mask: url('/brand/ssxz-cat-dog-static.svg') center / contain no-repeat;
  -webkit-mask: url('/brand/ssxz-cat-dog-static.svg') center / contain no-repeat;
}

.brand-logo__artwork {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.brand-logo__artwork--static {
  display: none;
}

@media (prefers-reduced-motion: reduce) {
  .brand-logo__artwork--animated {
    display: none;
  }

  .brand-logo__artwork--static {
    display: block;
  }
}
</style>
