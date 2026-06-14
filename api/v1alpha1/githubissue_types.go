package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionTypeIssueSync  = "IssueSynced"
	ConditionTypeIssueOpen  = "IssueOpen"
	ConditionTypeIssueHasPR = "IssueHasPR"

	ConditionReasonSync        = "IssueSynced"
	ConditionReasonSyncErr     = "SyncErr"
	ConditionReasonIssueFound  = "IssueOpen"
	ConditionReasonIssueClosed = "IssueClosed"
	ConditionReasonPRFound     = "PRLinked"
	ConditionReasonPRNotFound  = "PRNotLinked"

	ConditionMsgIssueSync   = "The GitHub issue is currently synchronized"
	ConditionMsgIssueOpen   = "The GitHub issue is currently open"
	ConditionMsgIssueClosed = "The GitHub issue is currently closed"
	ConditionMsgPRLinked    = "A pull request is linked to this issue"
	ConditionMsgPRNotLinked = "No pull request is linked to this issue"
)

type GithubIssueSpec struct {
	// +kubebuilder:validation:Pattern=`^https://github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+(?:\.git)?$`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="repo is immutable once created"
	Repo string `json:"repo"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="title is immutable once created"
	Title string `json:"title"`

	// +optional
	Description string `json:"description,omitempty"`

	// +optional
	Assignees []string `json:"assignees,omitempty"`

	// +optional
	Labels []string `json:"labels,omitempty"`
}

type GithubIssueStatus struct {
	Number int `json:"number,omitempty"`

	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type GithubIssue struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec GithubIssueSpec `json:"spec"`

	// +optional
	Status GithubIssueStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

type GithubIssueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []GithubIssue `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GithubIssue{}, &GithubIssueList{})
}
