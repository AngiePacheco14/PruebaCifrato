package config

import (
	"os"
	"strconv"
)

// Config holds application-level settings: whether the buyer is a VAT
// withholding agent. When false, RETEIVA is never withheld.
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
