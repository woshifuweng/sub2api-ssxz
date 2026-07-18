<template>
  <div class="admin-page-header">
    <div class="admin-page-header__title-block">
      <h1 class="admin-page-header__title">{{ title }}</h1>
      <p v-if="description" class="admin-page-header__desc">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="admin-page-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title: string; description?: string }>()
</script>

<style scoped>
.admin-page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--ssxz-border);
}

.admin-page-header__title-block {
  min-width: 0;
}

.admin-page-header__title {
  margin: 0;
  color: var(--ssxz-text-primary);
  font-size: 20px;
  font-synthesis: none;
  font-weight: 700;
  line-height: 1.35;
}

.admin-page-header__desc {
  margin: 4px 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.admin-page-header__actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
}
</style>
