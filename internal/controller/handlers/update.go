package handlers

import (
	"context"
	"fmt"

	"github.com/Afik17/GithubIssue/internal/controller/core"
	"github.com/Afik17/GithubIssue/internal/controller/resources"
	gh "github.com/Afik17/GithubIssue/internal/github"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
)

// Update ensures the GitHub issue exists and is synchronized with the GithubIssue spec.
// It handles both creation and updates of the GitHub issue. If the issue already exists, it will be updated to match the spec.
// After applying the issue, it ensures that the GithubIssue resource has the correct issue number annotation.
func Update(ctx context.Context, k8sClient client.Client, ghClient gh.IGitHub, ghIssue *githubissuev1alpha1.GithubIssue, repoOwner, repoName string, issueNumberByAnnotation int) (*gh.Issue, error) {
	var existingIssue *gh.Issue
	var err error

	if issueNumberByAnnotation != 0 {
		existingIssue, err = ghClient.GetIssueByNumber(ctx, repoOwner, repoName, issueNumberByAnnotation)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch issue by number: %w", err)
		}
	} else {
		existingIssue, err = ghClient.GetIssueByTitle(ctx, repoOwner, repoName, ghIssue.Spec.Title, core.IssueOpenState)
		if err != nil {
			return nil, fmt.Errorf("failed to search issue by title: %w", err)
		}
	}

	appliedIssue, err := resources.ApplyGithubIssue(ctx, ghClient, ghIssue, existingIssue, repoOwner, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to apply GitHub issue: %w", err)
	}

	if appliedIssue.Number == issueNumberByAnnotation {
		return appliedIssue, nil
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latestIssue := &githubissuev1alpha1.GithubIssue{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ghIssue), latestIssue); err != nil {
			return err
		}

		if err := resources.EnsureGithubIssueNumberAnnotation(ctx, latestIssue, appliedIssue.Number); err != nil {
			return err
		}

		return k8sClient.Update(ctx, latestIssue)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update GithubIssue with issue number annotation: %w", err)
	}

	return appliedIssue, nil
}
