package github

import "github.com/google/go-github/v88/github"

type Issue struct {
	Number      int
	Title       string
	Description string
	State       string
	Labels      []string
	Assignees   []string
	HasPR       bool
}

func toDomainIssue(ghIssue *github.Issue) *Issue {
	if ghIssue == nil {
		return nil
	}

	domainIssue := &Issue{
		Number:      ghIssue.GetNumber(),
		Title:       ghIssue.GetTitle(),
		Description: ghIssue.GetBody(),
		State:       ghIssue.GetState(),
		HasPR:       ghIssue.PullRequestLinks != nil,
	}

	for _, label := range ghIssue.Labels {
		domainIssue.Labels = append(domainIssue.Labels, label.GetName())
	}

	for _, assignee := range ghIssue.Assignees {
		domainIssue.Assignees = append(domainIssue.Assignees, assignee.GetLogin())
	}

	return domainIssue
}
