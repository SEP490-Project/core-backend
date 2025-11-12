// Package config defines the application configuration settings.
package config

import (
	"core-backend/pkg/crypto"
	"crypto/rsa"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

// region: ============== Configuration Structs ==============

type AppConfig struct {
	Server            ServerConfig            `mapstructure:"server"`
	Database          DatabaseConfig          `mapstructure:"database"`
	Cache             CacheConfig             `mapstructure:"cache"`
	JWT               JWTConfig               `mapstructure:"jwt"`
	Log               LogConfig               `mapstructure:"log"`
	CORS              CORSConfig              `mapstructure:"cors"`
	Otel              OtelConfig              `mapstructure:"otel"`
	RabbitMQ          RabbitMQConfig          `mapstructure:"rabbitmq"`
	WebSocket         WebSocketConfig         `mapstructure:"websocket"`
	S3Bucket          S3BucketConfig          `mapstructure:"aws_s3_bucket"`
	S3StreamingBucket S3StreamingBucketConfig `mapstructure:"aws_s3_streaming_bucket"`
	PayOS             PayOSConfig             `mapstructure:"payos"`
	GHN               GHNConfig               `mapstructure:"ghn"`
	AdminConfig       AdminConfig             `mapstructure:"admin_config"`
	GmailSMTP         EmailConfig             `mapstructure:"gmail_smtp"`
	FirebaseFCM       FirebaseFCMConfig       `mapstructure:"firebase_fcm"`
	Notification      NotificationConfig      `mapstructure:"notification"`
	TaskScheduler     TaskSchedulerConfig     `mapstructure:"task_scheduler"`
	HTTPClient        HTTPClientConfig        `mapstructure:"http_client"`
	Social            SocialConfig            `mapstructure:"social"`
	TokenStorage      TokenStorageConfig      `mapstructure:"token_storage"`
}

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	ServiceName     string `mapstructure:"service_name"`
	Environment     string `mapstructure:"environment"`
	Timeout         int    `mapstructure:"timeout"`           // in seconds
	PayOSLinkExpiry int    `mapstructure:"payos_link_expiry"` // in seconds
	Timezone        string `mapstructure:"timezone"`
	BaseURL         string `mapstructure:"base_url"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type CacheConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Algorithm          string       `mapstructure:"algorithm"`
	ExpiryHours        int          `mapstructure:"expiry_hours"`
	AccessExpiryHours  int          `mapstructure:"access_expiry_hours"`
	RefreshExpiryHours int          `mapstructure:"refresh_expiry_hours"`
	PrivateKeyFile     string       `mapstructure:"private_key_file"`
	PublicKeyFile      string       `mapstructure:"public_key_file"`
	PrivateKey         string       `mapstructure:"private_key"`
	PublicKey          string       `mapstructure:"public_key"`
	Vault              *VaultConfig `mapstructure:"vault"`

	// Internal fields to hold parsed keys
	parsedPrivateKey *rsa.PrivateKey `mapstructure:"-"`
	parsedPublicKey  *rsa.PublicKey  `mapstructure:"-"`
}

type VaultConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Address         string `mapstructure:"address"`
	Token           string `mapstructure:"token"`
	SecretPath      string `mapstructure:"secret_path"`
	PrivateKeyField string `mapstructure:"private_key_field"`
	PublicKeyField  string `mapstructure:"public_key_field"`
}

type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	ExposedHeaders   []string `mapstructure:"exposed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type OtelConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
	Insecure    bool   `mapstructure:"insecure"`
	ServiceName string `mapstructure:"service_name"`
}

type RabbitMQConfig struct {
	URL                 string                   `mapstructure:"url"`
	Host                string                   `mapstructure:"host"`
	Username            string                   `mapstructure:"username"`
	Password            string                   `mapstructure:"password"`
	Port                int                      `mapstructure:"port"`
	VHost               string                   `mapstructure:"vhost"`
	ReconnectDelayMs    int                      `mapstructure:"reconnect_delay_ms"`
	ConnectionTimeoutMs int                      `mapstructure:"connection_timeout_ms"`
	Heartbeat           int                      `mapstructure:"heartbeat"`
	Topology            RabbitMQTopologyConfig   `mapstructure:"topology" json:"topology" yaml:"topology"`
	Producers           []RabbitMQProducerConfig `mapstructure:"producers" json:"producers" yaml:"producers"`
	Consumers           []RabbitMQConsumerConfig `mapstructure:"consumers" json:"consumers" yaml:"consumers"`
}

type WebSocketConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	Endpoint        string   `mapstructure:"endpoint"`
	AllowedOrigins  []string `mapstructure:"allowed_origins"`
	ReadBufferSize  int      `mapstructure:"read_buffer_size"`
	WriteBufferSize int      `mapstructure:"write_buffer_size"`
}

type S3BucketConfig struct {
	BucketName string `mapstructure:"bucket_name"`
	Region     string `mapstructure:"region"`
	Endpoint   string `mapstructure:"endpoint"`
	AccessKey  string `mapstructure:"access_key"`
	SecretKey  string `mapstructure:"secret_key"`
}

type S3StreamingBucketConfig struct {
	BucketName       string `mapstructure:"bucket_name"`
	Region           string `mapstructure:"region"`
	CloudfrontDomain string `mapstructure:"cloudfront_domain"`
	AccessKey        string `mapstructure:"access_key"`
	SecretKey        string `mapstructure:"secret_key"`
}

type PayOSConfig struct {
	BaseURL           string `mapstructure:"base_url"`
	ClientID          string `mapstructure:"client_id"`
	APIKey            string `mapstructure:"api_key"`
	ChecksumKey       string `mapstructure:"checksum_key"`
	CancelURL         string `mapstructure:"cancel_url"`
	ReturnURL         string `mapstructure:"return_url"`
	FrontendCancelURL string `mapstructure:"frontend_cancel_url"`
	FrontendReturnURL string `mapstructure:"frontend_return_url"`
}

type GHNConfig struct {
	BaseURL         string `mapstructure:"base_url"`
	FeeBaseURL      string `mapstructure:"fee_base_url"`
	Token           string `mapstructure:"token"`
	ShopID          int    `mapstructure:"shop_id"`
	DistrictID      int    `mapstructure:"district_id"`
	MockSessionInfo struct {
		MockURL   string `mapstructure:"mock_url"`
		UserID    int    `mapstructure:"user_id"`
		Password  string `mapstructure:"password"`
		DeviceID  string `mapstructure:"device_id"`
		UserAgent string `mapstructure:"user_agent"`
		AppKey    string `mapstructure:"app_key"`
	} `mapstructure:"mock_session_info"`
}

type EmailConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	FromName  string `mapstructure:"from_name"`
	FromEmail string `mapstructure:"from_email"`
}

type FirebaseFCMConfig struct {
	ServiceAccountPath string `mapstructure:"service_account_path"`
	ProjectID          string `mapstructure:"project_id"`
}

type NotificationConfig struct {
	TemplateDir         string                  `mapstructure:"template_dir"`
	MaxRetries          int                     `mapstructure:"max_retries"`
	RetryDelays         []int                   `mapstructure:"retry_delays"`
	RateLimits          NotificationRateLimits  `mapstructure:"rate_limits"`
	ConsumerConcurrency NotificationConcurrency `mapstructure:"consumer_concurrency"`
}

type NotificationRateLimits struct {
	EmailPerMinute int `mapstructure:"email_per_minute"`
	PushPerMinute  int `mapstructure:"push_per_minute"`
}

type NotificationConcurrency struct {
	Email int `mapstructure:"email"`
	Push  int `mapstructure:"push"`
}

type HTTPClientConfig struct {
	Timeout               int `mapstructure:"timeout"`                 // in seconds
	MaxIdleConns          int `mapstructure:"max_idle_conns"`          // maximum number of idle connections
	MaxIdleConnsPerHost   int `mapstructure:"max_idle_conns_per_host"` // maximum number of idle connections per host
	IdleConnTimeout       int `mapstructure:"idle_conn_timeout"`       // in seconds (default: 90)
	TLSHandshakeTimeout   int `mapstructure:"tls_handshake_timeout"`   // in seconds (default: 10)
	ExpectContinueTimeout int `mapstructure:"expect_continue_timeout"` // in seconds (default: 1)
}

// TokenStorageConfig controls where and how OAuth tokens are stored
type TokenStorageConfig struct {
	EncryptionKey   string `mapstructure:"encryption_key"`
	VaultPathPrefix string `mapstructure:"vault_path_prefix"`
	UseVault        bool   `mapstructure:"use_vault"`
}

