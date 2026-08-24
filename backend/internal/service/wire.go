package service

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/curlcffi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

const (
	// Deprecated: retained for backward-compatible tests and env hygiene.
	backgroundServicesEnvVar = "SUB2API_ENABLE_BACKGROUND_SERVICES"
	processRoleEnvVar        = "SUB2API_PROCESS_ROLE"
	processRoleWorker        = "worker"
	processRoleCoordinator   = "coordinator"
	appRuntimeRoleEnvVar     = "APP_RUNTIME_ROLE"
	stagingAPIOnlyEnvVar     = "STAGING_API_ONLY"
	backgroundJobsEnvVar     = "BACKGROUND_JOBS_ENABLED"
	schedulersEnvVar         = "SCHEDULERS_ENABLED"
	appRuntimeRoleAPI        = "api"
)

func currentProcessRole() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv(processRoleEnvVar)))
}

func envBoolValue(key string) (bool, bool) {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "y", "on", "enabled":
		return true, true
	case "0", "false", "no", "n", "off", "disabled":
		return false, true
	default:
		return false, false
	}
}

func stagingAPIOnlyRuntimeEnabled() bool {
	role := strings.ToLower(strings.TrimSpace(os.Getenv(appRuntimeRoleEnvVar)))
	if role == appRuntimeRoleAPI {
		return true
	}
	enabled, ok := envBoolValue(stagingAPIOnlyEnvVar)
	return ok && enabled
}

func runtimeBackgroundJobsEnabled() bool {
	if stagingAPIOnlyRuntimeEnabled() {
		return false
	}
	enabled, ok := envBoolValue(backgroundJobsEnvVar)
	if ok {
		return enabled
	}
	return true
}

func runtimeSchedulersEnabled() bool {
	if !runtimeBackgroundJobsEnabled() {
		return false
	}
	enabled, ok := envBoolValue(schedulersEnvVar)
	if ok {
		return enabled
	}
	return true
}

func singletonBackgroundServicesEnabled() bool {
	if !runtimeBackgroundJobsEnabled() {
		return false
	}
	switch currentProcessRole() {
	case "", processRoleCoordinator:
		return true
	default:
		return false
	}
}

func singletonSchedulerServicesEnabled() bool {
	return runtimeSchedulersEnabled() && singletonBackgroundServicesEnabled()
}

func workerLocalBackgroundServicesEnabled() bool {
	if !runtimeBackgroundJobsEnabled() {
		return false
	}
	switch currentProcessRole() {
	case "", processRoleWorker:
		return true
	default:
		return false
	}
}

func requestPathCacheSyncEnabled() bool {
	if !runtimeBackgroundJobsEnabled() {
		return false
	}
	switch currentProcessRole() {
	case "", processRoleWorker:
		return true
	default:
		return false
	}
}

func coordinatorOrSingleProcess() bool {
	if !runtimeBackgroundJobsEnabled() {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(os.Getenv(processRoleEnvVar)))
	switch role {
	case "", processRoleCoordinator:
		return true
	default:
		return false
	}
}

// BuildInfo contains build information
type BuildInfo struct {
	Version     string
	BuildType   string
	ReleaseRepo string
}

// ProvidePricingService creates and initializes PricingService
func ProvidePricingService(cfg *config.Config, remoteClient PricingRemoteClient) (*PricingService, error) {
	svc := NewPricingService(cfg, remoteClient)
	if err := svc.InitializeWithBackground(singletonSchedulerServicesEnabled()); err != nil {
		// Pricing service initialization failure should not block startup, use fallback prices
		println("[Service] Warning: Pricing service initialization failed:", err.Error())
	}
	return svc, nil
}

// ProvideUpdateService creates UpdateService with BuildInfo
func ProvideUpdateService(cache UpdateCache, githubClient GitHubReleaseClient, buildInfo BuildInfo, cfg *config.Config) *UpdateService {
	releaseRepo := buildInfo.ReleaseRepo
	if cfg != nil && strings.TrimSpace(cfg.Update.Repo) != "" {
		releaseRepo = cfg.Update.Repo
	}
	return NewUpdateService(cache, githubClient, buildInfo.Version, buildInfo.BuildType, releaseRepo)
}

// ProvideEmailQueueService creates EmailQueueService with default worker count
func ProvideEmailQueueService(emailService *EmailService) *EmailQueueService {
	return NewEmailQueueService(emailService, 3)
}

// ProvideAuthService wires the optional captcha providers into AuthService while
// keeping NewAuthService's public constructor compatible with existing tests.
func ProvideAuthService(
	entClient *dbent.Client,
	userRepo UserRepository,
	redeemRepo RedeemCodeRepository,
	refreshTokenCache RefreshTokenCache,
	cfg *config.Config,
	settingService *SettingService,
	emailService *EmailService,
	turnstileService *TurnstileService,
	tencentCaptchaService *TencentCaptchaService,
	aliyunCaptchaService *AliyunCaptchaService,
	emailQueueService *EmailQueueService,
	promoService *PromoService,
	defaultSubAssigner DefaultSubscriptionAssigner,
	affiliateService *AffiliateService,
	userPlatformQuotaRepo UserPlatformQuotaRepository,
) *AuthService {
	svc := NewAuthService(
		entClient,
		userRepo,
		redeemRepo,
		refreshTokenCache,
		cfg,
		settingService,
		emailService,
		turnstileService,
		emailQueueService,
		promoService,
		defaultSubAssigner,
		affiliateService,
		userPlatformQuotaRepo,
	)
	svc.SetTencentCaptchaService(tencentCaptchaService)
	svc.SetAliyunCaptchaService(aliyunCaptchaService)
	return svc
}

