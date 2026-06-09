/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	gh "github.com/Afik17/GithubIssue/internal/github"
	"github.com/Afik17/GithubIssue/internal/github/mocks"
)

var _ = Describe("ApplyGithubIssue", func() {
	var (
		ctx       context.Context
		ctrl      *gomock.Controller
		ghClient  *mocks.MockIGitHub
		newIssue  *githubissuev1alpha1.GithubIssue
		repoOwner string
		repoName  string
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		ghClient = mocks.NewMockIGitHub(ctrl)
		repoOwner = "Afik17"
		repoName = "GithubIssue"
		newIssue = &githubissuev1alpha1.GithubIssue{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ghissue", Namespace: "default"},
			Spec: githubissuev1alpha1.GithubIssueSpec{
				Repo:        "https://github.com/Afik17/GithubIssue",
				Title:       "testing",
				Description: "fix the problem",
				Labels:      []string{"operator", "bug"},
				Assignees:   []string{"afik", "dvir"},
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("when no existing GitHub issue", func() {
		It("should create a new issue", func() {
			created := &gh.Issue{Number: 42, Title: newIssue.Spec.Title, State: "open"}
			ghClient.EXPECT().
				CreateIssue(ctx, repoOwner, repoName, gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ string, req *gh.Issue) (*gh.Issue, error) {
					Expect(req).To(Equal(&gh.Issue{
						Title:       "testing",
						Description: "fix the problem",
						Labels:      []string{"operator", "bug"},
						Assignees:   []string{"afik", "dvir"},
						State:       "open",
					}))
					return created, nil
				})

			result, err := ApplyGithubIssue(ctx, ghClient, newIssue, nil, repoOwner, repoName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(created))
		})

		It("should normalize nil labels and assignees to empty slices on create", func() {
			newIssue.Spec.Labels = nil
			newIssue.Spec.Assignees = nil

			ghClient.EXPECT().
				CreateIssue(ctx, repoOwner, repoName, gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ string, req *gh.Issue) (*gh.Issue, error) {
					Expect(req.Labels).To(Equal([]string{}))
					Expect(req.Assignees).To(Equal([]string{}))
					return &gh.Issue{Number: 1}, nil
				})

			_, err := ApplyGithubIssue(ctx, ghClient, newIssue, nil, repoOwner, repoName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return a wrapped error when create fails", func() {
			ghClient.EXPECT().
				CreateIssue(ctx, repoOwner, repoName, gomock.Any()).
				Return(nil, errors.New("rate limited"))

			_, err := ApplyGithubIssue(ctx, ghClient, newIssue, nil, repoOwner, repoName)
			Expect(err).To(MatchError(ContainSubstring("failed to create GitHub issue")))
			Expect(err).To(MatchError(ContainSubstring("rate limited")))
		})
	})

	Context("when an existing GitHub issue matches the spec", func() {
		It("should return the existing issue without calling the API", func() {
			existing := &gh.Issue{
				Number:      7,
				Title:       "testing",
				Description: "fix the problem",
				State:       "open",
				Labels:      []string{"bug", "operator"},
				Assignees:   []string{"dvir", "afik"},
			}

			result, err := ApplyGithubIssue(ctx, ghClient, newIssue, existing, repoOwner, repoName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(existing))
		})
	})

	Context("when an existing GitHub issue differs from the spec", func() {
		It("should update the issue", func() {
			existing := &gh.Issue{
				Number:      7,
				Title:       "testing",
				Description: "fixing error",
				State:       "open",
				Labels:      []string{"operator"},
				Assignees:   []string{"afik"},
			}
			updated := &gh.Issue{
				Number:      7,
				Title:       "testing",
				Description: "fix the problem",
				State:       "open",
				Labels:      []string{"operator", "sync"},
				Assignees:   []string{"afik", "dvir"},
			}

			ghClient.EXPECT().
				EditIssue(ctx, repoOwner, repoName, 7, gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ string, number int, req *gh.Issue) (*gh.Issue, error) {
					Expect(number).To(Equal(7))
					Expect(req.Description).To(Equal("fix the problem"))
					return updated, nil
				})

			result, err := ApplyGithubIssue(ctx, ghClient, newIssue, existing, repoOwner, repoName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(updated))
		})

		It("should return a wrapped error when update fails", func() {
			existing := &gh.Issue{
				Number:      7,
				Description: "fixing error",
				State:       "open",
			}

			ghClient.EXPECT().
				EditIssue(ctx, repoOwner, repoName, 7, gomock.Any()).
				Return(nil, errors.New("forbidden"))

			_, err := ApplyGithubIssue(ctx, ghClient, newIssue, existing, repoOwner, repoName)
			Expect(err).To(MatchError(ContainSubstring("failed to update GitHub issue 7")))
			Expect(err).To(MatchError(ContainSubstring("forbidden")))
		})
	})
})

var _ = Describe("GetGithubIssueNumberByAnnotation", func() {
	var ghIssue *githubissuev1alpha1.GithubIssue

	BeforeEach(func() {
		ghIssue = &githubissuev1alpha1.GithubIssue{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ghissue"},
		}
	})

	It("should return 0 when the annotation is missing", func() {
		Expect(GetGithubIssueNumberByAnnotation(context.Background(), ghIssue)).To(Equal(0))
	})

	It("should return the parsed issue number when the annotation is valid", func() {
		ghIssue.SetAnnotations(map[string]string{
			core.AnnotationIssueNumber: "123",
		})
		Expect(GetGithubIssueNumberByAnnotation(context.Background(), ghIssue)).To(Equal(123))
	})

	It("should return 0 when the annotation is not a valid integer", func() {
		ghIssue.SetAnnotations(map[string]string{
			core.AnnotationIssueNumber: "issue-7",
		})
		Expect(GetGithubIssueNumberByAnnotation(context.Background(), ghIssue)).To(Equal(0))
	})
})

var _ = Describe("EnsureGithubIssueNumberAnnotation", func() {
	var ghIssue *githubissuev1alpha1.GithubIssue

	BeforeEach(func() {
		ghIssue = &githubissuev1alpha1.GithubIssue{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ghissue"},
		}
	})

	It("should set the issue number annotation on the resource", func() {
		err := EnsureGithubIssueNumberAnnotation(context.Background(), ghIssue, 99)
		Expect(err).NotTo(HaveOccurred())
		Expect(ghIssue.GetAnnotations()).To(HaveKeyWithValue(core.AnnotationIssueNumber, "99"))
	})

	It("should initialize annotations when they are nil", func() {
		Expect(ghIssue.GetAnnotations()).To(BeNil())

		err := EnsureGithubIssueNumberAnnotation(context.Background(), ghIssue, 42)
		Expect(err).NotTo(HaveOccurred())
		Expect(ghIssue.GetAnnotations()).NotTo(BeNil())
		Expect(ghIssue.GetAnnotations()[core.AnnotationIssueNumber]).To(Equal("42"))
	})

	It("should preserve existing annotations", func() {
		ghIssue.SetAnnotations(map[string]string{"kubectl.kubernetes.io/last-applied-configuration": "{}"})

		err := EnsureGithubIssueNumberAnnotation(context.Background(), ghIssue, 5)
		Expect(err).NotTo(HaveOccurred())
		Expect(ghIssue.GetAnnotations()).To(HaveKey("kubectl.kubernetes.io/last-applied-configuration"))
		Expect(ghIssue.GetAnnotations()).To(HaveKeyWithValue(core.AnnotationIssueNumber, "5"))
	})
})

var _ = Describe("isGithubIssuesEqual", func() {
	var ghIssue *githubissuev1alpha1.GithubIssue

	BeforeEach(func() {
		ghIssue = &githubissuev1alpha1.GithubIssue{
			Spec: githubissuev1alpha1.GithubIssueSpec{
				Description: "fix critical bug in capp",
				Labels:      []string{"operator", "bug"},
				Assignees:   []string{"afik", "dvir"},
			},
		}
	})

	It("should return true when labels, assignees, description, and state match", func() {
		existing := &gh.Issue{
			State:       "open",
			Labels:      []string{"bug", "operator"},
			Assignees:   []string{"dvir", "afik"},
			Description: "fix critical bug in capp",
		}
		Expect(isGithubIssuesEqual(ghIssue, existing)).To(BeTrue())
	})

	It("should return false when the GitHub issue is not open", func() {
		existing := &gh.Issue{State: "closed"}
		Expect(isGithubIssuesEqual(ghIssue, existing)).To(BeFalse())
	})

	It("should return false when labels differ", func() {
		existing := &gh.Issue{
			State:       "open",
			Labels:      []string{"operator"},
			Assignees:   []string{"dvir", "afik"},
			Description: "fix critical bug in capp",
		}
		Expect(isGithubIssuesEqual(ghIssue, existing)).To(BeFalse())
	})

	It("should return false when assignees differ", func() {
		existing := &gh.Issue{
			State:       "open",
			Labels:      []string{"bug", "operator"},
			Assignees:   []string{"afik"},
			Description: "fix critical bug in capp",
		}
		Expect(isGithubIssuesEqual(ghIssue, existing)).To(BeFalse())
	})

	It("should return false when description differs", func() {
		existing := &gh.Issue{
			State:       "open",
			Labels:      []string{"bug", "operator"},
			Assignees:   []string{"dvir", "afik"},
			Description: "only happens when the finalizer is still present",
		}
		Expect(isGithubIssuesEqual(ghIssue, existing)).To(BeFalse())
	})
})
