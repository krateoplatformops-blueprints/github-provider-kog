package collaborator

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
)

// Handler constructors
func GetCollaborator(opts handlers.HandlerOptions) handlers.Handler {
	return &getHandler{baseHandler: newBaseHandler(opts)}
}

func PostCollaborator(opts handlers.HandlerOptions) handlers.Handler {
	return &postHandler{baseHandler: newBaseHandler(opts)}
}

func PatchCollaborator(opts handlers.HandlerOptions) handlers.Handler {
	return &patchHandler{baseHandler: newBaseHandler(opts)}
}

func DeleteCollaborator(opts handlers.HandlerOptions) handlers.Handler {
	return &deleteHandler{baseHandler: newBaseHandler(opts)}
}

// Interface compliance verification
var _ handlers.Handler = &getHandler{}
var _ handlers.Handler = &postHandler{}
var _ handlers.Handler = &patchHandler{}
var _ handlers.Handler = &deleteHandler{}

// Base handler with common functionality
type baseHandler struct {
	handlers.HandlerOptions
}

// Constructor for the base handler
func newBaseHandler(opts handlers.HandlerOptions) *baseHandler {
	return &baseHandler{HandlerOptions: opts}
}

// Handler types embedding the base handler
type getHandler struct {
	*baseHandler
}

type postHandler struct {
	*baseHandler
}

type patchHandler struct {
	*baseHandler
}

type deleteHandler struct {
	*baseHandler
}

// Common types and constants
type CollaboratorStatus int

const (
	StatusNotCollaborator CollaboratorStatus = iota
	StatusCollaborator
	StatusPendingInvitation
)

// Common methods, defined once on baseHandler
func (h *baseHandler) makeGitHubRequest(method, url, authHeader string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	if body != nil && len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func (h *baseHandler) checkCollaboratorStatus(owner, repo, username, authHeader string) (CollaboratorStatus, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := h.makeGitHubRequest("GET", url, authHeader, nil)
	if err != nil {
		return StatusNotCollaborator, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return StatusCollaborator, nil
	case http.StatusNotFound:
		return StatusNotCollaborator, nil
	default:
		return StatusNotCollaborator, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (h *baseHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	h.Log.Print(message)
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

func (h *baseHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(body)
}

func (h *baseHandler) findUserInvitation(owner, repo, username, authHeader string) (*GitHubInvitation, bool, error) {
	return findUserInvitationHelper(h.Client, h.Log, owner, repo, username, authHeader)
}

// GET handler implementation
// @Summary Get the permission of a user in a repository
// @Description Get the permission of a user in a repository
// @ID get-repo-permission
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator"
// @Produce json
// @Success 200 {object} collaborator.RepoPermissions
// @Router /repository/{owner}/{repo}/collaborators/{username}/permission [get]
func (h *getHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Getting permission for user %s in repository %s/%s", username, owner, repo)

	status, err := h.checkCollaboratorStatus(owner, repo, username, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error checking collaborator status: %v", err))
		return
	}

	if status != StatusCollaborator {
		h.Log.Printf("User %s is not a collaborator of repository %s/%s, or the user does not exist", username, owner, repo)
		h.writeErrorResponse(w, http.StatusNotFound, "User is not a collaborator of the repository or the user does not exist")
		return
	}

	// Get user permission
	err = h.getUserPermissionAndRespond(w, owner, repo, username, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error getting user permission: %v", err))
	}
}

func (h *getHandler) getUserPermissionAndRespond(w http.ResponseWriter, owner, repo, username, authHeader string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s/permission", owner, repo, username)
	resp, err := h.makeGitHubRequest("GET", url, authHeader, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Process the response body through the transformation pipeline
	processedBody, err := h.processPermissionResponse(body, owner, repo, username)
	if err != nil {
		h.Log.Printf("Failed to process response, returning original: %v", err)
		h.writeJSONResponse(w, http.StatusOK, body)
		return nil
	}

	h.writeJSONResponse(w, http.StatusOK, processedBody)
	h.Log.Printf("Successfully retrieved permission for user %s", username)
	return nil
}

func (h *getHandler) processPermissionResponse(body []byte, owner, repo, username string) ([]byte, error) {
	// Flatten the response
	flattenedBody, err := FlattenGitHubUserPermissionBytes(body)
	if err != nil {
		return nil, fmt.Errorf("failed to flatten response: %w", err)
	}

	// Correct the permission field
	correctedBody, err := CorrectGitHubUserPermissionField(flattenedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to correct permission field: %w", err)
	}

	// Read permission from corrected body
	permission, err := ReadFieldFromBody(correctedBody, "permission")
	if err != nil {
		return nil, fmt.Errorf("failed to read permission: %w", err)
	}

	// Add message field
	message := fmt.Sprintf("User is a collaborator of the repository %s/%s with permission %s", owner, repo, permission)
	finalBody, err := AddFieldToResponse(correctedBody, "message", message)
	if err != nil {
		return nil, fmt.Errorf("failed to add message field: %w", err)
	}

	return finalBody, nil
}

// POST handler implementation
// @Summary Add a repository collaborator
// @Description Add a repository collaborator or invite a user to collaborate on a repository
// @ID post-repo-collaborator
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator to add"
// @Param permission body collaborator.Permission true "Permission to grant to the collaborator"
// @Accept json
// @Produce json
// @Success 202  {object} collaborator.Message "Invitation sent to user"
// @Success 204 "User already collaborator"
// @Router /repository/{owner}/{repo}/collaborators/{username} [post]
func (h *postHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Adding collaborator %s to repository %s/%s", username, owner, repo)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Error reading request body: %v", err))
		return
	}
	defer r.Body.Close()

	permission, err := ReadFieldFromBody(body, "permission")
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Error reading permission from request body: %v", err))
		return
	}

	err = h.addCollaborator(w, owner, repo, username, authHeader, body, fmt.Sprintf("%s", permission))
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error adding collaborator: %v", err))
	}
}

