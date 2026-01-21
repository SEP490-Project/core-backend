package config

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AdminConfig holds the admin-related configuration settings
type AdminConfig struct {
	PayOSLinkExpiry                    int   `mapstructure:"payos_link_expiry"`
	MinimumDayBeforeContractPaymentDue int   `mapstructure:"minimum_day_before_contract_payment_due"`
	ForgetPasswordExpiryInSeconds      int64 `mapstructure:"forget_password_expiry_in_seconds"`

	// Representative Information
	RepresentativeName              string `mapstructure:"representative_name"`
	RepresentativeRole              string `mapstructure:"representative_role"`
	RepresentativePhone             string `mapstructure:"representative_phone"`
	RepresentativeEmail             string `mapstructure:"representative_email"`
	RepresentativeTaxNumber         string `mapstructure:"representative_tax_number"`
	RepresentativeBankName          string `mapstructure:"representative_bank_name"`
	RepresentativeBankAccountNumber string `mapstructure:"representative_bank_account_number"`
	RepresentativeBankAccountHolder string `mapstructure:"representative_bank_account_holder"`
	RepresentativeCompanyAddress    string `mapstructure:"representative_company_address"`
	// Representative use to create GHN Order
	RepresentativeGHNCompanyName  string `mapstructure:"representative_company_name"` // From name: max:1024
	RepresentativeGHNPhone        string `mapstructure:"representative_ghn_phone"`
	RepresentativeGHNWardName     string `mapstructure:"representative_ghn_ward_name"`
	RepresentativeGHNDistrictName string `mapstructure:"representative_ghn_district_name"`
	RepresentativeGHNProvinceName string `mapstructure:"representative_ghn_province_name"`

	// Affiliate Link Tracking Configuration
	TrackingLinkTrustedDomains []string `mapstructure:"tracking_link_trusted_domains"`
	BotSignatures              []string `mapstructure:"bot_signatures"`

	// Cron Jobs Configuration
	CTRAggregationEnabled               bool   `mapstructure:"ctr_aggregation_enabled" job:"ctr_aggregation_job"`
	CTRAggregationIntervalMinutes       int    `mapstructure:"ctr_aggregation_interval_minutes" job:"ctr_aggregation_job"`
	ExpiredLinkCleanupEnabled           bool   `mapstructure:"expired_link_cleanup_enabled" job:"expired_link_cleanup_job"`
	ExpiredLinkCleanupCronExpr          string `mapstructure:"expired_link_cleanup_cron_expr" job:"expired_link_cleanup_job"`
	PayOSExpiryCheckEnabled             bool   `mapstructure:"payos_expiry_check_enabled" job:"payos_expiry_check_job"`
	PayOSExpiryCheckIntervalMinutes     int    `mapstructure:"payos_expiry_check_interval_minutes" job:"payos_expiry_check_job"`
	PreOrderOpeningCheckEnable          bool   `mapstructure:"preorder_opening_check_enabled" job:"pre_order_opening_check_job"`
	PreOrderOpeningCheckIntervalMinutes int    `mapstructure:"preorder_opening_check_interval_minutes" job:"pre_order_opening_check_job"`
	TikTokStatusPollerEnabled           bool   `mapstructure:"tiktok_status_poller_enabled" job:"tiktok_status_poller_job"`
	TikTokStatusPollerCronExpr          string `mapstructure:"tiktok_status_poller_cron_expr" job:"tiktok_status_poller_job"`
	ContentMetricsPollerEnabled         bool   `mapstructure:"content_metrics_poller_enabled" job:"content_metrics_poller_job"`
	ContentMetricsPollerCronExpr        string `mapstructure:"content_metrics_poller_cron_expr" job:"content_metrics_poller_job"`
	DailyCronJobEnabled                 bool   `mapstructure:"daily_cron_job_enabled" job:"daily_job"`
	DailyCronJobCronExpr                string `mapstructure:"daily_cron_job_cron_expr" job:"daily_job"`
	DailyCronJobWorkerCount             int    `mapstructure:"daily_cron_job_worker_count" job:"daily_job"`

	// Daily Job configuration fields
	ContractPaymentAllowedOverdueDays int `mapstructure:"contract_payment_allowed_overdue_days"`
	ContractPaymentNotificationHour   int `mapstructure:"contract_payment_notification_hour"`

	// Order - PreOrder
	CensorshipIntervalMinutes     int   `mapstructure:"censorship_interval_minutes"`
	AutoReceiveOrderIntervalMs    int64 `mapstructure:"auto_receive_order_interval_ms"`    // Milliseconds (default: 72 hours = 259200000ms)
	AutoReceivePreOrderIntervalMs int64 `mapstructure:"auto_receive_preorder_interval_ms"` // Milliseconds (default: 30 days = 2592000000ms)

	// Products
	ProductMaximumVariants int `mapstructure:"product_maximum_variants"`

	// Social Media Integration
	// This is used to determine when to send notifications for expiring OAuth tokens

	// ========= Facebook =========
	FacebookHomepageURL                  string `mapstructure:"facebook_homepage_url"`
	FacebookExpiryThresholdNotifications int    `mapstructure:"facebook_expiry_threshold_notifications"` // in days
	FacebookVideoUploadChunkSizeInMB     int    `mapstructure:"facebook_video_upload_chunk_size_in_mb"`
	FacebookVideoUploadMaxRetries        int    `mapstructure:"facebook_video_upload_max_retries"`

	// ========= TikTok =========
	TikTokHomepageURL                  string `mapstructure:"tiktok_homepage_url"`
	TikTokExpiryThresholdNotifications int    `mapstructure:"tiktok_expiry_threshold_notifications"` // in days

	// ======== General ========
	SystemEmail   string `mapstructure:"system_email"`
	SystemName    string `mapstructure:"system_name"`
	TermOfService string `mapstructure:"term_of_service"`
	PrivacyPolicy string `mapstructure:"privacy_policy"`

	// AI Content Generation
	ContentGenerationPromptTemplate string `mapstructure:"content_generation_prompt_template" type:"textarea"`

	// Affiliate Links config
	AffiliateHashLength int    `mapstructure:"affiliate_hash_length"`
	AffiliateURLFormat  string `mapstructure:"affiliate_url_format"`

	// Cache TTLs
	ContentViewUniqueCacheTTLHours int `mapstructure:"content_view_unique_cache_ttl_hours"`

	// Contract Violation Configuration
	ViolationPaymentDeadlineDays int `mapstructure:"violation_payment_deadline_days"`
	ViolationProofMaxAttempts    int `mapstructure:"violation_proof_max_attempts"`                // Max times KOL can resubmit rejected proof
	ViolationProofReviewDays     int `mapstructure:"violation_proof_review_days" job:"daily_job"` // Days brand has to review proof before auto-approval

	// CO_PRODUCING Refund Settings
	CoProducingRefundProofMaxAttempts int `mapstructure:"co_producing_refund_proof_max_attempts"`          // Max times Marketing can resubmit rejected proof
	CoProducingRefundReviewDays       int `mapstructure:"co_producing_refund_review_days" job:"daily_job"` // Days brand has to review proof before auto-approval
}