// TaskSchedulerConfig holds configuration for task schedulers
type TaskSchedulerConfig struct {
	LocationSync locationSynchronizationConfig `mapstructure:"location_synchronization"`
}

// Scheduler configuration
type locationSynchronizationConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	SyncHour    int  `mapstructure:"sync_hour"`
	Concurrency int  `mapstructure:"concurrency"`
}

type SocialConfig struct {
	Facebook FacebookSocialConfig `mapstructure:"facebook"`
	TikTok   TikTokSocialConfig   `mapstructure:"tiktok"`
}

type FacebookSocialConfig struct {
	BaseURL             string   `mapstructure:"base_url"`
	ClientID            string   `mapstructure:"client_id"`
	ClientSecret        string   `mapstructure:"client_secret"`
	APIVersion          string   `mapstructure:"api_version"`
	RedirectURL         string   `mapstructure:"redirect_url"`
	FrontendRedirectURL string   `mapstructure:"frontend_redirect_url"`
	FrontendCancelURL   string   `mapstructure:"frontend_cancel_url"`
	Scopes              []string `mapstructure:"scopes"`
	ResponseType        string   `mapstructure:"response_type"`
}

type TikTokSocialConfig struct {
	BaseURL             string   `mapstructure:"base_url"`
	ClientKey           string   `mapstructure:"client_key"` // TikTok uses "client_key" not "client_id"
	ClientSecret        string   `mapstructure:"client_secret"`
	APIVersion          string   `mapstructure:"api_version"`
	RedirectURL         string   `mapstructure:"redirect_url"`
	FrontendRedirectURL string   `mapstructure:"frontend_redirect_url"`
	FrontendCancelURL   string   `mapstructure:"frontend_cancel_url"`
	UserScopes          []string `mapstructure:"user_scopes"`
	Scopes              []string `mapstructure:"scopes"`
	ResponseType        string   `mapstructure:"response_type"`
}

//End of Schedulers

// endregion

var (
	appConfig *AppConfig
)

func LoadConfig(configPath string) error {
	// Priority 3: Default values
	setDefaultValues()

	// Priority 2: Configuration file in config.yaml format (if exists)
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Println("Config file not found, using defaults and environment variables.")
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Priority 1: Environment variables
	_ = viper.BindEnv("aws_s3_bucket.access_key", "AWS_S3_BUCKET_ACCESS_KEY")
	_ = viper.BindEnv("aws_s3_bucket.secret_key", "AWS_S3_BUCKET_SECRET_KEY")
	_ = viper.BindEnv("aws_s3_streaming_bucket.access_key", "AWS_S3_STREAMING_BUCKET_ACCESS_KEY")
	_ = viper.BindEnv("aws_s3_streaming_bucket.secret_key", "AWS_S3_STREAMING_BUCKET_SECRET_KEY")
	_ = viper.BindEnv("aws_s3_bucket.access_key", "AWS_S3_BUCKET_ACCESS_KEY")
	_ = viper.BindEnv("aws_s3_bucket.secret_key", "AWS_S3_BUCKET_SECRET_KEY")
	_ = viper.BindEnv("aws_s3_streaming_bucket.access_key", "AWS_S3_STREAMING_BUCKET_ACCESS_KEY")
	_ = viper.BindEnv("aws_s3_streaming_bucket.secret_key", "AWS_S3_STREAMING_BUCKET_SECRET_KEY")
	_ = viper.BindEnv("gmail_smtp.username", "GMAIL_SMTP_USERNAME")
	_ = viper.BindEnv("gmail_smtp.password", "GMAIL_APP_PASSWORD")
	_ = viper.BindEnv("gmail_smtp.from_email", "GMAIL_SMTP_FROM_EMAIL")
	_ = viper.BindEnv("firebase_fcm.service_account_path", "FIREBASE_SERVICE_ACCOUNT_PATH")
	_ = viper.BindEnv("firebase_fcm.project_id", "FIREBASE_PROJECT_ID")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.Unmarshal(&appConfig)
	if err != nil {
		return fmt.Errorf("unable to decode into struct: %w", err)
	}

	// Override RabbitMQ URL if individual components are set in config
	if appConfig.RabbitMQ.Host != "" && appConfig.RabbitMQ.Username != "" && appConfig.RabbitMQ.Password != "" {
		appConfig.RabbitMQ.URL = fmt.Sprintf("amqp://%s:%s@%s:%d/", appConfig.RabbitMQ.Username, appConfig.RabbitMQ.Password, appConfig.RabbitMQ.Host, appConfig.RabbitMQ.Port)
	}

	fmt.Println("Loaded server port from config:", appConfig.Server.Port)

	// Parse RSA keys
	if err := appConfig.JWT.parseRSAKeys(); err != nil {
		return fmt.Errorf("error parsing RSA keys: %w", err)
	}

	// Load RabbitMQ advanced configuration from separate file
	if err := loadRabbitMQConfig(configPath); err != nil {
		// Log warning but don't fail - RabbitMQ advanced config is optional
		fmt.Printf("Warning: Could not load RabbitMQ advanced config: %v\n", err)
		fmt.Println("Continuing with basic RabbitMQ configuration...")
	}

	// Load Admin configuration from separate file
	if err := loadAdminConfig(configPath); err != nil {
		fmt.Printf("Warning: Could not load Admin config: %v\n", err)
		fmt.Println("Continuing with default Admin config...")
	}

	return nil
}

