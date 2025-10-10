package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

// Config holds all application configuration
type Config struct {
	Auth         *AuthConfig
	DebugEnabled bool
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	BasicAuthUser string
	BasicAuthPass string
}

// LoadConfig loads configuration from environment variables
// .env file is automatically loaded via autoload import
func LoadConfig() *Config {
	authConfig := &AuthConfig{
		BasicAuthUser: getEnvWithDefault("BACKOFFICE_BASIC_AUTH_USER", "admin"),
		BasicAuthPass: getEnvWithDefault("BACKOFFICE_BASIC_AUTH_PASS", "admin123"),
	}

	debugEnabled := getBoolEnvWithDefault("DEBUG", false)

	config := &Config{
		Auth:         authConfig,
		DebugEnabled: debugEnabled,
	}

	fmt.Printf("🔐 DEBUG: Loaded auth config - User: '%s', Pass: '%s'\n", authConfig.BasicAuthUser, authConfig.BasicAuthPass)
	if debugEnabled {
		fmt.Printf("🐛 DEBUG: SQL debug logging enabled\n")
	}

	return config
}

// LoadAuthConfig loads only authentication configuration (for backward compatibility)
func LoadAuthConfig() *AuthConfig {
	return LoadConfig().Auth
}

// getEnvWithDefault gets an environment variable with a default fallback
func getEnvWithDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		fmt.Printf("🔐 DEBUG: Using environment variable %s='%s'\n", key, value)
		return value
	}
	fmt.Printf("🔐 DEBUG: Using default value for %s='%s'\n", key, defaultValue)
	return defaultValue
}

// getBoolEnvWithDefault gets a boolean environment variable with a default fallback
func getBoolEnvWithDefault(key string, defaultValue bool) bool {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			fmt.Printf("🐛 DEBUG: Using environment variable %s=%t\n", key, parsed)
			return parsed
		}
		fmt.Printf("🐛 DEBUG: Invalid boolean value for %s='%s', using default %t\n", key, value, defaultValue)
	}
	return defaultValue
}
