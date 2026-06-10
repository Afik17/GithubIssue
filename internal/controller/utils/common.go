package utils

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

var supportedProtocols = []string{"http", "https"}

func ParseRepoURL(repoURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil {
		return "", "", err
	}

	if !slices.Contains(supportedProtocols, u.Scheme) {
		return "", "", fmt.Errorf("unsupported protocol %q", u.Scheme)
	}

	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", errors.New("invalid repository path: expected format 'owner/repo'")
	}

	owner, repo := parts[0], parts[1]

	if owner == "" || repo == "" {
		return "", "", errors.New("repository owner and name cannot be empty")
	}

	return owner, repo, nil
}
