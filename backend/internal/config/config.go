package config

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TrustedProxies                                                         []netip.Prefix
	Database, Artifacts, Contract, Addr, AdminAddr, User, Password, Origin string
	Pool                                                                   int32
}

func Load() (Config, error) {
	c := Config{
		Database:  os.Getenv("DATABASE_URL"),
		Artifacts: env("ARTIFACTS_ROOT", "artifacts"),
		Contract:  env("OPENAPI_PATH", "openapi/openapi.yaml"),
		Addr:      env("HTTP_ADDR", "127.0.0.1:8080"),
		AdminAddr: env("ADMIN_ADDR", "127.0.0.1:9090"),
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
	for _, value := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		if strings.TrimSpace(value) == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return c, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS")
		}
		c.TrustedProxies = append(c.TrustedProxies, prefix)
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
