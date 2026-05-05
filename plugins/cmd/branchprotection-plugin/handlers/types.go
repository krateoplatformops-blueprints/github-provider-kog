package branchprotection

// BranchProtectionRequest is the GitHub PUT request body for branch protection.
// The four top-level object fields (required_status_checks, enforce_admins,
// required_pull_request_reviews, restrictions) are required by the GitHub API;
// the plugin fills in null/false defaults for any that are absent.
type BranchProtectionRequest struct {
	// Require status checks to pass before merging. Set to `null` to disable.
	RequiredStatusChecks *RequiredStatusChecks `json:"required_status_checks" validate:"required" extensions:"x-nullable"`
	// Enforce all configured restrictions for administrators. Set to `true` to enforce required status checks for repository administrators. Set to `null` to disable.
	EnforceAdmins *bool `json:"enforce_admins" validate:"required" extensions:"x-nullable"`
	// Require at least one approving review on a pull request, before merging. Set to `null` to disable.
	RequiredPullRequestReviews *RequiredPullRequestReviews `json:"required_pull_request_reviews" validate:"required" extensions:"x-nullable"`
	// Restrict who can push to the protected branch. User, app, and team restrictions are only available for organization-owned repositories. Set to null to disable.
	Restrictions *Restrictions `json:"restrictions" validate:"required" extensions:"x-nullable"`
	// Enforces a linear commit Git history, which prevents anyone from pushing merge commits to a branch. Set to true to enforce a linear commit history. Set to false to disable. Default: false.
	RequiredLinearHistory *bool `json:"required_linear_history,omitempty"`
	// Permits force pushes to the protected branch by anyone with write access to the repository. Set to true to allow force pushes. Set to false or null to block force pushes. Default: false.
	AllowForcePushes *bool `json:"allow_force_pushes,omitempty" extensions:"x-nullable"`
	// Allows deletion of the protected branch by anyone with write access to the repository. Set to false to prevent deletion of the protected branch. Default: false.
	AllowDeletions *bool `json:"allow_deletions,omitempty"`
	// If set to true, the restrictions branch protection settings which limits who can push will also block pushes which create new branches, unless the push is initiated by a user, team, or app which has the ability to push. Default: false.
	BlockCreations *bool `json:"block_creations,omitempty"`
	// Requires all conversations on code to be resolved before a pull request can be merged into a branch that matches this rule. Set to false to disable. Default: false.
	RequiredConversationResolution *bool `json:"required_conversation_resolution,omitempty"`
	// Whether to set the branch as read-only. If this is true, users will not be able to push to the branch. Default: false.
	LockBranch *bool `json:"lock_branch,omitempty" default:"false"`
	// Whether users can pull changes from upstream when the branch is locked. Set to true to allow fork syncing. Set to false to prevent fork syncing. Default: false.
	AllowForkSyncing *bool `json:"allow_fork_syncing,omitempty" default:"false"`
}

// Require status checks to pass before merging. Set to `null` to disable.
type RequiredStatusChecks struct {
	// Require branches to be up to date before merging.
	Strict bool `json:"strict" validate:"required"`
	// The list of status checks to require in order to merge into this branch.
	Checks []StatusCheck `json:"checks,omitempty" validate:"required"`
	// Note: `contexts` field is deprectaed by GitHub API in favor of `checks`.
	// Not supported by GitHub Provider KOG.
}

// The list of status checks to require in order to merge into this branch.
type StatusCheck struct {
	// The name of the required check.
	Context string `json:"context" validate:"required"`
	// The ID of the GitHub App that must provide this check. Omit this field to automatically select the GitHub App that has recently provided this check, or any app if it was not set by a GitHub App. Pass -1 to explicitly allow any app to set the status.
	AppID *int `json:"app_id"`
}

// RequiredPullRequestReviews defines PR review requirements.
type RequiredPullRequestReviews struct {
	// Specify which users, teams, and apps can dismiss pull request reviews. Pass an empty `dismissal_restrictions` object to disable. User and team `dismissal_restrictions` are only available for organization-owned repositories. Omit this parameter for personal repositories.
	DismissalRestrictions *UserTeamAppList `json:"dismissal_restrictions,omitempty"`
	// Set to true if you want to automatically dismiss approving reviews when someone pushes a new commit.
	DismissStaleReviews bool `json:"dismiss_stale_reviews,omitempty"`
	// Blocks merging pull requests until code owners review them.
	RequireCodeOwnerReviews bool `json:"require_code_owner_reviews,omitempty"`
	// Specify the number of reviewers required to approve pull requests. Use a number between 1 and 6 or 0 to not require reviewers.
	RequiredApprovingReviewCount int `json:"required_approving_review_count,omitempty"`
	// Whether the most recent push must be approved by someone other than the person who pushed it. Default: false.
	RequireLastPushApproval bool `json:"require_last_push_approval,omitempty" default:"false"`
	// Allow specific users, teams, or apps to bypass pull request requirements.
	BypassPullRequestAllowances *BypassAllowances `json:"bypass_pull_request_allowances,omitempty"`
}

