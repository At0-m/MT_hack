package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Database, Artifacts, Contract, Addr, User, Password, Origin string
	Pool                                                        int32
}

func Load() (Config, error) {
	c := Config{
		Database:  os.Getenv("DATABASE_URL"),
		Artifacts: env("ARTIFACTS_ROOT", "artifacts"),
		Contract:  env("OPENAPI_PATH", "openapi/openapi.yaml"),
		Addr:      env("HTTP_ADDR", "127.0.0.1:8080"),
		User:      os.Getenv("API_USER"),
		Password:  os.Getenv("API_PASSWORD"),
		Origin:    os.Getenv("CORS_ORIGIN"),
		Pool:      8,
	}
	if c.Database == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	n, err := strconv.Atoi(env("DB_MAX_CONNECTIONS", "8"))
	if err != nil || n < 1 || n > 32 {
		return c, fmt.Errorf("DB_MAX_CONNECTIONS must be 1..32")
	}
	c.Pool = int32(n)
	return c, nil
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
