package service

import (
	"context"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"net/http"
	"time"
)

type AdminService interface {
	ListUsers(ctx context.Context, page, pageSize int, filters UserListFilters) ([]User, int64, error)
	GetUser(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, input *CreateUserInput) (*User, error)
	UpdateUser(ctx context.Context, id int64, input *UpdateUserInput) (*User, error)
	DeleteUser(ctx context.Context, id int64) error
	UpdateUserBalance(ctx context.Context, userID int64, balance float64, operation string, notes string) (*User, error)
	GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int) ([]APIKey, int64, error)
	GetUserUsageStats(ctx context.Context, userID int64, period string) (any, error)
	GetUserBalanceHistory(ctx context.Context, userID int64, page, pageSize int, codeType string) ([]RedeemCode, int64, float64, error)
	ListGroups(ctx context.Context, page, pageSize int, platform, status, search string, isExclusive *bool) ([]Group, int64, error)
	GetAllGroups(ctx context.Context) ([]Group, error)
	GetAllGroupsByPlatform(ctx context.Context, platform string) ([]Group, error)
	GetGroup(ctx context.Context, id int64) (*Group, error)
	GetGroupByName(ctx context.Context, name string) (*Group, error)
	CreateGroup(ctx context.Context, input *CreateGroupInput) (*Group, error)
	UpdateGroup(ctx context.Context, id int64, input *UpdateGroupInput) (*Group, error)
	DeleteGroup(ctx context.Context, id int64) error
	GetGroupAPIKeys(ctx context.Context, groupID int64, page, pageSize int) ([]APIKey, int64, error)
	UpdateGroupSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error
	GetGroupRateMultipliers(ctx context.Context, groupID int64) ([]UserGroupRateEntry, error)
	ClearGroupRateMultipliers(ctx context.Context, groupID int64) error
	BatchSetGroupRateMultipliers(ctx context.Context, groupID int64, entries []GroupRateMultiplierInput) error
	AdminUpdateAPIKeyGroupID(ctx context.Context, keyID int64, groupID *int64, modelDisplayMode *string) (*AdminUpdateAPIKeyGroupIDResult, error)
	ReplaceUserGroup(ctx context.Context, userID, oldGroupID, newGroupID int64) (*ReplaceUserGroupResult, error)
	ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]Account, int64, error)
	GetAccountStatusSummary(ctx context.Context, filters AccountStatusSummaryFilters) (*AccountStatusSummary, error)
	ListArchivedGroups(ctx context.Context, filters ArchivedAccountGroupFilters) ([]ArchivedAccountGroupSummary, error)
	GetAccount(ctx context.Context, id int64) (*Account, error)
	GetAccountsByIDs(ctx context.Context, ids []int64) ([]*Account, error)
	CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error)
	UpdateAccount(ctx context.Context, id int64, input *UpdateAccountInput) (*Account, error)
	BackfillAccountModelPolicies(ctx context.Context, registry *ModelRegistryService, pageSize int) (*AccountModelPolicyBackfillResult, error)
	DeleteAccount(ctx context.Context, id int64) error
	BatchDeleteBlacklistedAccounts(ctx context.Context, ids []int64, deleteAll bool) (*BlacklistedBatchDeleteResult, error)
	RefreshAccountCredentials(ctx context.Context, id int64) (*Account, error)
	ClearAccountError(ctx context.Context, id int64) (*Account, error)
	SetAccountError(ctx context.Context, id int64, errorMsg string) error
	SetAccountSchedulable(ctx context.Context, id int64, schedulable bool) (*Account, error)
	EnsureOpenAIPrivacy(ctx context.Context, account *Account) string
	EnsureAntigravityPrivacy(ctx context.Context, account *Account) string
	ForceOpenAIPrivacy(ctx context.Context, account *Account) string
	BlacklistAccount(ctx context.Context, id int64, input *BlacklistAccountInput) (*Account, error)
	RestoreBlacklistedAccount(ctx context.Context, id int64) (*Account, error)
	UnarchiveAccounts(ctx context.Context, input *UnarchiveAccountsInput) (*UnarchiveAccountsResult, error)
	BulkUpdateAccounts(ctx context.Context, input *BulkUpdateAccountsInput) (*BulkUpdateAccountsResult, error)
	CheckMixedChannelRisk(ctx context.Context, currentAccountID int64, currentAccountPlatform string, groupIDs []int64) error
	ListProxies(ctx context.Context, page, pageSize int, protocol, status, search string) ([]Proxy, int64, error)
	ListProxiesWithAccountCount(ctx context.Context, page, pageSize int, protocol, status, search string) ([]ProxyWithAccountCount, int64, error)
	GetAllProxies(ctx context.Context) ([]Proxy, error)
	GetAllProxiesWithAccountCount(ctx context.Context) ([]ProxyWithAccountCount, error)
	GetProxy(ctx context.Context, id int64) (*Proxy, error)
	GetProxiesByIDs(ctx context.Context, ids []int64) ([]Proxy, error)
	CreateProxy(ctx context.Context, input *CreateProxyInput) (*Proxy, error)
	UpdateProxy(ctx context.Context, id int64, input *UpdateProxyInput) (*Proxy, error)
	DeleteProxy(ctx context.Context, id int64) error
	BatchDeleteProxies(ctx context.Context, ids []int64) (*ProxyBatchDeleteResult, error)
	GetProxyAccounts(ctx context.Context, proxyID int64) ([]ProxyAccountSummary, error)
	CheckProxyExists(ctx context.Context, host string, port int, username, password string) (bool, error)
	TestProxy(ctx context.Context, id int64) (*ProxyTestResult, error)
	CheckProxyQuality(ctx context.Context, id int64) (*ProxyQualityCheckResult, error)
	ListRedeemCodes(ctx context.Context, page, pageSize int, codeType, status, search string) ([]RedeemCode, int64, error)
	GetRedeemCode(ctx context.Context, id int64) (*RedeemCode, error)
	GenerateRedeemCodes(ctx context.Context, input *GenerateRedeemCodesInput) ([]RedeemCode, error)
	DeleteRedeemCode(ctx context.Context, id int64) error
	BatchDeleteRedeemCodes(ctx context.Context, ids []int64) (int64, error)
	ExpireRedeemCode(ctx context.Context, id int64) (*RedeemCode, error)
	ResetAccountQuota(ctx context.Context, id int64) error
}
type CreateUserInput struct {
	Email         string
	Password      string
	Username      string
	Notes         string
	Balance       float64
	Concurrency   int
	AllowedGroups []int64
}
type UpdateUserInput struct {
	Email                string
	Password             string
	Username             *string
	Notes                *string
	Balance              *float64
	Concurrency          *int
	AdminFreeBilling     *bool
	RequestDetailsReview *bool
	Status               string
	AllowedGroups        *[]int64
	GroupRates           map[int64]*float64
}
type CreateGroupInput struct {
	Name                            string
	Description                     string
	Platform                        string
	Priority                        int
	RateMultiplier                  float64
	IsExclusive                     bool
	SubscriptionType                string
	DailyLimitUSD                   *float64
	WeeklyLimitUSD                  *float64
	MonthlyLimitUSD                 *float64
	ImagePrice1K                    *float64
	ImagePrice2K                    *float64
	ImagePrice4K                    *float64
	ImageProtocolMode               string
	ClaudeCodeOnly                  bool
	FallbackGroupID                 *int64
	FallbackGroupIDOnInvalidRequest *int64
	ModelRouting                    map[string][]int64
	ModelRoutingEnabled             bool
	GeminiMixedProtocolEnabled      bool
	MCPXMLInject                    *bool
	SupportedModelScopes            []string
	AllowMessagesDispatch           bool
	DefaultMappedModel              string
	CopyAccountsFromGroupIDs        []int64
}
type UpdateGroupInput struct {
	Name                            string
	Description                     string
	Platform                        string
	Priority                        *int
	RateMultiplier                  *float64
	IsExclusive                     *bool
	Status                          string
	SubscriptionType                string
	DailyLimitUSD                   *float64
	WeeklyLimitUSD                  *float64
	MonthlyLimitUSD                 *float64
	ImagePrice1K                    *float64
	ImagePrice2K                    *float64
	ImagePrice4K                    *float64
	ImageProtocolMode               string
	ClaudeCodeOnly                  *bool
	FallbackGroupID                 *int64
	FallbackGroupIDOnInvalidRequest *int64
	ModelRouting                    map[string][]int64
	ModelRoutingEnabled             *bool
	GeminiMixedProtocolEnabled      *bool
	MCPXMLInject                    *bool
	SupportedModelScopes            *[]string
	AllowMessagesDispatch           *bool
	DefaultMappedModel              *string
	CopyAccountsFromGroupIDs        []int64
}
type CreateAccountInput struct {
	Name                   string
	Notes                  *string
	Platform               string
	Type                   string
	Credentials            map[string]any
	Extra                  map[string]any
	ProxyID                *int64
	Concurrency            int
	Priority               int
	RateMultiplier         *float64
	LoadFactor             *int
	GroupIDs               []int64
	Status                 string
	LifecycleState         string
	LifecycleReasonCode    string
	LifecycleReasonMessage string
	ExpiresAt              *int64
	AutoPauseOnExpired     *bool
	SkipDefaultGroupBind   bool
	SkipMixedChannelCheck  bool
}
type UpdateAccountInput struct {
	Name                  string
	Notes                 *string
	Type                  string
	Credentials           map[string]any
	Extra                 map[string]any
	ProxyID               *int64
	Concurrency           *int
	Priority              *int
	RateMultiplier        *float64
	LoadFactor            *int
	Status                string
	GroupIDs              *[]int64
	ExpiresAt             *int64
	AutoPauseOnExpired    *bool
	SkipMixedChannelCheck bool
}
type BulkUpdateAccountsInput struct {
	AccountIDs             []int64
	Name                   string
	ProxyID                *int64
	Concurrency            *int
	Priority               *int
	RateMultiplier         *float64
	LoadFactor             *int
	Status                 string
	Schedulable            *bool
	GroupIDs               *[]int64
	Credentials            map[string]any
	Extra                  map[string]any
	LifecycleState         string
	LifecycleReasonCode    string
	LifecycleReasonMessage string
	SkipMixedChannelCheck  bool
}
type BulkUpdateAccountResult struct {
	AccountID int64  `json:"account_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}
type BlacklistAccountInput struct {
	Source   string                  `json:"source,omitempty"`
	Feedback *BlacklistFeedbackInput `json:"feedback,omitempty"`
}
type AdminUpdateAPIKeyGroupIDResult struct {
	APIKey                 *APIKey
	AutoGrantedGroupAccess bool
	GrantedGroupID         *int64
	GrantedGroupName       string
}
type AdminAPIKeyGroupUpdateInput struct {
	GroupID       int64
	Quota         float64
	ModelPatterns []string
}
type AdminGrantedGroupAccess struct {
	GroupID   int64
	GroupName string
}
type AdminUpdateAPIKeyGroupsResult struct {
	APIKey                 *APIKey
	AutoGrantedGroupAccess bool
	GrantedGroupID         *int64
	GrantedGroupName       string
	GrantedGroups          []AdminGrantedGroupAccess
}
type ReplaceUserGroupResult struct {
	MigratedKeys int64
}
type BulkUpdateAccountsResult struct {
	Success    int                       `json:"success"`
	Failed     int                       `json:"failed"`
	SuccessIDs []int64                   `json:"success_ids"`
	FailedIDs  []int64                   `json:"failed_ids"`
	Results    []BulkUpdateAccountResult `json:"results"`
}
type CreateProxyInput struct {
	Name     string
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
}
type UpdateProxyInput struct {
	Name     string
	Protocol string
	Host     string
	Port     int
	Username string
	Password string
	Status   string
}
type GenerateRedeemCodesInput struct {
	Count        int
	Type         string
	Value        float64
	GroupID      *int64
	ValidityDays int
}
type ProxyBatchDeleteResult struct {
	DeletedIDs []int64                   `json:"deleted_ids"`
	Skipped    []ProxyBatchDeleteSkipped `json:"skipped"`
}
type ProxyBatchDeleteSkipped struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}
type BlacklistedBatchDeleteResult struct {
	DeletedIDs   []int64                         `json:"deleted_ids"`
	Failed       []BlacklistedBatchDeleteFailure `json:"failed"`
	DeletedCount int                             `json:"deleted_count"`
	FailedCount  int                             `json:"failed_count"`
}
type BlacklistedBatchDeleteFailure struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}
type ProxyTestResult struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	LatencyMs   int64  `json:"latency_ms,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	City        string `json:"city,omitempty"`
	Region      string `json:"region,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
}
type ProxyQualityCheckResult struct {
	ProxyID        int64                   `json:"proxy_id"`
	Score          int                     `json:"score"`
	Grade          string                  `json:"grade"`
	Summary        string                  `json:"summary"`
	ExitIP         string                  `json:"exit_ip,omitempty"`
	Country        string                  `json:"country,omitempty"`
	CountryCode    string                  `json:"country_code,omitempty"`
	BaseLatencyMs  int64                   `json:"base_latency_ms,omitempty"`
	PassedCount    int                     `json:"passed_count"`
	WarnCount      int                     `json:"warn_count"`
	FailedCount    int                     `json:"failed_count"`
	ChallengeCount int                     `json:"challenge_count"`
	CheckedAt      int64                   `json:"checked_at"`
	Items          []ProxyQualityCheckItem `json:"items"`
}
type ProxyQualityCheckItem struct {
	Target     string `json:"target"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	LatencyMs  int64  `json:"latency_ms,omitempty"`
	Message    string `json:"message,omitempty"`
	CFRay      string `json:"cf_ray,omitempty"`
}
type ProxyExitInfo struct {
	IP          string
	City        string
	Region      string
	Country     string
	CountryCode string
}
type ProxyExitInfoProber interface {
	ProbeProxy(ctx context.Context, proxyURL string) (*ProxyExitInfo, int64, error)
}
type proxyQualityTarget struct {
	Target          string
	URL             string
	Method          string
	AllowedStatuses map[int]struct{}
}