// UserTeamAppList holds lists of user logins, team slugs, and app slugs with dismissal access.
type UserTeamAppList struct {
	// The list of user `login`s with dismissal access
	Users []string `json:"users,omitempty"`
	// The list of team `slug`s with dismissal access
	Teams []string `json:"teams,omitempty"`
	// The list of app `slug`s with dismissal access
	Apps []string `json:"apps,omitempty"`
}

// BypassAllowances holds lists of users, teams, and app slugs allowed to bypass pull request requirements.
type BypassAllowances struct {
	// The list of user `login`s allowed to bypass pull request requirements.
	Users []string `json:"users,omitempty"`
	// The list of team `slug`s allowed to bypass pull request requirements.
	Teams []string `json:"teams,omitempty"`
	// The list of app `slug`s allowed to bypass pull request requirements.
	Apps []string `json:"apps,omitempty"`
}

// Restrictions defines push access restrictions (org-owned repos only).
type Restrictions struct {
	// The list of user logins with push access.
	Users []string `json:"users" validate:"required"`
	// The list of team slugs with push access.
	Teams []string `json:"teams" validate:"required"`
	// The list of app slugs with push access.
	Apps []string `json:"apps,omitempty"`
}

// BranchProtectionResponse is the normalized GET/PUT response returned by the plugin.
// Boolean fields that GitHub wraps in { "enabled": bool } objects are unwrapped here.
type BranchProtectionResponse struct {
	// The URL of the branch protection resource.
	URL string `json:"url,omitempty"`
	// Require status checks to pass before merging.
	RequiredStatusChecks *RequiredStatusChecks `json:"required_status_checks,omitempty"`
	// Whether admins are subject to the branch protection rules.
	EnforceAdmins bool `json:"enforce_admins,omitempty"`
	// Pull request review requirements.
	RequiredPullRequestReviews *RequiredPullRequestReviewsResponse `json:"required_pull_request_reviews,omitempty"`
	// Push access restrictions.
	Restrictions *Restrictions `json:"restrictions,omitempty"`
	// Whether a linear commit Git history is enforced.
	RequiredLinearHistory bool `json:"required_linear_history,omitempty"`
	// Whether force pushes are permitted.
	AllowForcePushes bool `json:"allow_force_pushes,omitempty"`
	// Whether deletion of the protected branch is allowed.
	AllowDeletions bool `json:"allow_deletions,omitempty"`
	// Whether pushes that create new branches are blocked.
	BlockCreations bool `json:"block_creations,omitempty"`
	// Whether all conversations on code must be resolved before merging.
	RequiredConversationResolution bool `json:"required_conversation_resolution,omitempty"`
	// Whether the branch is read-only.
	LockBranch bool `json:"lock_branch,omitempty"`
	// Whether users can pull changes from upstream when the branch is locked.
	AllowForkSyncing bool `json:"allow_fork_syncing,omitempty"`
}

// RequiredPullRequestReviewsResponse mirrors RequiredPullRequestReviews for response bodies.
type RequiredPullRequestReviewsResponse struct {
	// Users, teams, and apps that can dismiss pull request reviews.
	DismissalRestrictions *UserTeamAppList `json:"dismissal_restrictions,omitempty"`
	// Whether approving reviews are automatically dismissed when someone pushes a new commit.
	DismissStaleReviews bool `json:"dismiss_stale_reviews,omitempty"`
	// Whether merging pull requests is blocked until code owners review them.
	RequireCodeOwnerReviews bool `json:"require_code_owner_reviews,omitempty"`
	// The number of reviewers required to approve pull requests.
	RequiredApprovingReviewCount int `json:"required_approving_review_count,omitempty"`
	// Whether the most recent push must be approved by someone other than the person who pushed it.
	RequireLastPushApproval bool `json:"require_last_push_approval,omitempty"`
	// Users, teams, or apps allowed to bypass pull request requirements.
	BypassPullRequestAllowances *BypassAllowances `json:"bypass_pull_request_allowances,omitempty"`
}

// ErrorResponse is the error body returned on API failures.
type ErrorResponse struct {
	// A message describing the error.
	Message string `json:"message"`
	// A URL to the documentation for this error.
	DocumentationURL string `json:"documentation_url,omitempty"`
}
