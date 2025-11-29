package config

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
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

	// Affiliate Link Tracking Configuration
	TrackingLinkTrustedDomains []string `mapstructure:"tracking_link_trusted_domains"`
	BotSignatures              []string `mapstructure:"bot_signatures"`

	// Cron Jobs Configuration
	CTRAggregationEnabled               bool   `mapstructure:"ctr_aggregation_enabled"`
	CTRAggregationIntervalMinutes       int    `mapstructure:"ctr_aggregation_interval_minutes"`
	ExpiredContractCleanupEnabled       bool   `mapstructure:"expired_contract_cleanup_enabled"`
	ExpiredContractCleanupCronExpr      string `mapstructure:"expired_contract_cleanup_cron_expr"`
	PayOSExpiryCheckEnabled             bool   `mapstructure:"payos_expiry_check_enabled"`
	PayOSExpiryCheckIntervalMinutes     int    `mapstructure:"payos_expiry_check_interval_minutes"`
	PreOrderOpeningCheckEnable          bool   `mapstructure:"preorder_opening_check_enabled"`
	PreOrderOpeningCheckIntervalMinutes int    `mapstructure:"preorder_opening_check_interval_minutes"`
	TikTokStatusPollerEnabled           bool   `mapstructure:"tiktok_status_poller_enabled"`
	TikTokStatusPollerIntervalSeconds   int    `mapstructure:"tiktok_status_poller_interval_seconds"`

	// Order - PreOrder
	CensorshipIntervalMinutes int `mapstructure:"censorship_interval_minutes"`

	// Social Media Integration
	// This is used to determine when to send notifications for expiring OAuth tokens

	// ========= Facebook =========
	FacebookExpiryThresholdNotifications int    `mapstructure:"facebook_expiry_threshold_notifications"` // in days
	FacebookVideoUploadChunkSizeInMB     int    `mapstructure:"facebook_video_upload_chunk_size_in_mb"`
	FacebookVideoUploadMaxRetries        int    `mapstructure:"facebook_video_upload_max_retries"`
	FacebookWebhookSecret                string `mapstructure:"facebook_webhook_secret"`

	// ========= TikTok =========
	TikTokExpiryThresholdNotifications int    `mapstructure:"tiktok_expiry_threshold_notifications"` // in days
	TikTokWebhookSecret                string `mapstructure:"tiktok_webhook_secret"`

	// ======== General ========
	SystemEmail string `mapstructure:"system_email"`
	SystemName  string `mapstructure:"system_name"`

	// AI Content Generation
	ContentGenerationPromptTemplate string `mapstructure:"content_generation_prompt_template" type:"textarea"`
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

	adminViper.SetDefault("tracking_link_trusted_domains", []string{"example.com", "trustedpartner.com"})
	adminViper.SetDefault("bot_signatures", []string{"example.com", "trustedpartner.com"})

	adminViper.SetDefault("tiktok_status_poller_enabled", true)
	adminViper.SetDefault("tiktok_status_poller_interval_seconds", 30)

	adminViper.SetDefault("facebook_expiry_threshold_notifications", 7)
	adminViper.SetDefault("facebook_video_upload_chunk_size_in_mb", 50)
	adminViper.SetDefault("facebook_video_upload_max_retries", 3)

	adminViper.SetDefault("tiktok_expiry_threshold_notifications", 7)

	// Webhook secrets (should be set via environment variables for security)
	adminViper.SetDefault("facebook_webhook_secret", "")
	adminViper.SetDefault("tiktok_webhook_secret", "")

	adminViper.SetDefault("content_generation_prompt_template", "Default prompt template")
}

// Override updates AdminConfig with values from the the model that was retrieved from the database
func (c *AdminConfig) Override(models []model.Config) error {
	// Create a map from config key to model.Config for easy lookup
	configMap := utils.MapKeyFromSlice(models, func(m model.Config) (string, model.Config) { return m.Key, m })

	// Override fields if they exist in the configMap with reflect
	for key, cfg := range configMap {
		if reflectVal := reflect.ValueOf(c).Elem().FieldByName(strings.ToLower(key)); reflectVal.IsValid() {
			value := reflect.ValueOf(cfg.Value)
			fmt.Printf("Overriding AdminConfig field %s with value %v\n", key, cfg.Value)
			reflectVal.Set(value)
		}
	}

	return nil
}
