package anthropic

import "os"

const defaultModel = "claude-haiku-4-5"

// ModelFromEnv resolves which Claude model classifies invoice lines from
// the CLASSIFIER_MODEL env var, defaulting to defaultModel. This is
// infrastructure config for this adapter (analogous to
// postgres.ConfigFromEnv()), not a business rule — it does not belong in
// application.Config.
func ModelFromEnv() string {
	if v := os.Getenv("CLASSIFIER_MODEL"); v != "" {
		return v
	}
	return defaultModel
}
