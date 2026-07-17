<template>
  <header class="f0-page-header">
    <div class="f0-page-header__copy">
      <div v-if="eyebrow" class="f0-page-header__eyebrow">
        <Icon :name="icon" size="xs" aria-hidden="true" />
        <span>{{ eyebrow }}</span>
      </div>
      <h1 class="f0-page-header__title">{{ title }}</h1>
      <p class="f0-page-header__subtitle">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="f0-page-header__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']

withDefaults(defineProps<{
  title: string
  subtitle: string
  eyebrow?: string
  icon?: IconName
}>(), {
  eyebrow: 'SSXZ AI',
  icon: 'sparkles'
})
</script>

<style scoped>
.f0-page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1.25rem;
  min-width: 0;
  margin-bottom: 1.5rem;
  border: 1px solid var(--ssxz-border, hsl(var(--border)));
  border-radius: var(--ssxz-radius-card, var(--radius));
  padding: 1.35rem 1.5rem;
  background: var(--ssxz-surface-raised, hsl(var(--card)));
  box-shadow: var(--ssxz-shadow-card, 0 1px 2px hsl(var(--shadow)));
}

.f0-page-header__copy {
  min-width: 0;
}

.f0-page-header__eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--ssxz-text-muted, hsl(var(--muted-foreground)));
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1rem;
}

.f0-page-header__title {
  margin: 0.55rem 0 0;
  color: var(--ssxz-text, hsl(var(--foreground)));
  font-size: clamp(1.55rem, 3vw, 2rem);
  font-weight: 680;
  letter-spacing: 0;
  line-height: 1.2;
  text-wrap: balance;
}

.f0-page-header__subtitle {
  max-width: 48rem;
  margin: 0.5rem 0 0;
  color: var(--ssxz-text-muted, hsl(var(--muted-foreground)));
  font-size: 0.875rem;
  line-height: 1.6;
}

.f0-page-header__actions {
  flex: 0 0 auto;
}

@media (max-width: 640px) {
  .f0-page-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 1rem;
    margin-bottom: 1rem;
    padding: 1.1rem 1rem;
  }

  .f0-page-header__actions {
    width: 100%;
  }
}
</style>
