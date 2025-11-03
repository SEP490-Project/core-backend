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
	PayOSLinkExpiry                   int    `mapstructure:"payos_link_expiry"`
	MinimumDayBeforeContracPaymentDue int    `mapstructure:"minimum_day_before_contract_payment_due"`
	RepresentativeName                string `mapstructure:"representative_name"`
	RepresentativeRole                string `mapstructure:"representative_role"`
	RepresentativePhone               string `mapstructure:"representative_phone"`
	RepresentativeEmail               string `mapstructure:"representative_email"`
	RepresentativeTaxNumber           string `mapstructure:"representative_tax_number"`
	RepresentativeBankName            string `mapstructure:"representative_bank_name"`
	RepresentativeBankAccountNumber   string `mapstructure:"representative_bank_account_number"`
	RepresentativeBankAccountHolder   string `mapstructure:"representative_bank_account_holder"`

	// Affiliate Link Tracking Configuration
	TrackingLinkTrustedDomains []string `mapstructure:"tracking_link_trusted_domains"`
	BotSignatures              []string `mapstructure:"bot_signatures"`

	// Cron Jobs Configuration
	CTRAggregationEnabled           bool   `mapstructure:"ctr_aggregation_enabled"`
	CTRAggregationIntervalMinutes   int    `mapstructure:"ctr_aggregation_interval_minutes"`
	ExpiredContractCleanupEnabled   bool   `mapstructure:"expired_contract_cleanup_enabled"`
	ExpiredContractCleanupCronExpr  string `mapstructure:"expired_contract_cleanup_cron_expr"`
	PayOSExpiryCheckEnabled         bool   `mapstructure:"payos_expiry_check_enabled"`
	PayOSExpiryCheckIntervalMinutes int    `mapstructure:"payos_expiry_check_interval_minutes"`
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
	adminViper.SetDefault("admin.payos_link_expiry", 300)
	adminViper.SetDefault("admin.minimum_day_before_contract_payment_due", 5)
	adminViper.SetDefault("admin.representative_name", "Đinh Thị Ngọc Trinh")
	adminViper.SetDefault("admin.representative_role", "Beauty Blogger")
	adminViper.SetDefault("admin.representative_phone", "+84917956697")
	adminViper.SetDefault("admin.representative_email", "mrstrinh.work@gmail.com")
	adminViper.SetDefault("admin.representative_tax_number", "01234567890")
	adminViper.SetDefault("admin.representative_bank_name", "")
	adminViper.SetDefault("admin.representative_bank_account_number", "")
	adminViper.SetDefault("admin.representative_bank_account_holder", "TRAN GIANH KHANH")
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