func (h *postHandler) addCollaborator(w http.ResponseWriter, owner, repo, username, authHeader string, body []byte, permission string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := h.makeGitHubRequest("PUT", url, authHeader, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusCreated: // Invitation sent
		message := fmt.Sprintf("Invitation sent to user %s for repository %s/%s with permission %s", username, owner, repo, permission)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			return fmt.Errorf("failed to add message field: %w", err)
		}
		h.writeJSONResponse(w, http.StatusAccepted, finalBody)
		h.Log.Printf("Invitation sent to user %s", username)

	case http.StatusNoContent: // User already collaborator
		h.Log.Printf("User %s is already a collaborator", username)
		w.WriteHeader(http.StatusNoContent)

	default:
		h.Log.Printf("GitHub API returned status %d", resp.StatusCode)
		w.WriteHeader(resp.StatusCode)
		if len(respBody) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write(respBody)
		} else {
			w.Write([]byte(fmt.Sprintf("Error: %s", resp.Status)))
		}
	}

	return nil
}

// PATCH handler implementation
// @Summary Update repository collaborator permission or invitation
// @Description Update the permission of an existing collaborator or pending invitation
// @ID patch-repo-collaborator
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator"
// @Param permissions body collaborator.Permission true "New permission to set (`pull`, `push`, `admin`, `maintain`, `triage`)"
// @Accept json
// @Produce json
// @Success 200 {object} collaborator.Message "Permission updated successfully"
// @Success 202 {object} collaborator.Message "Invitation permission updated"
// @Router /repository/{owner}/{repo}/collaborators/{username} [patch]
func (h *patchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Updating permission for user %s in repository %s/%s", username, owner, repo)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Error reading request body: %v", err))
		return
	}
	defer r.Body.Close()

	permission, err := ReadFieldFromBody(body, "permission")
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error reading permission from request body: %v", err))
		return
	}

	status, err := h.checkCollaboratorStatus(owner, repo, username, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error checking collaborator status: %v", err))
		return
	}

	switch status {
	case StatusCollaborator:
		err = h.updateCollaboratorPermission(w, owner, repo, username, authHeader, body, fmt.Sprintf("%s", permission))
	case StatusNotCollaborator:
		err = h.updateInvitationPermission(w, owner, repo, username, authHeader, body, fmt.Sprintf("%s", permission))
	}

	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error updating permission: %v", err))
	}
}

