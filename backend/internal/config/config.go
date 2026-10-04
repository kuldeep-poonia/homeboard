package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                     string
	Host                     string
	DBPath                   string
	AppEnv                   string
	BaseURL                  string
	CORSAllowedOrigins       []string
	TokenTTL                 time.Duration
	MaxItemsPerBoard         int
	MaxDevicesPerBoard       int
	RateLimitRedeemPerMin    int
	RateLimitTokenGenPerMin  int
	BedrockEnabled           bool
	AWSRegion                string
	BedrockModelID           string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	host := getEnv("HOST", "0.0.0.0")
	dbPath := getEnv("DB_PATH", "./homeboard.db")
	appEnv := getEnv("APP_ENV", "development")
	baseURL := strings.TrimRight(getEnv("BASE_URL", "http://localhost:"+port), "/")

	corsOriginsStr := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:8080,http://127.0.0.1:8080")
	corsOrigins := strings.Split(corsOriginsStr, ",")
	for i := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
	}

	tokenTTLMin, _ := strconv.Atoi(getEnv("TOKEN_TTL_MINUTES", "10"))
	if tokenTTLMin <= 0 {
		tokenTTLMin = 10
	}

	maxItems, _ := strconv.Atoi(getEnv("MAX_ITEMS_PER_BOARD", "200"))
	if maxItems <= 0 {
		maxItems = 200
	}

	maxDevices, _ := strconv.Atoi(getEnv("MAX_DEVICES_PER_BOARD", "10"))
	if maxDevices <= 0 {
		maxDevices = 10
	}

	redeemRate, _ := strconv.Atoi(getEnv("RATE_LIMIT_REDEEM_PER_MINUTE", "20"))
	tokenRate, _ := strconv.Atoi(getEnv("RATE_LIMIT_TOKEN_GEN_PER_MINUTE", "10"))

	bedrockEnabled := strings.ToLower(getEnv("BEDROCK_ENABLED", "false")) == "true"
	awsRegion := getEnv("AWS_REGION", "us-east-1")
	bedrockModel := getEnv("BEDROCK_MODEL_ID", "anthropic.claude-haiku-4-5-20251001-v1:0")

	return &Config{
		Port:                    port,
		Host:                    host,
		DBPath:                  dbPath,
		AppEnv:                  appEnv,
		BaseURL:                 baseURL,
		CORSAllowedOrigins:      corsOrigins,
		TokenTTL:                time.Duration(tokenTTLMin) * time.Minute,
		MaxItemsPerBoard:        maxItems,
		MaxDevicesPerBoard:      maxDevices,
		RateLimitRedeemPerMin:   redeemRate,
		RateLimitTokenGenPerMin: tokenRate,
		BedrockEnabled:          bedrockEnabled,
		AWSRegion:               awsRegion,
		BedrockModelID:          bedrockModel,
	}
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
