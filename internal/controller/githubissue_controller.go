package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	"github.com/Afik17/GithubIssue/internal/controller/finalizer"
	"github.com/Afik17/GithubIssue/internal/controller/handlers"
	"github.com/Afik17/GithubIssue/internal/controller/resources"
	"github.com/Afik17/GithubIssue/internal/controller/status"
	"github.com/Afik17/GithubIssue/internal/controller/utils"
	gh "github.com/Afik17/GithubIssue/internal/github"
)

// GithubIssueReconciler reconciles a GithubIssue object
type GithubIssueReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	GithubManager gh.IGitHub
}

// +kubebuilder:rbac:groups=githubissue.dana.io,resources=githubissues,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=githubissue.dana.io,resources=githubissues/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=githubissue.dana.io,resources=githubissues/finalizers,verbs=update

func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Reconciling")

	ghIssue := &githubissuev1alpha1.GithubIssue{}
	if err := r.Get(ctx, req.NamespacedName, ghIssue); err != nil {
		logger.Info("unable to fetch GithubIssue", "name", ghIssue.Name, "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Delete stage
	if ghIssue.DeletionTimestamp != nil {
		repoOwner, repoName, _ := utils.ParseRepoURL(ghIssue.Spec.Repo)
		err := handlers.HandleDelete(ctx, r.Client, r.GithubManager, ghIssue, repoOwner, repoName)
		if err != nil {
			logger.Error(err, "failed to cleanup GitHub issue during deletion")
			return ctrl.Result{RequeueAfter: time.Second * core.RequeueIntervalSeconds}, nil
		}
	}

	change, err := finalizer.Ensure(ctx, r.Client, ghIssue, core.GHIssueDeletionFinalizer)
	if err != nil {
		logger.Error(err, "failed to ensure finalizer")
		return ctrl.Result{}, fmt.Errorf("failed to ensure finalizer: %w", err)
	}
	if change {
		logger.Info("Finalizer added, requeuing for next reconciliation")
		return ctrl.Result{}, nil
	}

	// Sync stage
	repoOwner, repoName, _ := utils.ParseRepoURL(ghIssue.Spec.Repo)
	issueNumberByAnnotation := resources.GetGithubIssueNumberByAnnotation(ctx, ghIssue)
	appliedIssue, syncErr := handlers.Update(ctx, r.Client, r.GithubManager, ghIssue, repoOwner, repoName, issueNumberByAnnotation)
	if syncErr != nil {
		return r.handleSyncError(ctx, logger, ghIssue, appliedIssue, syncErr)
	}

	if issueNumberByAnnotation != appliedIssue.Number {
		logger.Info("Issue number annotation updated, requeuing for next reconciliation")
		return ctrl.Result{}, nil
	}

	hasPR, err := r.GithubManager.IssueHasLinkedPR(ctx, repoOwner, repoName, appliedIssue.Number)
	if err != nil {
		logger.Error(err, "failed to check if GitHub issue has linked PR")
		return ctrl.Result{}, fmt.Errorf("failed to check if GitHub issue has linked PR: %w", err)
	}
	appliedIssue.HasPR = hasPR

	if err := status.UpdateStatus(ctx, r.Client, r.Recorder, ghIssue, appliedIssue, syncErr); err != nil {
		logger.Error(err, "failed to update GithubIssue status")
		return ctrl.Result{}, fmt.Errorf("failed to update GithubIssue status on success sync: %w", err)
	}

	logger.Info("Reconciliation completed")
	return ctrl.Result{}, nil
}

// handleSyncError processes sync errors and updates the K8s status
func (r *GithubIssueReconciler) handleSyncError(ctx context.Context, logger logr.Logger, ghIssue *githubissuev1alpha1.GithubIssue, appliedIssue *gh.Issue, syncErr error) (ctrl.Result, error) {
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

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubissuev1alpha1.GithubIssue{}).
		Named("githubissue").
		Complete(r)
}
