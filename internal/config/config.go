package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                    string
	AzureTenantID           string
	AzureClientID           string
	StorageAccountName      string
	StorageContainerName    string
	StorageConnectionString string
	AppTitle                string
	AppIcon                 string
	LoginNotice             string
	LogLevel                string
	LogFile                 string
}

func Load() (*Config, error) {
	// Load .env file if present (dev convenience, ignored in production)
	_ = godotenv.Load()

	cfg := &Config{
		Port:                    getEnv("PORT", "8080"),
		AzureTenantID:           os.Getenv("MSAL_TENANT_ID"),
		AzureClientID:           os.Getenv("MSAL_CLIENT_ID"),
		StorageAccountName:      os.Getenv("STORAGE_ACCOUNT_NAME"),
		StorageContainerName:    getEnv("STORAGE_CONTAINER_NAME", "bicep"),
		StorageConnectionString: os.Getenv("AZURE_STORAGE_CONNECTION_STRING"),
		AppTitle:                getEnv("APP_TITLE", "Bicep Deployer"),
		AppIcon:                 getEnv("APP_ICON", ""),
		LoginNotice:             os.Getenv("LOGIN_NOTICE"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		LogFile:                 os.Getenv("LOG_FILE"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.AzureTenantID == "" {
		return fmt.Errorf("MSAL_TENANT_ID is required")
	}
	if c.AzureClientID == "" {
		return fmt.Errorf("MSAL_CLIENT_ID is required")
	}
	if c.StorageAccountName == "" && c.StorageConnectionString == "" {
		return fmt.Errorf("either STORAGE_ACCOUNT_NAME (for Managed Identity) or AZURE_STORAGE_CONNECTION_STRING is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
