<template>
  <div v-if="mobileOpen" class="f0-sidebar-overlay" aria-hidden="true" @click="emit('close')" />
  <aside class="f0-sidebar" :class="{ 'f0-sidebar--mobile-open': mobileOpen }">
    <header class="f0-sidebar-header">
      <span class="f0-sidebar-brand-icon" aria-hidden="true">
        <Boxes />
      </span>
      <div class="f0-sidebar-brand-copy">
        <div class="f0-sidebar-brand-title">{{ title }}</div>
        <div class="f0-sidebar-brand-subtitle">{{ subtitle }}</div>
      </div>
      <button
        class="f0-sidebar-mobile-close"
        type="button"
        aria-label="关闭导航"
        @click="emit('close')"
      >
        <X aria-hidden="true" />
      </button>
    </header>

    <nav class="f0-sidebar-nav" aria-label="样板导航">
      <section v-for="section in sections" :key="section.label" class="f0-sidebar-group">
        <div class="f0-sidebar-label">{{ section.label }}</div>
        <button
          v-for="item in section.items"
          :key="item.id"
          class="f0-sidebar-item"
          type="button"
          :aria-current="item.id === activeId ? 'page' : undefined"
          @click="selectItem(item.id)"
        >
          <component :is="item.icon" v-if="item.icon" aria-hidden="true" />
          <span class="f0-sidebar-item-label">{{ item.label }}</span>
          <FoundationBadge v-if="item.badge" variant="secondary">{{ item.badge }}</FoundationBadge>
        </button>
      </section>
    </nav>

    <footer class="f0-sidebar-footer">
      <slot name="footer" />
    </footer>
  </aside>
</template>

<script setup lang="ts">
import { Boxes, X } from '@lucide/vue'
import FoundationBadge from './FoundationBadge.vue'
import type { FoundationSidebarSection } from './types'

withDefaults(
  defineProps<{
    title?: string
    subtitle?: string
    activeId?: string
    mobileOpen?: boolean
    sections: FoundationSidebarSection[]
  }>(),
  {
    title: 'SSXZ UI',
    subtitle: 'Foundation preview',
    activeId: '',
    mobileOpen: false
  }
)

const emit = defineEmits<{
  (event: 'select', id: string): void
  (event: 'close'): void
}>()

const selectItem = (id: string) => {
  emit('select', id)
  emit('close')
}
</script>
