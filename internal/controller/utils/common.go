package utils

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var supportedProtocols = []string{"http", "https"}

func ParseRepoURL(repoURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(repoURL))
	if err != nil {
		return "", "", err
	}

	// You must manually loop through the slice to find a match
	isValid := false
	for _, p := range supportedProtocols {
		if u.Scheme == p {
			isValid = true
			break
		}
	}

	if !isValid {
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
