package utils

import (
	"errors"
	"net/url"
	"strings"
)

// ParseRepoURL extracts the owner and repository name from a Git URL.
func ParseRepoURL(repoURL string) (string, string, error) {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return "", "", errors.New("repository URL cannot be empty")
	}

	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimSuffix(repoURL, "/")

	var path string

	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		u, err := url.Parse(repoURL)
		if err != nil {
			return "", "", errors.New("failed to parse HTTPS URL")
		}
		path = u.Path
	} else {
		return "", "", errors.New("unsupported protocol: URL must start with http://, https://")
	}

	path = strings.TrimPrefix(path, "/")

	pathParts := strings.Split(path, "/")
	if len(pathParts) != 2 {
		return "", "", errors.New("invalid repository path: expected format 'owner/repo'")
	}

	owner := pathParts[0]
	repo := pathParts[1]

	if owner == "" || repo == "" {
		return "", "", errors.New("repository owner and name cannot be empty")
	}

	return owner, repo, nil
}
