package controller

import (
	"context"
	"errors"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/handlers"
	"github.com/Afik17/GithubIssue/internal/controller/resources"
	"github.com/Afik17/GithubIssue/internal/controller/status"
	"github.com/Afik17/GithubIssue/internal/controller/utils"
	gh "github.com/Afik17/GithubIssue/internal/github"
)

// Sync represents the sync stage. It orchestrates the creation, updating, and validation of the GitHub Issue.
func (r *GithubIssueReconciler) Sync(ctx context.Context, ghIssue *githubissuev1alpha1.GithubIssue) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	repoOwner, repoName, err := utils.ParseRepoURL(ghIssue.Spec.Repo)
	if err != nil {
		logger.Error(err, "invalid repository URL provided in Spec")
		return ctrl.Result{}, nil
	}

	issueNumberByAnnotation := resources.GetGithubIssueNumberByAnnotation(ctx, ghIssue)
	appliedIssue, syncErr := handlers.UpdateGithubIssue(ctx, r.Client, r.GithubManager, ghIssue, repoOwner, repoName, issueNumberByAnnotation)

	if syncErr != nil {
		return r.handleSyncError(ctx, ghIssue, appliedIssue, syncErr)
	}

	if issueNumberByAnnotation != appliedIssue.Number {
		logger.Info("Issue number annotation updated, waiting for K8s to re-trigger reconcile")
		return ctrl.Result{}, nil
	}

	hasPR, err := r.GithubManager.IssueHasLinkedPR(ctx, repoOwner, repoName, appliedIssue.Number)
	if err != nil {
		logger.Error(err, "failed to check if GitHub issue has linked PR")
		return ctrl.Result{}, fmt.Errorf("failed to check if GitHub issue has linked PR: %w", err)
	}
	appliedIssue.HasPR = hasPR

	// 6. Update the Kubernetes Status to reflect success
	if err := status.UpdateStatus(ctx, r.Client, r.Recorder, ghIssue, appliedIssue, nil); err != nil {
		logger.Error(err, "failed to update GithubIssue status after successful sync")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleSyncError processes sync errors and updates the K8s status
func (r *GithubIssueReconciler) handleSyncError(ctx context.Context, ghIssue *githubissuev1alpha1.GithubIssue, appliedIssue *gh.Issue, syncErr error) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	if err := status.UpdateStatus(ctx, r.Client, r.Recorder, ghIssue, appliedIssue, syncErr); err != nil {
		logger.Error(err, "failed to update GithubIssue status")
		return ctrl.Result{}, err
	}

	var ghSyncErr *gh.GitHubError
	if errors.As(syncErr, &ghSyncErr) {
		logger.Info("failed to sync GitHub Issue", "reason", ghSyncErr.Error())
		return ctrl.Result{}, nil
	}

	// Handle unexpected infra errors
	logger.Error(syncErr, "failed to sync Github Issue")
	return ctrl.Result{}, fmt.Errorf("failed to sync GitHub issue: %w", syncErr)
}
