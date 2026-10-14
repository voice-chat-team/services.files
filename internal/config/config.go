package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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

	secrets := map[string]string{}
	for _, key := range []string{
		"DATABASE_URL",
		"S3_ENDPOINT",
		"S3_REGION",
		"S3_ACCESS_KEY",
		"S3_SECRET_KEY",
		"S3_BUCKET",
		"S3_PUBLIC_URL",
	} {
		value, err := getSecret(key)
		if err != nil {
			return nil, err
		}
		secrets[key] = value
	}

	cfg := &Config{
		GRPCPort:    getEnv("GRPC_PORT", "5060"),
		DatabaseURL: secrets["DATABASE_URL"],
		S3Endpoint:  secrets["S3_ENDPOINT"],
		S3Region:    secrets["S3_REGION"],
		S3AccessKey: secrets["S3_ACCESS_KEY"],
		S3SecretKey: secrets["S3_SECRET_KEY"],
		S3Bucket:    secrets["S3_BUCKET"],
		S3UseSSL:    useSSL,
		S3PublicURL: secrets["S3_PUBLIC_URL"],
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

// getSecret поддерживает Docker/Swarm secrets: если задана `<KEY>_FILE`
func getSecret(key string) (string, error) {
	if v := os.Getenv(key); v != "" {
		return v, nil
	}

	path := os.Getenv(key + "_FILE")
	if path == "" {
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s_FILE: %w", key, err)
	}

	return strings.TrimSpace(string(data)), nil
}
