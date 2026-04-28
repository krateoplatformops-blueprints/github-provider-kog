package branchprotection

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/krateoplatformops/github-provider-kog/pkg/handlers"
)

// Handler constructors.
func GetBranchProtection(opts handlers.HandlerOptions) handlers.Handler {
	return &getHandler{baseHandler: newBaseHandler(opts)}
}

func PutBranchProtection(opts handlers.HandlerOptions) handlers.Handler {
	return &putHandler{baseHandler: newBaseHandler(opts)}
}

func DeleteBranchProtection(opts handlers.HandlerOptions) handlers.Handler {
	return &deleteHandler{baseHandler: newBaseHandler(opts)}
}

// Interface compliance verification.
var _ handlers.Handler = &getHandler{}
var _ handlers.Handler = &putHandler{}
var _ handlers.Handler = &deleteHandler{}

// baseHandler holds shared dependencies.
type baseHandler struct {
	handlers.HandlerOptions
}

func newBaseHandler(opts handlers.HandlerOptions) *baseHandler {
	return &baseHandler{HandlerOptions: opts}
}

type getHandler struct{ *baseHandler }
type putHandler struct{ *baseHandler }
type deleteHandler struct{ *baseHandler }

// makeGitHubRequest builds and executes an HTTP request against the GitHub API,
// forwarding the Authorization header from the incoming request.
func (h *baseHandler) makeGitHubRequest(method, url, authHeader string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	}

	return h.Client.Do(req)
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

// GET handler — fetches branch protection from GitHub and normalizes the response
// so boolean fields match the request body structure expected by the controller.
//
// @Summary      Get branch protection
// @Description  Retrieves branch protection settings and normalizes the response for drift-free reconciliation.
// @ID           get-branch-protection
// @Param        owner   path   string  true  "Repository owner (user or organization)"
// @Param        repo    path   string  true  "Repository name"
// @Param        branch  path   string  true  "Branch name"
// @Produce      json
// @Success      200  {object}  branchprotection.BranchProtectionResponse
// @Failure      404  {object}  branchprotection.ErrorResponse
// @Router       /api/{owner}/{repo}/branches/{branch}/protection [get]
func (h *getHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	branch := r.PathValue("branch")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Getting branch protection for %s/%s branch %s", owner, repo, branch)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s/protection", owner, repo, branch)
	resp, err := h.makeGitHubRequest("GET", url, authHeader, nil)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to call GitHub API: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.writeJSONResponse(w, resp.StatusCode, body)
		return
	}

	normalized, err := normalizeGetResponse(body)
	if err != nil {
		h.Log.Printf("response normalization failed, returning raw response: %v", err)
		h.writeJSONResponse(w, http.StatusOK, body)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, normalized)
	h.Log.Printf("Successfully retrieved branch protection for %s/%s branch %s", owner, repo, branch)
}

// PUT handler — creates or updates branch protection. GitHub uses PUT for both
// operations, making this endpoint idempotent. Missing GitHub-required fields
// (required_status_checks, enforce_admins, required_pull_request_reviews,
// restrictions) are filled in with null/false defaults before forwarding.
//
// @Summary      Create or update branch protection
// @Description  Creates or updates branch protection rules. Used for both create and update actions.
// @ID           put-branch-protection
// @Param        owner   path   string                                               true  "Repository owner (user or organization)"
// @Param        repo    path   string                                               true  "Repository name"
// @Param        branch  path   string                                               true  "Branch name"
// @Param        body    body   branchprotection.BranchProtectionRequest             true  "Branch protection settings"
// @Accept       json
// @Produce      json
// @Success      200  {object}  branchprotection.BranchProtectionResponse
// @Failure      422  {object}  branchprotection.ErrorResponse
// @Router       /api/{owner}/{repo}/branches/{branch}/protection [put]
func (h *putHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	branch := r.PathValue("branch")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Updating branch protection for %s/%s branch %s", owner, repo, branch)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err))
		return
	}
	defer r.Body.Close()

	enriched, err := ensureGitHubRequiredFields(body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to process request body: %v", err))
		return
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s/protection", owner, repo, branch)
	resp, err := h.makeGitHubRequest("PUT", url, authHeader, enriched)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to call GitHub API: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to read response: %v", err))
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.writeJSONResponse(w, resp.StatusCode, respBody)
		return
	}

	normalized, err := normalizeGetResponse(respBody)
	if err != nil {
		h.Log.Printf("response normalization failed, returning raw response: %v", err)
		h.writeJSONResponse(w, http.StatusOK, respBody)
		return
	}

	h.writeJSONResponse(w, http.StatusOK, normalized)
	h.Log.Printf("Successfully updated branch protection for %s/%s branch %s", owner, repo, branch)
}

// DELETE handler — removes branch protection. GitHub returns 204 on success and
// 404 when the branch is not protected; both are treated as successful deletions
// to ensure idempotent behavior.
//
// @Summary      Delete branch protection
// @Description  Removes branch protection rules. Idempotent: returns 204 even if branch was not protected.
// @ID           delete-branch-protection
// @Param        owner   path   string  true  "Repository owner (user or organization)"
// @Param        repo    path   string  true  "Repository name"
// @Param        branch  path   string  true  "Branch name"
// @Produce json
// @Success      204
// @Failure      403  {object}  branchprotection.ErrorResponse
// @Router       /api/{owner}/{repo}/branches/{branch}/protection [delete]
func (h *deleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	branch := r.PathValue("branch")
	authHeader := r.Header.Get("Authorization")

	h.Log.Printf("Deleting branch protection for %s/%s branch %s", owner, repo, branch)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s/protection", owner, repo, branch)
	resp, err := h.makeGitHubRequest("DELETE", url, authHeader, nil)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("failed to call GitHub API: %v", err))
		return
	}
	defer resp.Body.Close()

	// 204 = deleted successfully; 404 = already not protected (idempotent success).
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		w.WriteHeader(http.StatusNoContent)
		h.Log.Printf("Successfully deleted branch protection for %s/%s branch %s", owner, repo, branch)
		return
	}

	body, _ := io.ReadAll(resp.Body)
	h.writeJSONResponse(w, resp.StatusCode, body)
}
