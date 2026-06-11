package handlers

import (
	"context"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/resources"
	gh "github.com/Afik17/GithubIssue/internal/github"
)

// DeleteGithubIssue clean up any resources associated with the issue, such as closing the corresponding GitHub issue if it exists
func DeleteGithubIssue(ctx context.Context, ghClient gh.IGitHub, ghIssue *githubissuev1alpha1.GithubIssue, repoOwner, repoName string) error {
	// If the issue was never created, just remove the finalizer on the next step
	if !resources.IsGithubIssueManaged(ctx, ghIssue) {
		return nil
	}

	err := ghClient.CloseIssue(ctx, repoOwner, repoName, ghIssue.Status.Number)
	if err != nil {
		return err
	}

	return nil
}