// loadAdminConfig loads the admin configuration from file and environment variables
func loadAdminConfig(configPath string) error {
	adminViper := viper.New()

	// Priprity 1: Default values
	setDefaultAdminConfig(adminViper)

	// Priority 2: Config file
	adminViper.AddConfigPath(configPath)
	adminViper.SetConfigName("admin_config")
	adminViper.SetConfigType("yaml")

	if err := adminViper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Println("Config file not found, using defaults and environment variables.")
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Priority 3: Environment variables
	adminViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	adminViper.AutomaticEnv()

	var adminConfig AdminConfig
	if err := adminViper.Unmarshal(&adminConfig); err != nil {
		fmt.Printf("Unable to decode admin config into struct: %v\n", err)
	}

	appConfig.AdminConfig = adminConfig

	return nil
}

func setDefaultAdminConfig(adminViper *viper.Viper) {
	adminViper.SetDefault("payos_link_expiry", 300)
	adminViper.SetDefault("minimum_day_before_contract_payment_due", 5)
	adminViper.SetDefault("forget_password_expiry_in_seconds", 300)

	adminViper.SetDefault("representative_name", "Đinh Thị Ngọc Trinh")
	adminViper.SetDefault("representative_role", "Beauty Blogger")
	adminViper.SetDefault("representative_phone", "+84917956697")
	adminViper.SetDefault("representative_email", "mrstrinh.work@gmail.com")
	adminViper.SetDefault("representative_tax_number", "01234567890")
	adminViper.SetDefault("representative_bank_name", "")
	adminViper.SetDefault("representative_bank_account_number", "")
	adminViper.SetDefault("representative_bank_account_holder", "TRAN GIANH KHANH")
	// GHN Info
	adminViper.SetDefault("representative_company_name", "Công ty TNHH Thương Mại Dịch Vụ BShowSell")
	adminViper.SetDefault("representative_ghn_phone", "0912312312")
	adminViper.SetDefault("representative_ghn_ward_name", "phường Long Thạnh Mỹ")
	adminViper.SetDefault("representative_ghn_district_name", "Thủ Đức")
	adminViper.SetDefault("representative_ghn_province_name", "TP Hồ Chí Minh")

	adminViper.SetDefault("tracking_link_trusted_domains", []string{"example.com", "trustedpartner.com"})
	adminViper.SetDefault("bot_signatures", []string{"example.com", "trustedpartner.com"})

	adminViper.SetDefault("tiktok_status_poller_enabled", true)
	adminViper.SetDefault("tiktok_status_poller_interval_seconds", 30)

	adminViper.SetDefault("social_metrics_poller_enabled", true)
	adminViper.SetDefault("social_metrics_poller_interval_minutes", 10)

	adminViper.SetDefault("facebook_homepage_url", "https://www.facebook.com/")
	adminViper.SetDefault("facebook_expiry_threshold_notifications", 7)
	adminViper.SetDefault("facebook_video_upload_chunk_size_in_mb", 50)
	adminViper.SetDefault("facebook_video_upload_max_retries", 3)

	adminViper.SetDefault("tiktok_homepage_url", "https://www.tiktok.com/")
	adminViper.SetDefault("tiktok_expiry_threshold_notifications", 7)

	// Webhook secrets (should be set via environment variables for security)
	adminViper.SetDefault("facebook_webhook_secret", "")
	adminViper.SetDefault("tiktok_webhook_secret", "")

	adminViper.SetDefault("content_generation_prompt_template", "Default prompt template")

	adminViper.SetDefault("affiliate_hash_length", 16)
	adminViper.SetDefault("affiliate_url_format", "%s/r/%s")

	adminViper.SetDefault("content_view_unique_cache_ttl_hours", 24)

	adminViper.SetDefault("contract_payment_allowed_overdue_days", 5)
	adminViper.SetDefault("contract_payment_notification_hour", 8)

	// Contract Violation Settings
	adminViper.SetDefault("violation_proof_max_attempts", 3)
	adminViper.SetDefault("violation_proof_review_days", 7)

	// CO_PRODUCING Refund Settings
	adminViper.SetDefault("co_producing_refund_proof_max_attempts", 3)
	adminViper.SetDefault("co_producing_refund_review_days", 7)

	//Mock
	//products
	adminViper.SetDefault("product_maximum_variants", 3)
	//orders - Auto receive intervals in milliseconds
	adminViper.SetDefault("auto_receive_order_interval_ms", 259200000)     // 72 hours = 72 * 60 * 60 * 1000 ms
	adminViper.SetDefault("auto_receive_preorder_interval_ms", 2592000000) // 30 days = 30 * 24 * 60 * 60 * 1000 ms
}

