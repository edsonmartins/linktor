package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	NATS     NATSConfig     `mapstructure:"nats"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
}

// CryptoConfig holds secret-at-rest encryption configuration. EncryptionKey is
// the primary key (used to encrypt). PreviousKeys are older keys still accepted
// for decryption during a rotation, until everything has been re-encrypted.
type CryptoConfig struct {
	EncryptionKey string   `mapstructure:"encryption_key"`
	PreviousKeys  []string `mapstructure:"previous_keys"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int    `mapstructure:"port"`
	Host            string `mapstructure:"host"`
	Mode            string `mapstructure:"mode"` // debug, release, test
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime"` // in minutes
}

// DSN returns the PostgreSQL connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode,
	)
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Addr returns the Redis address
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// NATSConfig holds NATS JetStream configuration
type NATSConfig struct {
	URL       string `mapstructure:"url"`
	ClusterID string `mapstructure:"cluster_id"`
	ClientID  string `mapstructure:"client_id"`
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	AccessTokenTTL  int    `mapstructure:"access_token_ttl"`  // in minutes
	RefreshTokenTTL int    `mapstructure:"refresh_token_ttl"` // in hours
	Issuer          string `mapstructure:"issuer"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/linktor")

	// Environment variables
	viper.SetEnvPrefix("LINKTOR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := validateProductionConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateProductionConfig(cfg *Config) error {
	if cfg.Server.Mode != "release" {
		return nil
	}

	secret := strings.TrimSpace(cfg.JWT.Secret)
	switch {
	case secret == "":
		return fmt.Errorf("jwt.secret is required in release mode")
	case secret == "change-me-in-production", secret == "change-me-in-production-use-strong-secret", secret == "change-me-in-development":
		return fmt.Errorf("jwt.secret must be changed in release mode")
	case len(secret) < 32:
		return fmt.Errorf("jwt.secret must be at least 32 characters in release mode")
	}

	encKey := strings.TrimSpace(cfg.Crypto.EncryptionKey)
	switch {
	case encKey == "":
		return fmt.Errorf("crypto.encryption_key is required in release mode")
	case encKey == "change-me-in-production",
		encKey == "change-me-in-production-use-strong-secret",
		encKey == secret:
		return fmt.Errorf("crypto.encryption_key must be changed to a unique value in release mode")
	case len(encKey) < 32:
		return fmt.Errorf("crypto.encryption_key must be at least 32 characters in release mode")
	}

	return nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.shutdown_timeout", 30)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "linktor")
	viper.SetDefault("database.password", "linktor")
	viper.SetDefault("database.database", "linktor")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.max_lifetime", 5)

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.cluster_id", "linktor-cluster")
	viper.SetDefault("nats.client_id", "linktor-server")

	// JWT defaults
	viper.SetDefault("jwt.secret", "change-me-in-production")
	viper.SetDefault("jwt.access_token_ttl", 15)
	viper.SetDefault("jwt.refresh_token_ttl", 168) // 7 days
	viper.SetDefault("jwt.issuer", "linktor")

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")

	// Crypto defaults (must be overridden in release mode)
	viper.SetDefault("crypto.encryption_key", "change-me-in-production")
}
