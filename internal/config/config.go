package config

import (
	"os"
	"path/filepath"
)

// Config holds the service configuration
type Config struct {
	HTTPAddr        string
	GRPCAddr        string
	DBPath          string
	AuthServiceAddr string
}

// NewConfig creates a new config instance
func NewConfig() *Config {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Construct absolute path for the database
	dbPath := filepath.Join(cwd, "data", "forum.db")

	return &Config{
		HTTPAddr:        getEnv("HTTP_ADDR", "localhost:8082"),
		GRPCAddr:        getEnv("GRPC_ADDR", "localhost:9082"),
		DBPath:          getEnv("DB_PATH", dbPath),
		AuthServiceAddr: getEnv("AUTH_SERVICE_ADDR", "localhost:9081"),
	}
}

// Helper function to get environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
