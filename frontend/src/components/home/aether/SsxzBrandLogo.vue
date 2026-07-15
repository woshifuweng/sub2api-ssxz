<template>
  <div class="ssxz-brand-logo" role="img" aria-label="SSXZ 双S猫狗交缠 Logo">
    <img
      :src="ssxzAnimatedLogo"
      :style="{ colorScheme: theme }"
      alt=""
      class="ssxz-brand-logo__artwork ssxz-brand-logo__artwork--animated"
      width="120"
      height="194"
      aria-hidden="true"
    />
    <img
      :src="ssxzStaticLogo"
      :style="{ colorScheme: theme }"
      alt=""
      class="ssxz-brand-logo__artwork ssxz-brand-logo__artwork--static"
      width="120"
      height="194"
      aria-hidden="true"
    />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  theme: 'light' | 'dark'
}>()

const ssxzAnimatedLogo = '/brand/ssxz-cat-dog-line-draw.svg'
const ssxzStaticLogo = '/brand/ssxz-cat-dog-static.svg'
</script>

<style scoped>
.ssxz-brand-logo {
  position: relative;
  width: 100%;
  height: 100%;
  isolation: isolate;
}

.ssxz-brand-logo__artwork {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.ssxz-brand-logo__artwork--static {
  display: none;
}

@media (prefers-reduced-motion: reduce) {
  .ssxz-brand-logo__artwork--animated {
    display: none;
  }

  .ssxz-brand-logo__artwork--static {
    display: block;
  }
}
</style>