// ProvideOAuthRefreshAPI creates OAuthRefreshAPI with the default lock TTL.
func ProvideOAuthRefreshAPI(accountRepo AccountRepository, tokenCache GeminiTokenCache) *OAuthRefreshAPI {
	return NewOAuthRefreshAPI(accountRepo, tokenCache)
}

func ProvideBatchImageModelPricingResolver(resolver *ModelPricingResolver) *BatchImageModelPricingResolver {
	return &BatchImageModelPricingResolver{Resolver: resolver}
}

func ProvideBatchImageCleanupService(repo BatchImageRepository, accountRepo AccountRepository, cfg *config.Config) *BatchImageCleanupService {
	svc := NewBatchImageCleanupService(repo, accountRepo, cfg)
	svc.Start()
	return svc
}

// ProvideTokenRefreshService creates and starts TokenRefreshService
func ProvideTokenRefreshService(
	accountRepo AccountRepository,
	soraAccountRepo SoraAccountRepository, // Sora 扩展表仓储，用于双表同步
	oauthService *OAuthService,
	openaiOAuthService *OpenAIOAuthService,
	geminiOAuthService *GeminiOAuthService,
	antigravityOAuthService *AntigravityOAuthService,
	cacheInvalidator TokenCacheInvalidator,
	schedulerCache SchedulerCache,
	cfg *config.Config,
	tempUnschedCache TempUnschedCache,
	privacyClientFactory PrivacyClientFactory,
	proxyRepo ProxyRepository,
	refreshAPI *OAuthRefreshAPI,
	rateLimitService *RateLimitService,
) *TokenRefreshService {
	svc := NewTokenRefreshService(accountRepo, oauthService, openaiOAuthService, geminiOAuthService, antigravityOAuthService, cacheInvalidator, schedulerCache, cfg, tempUnschedCache)
	// 注入 Sora 账号扩展表仓储，用于 OpenAI Token 刷新时同步 sora_accounts 表
	svc.SetSoraAccountRepo(soraAccountRepo)
	// 注入 OpenAI privacy opt-out 依赖
	svc.SetPrivacyDeps(privacyClientFactory, proxyRepo)
	// 注入统一 OAuth 刷新 API（消除 TokenRefreshService 与 TokenProvider 之间的竞争条件）
	svc.SetRefreshAPI(refreshAPI)
	// 调用侧显式注入后台刷新策略，避免策略漂移
	svc.SetRefreshPolicy(DefaultBackgroundRefreshPolicy())
	svc.SetRateLimitService(rateLimitService)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideClaudeTokenProvider creates ClaudeTokenProvider with OAuthRefreshAPI injection
func ProvideClaudeTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	oauthService *OAuthService,
	refreshAPI *OAuthRefreshAPI,
) *ClaudeTokenProvider {
	p := NewClaudeTokenProvider(accountRepo, tokenCache, oauthService)
	executor := NewClaudeTokenRefresher(oauthService)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(ClaudeProviderRefreshPolicy())
	return p
}

// ProvideOpenAITokenProvider creates OpenAITokenProvider with OAuthRefreshAPI injection
func ProvideOpenAITokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	openaiOAuthService *OpenAIOAuthService,
	refreshAPI *OAuthRefreshAPI,
) *OpenAITokenProvider {
	p := NewOpenAITokenProvider(accountRepo, tokenCache, openaiOAuthService)
	executor := NewOpenAITokenRefresher(openaiOAuthService, accountRepo)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(OpenAIProviderRefreshPolicy())
	return p
}

// ProvideOpenAIQuotaService wires the OpenAI quota query/reset service.
// It depends on the OpenAI token provider for refreshed access tokens and the
// privacy client factory for the impersonated upstream HTTP client.
func ProvideOpenAIQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *OpenAITokenProvider,
	privacyClientFactory PrivacyClientFactory,
	openAIGatewayService *OpenAIGatewayService,
) *OpenAIQuotaService {
	service := NewOpenAIQuotaService(accountRepo, proxyRepo, tokenProvider, privacyClientFactory)
	service.agentIdentityWS = openAIGatewayService
	return service
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
	service := NewAccountUsageService(
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
	service.agentIdentityWS = openAIGatewayService
	return service
}

func ProvideAccountTestService(
	accountRepo AccountRepository,
	geminiTokenProvider *GeminiTokenProvider,
	claudeTokenProvider *ClaudeTokenProvider,
	grokTokenProvider *GrokTokenProvider,
	antigravityGatewayService *AntigravityGatewayService,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	tlsFPProfileService *TLSFingerprintProfileService,
	openAIGatewayService *OpenAIGatewayService,
	settingService *SettingService,
) *AccountTestService {
	service := NewAccountTestService(
		accountRepo,
		geminiTokenProvider,
		claudeTokenProvider,
		grokTokenProvider,
		antigravityGatewayService,
		httpUpstream,
		cfg,
		tlsFPProfileService,
	)
	service.agentIdentityWS = openAIGatewayService
	service.SetSettingService(settingService)
	return service
}

func ProvideGrokQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *GrokTokenProvider,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
	usageLogRepo UsageLogRepository,
	settingService *SettingService,
) *GrokQuotaService {
	service := NewGrokQuotaService(accountRepo, proxyRepo, tokenProvider, httpUpstream, cfg, usageLogRepo)
	service.SetSettingService(settingService)
	return service
}

// ProvideCNProviderQuotaService 构造国产供应商 Coding Plan 额度探测服务。
func ProvideCNProviderQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *CNProviderQuotaService {
	return NewCNProviderQuotaService(accountRepo, proxyRepo, httpUpstream, cfg)
}

