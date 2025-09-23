# Krateo Github Plugin for `rest-dynamic-controller`

A specialized web service that addresses some inconsistencies in the GitHub REST API
It is designed to work with the [`rest-dynamic-controller`](https://github.com/krateoplatformops/rest-dynamic-controller/) and [`github-provider-kog`](https://github.com/krateoplatformops/github-provider-kog-chart)

## Summary

- [Summary](#summary)
- [API Endpoints](#api-endpoints)
  - [Collaborator](#collaborator)
    - [Get Repository Collaborator Permission](#get-repository-collaborator-permission)
    - [Add Repository Collaborator](#add-repository-collaborator)
    - [Update Repository Collaborator Permission](#update-repository-collaborator-permission)
    - [Remove Repository Collaborator](#remove-repository-collaborator)
  - [TeamRepo](#teamrepo)
    - [Get TeamRepo Permission](#get-teamrepo-permission)
- [Swagger Documentation](#swagger-documentation)
- [GitHub API Reference](#github-api-reference)
- [Authentication](#authentication)


## API Endpoints

### Collaborator

All "Collaborator" endpoints handle both direct repository collaborators and external collaborators invitations (users not in the organization).

#### Get Repository Collaborator Permission

```http
GET /repository/{owner}/{repo}/collaborators/{username}/permission
```

**Description**: 
It retrieves the permission level of a user for a specific repository if the user is a collaborator.
If the user is not a collaborator, it returns `404 Not Found`.
Therefore, even if the user is invited to be an external collaborator, it will return `404 Not Found` if the user has not accepted the invitation yet.

**Why This Endpoint Exists**: 
- The standard GitHub API returns `200 OK` instead of `404 Not Found` when checking permissions for users who were previously collaborators but have been removed.
- It normalizes permission values: `write` → `push`, `read` → `pull`.
- It includes `html_url`, `id`, and `permissions` at root level.

**Path parameters**:
- `owner` (string, required): Repository owner
- `repo` (string, required): Repository name  
- `username` (string, required): Username to check permission for

<details>
<summary><b>Response example</b></summary>

```json
{
  "html_url":"<REDACTED>",  // Adjusted field
  "id": 12345678,           // Adjusted field
  "permission":"admin",     // Adjusted field
  "permissions":{           // Adjusted field
    "admin":true,
    "maintain":true,
    "pull":true,
    "push":true,
    "triage":true
  },
  "role_name":"admin",
  "user":{
    "avatar_url":"<REDACTED>",
    "events_url":"<REDACTED>",
    "followers_url":"<REDACTED>",
    "following_url":"<REDACTED>",
    "gists_url":"<REDACTED>",
    "gravatar_id":"<REDACTED>",
    "html_url":"<REDACTED>",
    "id":12345678,
    "login":"<REDACTED>",
    "node_id":"<REDACTED>",
    "organizations_url":"<REDACTED>",
    "permissions":{
      "admin":true,
      "maintain":true,
      "pull":true,
      "push":true,
      "triage":true
    },
    "received_events_url":"<REDACTED>",
    "repos_url":"<REDACTED>",
    "role_name":"admin",
    "site_admin":false,
    "starred_url":"<REDACTED>",
    "subscriptions_url":"<REDACTED>",
    "type":"User",
    "url":"<REDACTED>",
    "user_view_type":"public"
  }
}
```
</details>

#### Add Repository Collaborator

```http
POST /repository/{owner}/{repo}/collaborators/{username}
```

**Description**: 
It adds a user as a repository collaborator or sends an invitation if they are not an organization member.

**Why This Endpoint Exists**:
- GitHub REST API already provides the dual functionality of adding collaborators and sending invitations with a single endpoint.
- However, this endpoint returns `202 Accepted` when an invitation is sent to external users, allowing the `rest-dynamic-controller` to maintain proper resource state management ("pending" state in Collaborator custom resource).

**Path parameters**:
- `owner` (string, required): Repository owner
- `repo` (string, required): Repository name
- `username` (string, required): Username to add as collaborator

**Request Body**:
```json
{
  "permission": "push"
}
```

**Permission Values (in request body)**:
`pull`, `push`, `admin`, `maintain`, `triage`

**Responses**:
- `202 Accepted`: Invitation sent to external user
- `204 No Content`: User added as direct collaborator

#### Update Repository Collaborator Permission

```http
PATCH /repository/{owner}/{repo}/collaborators/{username}
```

**Description**: 
It updates permission level for existing collaborators or pending invitations.

**Why This Endpoint Exists**:
- It handles both active collaborators and pending invitations with 2 differents calls to the GitHub API.
- It returns `202 Accepted` when an invitation is sent to external users, allowing the `rest-dynamic-controller` to maintain proper resource state management ("pending" state in Collaborator custom resource).
- It normalizes permission values (`write` → `push`, `read` → `pull`) when necessary.

**Path parameters**:
- `owner` (string, required): Repository owner
- `repo` (string, required): Repository name
- `username` (string, required): Username to update permission for

**Request Body**:
```json
{
  "permission": "admin"
}
```

**Permission Values (in request body)**:
`pull`, `push`, `admin`, `maintain`, `triage`

**Responses**:
- `200 OK`: Collaborator permission updated
- `202 Accepted`: Invitation permission updated

#### Remove Repository Collaborator

```http
DELETE /repository/{owner}/{repo}/collaborators/{username}
```

**Description**: 
It removes a collaborator or cancels a pending invitation.

**Why This Endpoint Exists**:
- It provides unified handling for both removing active collaborators and canceling pending invitations with 2 different calls to the GitHub API based on the user status.

**Path parameters**:
- `owner` (string, required): Repository owner
- `repo` (string, required): Repository name
- `username` (string, required): Username to remove as collaborator or cancel invitation for

**Responses**:
- `200 OK`: Collaborator removed
- `202 Accepted`: Invitation cancelled  
- `404 Not Found`: User not found as collaborator or invitee

### TeamRepo

#### Get TeamRepo Permission

```http
GET /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
```

**Description**: 
It retrieves repository permissions for a specific team.

**Why This Endpoint Exists**:
- It sets the required `application/vnd.github.v3.repository+json` Accept header. Without this header, GitHub API returns `204 No Content` instead of permission details.
- It normalizes permission values (`write` → `push`, `read` → `pull`).
- It adds `owner` field at root level for easier access.

**Parameters**:
- `org` (string, required): Organization name
- `team_slug` (string, required): Team slug
- `owner` (string, required): Repository owner
- `repo` (string, required): Repository name

<details>
<summary><b>Response example</b></summary>

```json
{
  "allow_auto_merge":false,
  "allow_forking":false,
  "allow_merge_commit":true,
  "allow_rebase_merge":true,
  "allow_squash_merge":true,
  "archive_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/{archive_format}{/ref}",
  "archived":false,
  "assignees_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/assignees{/user}",
  "blobs_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/blobs{/sha}",
  "branches_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/branches{/branch}",
  "clone_url":"https://github.com/krateoplatformops-test/test-teamrepo.git",
  "collaborators_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/collaborators{/collaborator}",
  "comments_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/comments{/number}",
  "commits_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/commits{/sha}",
  "compare_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/compare/{base}...{head}",
  "contents_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/contents/{+path}",
  "contributors_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/contributors",
  "created_at":"2025-06-10T17:15:43Z",
  "default_branch":"main",
  "delete_branch_on_merge":false,
  "deployments_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/deployments",
  "description":null,
  "disabled":false,
  "downloads_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/downloads",
  "events_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/events",
  "fork":false,
  "forks":0,
  "forks_count":0,
  "forks_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/forks",
  "full_name":"krateoplatformops-test/test-teamrepo",
  "git_commits_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/commits{/sha}",
  "git_refs_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/refs{/sha}",
  "git_tags_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/tags{/sha}",
  "git_url":"git://github.com/krateoplatformops-test/test-teamrepo.git",
  "has_downloads":true,
  "has_issues":true,
  "has_pages":false,
  "has_projects":true,
  "has_wiki":false,
  "homepage":null,
  "hooks_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/hooks",
  "html_url":"https://github.com/krateoplatformops-test/test-teamrepo",
  "id":999717066,
  "issue_comment_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/issues/comments{/number}",
  "issue_events_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/issues/events{/number}",
  "issues_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/issues{/number}",
  "keys_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/keys{/key_id}",
  "labels_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/labels{/name}",
  "language":null,
  "languages_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/languages",
  "license":null,
  "merges_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/merges",
  "milestones_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/milestones{/number}",
  "mirror_url":null,
  "name":"test-teamrepo",
  "node_id":"R_kgDOO5Z4yg",
  "notifications_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/notifications{?since,all,participating}",
  "open_issues":0,
  "open_issues_count":0,
  "owner":"krateoplatformops-test", // Adjusted field
  "permission":"pull",              // Adjusted field
  "private":true,
  "pulls_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/pulls{/number}",
  "pushed_at":"2025-06-10T17:15:43Z",
  "releases_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/releases{/id}",
  "role_name":"read",
  "size":0,
  "ssh_url":"git@github.com:krateoplatformops-test/test-teamrepo.git",
  "stargazers_count":0,
  "stargazers_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/stargazers",
  "statuses_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/statuses/{sha}",
  "subscribers_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/subscribers",
  "subscription_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/subscription",
  "svn_url":"https://github.com/krateoplatformops-test/test-teamrepo",
  "tags_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/tags",
  "teams_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/teams",
  "topics":[],
  "trees_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/trees{/sha}",
  "updated_at":"2025-06-10T17:15:43Z",
  "url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo",
  "visibility":"private",
  "watchers":0,
  "watchers_count":0
}   
```
</details>

## Swagger Documentation

For more detailed information about the API endpoints, please refer to the Swagger documentation available at `/swagger/index.html` endpoint of the service.

## GitHub API Reference

For complete GitHub REST API documentation, visit: [GitHub REST API docs](https://docs.github.com/en/rest?apiVersion=2022-11-28)

## Authentication

The plugin will forward the `Authorization` header passed in the request to this plugin to the GitHub REST API.
