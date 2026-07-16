<template>
  <div class="profile-info-card">
    <div class="profile-info-card__identity">
      <div class="profile-info-card__avatar-wrap">
        <Avatar :src="avatar?.url" :name="displayName" :size="76" />
        <button
          type="button"
          class="btn btn-secondary profile-info-card__avatar-button"
          :aria-label="t('profile.avatar.change')"
          @click="emit('change-avatar')"
        >
          <Icon name="upload" size="xs" />
          {{ t('profile.avatar.change') }}
        </button>
      </div>
      <div class="min-w-0 flex-1">
        <h2 class="profile-info-card__email" :title="user?.email || ''">{{ user?.email }}</h2>
        <div class="profile-info-card__badges">
          <span :class="['badge', user?.role === 'admin' ? 'badge-primary' : 'badge-gray']">
            {{ user?.role === 'admin' ? t('profile.administrator') : t('profile.user') }}
          </span>
          <span :class="['badge', user?.status === 'active' ? 'badge-success' : 'badge-danger']">
            {{ user?.status === 'active' ? t('profile.statusActive') : t('profile.statusDisabled') }}
          </span>
        </div>
      </div>
    </div>

    <dl class="profile-info-card__details">
      <div>
        <dt><Icon name="mail" size="sm" />{{ t('profile.email') }}</dt>
        <dd>{{ user?.email }}</dd>
      </div>
      <div v-if="user?.username">
        <dt><Icon name="user" size="sm" />{{ t('profile.username') }}</dt>
        <dd>{{ user.username }}</dd>
      </div>
    </dl>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Avatar from '@/components/common/Avatar.vue'
import Icon from '@/components/icons/Icon.vue'
import type { User, UserAvatar } from '@/types'

const emit = defineEmits<{
  (event: 'change-avatar'): void
}>()

const { t } = useI18n()
const props = defineProps<{ user: User | null; avatar?: UserAvatar | null }>()
const displayName = computed(() => props.user?.username || props.user?.email || t('profile.user'))
</script>

<style scoped>
.profile-info-card { min-width: 0; padding: 1.25rem; }
.profile-info-card__identity { display: flex; min-width: 0; align-items: center; gap: 1.1rem; }
.profile-info-card__avatar-wrap { display: flex; flex-direction: column; align-items: center; gap: 0.65rem; }
.profile-info-card__avatar-button { min-height: 2rem; padding: 0.35rem 0.65rem; font-size: 0.74rem; }
.profile-info-card__email { max-width: 100%; overflow: hidden; color: var(--ssxz-text-primary); font-size: 1.05rem; font-weight: 850; text-overflow: ellipsis; white-space: nowrap; }
.profile-info-card__badges { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-top: 0.55rem; }
.profile-info-card__details { display: grid; gap: 0.7rem; margin: 1.25rem 0 0; padding-top: 1rem; border-top: 1px solid var(--ssxz-border); }
.profile-info-card__details > div { display: flex; align-items: center; justify-content: space-between; gap: 1rem; }
.profile-info-card__details dt { display: inline-flex; align-items: center; gap: 0.5rem; color: var(--ssxz-text-secondary); font-size: 0.82rem; }
.profile-info-card__details dt :deep(svg) { color: var(--ssxz-text-subtle); }
.profile-info-card__details dd { max-width: 70%; overflow: hidden; color: var(--ssxz-text-primary); font-size: 0.88rem; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }

@media (max-width: 480px) {
  .profile-info-card__identity { align-items: flex-start; }
  .profile-info-card__details > div { align-items: flex-start; flex-direction: column; gap: 0.25rem; }
  .profile-info-card__details dd { max-width: 100%; padding-left: 1.4rem; }
}
</style>
