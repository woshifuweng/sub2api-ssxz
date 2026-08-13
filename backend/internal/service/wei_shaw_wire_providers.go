package service

import (
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

func ProvideAuditLogService(repo AuditLogRepository, settingService *SettingService) *AuditLogService {
	svc := NewAuditLogService(repo, settingService)
	svc.Start()
	return svc
}

func ProvideOpsIngressRejectAggregator(opsRepo OpsRepository, opsService *OpsService) *OpsIngressRejectAggregator {
	repo, ok := opsRepo.(OpsIngressRejectRepository)
	if !ok {
		return nil
	}
	aggregator := NewOpsIngressRejectAggregator(repo)
	aggregator.Start()
	opsService.SetIngressRejectAggregator(aggregator)
	return aggregator
}

func ProvideUserPlatformQuotaUsageFlusher(
	cfg *config.Config,
	cache BillingCache,
	quotaRepo UserPlatformQuotaRepository,
	tw *TimingWheelService,
) *UserPlatformQuotaUsageFlusher {
	svc := NewUserPlatformQuotaUsageFlusher(cfg, cache, quotaRepo, tw)
	svc.Start()
	return svc
}

func ProvideOpenAIQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *OpenAITokenProvider,
	privacyClientFactory PrivacyClientFactory,
	openAIGatewayService *OpenAIGatewayService,
) *OpenAIQuotaService {
	svc := NewOpenAIQuotaService(accountRepo, proxyRepo, tokenProvider, privacyClientFactory)
	svc.agentIdentityWS = openAIGatewayService
	return svc
}

func ProvideOpenAIGatewayService(
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	usageLogRepo UsageLogRepository,
	usageBillingRepo UsageBillingRepository,
	userRepo UserRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache GatewayCache,
	cfg *config.Config,
	schedulerSnapshot *SchedulerSnapshotService,
	concurrencyService *ConcurrencyService,
	billingService *BillingService,
	modelPricingResolver *ModelPricingResolver,
	rateLimitService *RateLimitService,
	billingCacheService *BillingCacheService,
	httpUpstream HTTPUpstream,
	deferredService *DeferredService,
	openAITokenProvider *OpenAITokenProvider,
	grokTokenProvider *GrokTokenProvider,
	channelService *ChannelService,
	balanceNotifyService *BalanceNotifyService,
	settingService *SettingService,
	userPlatformQuotaRepo UserPlatformQuotaRepository,
	tlsFPProfileService *TLSFingerprintProfileService,
) *OpenAIGatewayService {
	svc := NewOpenAIGatewayService(
		accountRepo,
		groupRepo,
		usageLogRepo,
		usageBillingRepo,
		userRepo,
		userSubRepo,
		userGroupRateRepo,
		cache,
		cfg,
		schedulerSnapshot,
		concurrencyService,
		billingService,
		modelPricingResolver,
		rateLimitService,
		billingCacheService,
		httpUpstream,
		deferredService,
		openAITokenProvider,
	)
	svc.ConfigureProductionDependencies(
		grokTokenProvider,
		channelService,
		balanceNotifyService,
		settingService,
		userPlatformQuotaRepo,
		tlsFPProfileService,
	)
	return svc
}

func ProvideAntigravityGatewayService(
	accountRepo AccountRepository,
	cache GatewayCache,
	schedulerSnapshot *SchedulerSnapshotService,
	tokenProvider *AntigravityTokenProvider,
	rateLimitService *RateLimitService,
	httpUpstream HTTPUpstream,
	settingService *SettingService,
	internal500Cache Internal500CounterCache,
) *AntigravityGatewayService {
	return NewAntigravityGatewayService(
		accountRepo,
		cache,
		schedulerSnapshot,
		tokenProvider,
		rateLimitService,
		httpUpstream,
		settingService,
		internal500Cache,
	)
}

func ProvideAccountUsageService(
	accountRepo AccountRepository,
	usageLogRepo UsageLogRepository,
	usageFetcher ClaudeUsageFetcher,
	geminiQuotaService *GeminiQuotaService,
	antigravityQuotaFetcher *AntigravityQuotaFetcher,
	grokQuotaFetcher *GrokQuotaFetcher,
	grokQuotaService *GrokQuotaService,
	openAIQuotaService *OpenAIQuotaService,
	cache *UsageCache,
	identityCache IdentityCache,
	tlsFPProfileService *TLSFingerprintProfileService,
	openAIGatewayService *OpenAIGatewayService,
) *AccountUsageService {
	svc := NewAccountUsageService(
		accountRepo,
		usageLogRepo,
		usageFetcher,
		geminiQuotaService,
		antigravityQuotaFetcher,
		grokQuotaFetcher,
		grokQuotaService,
		openAIQuotaService,
		cache,
		identityCache,
		tlsFPProfileService,
	)
	svc.agentIdentityWS = openAIGatewayService
	return svc
}

