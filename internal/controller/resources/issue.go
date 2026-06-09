package resources

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	gh "github.com/Afik17/GithubIssue/internal/github"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyGithubIssue ensures the remote GitHub issue matches the K8s Spec using the internal domain struct.
func ApplyGithubIssue(ctx context.Context, ghClient gh.IGitHub, newIssue *githubissuev1alpha1.GithubIssue, existingIssue *gh.Issue, repoOwner, repoName string) (*gh.Issue, error) {

	assignees := newIssue.Spec.Assignees
	if assignees == nil {
		assignees = []string{}
	}

	labels := newIssue.Spec.Labels
	if labels == nil {
		labels = []string{}
	}
	desiredIssue := &gh.Issue{
		Title:       newIssue.Spec.Title,
		Description: newIssue.Spec.Description,
		Labels:      labels,
		Assignees:   assignees,
		State:       "open",
	}

	if existingIssue == nil {
		createdIssue, err := ghClient.CreateIssue(ctx, repoOwner, repoName, desiredIssue)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub issue: %w", err)
		}
		return createdIssue, nil
	}

	if !isGithubIssuesEqual(newIssue, existingIssue) {
		updatedIssue, err := ghClient.EditIssue(ctx, repoOwner, repoName, existingIssue.Number, desiredIssue)
		if err != nil {
			return nil, fmt.Errorf("failed to update GitHub issue %d: %w", existingIssue.Number, err)
		}
		return updatedIssue, nil
	}

	return existingIssue, nil
}

func GetGithubIssueNumberByAnnotation(ctx context.Context, ghIssue *githubissuev1alpha1.GithubIssue) int {
	annotations := ghIssue.GetAnnotations()
	if issueNumber, ok := annotations[core.AnnotationIssueNumber]; ok {
		if num, err := strconv.Atoi(issueNumber); err == nil {
			return num
		}
	}
	return 0
}

func EnsureGithubIssueNumberAnnotation(ctx context.Context, k8sClient client.Client, ghIssue *githubissuev1alpha1.GithubIssue, issueNumber int) error {
	annotations := ghIssue.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[core.AnnotationIssueNumber] = strconv.Itoa(issueNumber)
	ghIssue.SetAnnotations(annotations)
	return nil
}

// isGithubIssuesEqual compares the K8s Spec to the internal domain issue.
func isGithubIssuesEqual(ghIssue *githubissuev1alpha1.GithubIssue, existingIssue *gh.Issue) bool {

	// Compare states
	if existingIssue.State != "open" {
		return false
	}

	// Compare Labels
	existingLabels := slices.Clone(existingIssue.Labels)
	sort.Strings(existingLabels)

	newLabels := slices.Clone(ghIssue.Spec.Labels)
	sort.Strings(newLabels)

	if !slices.Equal(existingLabels, newLabels) {
		return false
	}

	// Compare Assignees
	existingAssignees := slices.Clone(existingIssue.Assignees)
	sort.Strings(existingAssignees)

	newAssignees := slices.Clone(ghIssue.Spec.Assignees)
	sort.Strings(newAssignees)

	if !slices.Equal(existingAssignees, newAssignees) {
		return false
	}

	// Compare Description
	existingDescription := strings.ReplaceAll(existingIssue.Description, "\r\n", "\n")
	newDescription := strings.ReplaceAll(ghIssue.Spec.Description, "\r\n", "\n")
	if existingDescription != newDescription {
		return false
	}

	return true
}
