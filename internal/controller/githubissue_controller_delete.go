package controller

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	"github.com/Afik17/GithubIssue/internal/controller/finalizer"
	"github.com/Afik17/GithubIssue/internal/controller/handlers"
)

// Delete represents the delete stage
func (r *GithubIssueReconciler) Delete(ctx context.Context, ghIssue *githubissuev1alpha1.GithubIssue, repoOwner, repoName string) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(ghIssue, core.GHIssueDeletionFinalizer) {
		return ctrl.Result{}, nil
	}

	if err := handlers.DeleteGithubIssue(ctx, r.GithubManager, ghIssue, repoOwner, repoName); err != nil {
		logger.Error(err, "failed to cleanup GitHub issue")
		return ctrl.Result{RequeueAfter: time.Second * core.RequeueIntervalSeconds}, nil
	}

	logger.Info("GitHub issue cleaned up successfully, removing finalizer")
	if err := finalizer.Remove(ctx, r.Client, ghIssue, core.GHIssueDeletionFinalizer); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
