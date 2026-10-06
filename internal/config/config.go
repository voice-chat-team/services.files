package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPCPort    string
	DatabaseURL string

	S3Endpoint  string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
	S3PublicURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	useSSL, err := strconv.ParseBool(getEnv("S3_USE_SSL", "true"))
	if err != nil {
		return nil, fmt.Errorf("S3_USE_SSL must be true or false: %w", err)
	}

	cfg := &Config{
		GRPCPort:    getEnv("GRPC_PORT", "5060"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3Region:    getEnv("S3_REGION", "ru-1"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		S3UseSSL:    useSSL,
		S3PublicURL: os.Getenv("S3_PUBLIC_URL"),
	}

	required := map[string]string{
		"DATABASE_URL":  cfg.DatabaseURL,
		"S3_ENDPOINT":   cfg.S3Endpoint,
		"S3_ACCESS_KEY": cfg.S3AccessKey,
		"S3_SECRET_KEY": cfg.S3SecretKey,
		"S3_BUCKET":     cfg.S3Bucket,
	}

	for name, value := range required {
		if value == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
