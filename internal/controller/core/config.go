package core

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
)

func LoadGitHubToken(logger logr.Logger) (string, error) {
	githubToken, foundEnv := os.LookupEnv("GH_TOKEN")
	if !foundEnv {
		logger.Info("GH_TOKEN env not set")
		return "", fmt.Errorf("GH_TOKEN environment variable is not set")
	}
	return githubToken, nil
}
