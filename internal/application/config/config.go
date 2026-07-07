package config

import (
	"os"
	"strconv"
)

// Config holds application-level (not infrastructure) settings. Today it is
// a single flag: whether the buyer (this company — single-tenant, no
// per-invoice variation) is a VAT withholding agent. When false, RETEIVA is
// never withheld, for any invoice or line.
type Config struct {
	IsVATWithholdingAgent bool
}

func FromEnv() Config {
	return Config{
		IsVATWithholdingAgent: getEnvBool("VAT_WITHHOLDING_AGENT", false),
	}
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
