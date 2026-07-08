package anthropic

import "os"

const defaultModel = "claude-haiku-4-5"

// ModelFromEnv resolves the Claude model from the CLASSIFIER_MODEL env var,
// defaulting to defaultModel.
func ModelFromEnv() string {
	if v := os.Getenv("CLASSIFIER_MODEL"); v != "" {
		return v
	}
	return defaultModel
}
