package finalizer

import (
	"context"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Remove(ctx context.Context, k8sClient client.Client, ghIssue *githubissuev1alpha1.GithubIssue, finalizer string) error {
	controllerutil.RemoveFinalizer(ghIssue, finalizer)
	if err := k8sClient.Update(ctx, ghIssue); err != nil {
		return err
	}
	return nil
}

func Ensure(ctx context.Context, k8sClient client.Client, ghIssue *githubissuev1alpha1.GithubIssue, finalizer string) (bool, error) {
	changed := false
	if !controllerutil.ContainsFinalizer(ghIssue, finalizer) {
		controllerutil.AddFinalizer(ghIssue, finalizer)
		if err := k8sClient.Update(ctx, ghIssue); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
}
