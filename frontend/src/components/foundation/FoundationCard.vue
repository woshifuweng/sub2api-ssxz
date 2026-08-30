<template>
  <section class="f0-card">
    <header v-if="title || description || $slots.action" class="f0-card-header">
      <div>
        <h2 v-if="title" class="f0-card-title">{{ title }}</h2>
        <p v-if="description" class="f0-card-description">{{ description }}</p>
      </div>
      <slot name="action" />
    </header>
    <div class="f0-card-content">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="f0-card-footer">
      <slot name="footer" />
    </footer>
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title?: string
  description?: string
}>()
</script>