var proxyQualityTargets = []proxyQualityTarget{{Target: "openai", URL: "https://api.openai.com/v1/models", Method: http.MethodGet, AllowedStatuses: map[int]struct{}{http.StatusUnauthorized: {}}}, {Target: "anthropic", URL: "https://api.anthropic.com/v1/messages", Method: http.MethodGet, AllowedStatuses: map[int]struct{}{http.StatusUnauthorized: {}, http.StatusMethodNotAllowed: {}, http.StatusNotFound: {}, http.StatusBadRequest: {}}}, {Target: "gemini", URL: "https://generativelanguage.googleapis.com/$discovery/rest?version=v1beta", Method: http.MethodGet, AllowedStatuses: map[int]struct{}{http.StatusOK: {}}}}

const (
	proxyQualityRequestTimeout        = 15 * time.Second
	proxyQualityResponseHeaderTimeout = 10 * time.Second
	proxyQualityMaxBodyBytes          = int64(8 * 1024)
	proxyQualityClientUserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"
)

type adminServiceImpl struct {
	userRepo             UserRepository
	groupRepo            GroupRepository
	accountRepo          AccountRepository
	proxyRepo            ProxyRepository
	apiKeyRepo           APIKeyRepository
	redeemCodeRepo       RedeemCodeRepository
	userGroupRateRepo    UserGroupRateRepository
	billingCacheService  *BillingCacheService
	proxyProber          ProxyExitInfoProber
	proxyLatencyCache    ProxyLatencyCache
	authCacheInvalidator APIKeyAuthCacheInvalidator
	privacyClientFactory PrivacyClientFactory
	entClient            *dbent.Client
	settingService       *SettingService
	defaultSubAssigner   DefaultSubscriptionAssigner
	userSubRepo          UserSubscriptionRepository
}
type userGroupRateBatchReader interface {
	GetByUserIDs(ctx context.Context, userIDs []int64) (map[int64]map[int64]float64, error)
}
type groupExistenceBatchReader interface {
	ExistsByIDs(ctx context.Context, ids []int64) (map[int64]bool, error)
}

