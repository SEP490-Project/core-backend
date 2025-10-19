// Package config defines the application configuration settings.
package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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
}

type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	ServiceName     string `mapstructure:"service_name"`
	Environment     string `mapstructure:"environment"`
	Timeout         int    `mapstructure:"timeout"`           // in seconds
	PayOSLinkExpiry int    `mapstructure:"payos_link_expiry"` // in seconds
	Timezone        string `mapstructure:"timezone"`
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
	BaseUrl           string `mapstructure:"base_url"`
	ClientID          string `mapstructure:"client_id"`
	ApiKey            string `mapstructure:"api_key"`
	ChecksumKey       string `mapstructure:"checksum_key"`
	CancelUrl         string `mapstructure:"cancel_url"`
	ReturnUrl         string `mapstructure:"return_url"`
	FrontendCancelUrl string `mapstructure:"frontend_cancel_url"`
	FrontendReturnUrl string `mapstructure:"frontend_return_url"`
}

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
	viper.BindEnv("aws_s3_bucket.access_key", "AWS_S3_BUCKET_ACCESS_KEY")
	viper.BindEnv("aws_s3_bucket.secret_key", "AWS_S3_BUCKET_SECRET_KEY")
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

	return nil
}

func setDefaultValues() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.service_name", "my_service")
	viper.SetDefault("server.environment", "development") // Options: development, production
	viper.SetDefault("server.timezone", "UTC")

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
	viper.SetDefault("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
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
}

// parseRSAKeys reads and parses the RSA private and public keys from the config.
// It prioritizes file paths over raw key content.
// If key files don't exist, it generates them automatically.
func (jc *JWTConfig) parseRSAKeys() error {
	var privateKeyBytes, publicKeyBytes []byte
	var err error

	// --- Load Private Key ---
	// Priority 1: From file path
	if jc.PrivateKeyFile != "" {
		privateKeyBytes, err = os.ReadFile(jc.PrivateKeyFile)
		if err != nil {
			// If file doesn't exist, generate key pair
			if os.IsNotExist(err) {
				fmt.Printf("RSA keys not found, generating new key pair...\n")
				if genErr := jc.generateKeyPair(); genErr != nil {
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
	} else if jc.PrivateKey != "" { // Priority 2: From embedded string
		privateKeyBytes = []byte(jc.PrivateKey)
	} else {
		// In the future, you would add Vault logic here.
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
		// In the future, you would add Vault logic here.
		return fmt.Errorf("public key is not provided (either file path or raw content)")
	}

	// --- Parse Keys ---
	parsedPrivKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	jc.parsedPrivateKey = parsedPrivKey

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
func (jc *JWTConfig) GetPrivateKey() *rsa.PrivateKey {
	return jc.parsedPrivateKey
}

// GetPublicKey returns the parsed public key.
func (jc *JWTConfig) GetPublicKey() *rsa.PublicKey {
	return jc.parsedPublicKey
}

// generateKeyPair generates RSA key pair and saves them to files
func (jc *JWTConfig) generateKeyPair() error {
	// Import necessary packages for key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Save private key
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privateKeyFile, err := os.Create(jc.PrivateKeyFile)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err = pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.Create(jc.PublicKeyFile)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("RSA key pair generated successfully:\n")
	fmt.Printf("  Private key: %s\n", jc.PrivateKeyFile)
	fmt.Printf("  Public key: %s\n", jc.PublicKeyFile)

	return nil
}
