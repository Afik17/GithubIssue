package github

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v88/github"
)

type GitHubError struct {
	Reason string
	Err    error
}

func (e *GitHubError) Error() string {
	return fmt.Sprintf("[%s] %v", e.Reason, e.Err)
}

func (e *GitHubError) Unwrap() error {
	return e.Err
}

func translateError(err error) error {
	if err == nil {
		return nil
	}

	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		switch ghErr.Response.StatusCode {
		case http.StatusNotFound:
			return &GitHubError{Reason: "NotFound", Err: err}
		case http.StatusUnprocessableEntity:
			return &GitHubError{Reason: "ValidationFailed", Err: err}
		case http.StatusForbidden:
			return &GitHubError{Reason: "RateLimited", Err: err}
		}
	}

	return err
}
