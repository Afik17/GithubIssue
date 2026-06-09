package core

import "time"

const (
	SyncPeriod             = time.Minute * 1
	RequeueIntervalSeconds = 5

	GHIssueDeletionFinalizer = "githubissue.dana.io/finalizer"
	AnnotationIssueNumber    = "githubissue.dana.io/issue-number"
)
