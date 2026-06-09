package status

import (
	"context"
	"errors"
	"fmt"

	"github.com/Afik17/GithubIssue/internal/controller/core"
	gh "github.com/Afik17/GithubIssue/internal/github"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatus updates the status of the GithubIssue CR based on the state of the corresponding GitHub issue
func UpdateStatus(
	ctx context.Context,
	k8sClient client.Client,
	recorder record.EventRecorder,
	ghIssue *githubissuev1alpha1.GithubIssue,
	appliedIssue *gh.Issue,
	syncErr error,
) error {

	if syncErr != nil {
		setFailedSyncStatus(recorder, ghIssue, syncErr)
	} else if appliedIssue != nil {
		setSuccessSyncStatus(recorder, ghIssue, appliedIssue)
	}

	if err := k8sClient.Status().Update(ctx, ghIssue); err != nil {
		return fmt.Errorf("failed to update GithubIssue status: %w", err)
	}

	return nil
}

func setSuccessSyncStatus(recorder record.EventRecorder, ghIssue *githubissuev1alpha1.GithubIssue, appliedIssue *gh.Issue) {
	ghIssue.Status.Number = appliedIssue.Number

	meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
		Type:    githubissuev1alpha1.ConditionTypeIssueSync,
		Status:  metav1.ConditionTrue,
		Reason:  githubissuev1alpha1.ConditionReasonSync,
		Message: githubissuev1alpha1.ConditionMsgIssueSync,
	})

	if appliedIssue.State == core.IssueOpenState {
		recorder.Eventf(ghIssue, corev1.EventTypeNormal, githubissuev1alpha1.ConditionReasonIssueFound, "GitHub issue #%d is open", appliedIssue.Number)
		meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
			Type:    githubissuev1alpha1.ConditionTypeIssueOpen,
			Status:  metav1.ConditionTrue,
			Reason:  githubissuev1alpha1.ConditionReasonIssueFound,
			Message: githubissuev1alpha1.ConditionMsgIssueOpen,
		})
	} else {
		recorder.Eventf(ghIssue, corev1.EventTypeNormal, githubissuev1alpha1.ConditionReasonIssueClosed, "GitHub issue #%d is closed", appliedIssue.Number)
		meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
			Type:    githubissuev1alpha1.ConditionTypeIssueOpen,
			Status:  metav1.ConditionFalse,
			Reason:  githubissuev1alpha1.ConditionReasonIssueClosed,
			Message: githubissuev1alpha1.ConditionMsgIssueClosed,
		})
	}

	if appliedIssue.HasPR {
		recorder.Eventf(ghIssue, corev1.EventTypeNormal, githubissuev1alpha1.ConditionReasonPRFound, "A pull request is linked to GitHub issue #%d", appliedIssue.Number)
		meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
			Type:    githubissuev1alpha1.ConditionTypeIssueHasPR,
			Status:  metav1.ConditionTrue,
			Reason:  githubissuev1alpha1.ConditionReasonPRFound,
			Message: githubissuev1alpha1.ConditionMsgPRLinked,
		})
	} else {
		recorder.Eventf(ghIssue, corev1.EventTypeNormal, githubissuev1alpha1.ConditionReasonPRNotFound, "No pull request is linked to GitHub issue #%d", appliedIssue.Number)
		meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
			Type:    githubissuev1alpha1.ConditionTypeIssueHasPR,
			Status:  metav1.ConditionFalse,
			Reason:  githubissuev1alpha1.ConditionReasonPRNotFound,
			Message: githubissuev1alpha1.ConditionMsgPRNotLinked,
		})
	}
}

func setFailedSyncStatus(recorder record.EventRecorder, ghIssue *githubissuev1alpha1.GithubIssue, syncErr error) {
	recorder.Eventf(ghIssue, corev1.EventTypeWarning, "GitHubError", "Failed to sync issue:%v", syncErr)

	reason := githubissuev1alpha1.ConditionReasonSyncErr
	var ghSyncErr *gh.GitHubError
	if errors.As(syncErr, &ghSyncErr) {
		reason = ghSyncErr.Reason
	}

	meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
		Type:    githubissuev1alpha1.ConditionTypeIssueSync,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: syncErr.Error(),
	})

	meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
		Type:    githubissuev1alpha1.ConditionTypeIssueOpen,
		Status:  metav1.ConditionUnknown,
		Reason:  githubissuev1alpha1.ConditionReasonSyncErr,
		Message: "Cannot determine issue state due to sync error",
	})

	meta.SetStatusCondition(&ghIssue.Status.Conditions, metav1.Condition{
		Type:    githubissuev1alpha1.ConditionTypeIssueHasPR,
		Status:  metav1.ConditionUnknown,
		Reason:  githubissuev1alpha1.ConditionReasonSyncErr,
		Message: "Cannot determine issue state due to sync error",
	})
}