// ProvideCNProviderBalanceService 构造国产供应商余额探测服务。
func ProvideCNProviderBalanceService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	httpUpstream HTTPUpstream,
	cfg *config.Config,
) *CNProviderBalanceService {
	return NewCNProviderBalanceService(accountRepo, proxyRepo, httpUpstream, cfg)
}

// ProvideCNProviderBalanceCheckService 构造并启动周期余额/额度检测任务。
// payg 账号探余额（低余额停调）；coding plan 账号探 5h/weekly 滚动窗口
// （落 extra 快照供调度阈值评估自动停调）。
// 间隔取自 gateway.cn_providers.balance_check_interval_minutes；<=0 或关闭时不启动。
func ProvideCNProviderBalanceCheckService(
	accountRepo AccountRepository,
	balanceService *CNProviderBalanceService,
	quotaService *CNProviderQuotaService,
	cfg *config.Config,
) *CNProviderBalanceCheckService {
	minutes := 10
	if cfg != nil && cfg.Gateway.CNProviders.BalanceCheckIntervalMinutes > 0 {
		minutes = cfg.Gateway.CNProviders.BalanceCheckIntervalMinutes
	}
	svc := NewCNProviderBalanceCheckService(accountRepo, balanceService, quotaService, cfg, time.Duration(minutes)*time.Minute)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideGeminiTokenProvider creates GeminiTokenProvider with OAuthRefreshAPI injection
func ProvideGeminiTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	geminiOAuthService *GeminiOAuthService,
	refreshAPI *OAuthRefreshAPI,
) *GeminiTokenProvider {
	p := NewGeminiTokenProvider(accountRepo, tokenCache, geminiOAuthService)
	executor := NewGeminiTokenRefresher(geminiOAuthService)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(GeminiProviderRefreshPolicy())
	return p
}

// ProvideAntigravityTokenProvider creates AntigravityTokenProvider with OAuthRefreshAPI injection
func ProvideAntigravityTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	antigravityOAuthService *AntigravityOAuthService,
	refreshAPI *OAuthRefreshAPI,
	tempUnschedCache TempUnschedCache,
) *AntigravityTokenProvider {
	p := NewAntigravityTokenProvider(accountRepo, tokenCache, antigravityOAuthService)
	executor := NewAntigravityTokenRefresher(antigravityOAuthService)
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(AntigravityProviderRefreshPolicy())
	p.SetTempUnschedCache(tempUnschedCache)
	return p
}

// ProvideKiroTokenProvider creates KiroTokenProvider with OAuthRefreshAPI injection
func ProvideKiroTokenProvider(
	accountRepo AccountRepository,
	tokenCache GeminiTokenCache,
	kiroUsageService *KiroUsageService,
	refreshAPI *OAuthRefreshAPI,
) *KiroTokenProvider {
	p := NewKiroTokenProvider(accountRepo, tokenCache, kiroUsageService)
	executor := NewKiroTokenRefresher()
	p.SetRefreshAPI(refreshAPI, executor)
	p.SetRefreshPolicy(ClaudeProviderRefreshPolicy())
	return p
}

