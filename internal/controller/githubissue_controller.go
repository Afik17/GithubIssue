package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	"github.com/Afik17/GithubIssue/internal/controller/finalizer"
	"github.com/Afik17/GithubIssue/internal/controller/utils"
	gh "github.com/Afik17/GithubIssue/internal/github"
)

// GithubIssueReconciler reconciles a GithubIssue object
type GithubIssueReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      events.EventRecorder
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

	if ghIssue.DeletionTimestamp != nil {
		repoOwner, repoName, _ := utils.ParseRepoURL(ghIssue.Spec.Repo)
		return r.Delete(ctx, ghIssue, repoOwner, repoName)
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

	return r.Sync(ctx, ghIssue)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubissuev1alpha1.GithubIssue{}).
		Named("githubissue").
		Complete(r)
}