func ProvideAccountTestService(
	accountRepo AccountRepository,
	geminiTokenProvider *GeminiTokenProvider,
	claudeTokenProvider *ClaudeTokenProvider,
	grokTokenProvider *GrokTokenProvider,
	openAITokenProvider *OpenAITokenProvider,
	antigravityGatewayService *AntigravityGatewayService,
	openAIGatewayService *OpenAIGatewayService,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	tlsFPProfileService *TLSFingerprintProfileService,
	settingService *SettingService,
) *AccountTestService {
	svc := NewAccountTestService(
		accountRepo,
		geminiTokenProvider,
		claudeTokenProvider,
		grokTokenProvider,
		openAITokenProvider,
		antigravityGatewayService,
		openAIGatewayService,
		httpUpstream,
		cfg,
		tlsFPProfileService,
	)
	svc.SetSettingService(settingService)
	return svc
}

func ProvideGrokQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *GrokTokenProvider,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	usageLogRepo UsageLogRepository,
) *GrokQuotaService {
	return NewGrokQuotaService(accountRepo, proxyRepo, tokenProvider, httpUpstream, cfg, usageLogRepo)
}

func ProvideGrokTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	grokOAuthService *GrokOAuthService,
	refreshAPI *OAuthRefreshAPI,
	tempUnschedCache TempUnschedCache,
) *GrokTokenProvider {
	p := NewGrokTokenProvider(accountRepo, tokenCache)
	p.SetRefreshAPI(refreshAPI, NewGrokTokenRefresher(grokOAuthService))
	p.SetRefreshPolicy(GrokProviderRefreshPolicy())
	p.SetTempUnschedCache(tempUnschedCache)
	return p
}

func ProvideImageStorageSettingService(
	settingRepo SettingRepository,
	encryptor SecretEncryptor,
	backup *BackupService,
	factory ImageStorageFactory,
	cfg *config.Config,
) *ImageStorageSettingService {
	if cfg.ImageStorage.Enabled && !cfg.ImageStorage.Active() {
		logger.L().Warn(
			"image_storage.enabled is true in config but object storage is not fully configured; configure it in the admin UI or complete the config file",
			zap.Strings("missing_keys", cfg.ImageStorage.MissingCredentialKeys()),
		)
	}
	return NewImageStorageSettingService(settingRepo, encryptor, backup, factory, cfg.ImageStorage)
}

func ProvideImageTaskService(store ImageTaskStore, settings *ImageStorageSettingService) *ImageTaskService {
	return NewImageTaskServiceWithResolver(store, settings.Resolver(), defaultImageTaskTTL, defaultImageTaskExecutionTimeout)
}

func ProvideBillingCacheService(
	cache BillingCache,
	userRepo UserRepository,
	subRepo UserSubscriptionRepository,
	apiKeyRepo APIKeyRepository,
	rpmCache UserRPMCache,
	rateRepo UserGroupRateRepository,
	cfg *config.Config,
	userPlatformQuotaRepo UserPlatformQuotaRepository,
) *BillingCacheService {
	return NewBillingCacheService(cache, userRepo, subRepo, apiKeyRepo, rpmCache, rateRepo, cfg, userPlatformQuotaRepo)
}

func ProvideAPIKeyService(
	apiKeyRepo APIKeyRepository,
	userRepo UserRepository,
	groupRepo GroupRepository,
	userSubRepo UserSubscriptionRepository,
	userGroupRateRepo UserGroupRateRepository,
	cache APIKeyCache,
	cfg *config.Config,
	billingCacheService *BillingCacheService,
	concurrencyService *ConcurrencyService,
) *APIKeyService {
	svc := NewAPIKeyService(apiKeyRepo, userRepo, groupRepo, userSubRepo, userGroupRateRepo, cache, cfg)
	svc.SetRateLimitCacheInvalidator(billingCacheService)
	svc.SetConcurrencyService(concurrencyService)
	return svc
}

func ProvidePaymentService(
	entClient *dbent.Client,
	registry *payment.Registry,
	loadBalancer payment.LoadBalancer,
	redeemService *RedeemService,
	subscriptionSvc *SubscriptionService,
	configService *PaymentConfigService,
	userRepo UserRepository,
	groupRepo GroupRepository,
	affiliateService *AffiliateService,
	notificationEmailService *NotificationEmailService,
) *PaymentService {
	svc := NewPaymentService(entClient, registry, loadBalancer, redeemService, subscriptionSvc, configService, userRepo, groupRepo, affiliateService)
	svc.SetNotificationEmailService(notificationEmailService)
	return svc
}