func setDefaultValues() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.service_name", "my_service")
	viper.SetDefault("server.environment", "development") // Options: development, production
	viper.SetDefault("server.timezone", "UTC")
	viper.SetDefault("server.base_url", "http://localhost:8080")

	viper.SetDefault("database.host", "postgres.trangiangkhanh.online")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "170504")
	viper.SetDefault("database.dbname", "sep490_db_stag")
	viper.SetDefault("database.sslmode", "disable")

	viper.SetDefault("cache.host", "localhost")
	viper.SetDefault("cache.port", 6379)
	viper.SetDefault("cache.password", "")
	viper.SetDefault("cache.db", 0)

	viper.SetDefault("jwt.algorithm", "RS256")
	viper.SetDefault("jwt.expiry_hours", 72)
	viper.SetDefault("jwt.private_key_file", "private.pem")
	viper.SetDefault("jwt.public_key_file", "public.pem")
	viper.SetDefault("jwt.private_key", "")
	viper.SetDefault("jwt.public_key", "")
	viper.SetDefault("jwt.vault.enabled", false)
	viper.SetDefault("jwt.vault.address", "")
	viper.SetDefault("jwt.vault.token", "")
	viper.SetDefault("jwt.vault.secret_path", "")
	viper.SetDefault("jwt.vault.private_key_field", "private_key_file")
	viper.SetDefault("jwt.vault.public_key_field", "public_key_file")

	viper.SetDefault("log.level", "info")

	viper.SetDefault("cors.allowed_origins", []string{"*"})
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"})
	viper.SetDefault("cors.allowed_headers", []string{"Origin", "Content-Type", "Accept", "Authorization"})
	viper.SetDefault("cors.exposed_headers", []string{"Content-Type", "Authorization"})
	viper.SetDefault("cors.allow_credentials", true)

	viper.SetDefault("otel.enabled", true)
	viper.SetDefault("otel.endpoint", "localhost:4317")
	viper.SetDefault("otel.insecure", true)
	viper.SetDefault("otel.service_name", "my_service")

	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")
	viper.SetDefault("rabbitmq.host", "localhost")
	viper.SetDefault("rabbitmq.username", "guest")
	viper.SetDefault("rabbitmq.password", "guest")
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.vhost", "/")
	viper.SetDefault("rabbitmq.reconnect_delay_ms", 5000)
	viper.SetDefault("rabbitmq.connection_timeout_ms", 10000)
	viper.SetDefault("rabbitmq.heartbeat", 10)

	viper.SetDefault("websocket.enabled", true)
	viper.SetDefault("websocket.endpoint", "/ws")
	viper.SetDefault("websocket.allowed_origins", []string{"*"})
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)

	//Task Scheduler Defaults
	viper.SetDefault("task_scheduler.location_synchronization.enabled", false)
	viper.SetDefault("task_scheduler.location_synchronization.sync_hour", 3) // 3 AM
	viper.SetDefault("task_scheduler.location_synchronization.concurrency", 1)

	// HTTP Client Defaults
	viper.SetDefault("http_client.timeout", 30) // seconds
	viper.SetDefault("http_client.max_idle_conns", 100)
	viper.SetDefault("http_client.max_idle_conns_per_host", 10)
	viper.SetDefault("http_client.idle_conn_timeout", 90)      // seconds
	viper.SetDefault("http_client.tls_handshake_timeout", 10)  // seconds
	viper.SetDefault("http_client.expect_continue_timeout", 1) // seconds
}

