package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// Database
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	DatabaseURL string

	// Redis
	EnableRedis bool
	RedisURL    string

	// JWT
	JWTSecret string

	// Server
	Port        string
	Environment string

	// CORS
	CORSOrigins []string

	// Upload
	UploadDir     string
	MaxUploadSize int64

	// Email
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Rate Limiting
	RateLimitRequests int
	RateLimitWindow   int

	// Features
	EnableCache       bool
	EnableEmail       bool
	EnableMetrics     bool
	EnableCompression bool

	// Site Meta
	SiteName        string
	SiteDescription string
	SiteURL         string
	SiteFavicon     string
}

func New() *Config {
	c := &Config{
		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "bloguser"),
		DBPassword: getEnv("DB_PASSWORD", "blogpassword"),
		DBName:     getEnv("DB_NAME", "blogdb"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// Redis
		EnableRedis: getEnvAsBool("ENABLE_REDIS", true),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),

		// JWT
		JWTSecret: getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),

		// Server
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// CORS
		CORSOrigins: strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:8080"), ","),

		// Upload
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadSize: 10 * 1024 * 1024, // 10MB

		// Email
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@blog.com"),

		// Rate Limiting
		RateLimitRequests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   getEnvAsInt("RATE_LIMIT_WINDOW", 60),

		// Features
		EnableCache:       getEnvAsBool("ENABLE_CACHE", true),
		EnableEmail:       getEnvAsBool("ENABLE_EMAIL", false),
		EnableMetrics:     getEnvAsBool("ENABLE_METRICS", true),
		EnableCompression: getEnvAsBool("ENABLE_COMPRESSION", true),

		// Site Meta
		SiteName:        getEnv("SITE_NAME", "Constructor Script"),
		SiteDescription: getEnv("SITE_DESCRIPTION", "Platform for building modern, high-performance websites using Go and templates."),
		SiteURL:         getEnv("SITE_URL", "http://localhost:8081"),
		SiteFavicon:     getEnv("SITE_FAVICON", "/favicon.ico"),
	}

	// Build DSN
	c.DatabaseURL = fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)

	return c
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1"
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