func (h *patchHandler) updateCollaboratorPermission(w http.ResponseWriter, owner, repo, username, authHeader string, body []byte, permission string) error {
	h.Log.Printf("User %s is already a collaborator, updating permission", username)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := h.makeGitHubRequest("PUT", url, authHeader, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		message := fmt.Sprintf("Permission updated successfully for collaborator %s with permission %s", username, permission)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			return fmt.Errorf("failed to add message field: %w", err)
		}
		h.writeJSONResponse(w, http.StatusOK, finalBody)
		h.Log.Printf("Successfully updated permission for collaborator %s", username)
		return nil
	}

	// Forward error from GitHub API
	respBody, _ := io.ReadAll(resp.Body)
	h.Log.Printf("GitHub API returned error %d when updating collaborator", resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	if len(respBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	} else {
		w.Write([]byte(fmt.Sprintf("Error: %s", resp.Status)))
	}
	return nil
}

func (h *patchHandler) updateInvitationPermission(w http.ResponseWriter, owner, repo, username, authHeader string, body []byte, permission string) error {
	h.Log.Printf("User %s is not a collaborator, checking for pending invitations", username)

	invitation, found, err := h.findUserInvitation(owner, repo, username, authHeader)
	if err != nil {
		return fmt.Errorf("error checking invitations: %w", err)
	}

	if !found {
		h.Log.Printf("User %s has no collaborator status or pending invitation", username)
		message := fmt.Sprintf("User %s is not a collaborator and has no pending invitation", username)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("User %s not found as collaborator or invitee", username)))
			return nil
		}
		h.writeJSONResponse(w, http.StatusNotFound, finalBody)
		return nil
	}

	return h.updateInvitation(w, owner, repo, username, invitation.ID, authHeader, body, permission)
}

func (h *patchHandler) updateInvitation(w http.ResponseWriter, owner, repo, username string, invitationID int64, authHeader string, body []byte, permission string) error {
	h.Log.Printf("Found pending invitation for user %s (ID: %d), updating permission", username, invitationID)

	// Correct the request body for invitation API
	correctedBody, err := CorrectGitHubUserPermissionsFieldReqBody(body)
	if err != nil {
		return fmt.Errorf("failed to correct permissions field: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/%d", owner, repo, invitationID)
	resp, err := h.makeGitHubRequest("PATCH", url, authHeader, correctedBody)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		message := fmt.Sprintf("Invitation permission updated successfully for user %s with permission %s", username, permission)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			return fmt.Errorf("failed to add message field: %w", err)
		}
		h.writeJSONResponse(w, http.StatusAccepted, finalBody)
		h.Log.Printf("Successfully updated invitation permission for user %s", username)
		return nil
	}

	// Forward error from GitHub API
	respBody, _ := io.ReadAll(resp.Body)
	h.Log.Printf("GitHub API returned error %d when updating invitation", resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	if len(respBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	} else {
		w.Write([]byte(fmt.Sprintf("Error: %s", resp.Status)))
	}
	return nil
}

// DELETE handler implementation
// @Summary Delete repository collaborator or cancel invitation
// @Description Remove a collaborator from repository or cancel a pending invitation
// @ID delete-repo-collaborator
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Param username path string true "Username of the collaborator to remove"
// @Produce json
// @Success 200 {object} collaborator.Message "Collaborator removed successfully"
// @Success 202 {object} collaborator.Message "Invitation cancelled successfully"
// @Success 404 {object} collaborator.Message "User not found as collaborator or invitee"
// @Router /repository/{owner}/{repo}/collaborators/{username} [delete]
func (h *deleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	username := r.PathValue("username")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Removing user %s from repository %s/%s", username, owner, repo)

	status, err := h.checkCollaboratorStatus(owner, repo, username, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error checking collaborator status: %v", err))
		return
	}

	switch status {
	case StatusCollaborator:
		err = h.removeCollaborator(w, owner, repo, username, authHeader)
	case StatusNotCollaborator:
		err = h.cancelInvitation(w, owner, repo, username, authHeader)
	}

	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error removing user: %v", err))
	}
}

