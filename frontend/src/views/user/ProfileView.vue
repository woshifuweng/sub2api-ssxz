<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div :class="['profile-workbench', { 'profile-workbench--app': useWorkbenchShell }]">
      <section v-if="useWorkbenchShell" class="profile-intro" aria-label="账户设置说明">
        <div>
          <span>账号与安全</span>
          <h3>管理你的登录信息和安全验证</h3>
          <p>这里只处理你的个人资料、密码和二次验证。余额、用量、充值和 API Key 已经拆到左侧对应入口，避免普通用户误进后台控制台。</p>
        </div>
      </section>

      <div class="profile-stat-grid">
        <StatCard :title="t('profile.accountBalance')" :value="formatCurrency(user?.balance || 0)" :icon="WalletIcon" icon-variant="success" />
        <StatCard :title="t('profile.accountStatus')" :value="accountStatusLabel" :icon="StatusIcon" :icon-variant="user?.status === 'active' ? 'success' : 'danger'" />
        <StatCard :title="t('profile.memberSince')" :value="formatDate(user?.created_at || '', { year: 'numeric', month: 'long' })" :icon="CalendarIcon" icon-variant="primary" />
      </div>

      <section class="profile-panel">
        <div class="profile-panel-heading">
          <span>基础资料</span>
          <strong>账号信息</strong>
        </div>
        <ProfileInfoCard :user="user" />
      </section>

      <div v-if="contactInfo" class="profile-support-card">
        <div class="flex items-center gap-4">
          <div class="profile-support-icon"><Icon name="chat" size="lg" /></div>
          <div><h3>{{ t('common.contactSupport') }}</h3><p>{{ contactInfo }}</p></div>
        </div>
      </div>

      <section class="profile-panel">
        <div class="profile-panel-heading">
          <span>显示名称</span>
          <strong>编辑个人资料</strong>
        </div>
        <ProfileEditForm :initial-username="user?.username || ''" />
      </section>

      <section class="profile-panel">
        <div class="profile-panel-heading">
          <span>登录保护</span>
          <strong>修改密码</strong>
        </div>
        <ProfilePasswordForm />
      </section>

      <section class="profile-panel">
        <div class="profile-panel-heading">
          <span>二次验证</span>
          <strong>账号安全</strong>
        </div>
        <ProfileTotpCard />
      </section>
    </div>
  </component>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'; import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'; import { formatDate } from '@/utils/format'
import { authAPI } from '@/api'; import AppLayout from '@/components/layout/AppLayout.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import StatCard from '@/components/common/StatCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import { Icon } from '@/components/icons'

const { t } = useI18n(); const authStore = useAuthStore(); const user = computed(() => authStore.user)
const route = useRoute()
const useWorkbenchShell = computed(() => route.path.startsWith('/app/'))
const pageShell = computed(() => useWorkbenchShell.value ? AppSectionShell : AppLayout)
const pageShellProps = computed(() => useWorkbenchShell.value
  ? {
      title: '账户设置',
      subtitle: '查看账户信息，更新资料、密码和安全验证设置。',
      eyebrow: '我的账户',
      icon: 'userCircle'
    }
  : {}
)
const contactInfo = ref('')
const accountStatusLabel = computed(() => user.value?.status === 'active'
  ? t('profile.statusActive')
  : t('profile.statusDisabled')
)

const WalletIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M21 12a2.25 2.25 0 00-2.25-2.25H15a3 3 0 11-6 0H5.25A2.25 2.25 0 003 12' })]) }
const StatusIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z' })]) }
const CalendarIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M6.75 3v2.25M17.25 3v2.25' })]) }

onMounted(async () => { try { const s = await authAPI.getPublicSettings(); contactInfo.value = s.contact_info || '' } catch (error) { console.error('Failed to load contact info:', error) } })
const formatCurrency = (v: number) => `$${v.toFixed(2)}`
</script>

<style scoped>
.profile-workbench {
  margin-inline: auto;
  max-width: 56rem;
  display: grid;
  gap: 1rem;
}

.profile-workbench--app {
  max-width: 62rem;
}

.profile-intro,
.profile-panel,
.profile-support-card {
  border: 1px solid var(--ssxz-border);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 90%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
}

.profile-intro {
  border-radius: 1.35rem;
  padding: 1.15rem;
}

.profile-intro span,
.profile-panel-heading span {
  color: var(--ssxz-action);
  font-size: 0.78rem;
  font-weight: 850;
}

.profile-intro h3 {
  margin: 0.35rem 0 0;
  color: var(--ssxz-text-primary);
  font-size: 1.1rem;
  font-weight: 900;
}

.profile-intro p {
  margin: 0.45rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 0.9rem;
  line-height: 1.7;
}

.profile-stat-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.profile-workbench--app :deep(.stat-card),
.profile-workbench--app :deep(.card) {
  border-color: var(--ssxz-border);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
}

.profile-panel,
.profile-support-card {
  overflow: hidden;
  border-radius: 1.25rem;
}

.profile-panel-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 0.95rem 1rem;
}

.profile-panel-heading strong {
  color: var(--ssxz-text-primary);
  font-size: 0.95rem;
  font-weight: 850;
}

.profile-panel :deep(.card) {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.profile-support-card {
  padding: 1rem;
}

.profile-support-icon {
  display: grid;
  width: 2.6rem;
  height: 2.6rem;
  place-items: center;
  border-radius: 0.9rem;
  background: color-mix(in srgb, var(--ssxz-action-soft) 75%, transparent);
  color: var(--ssxz-action);
}

.profile-support-card h3 {
  color: var(--ssxz-text-primary);
  font-weight: 850;
}

.profile-support-card p {
  margin-top: 0.15rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.9rem;
  font-weight: 650;
}

@media (max-width: 767px) {
  .profile-stat-grid {
    grid-template-columns: 1fr;
  }
}
</style>
