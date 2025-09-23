package collaborator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	teamrepo "github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers/teamRepo"
	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/utils"
)

// NewResponseFlattener creates a new flattener with field mappings
func NewResponseFlattener(mappings []utils.FieldMapping) *utils.ResponseFlattener {
	return &utils.ResponseFlattener{Mappings: mappings}
}

// GitHubUserPermissionFlattener for GitHub user permission responses
// Preserves all root fields and brings nested user fields to root
var GitHubUserPermissionFlattener = NewResponseFlattener([]utils.FieldMapping{
	{SourcePath: "user.permissions", TargetKey: "permissions"},
	{SourcePath: "user.html_url", TargetKey: "html_url"},
	{SourcePath: "user.id", TargetKey: "id"},
})

// Convenience function for GitHub user permission responses
func FlattenGitHubUserPermission(resp *http.Response) ([]byte, error) {
	return GitHubUserPermissionFlattener.FlattenResponse(resp)
}

// FlattenGitHubUserPermissionBytes flattens GitHub permission response from bytes
func FlattenGitHubUserPermissionBytes(body []byte) ([]byte, error) {
	return GitHubUserPermissionFlattener.FlattenBytes(body)
}

/*
The following is a utility function to deal with GitHub permissions discrepancies in the API responses

Discrepancies for a Collaborator:

'permission' in CR		'permission' in GitHub RESPONSE		'role_name' in GitHub RESPONSE
pull              		read                                read
push                  	write                               write
admin                 	admin                               admin
maintain              	write                               maintain
triage                	read                                triage
*/

// CorrectGitHubPermissionField corrects the permission field in GitHub API responses
func CorrectGitHubUserPermissionField(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// change the permission filed based on the role_name
	// but if the role_name is read, then set the permission to pull
	// and if the role_name is write, then set the permission to push
	roleName, exists := data["role_name"].(string)
	if !exists {
		return body, nil // No roleName field to correct
	}
	var permission string
	switch roleName {
	case "read":
		permission = "pull"
	case "write":
		permission = "push"
	case "admin":
		permission = "admin"
	case "maintain":
		permission = "maintain"
	case "triage":
		permission = "triage"
	default:
		permission = roleName // Use the roleName as is if it doesn't match any known roles
	}
	if permission == "" {
		return body, nil // No permission to correct
	}

	// Update the permission field in the data map
	if data["permission"] == nil {
		data["permission"] = make(map[string]interface{})
	}
	data["permission"] = permission

	// Marshal the updated data back to JSON
	return json.Marshal(data)
}

// function to add a field to the response body
func AddFieldToResponse(body []byte, fieldName string, fieldValue interface{}) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	data[fieldName] = fieldValue

	return json.Marshal(data)
}

// Invitations handling
// This section deals with GitHub repository invitations, allowing us to check if a user has been invited to collaborate on a repository.
// It includes parsing the invitation response and checking if a user exists in the invitations

// GitHubInvitation represents a GitHub repository invitation
type GitHubInvitation struct {
	ID         int64               `json:"id"`
	NodeID     string              `json:"node_id"`
	Repository teamrepo.Repository `json:"repository"`
	Invitee    struct {
		Login     string `json:"login"`
		ID        int64  `json:"id"`
		NodeID    string `json:"node_id"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
		Type      string `json:"type"`
	} `json:"invitee"`
	Inviter struct {
		Login     string `json:"login"`
		ID        int64  `json:"id"`
		NodeID    string `json:"node_id"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
		Type      string `json:"type"`
	} `json:"inviter"`
	Permissions string `json:"permissions"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	Expired     bool   `json:"expired"`
}

// parseInvitations parses invitation response body into slice of GitHubInvitation(s)
func parseInvitations(inviteBody []byte) ([]GitHubInvitation, error) {
	var invitations []GitHubInvitation
	if err := json.Unmarshal(inviteBody, &invitations); err != nil {
		return nil, err
	}
	return invitations, nil
}

// getUserInvitationFromPage checks if a username exists in a page of invitations
// returns the invitation (if found) along with a boolean indicating if it was found
// Here we could potentially check if the invitation is expired or not, if needed
func getUserInvitationFromPage(inviteBody []byte, username string) (*GitHubInvitation, bool) {
	invitations, err := parseInvitations(inviteBody)
	if err != nil {
		return nil, false
	}

	for _, invitation := range invitations {
		if strings.EqualFold(invitation.Invitee.Login, username) {
			return &invitation, true
		}
	}
	return nil, false
}

// Function to change the `permissions` (with the `s`) field in the request body of a PATCH request of an invitation
// before sending it to the GitHub API
// We need to change the `permission` field to `permissions` and map the permission value
// Note that in the case of Invitations, the `permissions` field is just a string with a single permission value
// and not an object like in the collaborator response.
func CorrectGitHubUserPermissionsFieldReqBody(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	// Check if the permission field exists,
	// save content, add a new `permissions` field with the mapping
	// and remove the old `permission` field
	permission, exists := data["permission"].(string)
	if !exists {
		return body, nil // No permission field to correct
	}

	// Map the permission to the permissions field
	var permissions string
	switch permission {
	case "pull":
		permissions = "read"
	case "push":
		permissions = "write"
	case "admin":
		permissions = "admin"
	case "maintain":
		permissions = "maintain"
	case "triage":
		permissions = "triage"
	default:
		permissions = permission // Use the permission as is if it doesn't match any known roles
	}

	// Update the data map with the new permissions field
	if data["permissions"] == nil {
		data["permissions"] = make(map[string]interface{})
	}
	data["permissions"] = permissions

	// Remove the old permission field
	if _, exists := data["permission"]; exists {
		delete(data, "permission")
	}

	return json.Marshal(data)
}

// function to read the field from a body and return the value (generic function)
func ReadFieldFromBody(body []byte, fieldName string) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if the field exists in the data map
	value, exists := data[fieldName]
	if !exists {
		return nil, fmt.Errorf("field %s not found", fieldName)
	}

	return value, nil
}