func (h *deleteHandler) removeCollaborator(w http.ResponseWriter, owner, repo, username, authHeader string) error {
	h.Log.Printf("User %s is a collaborator, removing from repository", username)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", owner, repo, username)
	resp, err := h.makeGitHubRequest("DELETE", url, authHeader, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		message := fmt.Sprintf("Collaborator %s removed successfully from repository %s/%s", username, owner, repo)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			return fmt.Errorf("failed to add message field: %w", err)
		}
		h.writeJSONResponse(w, http.StatusOK, finalBody)
		h.Log.Printf("Successfully removed collaborator %s", username)
		return nil
	}

	// Forward error from GitHub API
	respBody, _ := io.ReadAll(resp.Body)
	h.Log.Printf("GitHub API returned error %d when removing collaborator", resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	if len(respBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	} else {
		w.Write([]byte(fmt.Sprintf("Error: %s", resp.Status)))
	}
	return nil
}

func (h *deleteHandler) cancelInvitation(w http.ResponseWriter, owner, repo, username, authHeader string) error {
	h.Log.Printf("User %s is not a collaborator, checking for pending invitations", username)

	invitation, found, err := h.findUserInvitation(owner, repo, username, authHeader)
	if err != nil {
		return fmt.Errorf("error checking invitations: %w", err)
	}

	if !found {
		h.Log.Printf("User %s has no collaborator status or pending invitation", username)
		message := fmt.Sprintf("User %s is not a collaborator and has no pending invitation", username)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("User %s not found as collaborator or invitee", username)))
			return nil
		}
		h.writeJSONResponse(w, http.StatusNotFound, finalBody)
		return nil
	}

	h.Log.Printf("Found pending invitation for user %s (ID: %d), cancelling invitation", username, invitation.ID)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/%d", owner, repo, invitation.ID)
	resp, err := h.makeGitHubRequest("DELETE", url, authHeader, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		message := fmt.Sprintf("Invitation cancelled successfully for user %s", username)
		finalBody, err := AddFieldToResponse([]byte("{}"), "message", message)
		if err != nil {
			return fmt.Errorf("failed to add message field: %w", err)
		}
		h.writeJSONResponse(w, http.StatusAccepted, finalBody)
		h.Log.Printf("Successfully cancelled invitation for user %s", username)
		return nil
	}

	// Forward error from GitHub API
	respBody, _ := io.ReadAll(resp.Body)
	h.Log.Printf("GitHub API returned error %d when cancelling invitation", resp.StatusCode)
	w.WriteHeader(resp.StatusCode)
	if len(respBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBody)
	} else {
		w.Write([]byte(fmt.Sprintf("Error: %s", resp.Status)))
	}
	return nil
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Common helper function for finding user invitations
func findUserInvitationHelper(client httpDoer, logger interface {
	Printf(string, ...interface{})
	Print(...interface{})
}, owner, repo, username, authHeader string) (*GitHubInvitation, bool, error) {
	logger.Printf("Checking invitations for user %s in repository %s/%s", username, owner, repo)
	page := 1
	perPage := 30

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations?per_page=%d&page=%d", owner, repo, perPage, page), nil)
		if err != nil {
			return nil, false, err
		}

		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}

		inviteResp, err := client.Do(req)
		if err != nil {
			return nil, false, err
		}
		defer inviteResp.Body.Close()

		// If we can't get invitations (not 200 OK), return not found
		if inviteResp.StatusCode != http.StatusOK {
			logger.Printf("Failed to get invitations, status: %d", inviteResp.StatusCode)
			return nil, false, nil
		}

		inviteBody, err := io.ReadAll(inviteResp.Body)
		if err != nil {
			return nil, false, err
		}

		// Check if username exists in current page of invitations
		if invitation, found := getUserInvitationFromPage(inviteBody, username); found {
			return invitation, true, nil
		}

		// Check if we have more pages
		// If we got less than perPage results, we've reached the last page
		invitations, err := parseInvitations(inviteBody)
		if err != nil {
			return nil, false, err
		}

		if len(invitations) < perPage {
			// Last page reached, user not found
			break
		}

		page++
	}

	return nil, false, nil
}
