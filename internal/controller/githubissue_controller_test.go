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

package controller

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	githubissuev1alpha1 "github.com/Afik17/GithubIssue/api/v1alpha1"
	"github.com/Afik17/GithubIssue/internal/controller/core"
	gh "github.com/Afik17/GithubIssue/internal/github"
	"github.com/Afik17/GithubIssue/internal/github/mocks"
)

const (
	testRepoOwner = "Afik17"
	testRepoName  = "GithubIssue"
	testRepoURL   = "https://github.com/Afik17/GithubIssue"
)

func newTestGithubIssue(name string) *githubissuev1alpha1.GithubIssue {
	return &githubissuev1alpha1.GithubIssue{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: githubissuev1alpha1.GithubIssueSpec{
			Repo:        testRepoURL,
			Title:       "new source for capp",
			Description: "new kafka source",
			Labels:      []string{"operator", "knative"},
			Assignees:   []string{"afik", "dvir"},
		},
	}
}

var _ = Describe("GithubIssue Controller", func() {
	var (
		testCtx    context.Context
		gomockCtrl *gomock.Controller
		ghMock     *mocks.MockIGitHub
		reconciler *GithubIssueReconciler
	)

	BeforeEach(func() {
		testCtx = context.Background()
		gomockCtrl = gomock.NewController(GinkgoT())
		ghMock = mocks.NewMockIGitHub(gomockCtrl)
		reconciler = &GithubIssueReconciler{
			Client:        k8sClient,
			Scheme:        scheme.Scheme,
			Recorder:      events.NewFakeRecorder(10),
			GithubManager: ghMock,
		}
	})

	AfterEach(func() {
		gomockCtrl.Finish()
	})

	ensureFinalizer := func(req reconcile.Request) {
		_, err := reconciler.Reconcile(testCtx, req)
		Expect(err).NotTo(HaveOccurred())

		ghIssue := &githubissuev1alpha1.GithubIssue{}
		Expect(k8sClient.Get(testCtx, req.NamespacedName, ghIssue)).To(Succeed())
		Expect(ghIssue.Finalizers).To(ContainElement(core.GHIssueDeletionFinalizer))
	}

	deleteGithubIssueIfExists := func(name string) {
		ghIssue := &githubissuev1alpha1.GithubIssue{}
		key := types.NamespacedName{Name: name, Namespace: "default"}
		err := k8sClient.Get(testCtx, key, ghIssue)
		if apierrors.IsNotFound(err) {
			return
		}
		Expect(err).NotTo(HaveOccurred())
		ghIssue.Finalizers = nil
		Expect(client.IgnoreNotFound(k8sClient.Update(testCtx, ghIssue))).To(Succeed())
		Expect(client.IgnoreNotFound(k8sClient.Delete(testCtx, ghIssue))).To(Succeed())
	}

	Context("when syncing with GitHub", func() {
		const resourceName = "sync-github-issue"

		var req reconcile.Request

		BeforeEach(func() {
			deleteGithubIssueIfExists(resourceName)
			ghIssue := newTestGithubIssue(resourceName)
			Expect(k8sClient.Create(testCtx, ghIssue)).To(Succeed())
			req = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ghIssue)}
			ensureFinalizer(req)
		})

		AfterEach(func() {
			deleteGithubIssueIfExists(resourceName)
		})

		It("should create a GitHub issue when one does not exist", func() {
			created := &gh.Issue{
				Number:      17,
				Title:       "new source for capp",
				Description: "new kafka source",
				State:       "open",
				Labels:      []string{"operator", "knative"},
				Assignees:   []string{"afik", "dvir"},
			}

			ghMock.EXPECT().
				GetIssueByTitle(testCtx, testRepoOwner, testRepoName, created.Title, "open").
				Return(nil, nil)
			ghMock.EXPECT().
				CreateIssue(testCtx, testRepoOwner, testRepoName, gomock.Any()).
				Return(created, nil)

			_, err := reconciler.Reconcile(testCtx, req)
			Expect(err).NotTo(HaveOccurred())

			ghIssue := &githubissuev1alpha1.GithubIssue{}
			Expect(k8sClient.Get(testCtx, req.NamespacedName, ghIssue)).To(Succeed())
			Expect(ghIssue.Annotations).To(HaveKeyWithValue(core.AnnotationIssueNumber, "17"))

			synced := &gh.Issue{
				Number:      17,
				Title:       created.Title,
				Description: created.Description,
				State:       "open",
				Labels:      []string{"knative", "operator"},
				Assignees:   []string{"dvir", "afik"},
			}
			ghMock.EXPECT().
				GetIssueByNumber(testCtx, testRepoOwner, testRepoName, 17).
				Return(synced, nil)
			ghMock.EXPECT().
				IssueHasLinkedPR(testCtx, testRepoOwner, testRepoName, 17).
				Return(false, nil)

			_, err = reconciler.Reconcile(testCtx, req)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(testCtx, req.NamespacedName, ghIssue)).To(Succeed())
			Expect(ghIssue.Status.Number).To(Equal(17))

			syncCond := meta.FindStatusCondition(ghIssue.Status.Conditions, githubissuev1alpha1.ConditionTypeIssueSync)
			Expect(syncCond).NotTo(BeNil())
			Expect(syncCond.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should record a failed sync when GitHub issue creation fails", func() {
			ghMock.EXPECT().
				GetIssueByTitle(testCtx, testRepoOwner, testRepoName, gomock.Any(), "open").
				Return(nil, nil)
			ghMock.EXPECT().
				CreateIssue(testCtx, testRepoOwner, testRepoName, gomock.Any()).
				Return(nil, &gh.GitHubError{Reason: "ValidationFailed", Err: errors.New("title is too long")})

			_, err := reconciler.Reconcile(testCtx, req)
			Expect(err).NotTo(HaveOccurred())

			ghIssue := &githubissuev1alpha1.GithubIssue{}
			Expect(k8sClient.Get(testCtx, req.NamespacedName, ghIssue)).To(Succeed())
			Expect(ghIssue.Status.Number).To(Equal(0))

			syncCond := meta.FindStatusCondition(ghIssue.Status.Conditions, githubissuev1alpha1.ConditionTypeIssueSync)
			Expect(syncCond).NotTo(BeNil())
			Expect(syncCond.Status).To(Equal(metav1.ConditionFalse))
			Expect(syncCond.Reason).To(Equal("ValidationFailed"))
		})

		It("should record a failed sync when updating an existing GitHub issue fails", func() {
			ghIssue := &githubissuev1alpha1.GithubIssue{}
			Expect(k8sClient.Get(testCtx, req.NamespacedName, ghIssue)).To(Succeed())

			ghIssue.Annotations = map[string]string{core.AnnotationIssueNumber: "7"}
			Expect(k8sClient.Update(testCtx, ghIssue)).To(Succeed())

			existing := &gh.Issue{
				Number:      7,
				Title:       ghIssue.Spec.Title,
				Description: "happens only on clusters running v0.1.0",
				State:       "open",
				Labels:      []string{"operator"},
				Assignees:   []string{"afik"},
			}

			ghMock.EXPECT().
				GetIssueByNumber(testCtx, testRepoOwner, testRepoName, 7).
				Return(existing, nil)
			ghMock.EXPECT().
				EditIssue(testCtx, testRepoOwner, testRepoName, 7, gomock.Any()).
				Return(nil, &gh.GitHubError{Reason: "RateLimited", Err: errors.New("secondary rate limit")})

			_, err := reconciler.Reconcile(testCtx, req)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(testCtx, req.NamespacedName, ghIssue)).To(Succeed())
			Expect(ghIssue.Status.Number).To(Equal(0))

			syncCond := meta.FindStatusCondition(ghIssue.Status.Conditions, githubissuev1alpha1.ConditionTypeIssueSync)
			Expect(syncCond).NotTo(BeNil())
			Expect(syncCond.Status).To(Equal(metav1.ConditionFalse))
			Expect(syncCond.Reason).To(Equal("RateLimited"))
		})
	})

	Context("when the GithubIssue is deleted", func() {
		const resourceName = "delete-github-issue"

		var req reconcile.Request

		BeforeEach(func() {
			deleteGithubIssueIfExists(resourceName)

			ghIssue := newTestGithubIssue(resourceName)
			ghIssue.Finalizers = []string{core.GHIssueDeletionFinalizer}

			if ghIssue.Annotations == nil {
				ghIssue.Annotations = make(map[string]string)
			}
			ghIssue.Annotations[core.AnnotationIssueNumber] = "11"

			Expect(k8sClient.Create(testCtx, ghIssue)).To(Succeed())

			ghIssue.Status.Number = 11
			Expect(k8sClient.Status().Update(testCtx, ghIssue)).To(Succeed())

			req = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(ghIssue)}
			Expect(k8sClient.Delete(testCtx, ghIssue)).To(Succeed())
		})

		AfterEach(func() {
			deleteGithubIssueIfExists(resourceName)
		})

		It("should close the GitHub issue and remove the finalizer", func() {
			ghMock.EXPECT().
				CloseIssue(testCtx, testRepoOwner, testRepoName, 11).
				Return(nil)

			result, err := reconciler.Reconcile(testCtx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			Eventually(func() error {
				ghIssue := &githubissuev1alpha1.GithubIssue{}
				return k8sClient.Get(testCtx, req.NamespacedName, ghIssue)
			}, time.Second*5, time.Millisecond*100).Should(MatchError(apierrors.IsNotFound, "not found"))
		})
	})
})
