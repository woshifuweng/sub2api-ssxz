<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div
      :class="[
        'profile-workbench',
        { 'profile-workbench--app': useWorkbenchShell },
      ]"
    >
      <section
        class="profile-hero-card"
        :aria-label="t('profile.workbench.introAriaLabel')"
      >
        <div class="profile-hero-card__identity">
          <Avatar :src="avatar?.url" :name="displayName" :size="96" />
          <div class="profile-hero-card__copy">
            <div class="profile-hero-card__name-row">
              <h2>{{ displayName }}</h2>
              <span
                :class="[
                  'badge',
                  user?.role === 'admin' ? 'badge-primary' : 'badge-gray',
                ]"
              >
                {{
                  user?.role === "admin"
                    ? t("profile.administrator")
                    : t("profile.user")
                }}
              </span>
            </div>
            <p>{{ user?.email }}</p>
          </div>
        </div>
        <dl class="profile-hero-card__stats">
          <div>
            <dt>
              <Icon name="dollar" size="sm" />{{ t("profile.accountBalance") }}
            </dt>
            <dd>{{ formatCurrency(user?.balance || 0) }}</dd>
          </div>
          <div>
            <dt>
              <Icon name="shield" size="sm" />{{ t("profile.accountStatus") }}
            </dt>
            <dd :class="{ 'is-active': user?.status === 'active' }">
              {{ accountStatusLabel }}
            </dd>
          </div>
          <div>
            <dt>
              <Icon name="calendar" size="sm" />{{ t("profile.memberSince") }}
            </dt>
            <dd>
              {{
                formatDate(user?.created_at || "", {
                  year: "numeric",
                  month: "long",
                })
              }}
            </dd>
          </div>
        </dl>
      </section>

      <section class="profile-panel profile-identity-panel">
        <div class="profile-panel-heading">
          <span>{{ t("profile.workbench.basicInfoKicker") }}</span>
          <strong>{{ t("profile.workbench.accountInfoTitle") }}</strong>
        </div>
        <div class="profile-identity-grid">
          <div class="profile-avatar-settings">
            <Avatar :src="avatar?.url" :name="displayName" :size="84" />
            <div class="profile-avatar-settings__copy">
              <strong>{{ t("profile.avatar.uploadTitle") }}</strong>
              <p>{{ t("profile.avatar.uploadHint") }}</p>
            </div>
            <LiquidButton
              type="button"
              @click="avatarDialogOpen = true"
              variant="outline"
              size="sm"
            >
              <Icon name="upload" size="sm" />
              {{ t("profile.avatar.change") }}
            </LiquidButton>
          </div>
          <div class="profile-edit-slot">
            <ProfileEditForm :initial-username="user?.username || ''" />
          </div>
        </div>
      </section>

      <section
        v-if="linuxdoOAuthEnabled"
        class="profile-panel profile-provider-panel"
      >
        <div class="profile-panel-heading">
          <span>{{ t("profile.thirdParty.kicker") }}</span>
          <strong>{{ t("profile.thirdParty.title") }}</strong>
        </div>
        <div class="profile-provider-row">
          <div>
            <strong>{{ t("profile.thirdParty.linuxdoTitle") }}</strong>
            <p>{{ t("profile.thirdParty.linuxdoDescription") }}</p>
          </div>
          <span class="profile-provider-status">{{
            t("profile.thirdParty.connected")
          }}</span>
        </div>
      </section>

      <div v-if="contactInfo" class="profile-support-card">
        <div class="flex items-center gap-4">
          <div class="profile-support-icon"><Icon name="chat" size="lg" /></div>
          <div>
            <h3>{{ t("common.contactSupport") }}</h3>
            <p>{{ contactInfo }}</p>
          </div>
        </div>
      </div>

      <section class="profile-panel">
        <div class="profile-panel-heading">
          <span>{{ t("profile.workbench.loginProtectionKicker") }}</span>
          <strong>{{ t("profile.workbench.changePasswordTitle") }}</strong>
        </div>
        <ProfilePasswordForm />
      </section>

      <ProfileBalanceNotifyCard
        v-if="user && balanceLowNotifyEnabled"
        :enabled="user.balance_notify_enabled ?? true"
        :threshold="user.balance_notify_threshold ?? null"
        :extra-emails="user.balance_notify_extra_emails ?? []"
        :system-default-threshold="systemDefaultThreshold"
        :user-email="user.email"
      />

      <section class="profile-panel">
        <div class="profile-panel-heading">
          <span>{{ t("profile.workbench.twoFactorKicker") }}</span>
          <strong>{{ t("profile.workbench.securityTitle") }}</strong>
        </div>
        <ProfileTotpCard />
      </section>

      <ProfilePasskeyCard :enabled="passkeyEnabled" />
    </div>

    <AvatarCropDialog
      :show="avatarDialogOpen"
      :saving="avatarSaving"
      @close="avatarDialogOpen = false"
      @save="handleAvatarSave"
    />
  </component>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { ref, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useAppStore } from "@/stores/app";
