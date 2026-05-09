package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds application configuration from environment.
type Config struct {
	DBHost               string
	DBPort               int
	DBUser               string
	DBPassword           string
	DBName               string
	DBSSLMode            string
	APIPort              int
	JWTSecret            string
	DefaultAdminPassword string
	GoogleClientID       string
	GoogleClientIDs      []string // all allowed audience values
	GoogleClientSecret   string

	// Billing integration (Venta system)
	BillingGoServiceURL string
	BillingTenantSlug   string
	BillingAPIPeruURL   string
	BillingAPIPeruToken string
	BillingDBHost       string
	BillingDBPort       int
	BillingDBUser       string
	BillingDBPassword   string
	BillingDBSSLMode    string
	BillingTenantEmail  string
	BillingTenantPass   string

	// Culqi payment gateway
	CulqiPublicKey  string
	CulqiPrivateKey string

	// Email (SMTP)
	MailHost     string
	MailPort     string
	MailUsername string
	MailPassword string
	MailFromAddr string
	MailFromName string

	// Yape directo / voucher uploads
	VoucherStoragePath  string
	VoucherMaxSizeMB    int
	YapeHoldMinutes     int
	YapeBusinessNumber  string
	WhatsAppNotifyPhone string
}

// Load reads configuration from environment variables.
// It attempts to load a .env file from config/.env if present.
func Load() (*Config, error) {
	// Try to load .env file relative to this source file (works from any working directory)
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = godotenv.Load(
		filepath.Join(projectRoot, "config", ".env"),
		"config/.env",
	)
	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	apiPort, _ := strconv.Atoi(getEnv("API_PORT", "8085"))
	billingDBPort, _ := strconv.Atoi(getEnv("BILLING_DB_PORT", "5432"))
	voucherMaxMB, _ := strconv.Atoi(getEnv("VOUCHER_MAX_SIZE_MB", "10"))
	yapeHoldMin, _ := strconv.Atoi(getEnv("YAPE_HOLD_MINUTES", "30"))

	cfg := &Config{
		DBHost:               getEnv("DB_HOST", "localhost"),
		DBPort:               port,
		DBUser:               getEnv("DB_USER", "postgres"),
		DBPassword:           getEnv("DB_PASSWORD", ""),
		DBName:               getEnv("DB_NAME", "pasaje"),
		DBSSLMode:            getEnv("DB_SSLMODE", "disable"),
		APIPort:              apiPort,
		JWTSecret:            getEnv("JWT_SECRET", "pasaje-dev-secret-change-me"),
		DefaultAdminPassword: getEnv("DEFAULT_ADMIN_PASSWORD", "admin123"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientIDs:      buildGoogleClientIDs(getEnv("GOOGLE_CLIENT_ID", ""), getEnv("GOOGLE_MOBILE_CLIENT_ID", "")),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),

		BillingGoServiceURL: getEnv("BILLING_GO_SERVICE_URL", "https://goventa.facturame.online"),
		BillingTenantSlug:   getEnv("BILLING_TENANT_SLUG", "ancalla"),
		BillingAPIPeruURL:   getEnv("BILLING_APIPERU_URL", "https://apiperub.facturame.online"),
		BillingAPIPeruToken: getEnv("BILLING_APIPERU_TOKEN", ""),
		BillingDBHost:       getEnv("BILLING_DB_HOST", "localhost"),
		BillingDBPort:       billingDBPort,
		BillingDBUser:       getEnv("BILLING_DB_USER", "postgres"),
		BillingDBPassword:   getEnv("BILLING_DB_PASSWORD", ""),
		BillingDBSSLMode:    getEnv("BILLING_DB_SSLMODE", "disable"),
		BillingTenantEmail:  getEnv("BILLING_TENANT_EMAIL", "admin"),
		BillingTenantPass:   getEnv("BILLING_TENANT_PASS", "admin123"),

		CulqiPublicKey:  getEnv("CULQI_PUBLIC_KEY", "pk_test_LlHMSnMSdFBFOs6M"),
		CulqiPrivateKey: getEnv("CULQI_PRIVATE_KEY", "sk_test_UTCQSGcXW8bP7Aws"),

		MailHost:     getEnv("MAIL_HOST", "smtp.gmail.com"),
		MailPort:     getEnv("MAIL_PORT", "465"),
		MailUsername: getEnv("MAIL_USERNAME", ""),
		MailPassword: getEnv("MAIL_PASSWORD", ""),
		MailFromAddr: getEnv("MAIL_FROM_ADDRESS", ""),
		MailFromName: getEnv("MAIL_FROM_NAME", "Pasaje Bus"),

		VoucherStoragePath:  getEnv("VOUCHER_STORAGE_PATH", "/data/vouchers"),
		VoucherMaxSizeMB:    voucherMaxMB,
		YapeHoldMinutes:     yapeHoldMin,
		YapeBusinessNumber:  getEnv("YAPE_BUSINESS_NUMBER", ""),
		WhatsAppNotifyPhone: getEnv("WHATSAPP_NOTIFY_PHONE", ""),
	}
	return cfg, nil
}

// BillingDSN returns the PostgreSQL connection string for the billing tenant DB.
func (c *Config) BillingDSN() string {
	dbName := fmt.Sprintf("tenant_%s", c.BillingTenantSlug)
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.BillingDBUser, url.QueryEscape(c.BillingDBPassword), c.BillingDBHost, c.BillingDBPort, dbName, c.BillingDBSSLMode,
	)
}

// DSN returns the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DBUser, url.QueryEscape(c.DBPassword), c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

func buildGoogleClientIDs(ids ...string) []string {
	var result []string
	for _, id := range ids {
		if id != "" {
			result = append(result, id)
		}
	}
	return result
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