// Override updates AdminConfig with values from the the model that was retrieved from the database
func (c *AdminConfig) Override(models []model.Config) error {
	// Create a map from config key to model.Config for easy lookup
	configMap := utils.MapKeyFromSlice(models, func(m model.Config) (string, model.Config) { return m.Key, m })

	// Override fields if they exist in the configMap with reflect
	overridenCount := 0
	val := reflect.ValueOf(c)
	typ := reflect.TypeFor[*AdminConfig]()
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
		typ = typ.Elem()
	}
	for i := 0; i < val.NumField(); i++ {
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("mapstructure")

		if cfg, exists := configMap[tag]; exists {
			zap.L().Debug("Overriding AdminConfig field from database",
				zap.String("tag", tag), zap.String("field", fieldType.Name), zap.String("value", cfg.Value))
			if err := utils.SetStringToReflectValue(c, fieldType.Name, cfg.Value, true); err == nil {
				overridenCount++
			} else {
				zap.L().Error("Failed to override AdminConfig field",
					zap.String("field", tag), zap.String("value", cfg.Value), zap.Error(err))
			}
		}
	}

	zap.L().Info("AdminConfig overridden with database values successfully",
		zap.Int("config_count", len(configMap)), zap.Int("overriden_count", overridenCount))
	return nil
}
