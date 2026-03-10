package config

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Config struct {
	AppEnv  string
	AppAddr string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	SessionCookieName  string
	SessionTTLHours    int
	CookieSecure       bool
	CookieSameSite     string
	CORSAllowedOrigins []string
}

func Load() Config {
	appEnv := getEnv("ENV", "development")

	// Prefer explicit full address if provided, but normalize common "just a port" input.
	appAddr := getAddr(
		getEnv("APP_ADDR", ""),
		getEnv("APP_HOST", "0.0.0.0"),
		getEnv("APP_PORT", "8080"),
	)
	return Config{
		AppEnv:  appEnv,
		AppAddr: appAddr,

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     getEnv("DB_NAME", "ironlytic"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		SessionCookieName:  getEnv("SESSION_COOKIE_NAME", "__Host-ironlytic_session"),
		SessionTTLHours:    getEnvInt("SESSION_TTL_HOURS", 720),
		CookieSecure:       getEnvBool("COOKIE_SECURE", false),
		CookieSameSite:     getEnv("COOKIE_SAMESITE", "Lax"),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
	}
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// getAddr builds a valid http.Server.Addr.
// Rules:
//   - if APP_ADDR is "host:port" or ":port" => use as-is
//   - if APP_ADDR is "port"                => normalize to "host:port" using defaultHost
//   - else                                 => JoinHostPort(APP_HOST, APP_PORT)
func getAddr(appAddr, defaultHost, defaultPort string) string {
	appAddr = strings.TrimSpace(appAddr)
	if appAddr != "" {
		// Full address already (includes colon): "0.0.0.0:30081" or ":8080"
		if strings.Contains(appAddr, ":") {
			return appAddr
		}
		// Likely just a port: "8080"
		return net.JoinHostPort(defaultHost, appAddr)
	}

	return net.JoinHostPort(defaultHost, defaultPort)
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var out int
	_, err := fmt.Sscanf(v, "%d", &out)
	if err != nil {
		return fallback
	}
	return out
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "on"
}

func getEnvList(key string, fallback []string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			out = append(out, item)
		}
	}

	if len(out) == 0 {
		return fallback
	}

	return out
}
