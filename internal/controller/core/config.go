package core

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
)

func LoadGitHubToken(logger logr.Logger) (string, error) {
	githubToken, foundEnv := os.LookupEnv("GITHUB_TOKEN")
	if !foundEnv {
		logger.Info("GITHUB_TOKEN env not set")
		return "", fmt.Errorf("GITHUB_TOKEN environment variable is not set")
	}
	return githubToken, nil
}
