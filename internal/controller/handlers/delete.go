package handlers

import (
	"context"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	"github.com/Afik17/GithubIssue/internal/controller/finalizer"
	"github.com/Afik17/GithubIssue/internal/controller/resources"
	gh "github.com/Afik17/GithubIssue/internal/github"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// HandleDelete clean up any resources associated with the issue, such as closing the corresponding GitHub issue if it exists
func HandleDelete(ctx context.Context, k8sClient client.Client, ghClient gh.IGitHub, ghIssue *githubissuev1alpha1.GithubIssue, repoOwner, repoName string) error {
	if !controllerutil.ContainsFinalizer(ghIssue, core.GHIssueDeletionFinalizer) {
		return nil
	}

	// If the issue was never created, just remove the finalizer on the next step
	if resources.IsGithubIssueManaged(ctx, ghIssue) {
		return nil
	}

	err := ghClient.CloseIssue(ctx, repoOwner, repoName, ghIssue.Status.Number)
	if err != nil {
		return err
	}

	if err := finalizer.Remove(ctx, k8sClient, ghIssue, core.GHIssueDeletionFinalizer); err != nil {
		return err
	}

	return nil
}