// ProvideDashboardAggregationService 创建并启动仪表盘聚合服务
func ProvideDashboardAggregationService(repo DashboardAggregationRepository, timingWheel *TimingWheelService, cfg *config.Config) *DashboardAggregationService {
	svc := NewDashboardAggregationService(repo, timingWheel, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideUsageCleanupService 创建并启动使用记录清理任务服务
func ProvideUsageCleanupService(repo UsageCleanupRepository, timingWheel *TimingWheelService, dashboardAgg *DashboardAggregationService, cfg *config.Config) *UsageCleanupService {
	svc := NewUsageCleanupService(repo, timingWheel, dashboardAgg, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideAccountExpiryService creates and starts AccountExpiryService.
func ProvideAccountExpiryService(accountRepo AccountRepository) *AccountExpiryService {
	svc := NewAccountExpiryService(accountRepo, time.Minute)
	svc.Start()
	return svc
}

// ProvideOpenAICodexVersionSyncService creates and starts OpenAICodexVersionSyncService.
// 出站 Codex 身份的版本号靠它跟随官方发布，无需为了跟版本而发新版本；面板可关闭。
func ProvideOpenAICodexVersionSyncService(
	settingRepo SettingRepository,
	settingService *SettingService,
	githubClient GitHubReleaseClient,
) *OpenAICodexVersionSyncService {
	svc := NewOpenAICodexVersionSyncService(settingRepo, settingService, githubClient, openAICodexVersionSyncInterval)
	svc.Start()
	return svc
}

// ProvideProxyExpiryService creates and starts ProxyExpiryService.
func ProvideProxyExpiryService(proxyRepo ProxyRepository) *ProxyExpiryService {
	svc := NewProxyExpiryService(proxyRepo, time.Minute)
	svc.Start()
	return svc
}

// ProvideSubscriptionExpiryService creates and starts SubscriptionExpiryService.
func ProvideSubscriptionExpiryService(userSubRepo UserSubscriptionRepository) *SubscriptionExpiryService {
	svc := NewSubscriptionExpiryService(userSubRepo, time.Minute)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideTimingWheelService creates and starts TimingWheelService
func ProvideTimingWheelService() (*TimingWheelService, error) {
	svc, err := NewTimingWheelService()
	if err != nil {
		return nil, err
	}
	if runtimeSchedulersEnabled() {
		svc.Start()
	}
	return svc, nil
}

// ProvideDeferredService creates and starts DeferredService
func ProvideDeferredService(accountRepo AccountRepository, timingWheel *TimingWheelService) *DeferredService {
	svc := NewDeferredService(accountRepo, timingWheel, 10*time.Second)
	if workerLocalBackgroundServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideConcurrencyService creates ConcurrencyService and starts slot cleanup worker.
func ProvideConcurrencyService(cache ConcurrencyCache, accountRepo AccountRepository, cfg *config.Config) *ConcurrencyService {
	svc := NewConcurrencyService(cache)
	if cfg != nil {
		svc.SetFairWaitQueueEnabled(cfg.Gateway.Scheduling.FairWaitQueueEnabled)
	}
	if coordinatorOrSingleProcess() {
		if err := svc.CleanupStaleProcessSlots(context.Background()); err != nil {
			logger.LegacyPrintf("service.concurrency", "Warning: startup cleanup stale process slots failed: %v", err)
		}
	}
	if cfg != nil && coordinatorOrSingleProcess() {
		svc.StartSlotCleanupWorker(accountRepo, cfg.Gateway.Scheduling.SlotCleanupInterval)
	}
	return svc
}

// ProvideUserMessageQueueService 创建用户消息串行队列服务并启动清理 worker
func ProvideUserMessageQueueService(cache UserMsgQueueCache, rpmCache RPMCache, cfg *config.Config) *UserMessageQueueService {
	svc := NewUserMessageQueueService(cache, rpmCache, &cfg.Gateway.UserMessageQueue)
	if cfg.Gateway.UserMessageQueue.CleanupIntervalSeconds > 0 && coordinatorOrSingleProcess() {
		svc.StartCleanupWorker(time.Duration(cfg.Gateway.UserMessageQueue.CleanupIntervalSeconds) * time.Second)
	}
	return svc
}

// ProvideSchedulerSnapshotService creates and starts SchedulerSnapshotService.
func ProvideSchedulerSnapshotService(
	cache SchedulerCache,
	outboxRepo SchedulerOutboxRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	cfg *config.Config,
) *SchedulerSnapshotService {
	svc := NewSchedulerSnapshotService(cache, outboxRepo, accountRepo, groupRepo, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

func ProvideSchedulerSnapshotAdmissionBinding(
	svc *SchedulerSnapshotService,
	tokenRefreshService *TokenRefreshService,
) *SchedulerSnapshotService {
	if svc != nil {
		svc.SetAdmissionTester(tokenRefreshService)
	}
	return svc
}

func ProvideSchedulerSnapshotServiceWithAdmission(
	cache SchedulerCache,
	outboxRepo SchedulerOutboxRepository,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	cfg *config.Config,
	tokenRefreshService *TokenRefreshService,
) *SchedulerSnapshotService {
	svc := ProvideSchedulerSnapshotService(cache, outboxRepo, accountRepo, groupRepo, cfg)
	return ProvideSchedulerSnapshotAdmissionBinding(svc, tokenRefreshService)
}

// ProvideRateLimitService creates RateLimitService with optional dependencies.
func ProvideRateLimitService(
	accountRepo AccountRepository,
	usageRepo UsageLogRepository,
	cfg *config.Config,
	geminiQuotaService *GeminiQuotaService,
	tempUnschedCache TempUnschedCache,
	timeoutCounterCache TimeoutCounterCache,
	openAI403CounterCache OpenAI403CounterCache,
	settingService *SettingService,
	tokenCacheInvalidator TokenCacheInvalidator,
) *RateLimitService {
	svc := NewRateLimitService(accountRepo, usageRepo, cfg, geminiQuotaService, tempUnschedCache)
	svc.SetTimeoutCounterCache(timeoutCounterCache)
	svc.SetOpenAI403CounterCache(openAI403CounterCache)
	svc.SetSettingService(settingService)
	svc.SetTokenCacheInvalidator(tokenCacheInvalidator)
	return svc
}

// ProvideOpsMetricsCollector creates and starts OpsMetricsCollector.
func ProvideOpsMetricsCollector(
	opsRepo OpsRepository,
	settingRepo SettingRepository,
	accountRepo AccountRepository,
	concurrencyService *ConcurrencyService,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsMetricsCollector {
	collector := NewOpsMetricsCollector(opsRepo, settingRepo, accountRepo, concurrencyService, db, redisClient, cfg)
	if singletonSchedulerServicesEnabled() {
		collector.Start()
	}
	return collector
}

// ProvideOpsAggregationService creates and starts OpsAggregationService (hourly/daily pre-aggregation).
func ProvideOpsAggregationService(
	opsRepo OpsRepository,
	settingRepo SettingRepository,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsAggregationService {
	svc := NewOpsAggregationService(opsRepo, settingRepo, db, redisClient, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideOpsAlertEvaluatorService creates and starts OpsAlertEvaluatorService.
func ProvideOpsAlertEvaluatorService(
	opsService *OpsService,
	opsRepo OpsRepository,
	emailService *EmailService,
	redisClient *redis.Client,
	cfg *config.Config,
	proxyRepo ProxyRepository,
) *OpsAlertEvaluatorService {
	svc := NewOpsAlertEvaluatorService(opsService, opsRepo, emailService, redisClient, cfg, proxyRepo)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideOpsCleanupService creates and starts OpsCleanupService (cron scheduled).
func ProvideOpsCleanupService(
	opsRepo OpsRepository,
	db *sql.DB,
	redisClient *redis.Client,
	cfg *config.Config,
	channelMonitorSvc *ChannelMonitorService,
	settingRepo SettingRepository,
) *OpsCleanupService {
	svc := NewOpsCleanupService(opsRepo, db, redisClient, cfg, channelMonitorSvc, settingRepo)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

func ProvideOpsSystemLogSink(opsRepo OpsRepository) *OpsSystemLogSink {
	sink := NewOpsSystemLogSink(opsRepo)
	if runtimeBackgroundJobsEnabled() {
		sink.Start()
		logger.SetSink(sink)
	}
	return sink
}

// ProvideSoraMediaStorage 初始化 Sora 媒体存储
func ProvideSoraMediaStorage(cfg *config.Config) *SoraMediaStorage {
	return NewSoraMediaStorage(cfg)
}

func ProvideSoraS3Storage(settingService *SettingService) *SoraS3Storage {
	svc := NewSoraS3Storage(settingService)
	if settingService != nil {
		settingService.SetOnS3UpdateCallback(svc.RefreshClient)
	}
	return svc
}

func ProvideSoraSDKClient(
	cfg *config.Config,
	httpUpstream HTTPUpstream,
	tokenProvider *OpenAITokenProvider,
	accountRepo AccountRepository,
	soraAccountRepo SoraAccountRepository,
) *SoraSDKClient {
	client := NewSoraSDKClient(cfg, httpUpstream, tokenProvider)
	client.SetAccountRepositories(accountRepo, soraAccountRepo)
	return client
}

// ProvideSoraMediaCleanupService 创建并启动 Sora 媒体清理服务
func ProvideSoraMediaCleanupService(storage *SoraMediaStorage, cfg *config.Config) *SoraMediaCleanupService {
	svc := NewSoraMediaCleanupService(storage, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

func buildIdempotencyConfig(cfg *config.Config) IdempotencyConfig {
	idempotencyCfg := DefaultIdempotencyConfig()
	if cfg != nil {
		if cfg.Idempotency.DefaultTTLSeconds > 0 {
			idempotencyCfg.DefaultTTL = time.Duration(cfg.Idempotency.DefaultTTLSeconds) * time.Second
		}
		if cfg.Idempotency.SystemOperationTTLSeconds > 0 {
			idempotencyCfg.SystemOperationTTL = time.Duration(cfg.Idempotency.SystemOperationTTLSeconds) * time.Second
		}
		if cfg.Idempotency.ProcessingTimeoutSeconds > 0 {
			idempotencyCfg.ProcessingTimeout = time.Duration(cfg.Idempotency.ProcessingTimeoutSeconds) * time.Second
		}
		if cfg.Idempotency.FailedRetryBackoffSeconds > 0 {
			idempotencyCfg.FailedRetryBackoff = time.Duration(cfg.Idempotency.FailedRetryBackoffSeconds) * time.Second
		}
		if cfg.Idempotency.MaxStoredResponseLen > 0 {
			idempotencyCfg.MaxStoredResponseLen = cfg.Idempotency.MaxStoredResponseLen
		}
		idempotencyCfg.ObserveOnly = cfg.Idempotency.ObserveOnly
	}
	return idempotencyCfg
}

func ProvideIdempotencyCoordinator(repo IdempotencyRepository, cache IdempotencyCache, cfg *config.Config) *IdempotencyCoordinator {
	coordinator := NewIdempotencyCoordinator(repo, cache, buildIdempotencyConfig(cfg))
	SetDefaultIdempotencyCoordinator(coordinator)
	return coordinator
}

func ProvideSystemOperationLockService(repo IdempotencyRepository, cfg *config.Config) *SystemOperationLockService {
	return NewSystemOperationLockService(repo, buildIdempotencyConfig(cfg))
}

func ProvideIdempotencyCleanupService(repo IdempotencyRepository, cfg *config.Config) *IdempotencyCleanupService {
	svc := NewIdempotencyCleanupService(repo, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideScheduledTestService creates ScheduledTestService.
func ProvideScheduledTestService(
	planRepo ScheduledTestPlanRepository,
	resultRepo ScheduledTestResultRepository,
) *ScheduledTestService {
	return NewScheduledTestService(planRepo, resultRepo)
}

// ProvideScheduledTestRunnerService creates and starts ScheduledTestRunnerService.
func ProvideScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	svc := NewScheduledTestRunnerService(planRepo, scheduledSvc, accountTestSvc, rateLimitSvc, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvidePaymentConfigService wraps NewPaymentConfigService to accept the named
// payment.EncryptionKey type instead of raw []byte, avoiding Wire ambiguity.
func ProvidePaymentConfigService(entClient *dbent.Client, settingRepo SettingRepository, key payment.EncryptionKey) *PaymentConfigService {
	return NewPaymentConfigService(entClient, settingRepo, []byte(key))
}

// ProvideBalanceNotifyService creates BalanceNotifyService.
func ProvideBalanceNotifyService(emailService *EmailService, settingRepo SettingRepository, accountRepo AccountRepository, notificationEmailService *NotificationEmailService) *BalanceNotifyService {
	svc := NewBalanceNotifyService(emailService, settingRepo, accountRepo)
	svc.SetNotificationEmailService(notificationEmailService)
	return svc
}

// ProvidePaymentService creates PaymentService and attaches notification email delivery.
func ProvidePaymentService(entClient *dbent.Client, registry *payment.Registry, loadBalancer payment.LoadBalancer, redeemService *RedeemService, subscriptionSvc *SubscriptionService, configService *PaymentConfigService, userRepo UserRepository, groupRepo GroupRepository, affiliateService *AffiliateService, notificationEmailService *NotificationEmailService) *PaymentService {
	svc := NewPaymentService(entClient, registry, loadBalancer, redeemService, subscriptionSvc, configService, userRepo, groupRepo, affiliateService)
	svc.SetNotificationEmailService(notificationEmailService)
	return svc
}

// ProvidePaymentOrderExpiryService creates and starts PaymentOrderExpiryService.
func ProvidePaymentOrderExpiryService(paymentSvc *PaymentService, lockCache LeaderLockCache, db *sql.DB) *PaymentOrderExpiryService {
	svc := NewPaymentOrderExpiryService(paymentSvc, 60*time.Second)
	svc.SetLeaderLock(lockCache, db)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideChannelMonitorService creates channel monitor CRUD/runtime service.
func ProvideChannelMonitorService(repo ChannelMonitorRepository, encryptor SecretEncryptor, settingService *SettingService) *ChannelMonitorService {
	svc := NewChannelMonitorService(repo, encryptor)
	svc.SetRuntimeReader(settingService)
	return svc
}

// ProvideChannelMonitorV2Service wires runtime privacy settings into the v2 read API.
func ProvideChannelMonitorV2Service(repo ChannelMonitorV2Repository, settingService *SettingService) *ChannelMonitorV2Service {
	svc := NewChannelMonitorV2Service(repo)
	svc.SetRuntimeReader(settingService)
	return svc
}

// ProvideChannelMonitorV2Aggregator starts the passive v2 rollup worker unless it is explicitly disabled for local demos.
func ProvideChannelMonitorV2Aggregator(repo ChannelMonitorV2Repository, db *sql.DB, settingService *SettingService) *ChannelMonitorV2Aggregator {
	aggregator := NewChannelMonitorV2Aggregator(repo, db, settingService)
	if os.Getenv("CHANNEL_MONITOR_V2_DISABLE_AGGREGATOR") == "1" {
		return aggregator
	}
	aggregator.Start()
	return aggregator
}

// ProvideGrokOAuthService wires the configured capability policy and the shared
// Redis-backed single-use session store into Grok OAuth handling.
func ProvideGrokOAuthService(proxyRepo ProxyRepository, oauthClient GrokOAuthClient, cfg *config.Config, redisClient *redis.Client) *GrokOAuthService {
	svc := NewGrokOAuthService(proxyRepo, oauthClient, cfg)
	if redisClient != nil {
		svc = svc.WithSessionStore(xai.NewRedisSessionStore(redisClient))
	}
	return svc
}

// ProvideChannelMonitorRunner wires the monitor service to its scheduler and starts it.
func ProvideChannelMonitorRunner(svc *ChannelMonitorService, settingService *SettingService, quotaFetcher *ChannelMonitorQuotaFetcher) *ChannelMonitorRunner {
	r := NewChannelMonitorRunner(svc, settingService)
	if svc != nil {
		svc.SetRuntimeReader(settingService)
		svc.SetScheduler(r)
		svc.SetQuotaFetcher(quotaFetcher)
	}
	if singletonSchedulerServicesEnabled() {
		r.Start()
	}
	return r
}

func ProvideProxyMaintenanceService(
	planRepo ProxyMaintenancePlanRepository,
	resultRepo ProxyMaintenanceResultRepository,
	adminSvc AdminService,
	settingSvc *SettingService,
) *ProxyMaintenanceService {
	return NewProxyMaintenanceService(planRepo, resultRepo, adminSvc, settingSvc)
}

func ProvideProxyMaintenanceRunnerService(
	svc *ProxyMaintenanceService,
	cfg *config.Config,
) *ProxyMaintenanceRunnerService {
	runner := NewProxyMaintenanceRunnerService(svc, cfg)
	if singletonSchedulerServicesEnabled() {
		runner.Start()
	}
	return runner
}

// ProvideAccountModelsRefreshService creates and starts AccountModelsRefreshService.
func ProvideAccountModelsRefreshService(
	accountRepo AccountRepository,
	accountTestSvc *AccountTestService,
) *AccountModelsRefreshService {
	svc := NewAccountModelsRefreshService(accountRepo, accountTestSvc, defaultAccountModelsRefreshInterval)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideOpsScheduledReportService creates and starts OpsScheduledReportService.
func ProvideOpsScheduledReportService(
	opsService *OpsService,
	userService *UserService,
	emailService *EmailService,
	redisClient *redis.Client,
	cfg *config.Config,
) *OpsScheduledReportService {
	svc := NewOpsScheduledReportService(opsService, userService, emailService, redisClient, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideAPIKeyAuthCacheInvalidator 提供 API Key 认证缓存失效能力
func ProvideAPIKeyAuthCacheInvalidator(apiKeyService *APIKeyService) APIKeyAuthCacheInvalidator {
	// Start Pub/Sub subscriber for L1 cache invalidation across instances
	if requestPathCacheSyncEnabled() {
		apiKeyService.StartAuthCacheInvalidationSubscriber(context.Background())
	}
	return apiKeyService
}

// ProvideBackupService creates and starts BackupService
func ProvideBackupService(
	settingRepo SettingRepository,
	cfg *config.Config,
	encryptor SecretEncryptor,
	storeFactory BackupObjectStoreFactory,
	dumper DBDumper,
	lockCache LeaderLockCache,
	db *sql.DB,
) *BackupService {
	svc := NewBackupService(settingRepo, cfg, encryptor, storeFactory, dumper)
	svc.SetLeaderLock(lockCache, db)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideSettingService wires SettingService with group reader for default subscription validation.
func ProvideSettingService(settingRepo SettingRepository, groupRepo GroupRepository, proxyRepo ProxyRepository, cfg *config.Config) *SettingService {
	svc := NewSettingService(settingRepo, cfg)
	svc.SetDefaultSubscriptionGroupReader(groupRepo)
	svc.SetProxyRepository(proxyRepo)
	if err := svc.LoadForwardedClientIPSettings(context.Background()); err != nil {
		logger.LegacyPrintf("service.setting", "Warning: load forwarded client IP settings failed: %v", err)
	}
	if err := svc.MigrateOpenAIAllowClaudeCodeCodexPluginSetting(context.Background()); err != nil {
		logger.LegacyPrintf("service.setting", "Warning: migrate openai allow Claude Code Codex plugin setting failed: %v", err)
	}
	if err := svc.MigrateCodexBodyFingerprintToSignals(context.Background()); err != nil {
		logger.LegacyPrintf("service.setting", "Warning: migrate codex body fingerprint to signals failed: %v", err)
	}
	antigravity.SetUserAgentVersionResolver(svc.GetAntigravityUserAgentVersion)
	// enforceCodexIdentityHeaders 是所有 Codex 出站路径共用的纯函数收口点，拿不到 ctx，
	// 故注入无参解析器；解析器内部自带 60s TTL 缓存，热路径不触库。
	SetCodexCanonicalUserAgentResolver(func() string {
		return svc.GetOpenAICodexCanonicalUserAgent(context.Background())
	})
	return svc
}

func ProvideAccountImportService(
	accountStore AccountImportAccountStore,
	batchRepo AccountImportBatchRepository,
	proxyRepo ProxyRepository,
	groupRepo GroupRepository,
	soraAccountRepo SoraAccountRepository,
	schedulerSnapshot *SchedulerSnapshotService,
	cfg *config.Config,
) *AccountImportService {
	svc := NewAccountImportService(accountStore, batchRepo, proxyRepo, groupRepo, soraAccountRepo, schedulerSnapshot, cfg)
	if singletonSchedulerServicesEnabled() {
		svc.Start()
	}
	return svc
}

// ProvideGatewayService wires optional proxy failover dependencies onto GatewayService
// without forcing every unit test to pass them through the raw constructor.
func ProvideGatewayService(
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
	rateLimitService *RateLimitService,
	billingCacheService *BillingCacheService,
	identityService *IdentityService,
	httpUpstream HTTPUpstream,
	deferredService *DeferredService,
	claudeTokenProvider *ClaudeTokenProvider,
	sessionLimitCache SessionLimitCache,
	rpmCache RPMCache,
	digestStore *DigestSessionStore,
	settingService *SettingService,
	tlsFPProfileService *TLSFingerprintProfileService,
	channelService *ChannelService,
	resolver *ModelPricingResolver,
	compositeRouteResolver *CompositeRouteResolver,
	balanceNotifyService *BalanceNotifyService,
	userPlatformQuotaRepo UserPlatformQuotaRepository,
	kiroTokenProvider *KiroTokenProvider,
	kiroGatewayService *KiroGatewayService,
) *GatewayService {
	svc := NewGatewayService(
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
		rateLimitService,
		billingCacheService,
		identityService,
		httpUpstream,
		deferredService,
		claudeTokenProvider,
		sessionLimitCache,
		rpmCache,
		digestStore,
		settingService,
		tlsFPProfileService,
		channelService,
		resolver,
		compositeRouteResolver,
		balanceNotifyService,
		userPlatformQuotaRepo,
	)
	svc.SetKiroDeps(kiroTokenProvider, kiroGatewayService)
	svc.SetBillingShortfallNotifier(balanceNotifyService)
	rateLimitService.SetBalanceNotifyService(balanceNotifyService)
	return svc
}

// ProvideOpenAIOAuthService wires optional ChatWeb curl_cffi sidecar dependencies onto OpenAIOAuthService.
func ProvideOpenAIOAuthService(
	cfg *config.Config,
	proxyRepo ProxyRepository,
	oauthClient OpenAIOAuthClient,
) *OpenAIOAuthService {
	svc := NewOpenAIOAuthService(proxyRepo, oauthClient)
	if cfg == nil || !cfg.OpenAI.ChatWeb.CurlCFFISidecar.Enabled {
		return svc
	}

	sidecarClient, err := curlcffi.NewClient(curlcffi.Config{
		BaseURL:             cfg.OpenAI.ChatWeb.CurlCFFISidecar.BaseURL,
		Impersonate:         cfg.OpenAI.ChatWeb.CurlCFFISidecar.Impersonate,
		TimeoutSeconds:      cfg.OpenAI.ChatWeb.CurlCFFISidecar.TimeoutSeconds,
		SessionReuseEnabled: cfg.OpenAI.ChatWeb.CurlCFFISidecar.SessionReuseEnabled,
		SessionTTLSeconds:   cfg.OpenAI.ChatWeb.CurlCFFISidecar.SessionTTLSeconds,
	})
	if err != nil {
		logger.LegacyPrintf("wire.openai_oauth", "openai chatweb curl_cffi sidecar disabled: %v", err)
		return svc
	}
	svc.SetOpenAIChatWebCurlCFFISidecarClient(sidecarClient)
	return svc
}

// ProviderSet is the Wire provider set for all services
var ProviderSet = wire.NewSet(
	payment.ProviderSet,

	// Core services
	ProvideAuthService,
	NewPasskeyService,
	NewUserService,
	ProvideAPIKeyService,
	ProvideAPIKeyAuthCacheInvalidator,
	ProvideAuthCacheInvalidationWorker,
	NewGroupService,
	NewCompositeRouteResolver,
	NewAccountService,
	NewProxyService,
	NewRedeemService,
	NewPromoService,
	NewUsageService,
	NewDashboardService,
	NewDashboardOperationsService,
	ProvidePricingService,
	NewBillingService,
	ProvideBillingCacheService,
	NewAnnouncementService,
	NewAdminService,
	ProvideGatewayService,
	ProvideSoraS3Storage,
	ProvideSoraMediaStorage,
	ProvideSoraMediaCleanupService,
	NewSoraQuotaService,
	NewSoraGenerationService,
	ProvideChatWorkspaceServiceWithSub2APITextBridge,
	ProvideSoraSDKClient,
	wire.Bind(new(SoraClient), new(*SoraSDKClient)),
	NewSoraGatewayService,
	ProvideOpenAIGatewayService,
	ProvideImageStorageSettingService,
	ProvideImageTaskService,
	ProvideBatchImageModelPricingResolver,
	NewBatchImagePublicService,
	NewBatchImageDownloadService,
	ProvideBatchImageCleanupService,
	ProvideBatchImageWorkerRuntime,
	wire.Bind(new(AccountRuntimeBlocker), new(*OpenAIGatewayService)),
	NewOAuthService,
	ProvideOpenAIOAuthService,
	ProvideGrokOAuthService,
	wire.Bind(new(GrokOAuthTokenService), new(*GrokOAuthService)),
	NewGeminiOAuthService,
	NewGeminiQuotaService,
	NewCompositeTokenCacheInvalidator,
	wire.Bind(new(TokenCacheInvalidator), new(*CompositeTokenCacheInvalidator)),
	NewAntigravityOAuthService,
	ProvideOAuthRefreshAPI,
	ProvideGeminiTokenProvider,
	NewGeminiMessagesCompatService,
	ProvideAntigravityTokenProvider,
	ProvideGrokTokenProvider,
	ProvideKiroTokenProvider,
	NewKiroUsageService,
	NewKiroGatewayService,
	ProvideOpenAITokenProvider,
	ProvideOpenAIQuotaService,
	ProvideGrokQuotaService,
	ProvideCNProviderQuotaService,
	ProvideCNProviderBalanceService,
	ProvideCNProviderBalanceCheckService,
	ProvideClaudeTokenProvider,
	ProvideAntigravityGatewayService,
	ProvideRateLimitService,
	ProvideAccountUsageService,
	NewAccountExportService,
	ProvideAccountImportService,
	ProvideAccountTestService,
	ProvideUpstreamBillingProbeService,
	ProvideAccountModelsRefreshService,
	ProvideOllamaCloudUsageService,
	ProvideSettingService,
	NewDataManagementService,
	ProvideBackupService,
	ProvideOpsSystemLogSink,
	ProvideOpsService,
	ProvideOpsIngressRejectAggregator,
	ProvideAuditLogService,
	ProvideOpsMetricsCollector,
	ProvideOpsAggregationService,
	ProvideOpsAlertEvaluatorService,
	ProvideOpsCleanupService,
	ProvideOpsScheduledReportService,
	NewEmailService,
	NewNotificationEmailService,
	ProvideEmailQueueService,
	NewTurnstileService,
	NewTencentCaptchaService,
	NewAliyunCaptchaService,
	NewSubscriptionService,
	wire.Bind(new(DefaultSubscriptionAssigner), new(*SubscriptionService)),
	ProvideConcurrencyService,
	ProvideWorkspaceWebSearchTool,
	NewWorkspaceToolService,
	ProvideWorkspaceWebSearchService,
	ProvideUserMessageQueueService,
	NewUsageRecordWorkerPool,
	ProvideSchedulerSnapshotServiceWithAdmission,
	NewIdentityService,
	NewCRSSyncService,
	ProvideUpdateService,
	ProvideTokenRefreshService,
	wire.Bind(new(GrokOAuthReconciler), new(*TokenRefreshService)),
	ProvideAccountExpiryService,
	ProvideOpenAICodexVersionSyncService,
	ProvideProxyExpiryService,
	ProvideSubscriptionExpiryService,
	ProvideTimingWheelService,
	ProvideDashboardAggregationService,
	ProvideUsageCleanupService,
	ProvideDeferredService,
	NewAntigravityQuotaFetcher,
	NewGrokQuotaFetcher,
	NewUserAttributeService,
	NewUsageCache,
	NewTotpService,
	NewErrorPassthroughService,
	NewTLSFingerprintProfileService,
	NewAffiliateService,
	NewResellerService,
	NewDigestSessionStore,
	ProvideIdempotencyCoordinator,
	ProvideSystemOperationLockService,
	ProvideIdempotencyCleanupService,
	ProvideScheduledTestService,
	ProvideScheduledTestRunnerService,
	ProvideProxyMaintenanceService,
	ProvideProxyMaintenanceRunnerService,
	NewGroupCapacityService,
	NewChannelService,
	ProvideWorkspaceSelectedModelCatalogChannelLister,
	wire.Bind(new(ChannelCacheInvalidator), new(*ChannelService)),
	NewModelPricingResolver,
	NewContentModerationService,
	ProvidePaymentConfigService,
	ProvidePaymentService,
	ProvidePaymentOrderExpiryService,
	ProvideBalanceNotifyService,
	ProvideChannelMonitorService,
	ProvideChannelMonitorRunner,
	NewChannelMonitorQuotaFetcher,
	ProvideChannelMonitorV2Service,
	ProvideChannelMonitorV2Aggregator,
	NewChannelMonitorRequestTemplateService,
	ProvideUserPlatformQuotaUsageFlusher,
)

func ProvideWorkspaceSelectedModelCatalogChannelLister(channelService *ChannelService) WorkspaceSelectedModelCatalogChannelLister {
	return channelService
}

func ProvideWorkspaceWebSearchService(toolService *WorkspaceToolService) WorkspaceWebSearchService {
	return toolService
}
