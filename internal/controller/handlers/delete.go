package handlers

import (
	"context"
	"fmt"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	gh "github.com/Afik17/GithubIssue/internal/github"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// HandleDelete clean up any resources associated with the issue, such as closing the corresponding GitHub issue if it exists
func HandleDelete(ctx context.Context, ghClient gh.IGitHub, ghIssue *githubissuev1alpha1.GithubIssue, repoOwner, repoName string) (bool, error) {
	if !controllerutil.ContainsFinalizer(ghIssue, core.GHIssueDeletionFinalizer) {
		return false, nil
	}

	// If the issue was never created, just remove the finalizer on the next step
	if ghIssue.Status.Number == 0 {
		return true, nil
	}

	err := ghClient.CloseIssue(ctx, repoOwner, repoName, ghIssue.Status.Number)
	if err != nil {
		return false, fmt.Errorf("cleanup error, failed to close issue %q: %w", ghIssue.Status.Number, err)
	}

	return true, nil
}
