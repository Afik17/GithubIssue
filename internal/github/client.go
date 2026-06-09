package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v88/github"
)

//go:generate mockgen -destination=./mocks/mock_github.go -package=mocks -source=client.go IGitHub

type IGitHub interface {
	ListIssues(ctx context.Context, owner, repo, state string) ([]*Issue, error)

	GetIssueByTitle(ctx context.Context, owner, repo, title, state string) (*Issue, error)

	GetIssueByNumber(ctx context.Context, owner, repo string, number int) (*Issue, error)

	CreateIssue(ctx context.Context, owner, repo string, issueRequest *Issue) (*Issue, error)

	EditIssue(ctx context.Context, owner, repo string, number int, issue *Issue) (*Issue, error)

	CloseIssue(ctx context.Context, owner, repo string, number int) error

	IssueHasLinkedPR(ctx context.Context, owner, repo string, number int) (bool, error)
}

type GitHubManager struct {
	ghClient *github.Client
}

func NewGitHubManager(token string) (IGitHub, error) {
	client, err := github.NewClient(github.WithAuthToken(token))
	if err != nil {
		return &GitHubManager{}, fmt.Errorf("Error creating GitHub client: %v", err)
	}

	return &GitHubManager{
		ghClient: client,
	}, nil
}

func (c *GitHubManager) ListIssues(ctx context.Context, owner, repo, state string) ([]*Issue, error) {
	opts := &github.IssueListByRepoOptions{State: state}
	ghIssues, _, err := c.ghClient.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	var domainIssues []*Issue
	for _, ghIssue := range ghIssues {
		domainIssues = append(domainIssues, toDomainIssue(ghIssue))
	}
	return domainIssues, nil
}

func (c *GitHubManager) GetIssueByTitle(ctx context.Context, owner, repo, title, state string) (*Issue, error) {
	issues, err := c.ListIssues(ctx, owner, repo, state)
	if err != nil {
		return nil, translateError(err)
	}

	for _, issue := range issues {
		if issue.Title == title {
			return issue, nil
		}
	}
	return nil, nil
}

func (c *GitHubManager) GetIssueByNumber(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	ghIssue, resp, err := c.ghClient.Issues.Get(ctx, owner, repo, number)
	if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toDomainIssue(ghIssue), nil
}

func (c *GitHubManager) CreateIssue(ctx context.Context, owner, repo string, issue *Issue) (*Issue, error) {
	req := &github.IssueRequest{
		Title:     github.String(issue.Title),
		Body:      github.String(issue.Description),
		Labels:    &issue.Labels,
		Assignees: &issue.Assignees,
	}

	created, _, err := c.ghClient.Issues.Create(ctx, owner, repo, req)
	if err != nil {
		return nil, translateError(err)
	}
	return toDomainIssue(created), nil
}

func (c *GitHubManager) EditIssue(ctx context.Context, owner, repo string, number int, issue *Issue) (*Issue, error) {
	req := &github.IssueRequest{
		Title:     github.String(issue.Title),
		Body:      github.String(issue.Description),
		Labels:    &issue.Labels,
		Assignees: &issue.Assignees,
		State:     github.String(issue.State),
	}

	edited, _, err := c.ghClient.Issues.Edit(ctx, owner, repo, number, req)
	if err != nil {
		return nil, translateError(err)
	}
	return toDomainIssue(edited), nil
}

func (c *GitHubManager) CloseIssue(ctx context.Context, owner, repo string, number int) error {
	req := &github.IssueRequest{State: github.String("closed")}
	_, resp, err := c.ghClient.Issues.Edit(ctx, owner, repo, number, req)

	if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone) {
		return nil
	}
	return err
}

func (c *GitHubManager) IssueHasLinkedPR(ctx context.Context, owner, repo string, number int) (bool, error) {
	opts := &github.ListOptions{PerPage: 100}
	hasPR := false

	for {
		events, resp, err := c.ghClient.Issues.ListIssueTimeline(ctx, owner, repo, number, opts)
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to fetch issue timeline: %w", err)
		}

		for _, event := range events {
			if event.GetEvent() == "connected" || event.GetEvent() == "cross-referenced" {
				hasPR = true
			}
			if event.GetEvent() == "disconnected" {
				hasPR = false
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return hasPR, nil
}