// parseRSAKeys reads and parses the RSA private and public keys from the config.
// Priority order:
// 1. HashiCorp Vault (if enabled)
// 2. File paths
// 3. Embedded strings
// If keys don't exist, it generates them automatically.
func (jc *JWTConfig) parseRSAKeys() error {
	// --- Priority 1: Try loading from Vault if enabled ---
	if jc.Vault != nil && jc.Vault.Enabled {
		fmt.Printf("Vault is enabled, attempting to load RSA keys from Vault from infrastructure registry...\n")
	} else if err := jc.LoadRSAKeysLocally(); err != nil {
		return fmt.Errorf("error loading RSA keys locally: %w", err)
	}

	return nil
}

func (jc *JWTConfig) LoadRSAKeysLocally() error {
	var privateKeyBytes, publicKeyBytes []byte
	var err error

	// --- Priority 2: Load from file path (if not loaded from Vault) ---
	if len(privateKeyBytes) == 0 && jc.PrivateKeyFile != "" {
		privateKeyBytes, err = os.ReadFile(jc.PrivateKeyFile)
		if err != nil {
			// If file doesn't exist, generate key pair
			if os.IsNotExist(err) {
				fmt.Printf("RSA keys not found, generating new key pair...\n")
				if genErr := crypto.GenerateRSAKeyPair(jc.PrivateKeyFile, jc.PublicKeyFile, 2048); genErr != nil {
					return fmt.Errorf("failed to generate RSA keys: %w", genErr)
				}
				// Try reading again after generation
				privateKeyBytes, err = os.ReadFile(jc.PrivateKeyFile)
				if err != nil {
					return fmt.Errorf("could not read generated private key file %s: %w", jc.PrivateKeyFile, err)
				}
			} else {
				return fmt.Errorf("could not read private key file %s: %w", jc.PrivateKeyFile, err)
			}
		}
	} else if len(privateKeyBytes) == 0 && jc.PrivateKey != "" { // Priority 3: From embedded string
		privateKeyBytes = []byte(jc.PrivateKey)
	} else {
		return fmt.Errorf("private key is not provided (either file path or raw content)")
	}

	// --- Load Public Key ---
	// Priority 1: From file path
	if jc.PublicKeyFile != "" {
		publicKeyBytes, err = os.ReadFile(jc.PublicKeyFile)
		if err != nil {
			return fmt.Errorf("could not read public key file %s: %w", jc.PublicKeyFile, err)
		}
	} else if jc.PublicKey != "" { // Priority 2: From embedded string
		publicKeyBytes = []byte(jc.PublicKey)
	} else {
		return fmt.Errorf("public key is not provided (either file path or raw content)")
	}

	// Parse private key
	parsedPrivKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	jc.parsedPrivateKey = parsedPrivKey

	// Parse public key
	parsedPubKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse RSA public key: %w", err)
	}
	jc.parsedPublicKey = parsedPubKey

	return nil

}

// GetAppConfig returns the loaded application configuration.
func GetAppConfig() *AppConfig {
	return appConfig
}

// GetPrivateKey returns the parsed private key.
func (config *AppConfig) GetPrivateKey() *rsa.PrivateKey {
	return config.JWT.parsedPrivateKey
}

// GetPublicKey returns the parsed public key.
func (config *AppConfig) GetPublicKey() *rsa.PublicKey {
	return config.JWT.parsedPublicKey
}

// UpdateRSAKeys updates the parsed RSA keys with new PEM-encoded keys
// This is used when keys are generated and stored in Vault
func (jc *JWTConfig) UpdateRSAKeys(privateKeyPEM, publicKeyPEM string) error {
	// Parse private key
	parsedPrivKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	jc.parsedPrivateKey = parsedPrivKey

	// Parse public key
	parsedPubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
	if err != nil {
		return fmt.Errorf("failed to parse RSA public key: %w", err)
	}
	jc.parsedPublicKey = parsedPubKey

	return nil
}