import { formatDate } from "@/utils/format";
import { authAPI } from "@/api";
import AppLayout from "@/components/layout/AppLayout.vue";
import { userAPI } from "@/api";
import AppSectionShell from "@/components/user/AppSectionShell.vue";
import Avatar from "@/components/common/Avatar.vue";
import ProfileEditForm from "@/components/user/profile/ProfileEditForm.vue";
import ProfileBalanceNotifyCard from "@/components/user/profile/ProfileBalanceNotifyCard.vue";
import ProfilePasswordForm from "@/components/user/profile/ProfilePasswordForm.vue";
import ProfileTotpCard from "@/components/user/profile/ProfileTotpCard.vue";
import ProfilePasskeyCard from "@/components/user/profile/ProfilePasskeyCard.vue";
import AvatarCropDialog from "@/components/user/profile/AvatarCropDialog.vue";
import type { UserAvatar } from "@/types";
import { Icon } from "@/components/icons";

const { t } = useI18n();
const authStore = useAuthStore();
const appStore = useAppStore();
const user = computed(() => authStore.user);
const route = useRoute();
const useWorkbenchShell = computed(() => route.path.startsWith("/app/"));
const pageShell = computed(() =>
  useWorkbenchShell.value ? AppSectionShell : AppLayout,
);
const pageShellProps = computed(() =>
  useWorkbenchShell.value
    ? {
        title: t("profile.workbench.title"),
        subtitle: t("profile.workbench.subtitle"),
        eyebrow: t("profile.workbench.eyebrow"),
        icon: "userCircle",
      }
    : {},
);
const contactInfo = ref("");
const linuxdoOAuthEnabled = ref(false);
const balanceLowNotifyEnabled = ref(false);
const systemDefaultThreshold = ref(0);
const passkeyEnabled = ref(false);
const avatar = ref<UserAvatar | null>(null);
const avatarDialogOpen = ref(false);
const avatarSaving = ref(false);
const displayName = computed(
  () => user.value?.username || user.value?.email || t("profile.user"),
);
const accountStatusLabel = computed(() =>
  user.value?.status === "active"
    ? t("profile.statusActive")
    : t("profile.statusDisabled"),
);

onMounted(async () => {
  try {
    const settings = await authAPI.getPublicSettings();
    contactInfo.value = (settings.contact_info || "").trim();
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled === true;
    balanceLowNotifyEnabled.value =
      settings.balance_low_notify_enabled ?? false;
    systemDefaultThreshold.value = settings.balance_low_notify_threshold ?? 0;
    passkeyEnabled.value = settings.passkey_enabled === true;
  } catch {
    contactInfo.value = "";
    linuxdoOAuthEnabled.value = false;
    balanceLowNotifyEnabled.value = false;
    systemDefaultThreshold.value = 0;
    passkeyEnabled.value = false;
  }
  try {
    avatar.value = await userAPI.getAvatar();
  } catch {
    avatar.value = null;
  }
});

const handleAvatarSave = async (dataUrl: string) => {
  avatarSaving.value = true;
  try {
    avatar.value = await userAPI.updateAvatar(dataUrl);
    avatarDialogOpen.value = false;
    appStore.showSuccess(t("profile.avatar.saved"));
  } catch (error: any) {
    appStore.showError(error?.message || t("profile.avatar.saveFailed"));
  } finally {
    avatarSaving.value = false;
  }
};
const formatCurrency = (v: number) => `$${v.toFixed(2)}`;
</script>

<style scoped>
.profile-workbench {
  margin-inline: auto;
  max-width: 76rem;
  min-width: 0;
  width: 100%;
  display: grid;
  gap: 1.5rem;
}

.profile-workbench--app {
  max-width: 76rem;
}

.profile-hero-card,
.profile-panel,
.profile-support-card {
  min-width: 0;
  border: 1px solid var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.profile-hero-card {
  display: grid;
  grid-template-columns: minmax(16rem, 0.85fr) minmax(28rem, 1.35fr);
  align-items: center;
  gap: 2rem;
  border-radius: var(--ssxz-radius-card);
  padding: 1.5rem;
}

.profile-hero-card__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 1rem;
}

.profile-hero-card__copy {
  min-width: 0;
}

.profile-hero-card__name-row {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
}

