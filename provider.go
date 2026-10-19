package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// PRTarget identifies a pull request or merge request across any git forge.
type PRTarget struct {
	Provider string // "github", "gitlab", etc.
	Owner    string // namespace/owner
	Repo     string // project/repo name
	Number   int    // PR/MR number
	URL      string // original URL
}

// PRMeta holds the normalized metadata px0 needs to check out a PR,
// compute diffs, and label the review UI.
type PRMeta struct {
	Number           int
	Title            string
	Author           string
	State            string
	Merged           bool
	MergedAt         string
	Draft            bool
	BaseRef          string
	HeadRef          string
	HeadSHA          string
	HeadRepoCloneURL string
	HeadIsFork       bool
}

// PRComment is a comment already posted on the pull request, fetched
// read-only from the forge -- distinct from pr.go's prComment, which is a
// draft held in memory until a review is submitted. Kind is "issue" (a
// top-level PR conversation comment) or "review" (anchored to a diff line).
type PRComment struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Path      string `json:"path,omitempty"`
	Line      int    `json:"line,omitempty"`
	Side      string `json:"side,omitempty"`
	InReplyTo int64  `json:"inReplyTo,omitempty"`
	Author    string `json:"author"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	URL       string `json:"url"`
}

// Merge methods, named for what lands on the base branch rather than for any
// one forge's API. Each provider maps them onto its own vocabulary (GitHub's
// merge/squash/rebase, GitLab's squash flag plus merge-method setting,
// Bitbucket's merge_commit/squash/fast_forward, Gitea's Do field).
const (
	MergeMethodMerge  = "merge"  // a merge commit joining the head into the base
	MergeMethodSquash = "squash" // the PR's commits squashed into one commit
	MergeMethodRebase = "rebase" // each commit replayed onto the base, no merge commit
)

// Merge states, normalized across forges. PRMergeStatus.State is one of these.
const (
	MergeStateClean     = "clean"     // mergeable, nothing in the way
	MergeStateUnstable  = "unstable"  // mergeable, but some non-required checks are failing
	MergeStateConflicts = "conflicts" // the head conflicts with the base branch
	MergeStateBehind    = "behind"    // no conflicts, but the base requires the head to be up to date
	MergeStateBlocked   = "blocked"   // branch protection (required reviews or checks) refuses it
	MergeStateDraft     = "draft"     // still a draft
	MergeStateUnknown   = "unknown"   // the forge hasn't computed it yet; ask again shortly
	MergeStateClosed    = "closed"
	MergeStateMerged    = "merged"
)

// PRMergeStatus is whether, and how, the PR can be merged right now.
type PRMergeStatus struct {
	State    string   `json:"state"`
	Reason   string   `json:"reason,omitempty"` // the forge's own detail, when it gives one
	Methods  []string `json:"methods"`          // allowed on this repository, in the forge's display order
	CanMerge bool     `json:"canMerge"`         // the token has the rights to merge
	HeadSHA  string   `json:"headSha"`          // the head the status was computed for
}

// PRMergeRequest is one merge, as the reviewer confirmed it.
type PRMergeRequest struct {
	Method  string
	Title   string // commit title; ignored for MergeMethodRebase
	Message string // commit body; ignored for MergeMethodRebase
	HeadSHA string // the head the reviewer saw; a PR that has moved past it must not be merged
}

// PRMergeResult is what the forge reports back about a completed merge.
type PRMergeResult struct {
	SHA     string `json:"sha"`
	Message string `json:"message,omitempty"`
}

// Errors a provider's Merge wraps, so callers can tell a stale head or a PR
// that can't merge from a transport failure without parsing forge text.
var (
	ErrPRHeadMoved    = errors.New("the PR head moved since it was loaded")
	ErrPRNotMergeable = errors.New("the PR cannot be merged")
)

// GitProvider abstracts forge-specific operations (GitHub, GitLab, etc.)
// for pull/merge request reviews.
type GitProvider interface {
	// Name returns the provider name (e.g. "github", "gitlab").
	Name() string

	// MatchURL reports whether this provider recognizes and handles the given URL.
	MatchURL(rawURL string) bool

	// ParseURL extracts the PR target from the URL.
	ParseURL(rawURL string) (PRTarget, error)

	// ResolveToken looks for an auth token across settings, env vars, and CLI tools.
	// An empty return means the session stays read-only.
	ResolveToken(cfg settings) (token, source string)

	// FetchPR fetches pull/merge request metadata from the forge API.
	FetchPR(ctx context.Context, target PRTarget, token string) (PRMeta, error)

	// CheckPushAccess reports whether the authenticated user has push access to the repository.
	CheckPushAccess(ctx context.Context, target PRTarget, token string) bool

	// SubmitReview posts draft comments and the overall review verdict back to the forge.
	SubmitReview(ctx context.Context, target PRTarget, token, headSHA string, comments []prComment, event, body string) error

	// FetchComments returns every comment already posted on the PR: top-level
	// ("issue") comments and inline ("review") comments anchored to a diff line.
	FetchComments(ctx context.Context, target PRTarget, token string) (issue, review []PRComment, err error)

	// PostIssueComment posts a new top-level PR comment immediately. GitHub has
	// no threading for these, so "replying" to one is just posting a new one.
	PostIssueComment(ctx context.Context, target PRTarget, token, body string) (PRComment, error)

	// ReplyToReviewComment posts an immediate, threaded reply to an existing
	// inline review comment.
	ReplyToReviewComment(ctx context.Context, target PRTarget, token string, commentID int64, body string) (PRComment, error)

	// MergeStatus reports whether the PR can merge right now and which merge
	// methods the repository allows. A forge that computes mergeability
	// lazily reports MergeStateUnknown until it has.
	MergeStatus(ctx context.Context, target PRTarget, token string) (PRMergeStatus, error)

	// Merge merges the PR on the forge. It must refuse, wrapping
	// ErrPRHeadMoved, when the PR head is no longer req.HeadSHA, and wrap
	// ErrPRNotMergeable when the forge refuses the merge itself.
	Merge(ctx context.Context, target PRTarget, token string, req PRMergeRequest) (PRMergeResult, error)
}

var defaultProviders = []GitProvider{
	&GitHubProvider{},
}

// RegisterProvider registers a custom or additional GitProvider.
func RegisterProvider(p GitProvider) {
	defaultProviders = append(defaultProviders, p)
}

// DetectPRURL checks if rawURL is a recognized pull request URL for any supported provider.
// If matched and valid, returns the matching provider, parsed target, and true.
func DetectPRURL(rawURL string) (GitProvider, PRTarget, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, PRTarget{}, false
	}
	for _, p := range defaultProviders {
		if p.MatchURL(rawURL) {
			target, err := p.ParseURL(rawURL)
			if err == nil {
				return p, target, true
			}
		}
	}
	return nil, PRTarget{}, false
}

// ParsePRURL attempts to parse a PR URL, returning an error if no provider matches
// or if the URL format is invalid.
func ParsePRURL(rawURL string) (GitProvider, PRTarget, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, PRTarget{}, fmt.Errorf("empty PR URL")
	}
	for _, p := range defaultProviders {
		if p.MatchURL(rawURL) {
			target, err := p.ParseURL(rawURL)
			if err != nil {
				return nil, PRTarget{}, err
			}
			return p, target, nil
		}
	}
	return nil, PRTarget{}, fmt.Errorf("unsupported or unrecognized PR URL: %q (expected full GitHub URL like https://github.com/owner/repo/pull/123)", rawURL)
}