func NewAdminService(userRepo UserRepository, groupRepo GroupRepository, accountRepo AccountRepository, proxyRepo ProxyRepository, apiKeyRepo APIKeyRepository, redeemCodeRepo RedeemCodeRepository, userGroupRateRepo UserGroupRateRepository, billingCacheService *BillingCacheService, proxyProber ProxyExitInfoProber, proxyLatencyCache ProxyLatencyCache, authCacheInvalidator APIKeyAuthCacheInvalidator, privacyClientFactory PrivacyClientFactory, entClient *dbent.Client, settingService *SettingService, defaultSubAssigner DefaultSubscriptionAssigner, userSubRepo UserSubscriptionRepository) AdminService {
	return &adminServiceImpl{userRepo: userRepo, groupRepo: groupRepo, accountRepo: accountRepo, proxyRepo: proxyRepo, apiKeyRepo: apiKeyRepo, redeemCodeRepo: redeemCodeRepo, userGroupRateRepo: userGroupRateRepo, billingCacheService: billingCacheService, proxyProber: proxyProber, proxyLatencyCache: proxyLatencyCache, authCacheInvalidator: authCacheInvalidator, privacyClientFactory: privacyClientFactory, entClient: entClient, settingService: settingService, defaultSubAssigner: defaultSubAssigner, userSubRepo: userSubRepo}
}
func (s *adminServiceImpl) CheckProxyExists(ctx context.Context, host string, port int, username, password string) (bool, error) {
	return s.proxyRepo.ExistsByHostPortAuth(ctx, host, port, username, password)
}
func (s *adminServiceImpl) probeProxyLatency(ctx context.Context, proxy *Proxy) {
	if s.proxyProber == nil || proxy == nil {
		return
	}
	exitInfo, latencyMs, err := s.proxyProber.ProbeProxy(ctx, proxy.URL())
	if err != nil {
		s.saveProxyLatency(ctx, proxy.ID, &ProxyLatencyInfo{Success: false, Message: err.Error(), UpdatedAt: time.Now()})
		return
	}
	latency := latencyMs
	s.saveProxyLatency(ctx, proxy.ID, &ProxyLatencyInfo{Success: true, LatencyMs: &latency, Message: "Proxy is accessible", IPAddress: exitInfo.IP, Country: exitInfo.Country, CountryCode: exitInfo.CountryCode, Region: exitInfo.Region, City: exitInfo.City, UpdatedAt: time.Now()})
}
func (s *adminServiceImpl) ResetAccountQuota(ctx context.Context, id int64) error {
	return s.accountRepo.ResetQuotaUsed(ctx, id)
}