.profile-hero-card__name-row h2 {
  margin: 0;
  overflow: hidden;
  color: var(--ssxz-text-primary);
  font-size: clamp(1.25rem, 2vw, 1.6rem);
  font-weight: 800;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-hero-card__copy > p {
  margin: 0.35rem 0 0;
  overflow: hidden;
  color: var(--ssxz-text-secondary);
  font-size: 0.875rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-hero-card__stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: calc(var(--ssxz-radius-card) - 0.2rem);
  background: color-mix(in srgb, var(--ssxz-surface-muted) 62%, transparent);
}

.profile-hero-card__stats > div {
  min-width: 0;
  padding: 0.9rem 1rem;
}

.profile-hero-card__stats > div + div {
  border-left: 1px solid var(--ssxz-border);
}

.profile-hero-card__stats dt {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
  font-weight: 650;
}

.profile-hero-card__stats dd {
  margin: 0.45rem 0 0;
  overflow: hidden;
  color: var(--ssxz-text-primary);
  font-size: 0.95rem;
  font-variant-numeric: tabular-nums;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-hero-card__stats dd.is-active {
  color: var(--ssxz-success);
}

.profile-panel-heading span {
  color: var(--ssxz-action);
  font-size: 0.78rem;
  font-weight: 850;
}

.profile-workbench--app :deep(.card) {
  border-color: var(--ssxz-border);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.profile-panel,
.profile-support-card {
  overflow: hidden;
  border-radius: var(--ssxz-radius-card);
}

.profile-panel-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 1rem 1.25rem;
}

.profile-panel-heading strong {
  color: var(--ssxz-text-primary);
  font-size: 0.95rem;
  font-weight: 750;
}

.profile-identity-grid {
  display: grid;
  grid-template-columns: minmax(17rem, 0.8fr) minmax(22rem, 1.2fr);
  min-width: 0;
}

.profile-avatar-settings {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-content: center;
  align-items: center;
  gap: 0.9rem 1rem;
  border-right: 1px solid var(--ssxz-border);
  padding: 1.5rem;
}

.profile-avatar-settings__copy {
  min-width: 0;
}

.profile-avatar-settings__copy strong {
  color: var(--ssxz-text-primary);
  font-size: 0.95rem;
  font-weight: 750;
}

.profile-avatar-settings__copy p {
  margin: 0.3rem 0 0;
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
  line-height: 1.55;
}

.profile-avatar-settings .btn {
  grid-column: 1 / -1;
  width: fit-content;
}

.profile-edit-slot {
  min-width: 0;
}

.profile-edit-slot :deep(.card) {
  height: 100%;
}

.profile-panel :deep(.card) {
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.profile-support-card {
  padding: 1.25rem;
}

.profile-support-icon {
  display: grid;
  width: 2.6rem;
  height: 2.6rem;
  place-items: center;
  border-radius: var(--ssxz-radius-button);
  background: color-mix(in srgb, var(--ssxz-action-soft) 75%, transparent);
  color: var(--ssxz-action);
}

.profile-support-card h3 {
  color: var(--ssxz-text-primary);
  font-weight: 750;
}

.profile-support-card p {
  margin-top: 0.15rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.9rem;
  font-weight: 650;
}

.profile-provider-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.1rem 1.25rem;
}

.profile-provider-row strong {
  color: var(--ssxz-text-primary);
  font-size: 0.95rem;
  font-weight: 800;
}

.profile-provider-row p {
  margin-top: 0.25rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
}

.profile-provider-status {
  flex: 0 0 auto;
  border: 1px solid
    color-mix(in srgb, var(--ssxz-success) 32%, var(--ssxz-border));
  border-radius: 999px;
  padding: 0.35rem 0.65rem;
  color: var(--ssxz-success);
  font-size: 0.75rem;
  font-weight: 800;
}

@media (max-width: 900px) {
  .profile-hero-card {
    grid-template-columns: 1fr;
  }

  .profile-identity-grid {
    grid-template-columns: 1fr;
  }

  .profile-avatar-settings {
    border-right: 0;
    border-bottom: 1px solid var(--ssxz-border);
  }
}

@media (max-width: 767px) {
  .profile-panel-heading {
    align-items: flex-start;
  }

  .profile-hero-card {
    padding: 1.1rem;
  }

  .profile-hero-card__stats {
    grid-template-columns: 1fr;
  }

  .profile-hero-card__stats > div + div {
    border-top: 1px solid var(--ssxz-border);
    border-left: 0;
  }

  .profile-avatar-settings {
    grid-template-columns: 1fr;
  }

  .profile-avatar-settings .btn {
    grid-column: auto;
  }
}
</style>
