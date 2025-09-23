package teamrepo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
)

func GetTeamRepo(opts handlers.HandlerOptions) handlers.Handler {
	return &handler{
		HandlerOptions: opts,
	}
}

var _ handlers.Handler = &handler{}

type handler struct {
	handlers.HandlerOptions
}

// @Summary Get the permission of a team in a repository
// @Description Get the permission of a team in a repository
// @ID get-team-repo-permission
// @Param org path string true "Organization of the repository"
// @Param team_slug path string true "Slug of the team"
// @Param owner path string true "Owner of the repository"
// @Param repo path string true "Name of the repository"
// @Produce json
// @Success 200 {object} teamrepo.TeamRepoPermissions
// @Router /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo} [get]
// GET /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("org")
	teamSlug := r.PathValue("team_slug")
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	h.Log.Println("Calling GitHub TeamRepository API")

	auth_header := r.Header.Get("Authorization")

	// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#check-team-permissions-for-a-repository
	// /orgs/krateoplatformops/teams/krateo-team/repos/krateoplatformops/azuredevops-oas3
	req, err := http.NewRequest("GET", "https://api.github.com/orgs/"+org+"/teams/"+teamSlug+"/repos/"+owner+"/"+repo, nil)
	if err != nil {
		h.Log.Println(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
	}
	req.Header.Add("Accept", "application/vnd.github.v3.repository+json")

	if len(auth_header) > 0 {
		req.Header.Set("Authorization", auth_header)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		h.Log.Print(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
	}

	// read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.Log.Print(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
	}

	head := resp.Header.Clone()

	for key, values := range head {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, string(body), resp.StatusCode)
		return
	}

	var repoPermissions Repository
	err = json.Unmarshal(body, &repoPermissions)
	if err != nil {
		h.Log.Print(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
	}

	var expected map[string]any

	err = json.Unmarshal(body, &expected)
	if err != nil {
		h.Log.Print(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
	}

	delete(expected, "permissions")
	delete(expected, "owner")

	expected["owner"] = owner
	if strings.EqualFold(repoPermissions.RoleName, "read") {
		expected["permission"] = "pull"
	} else if strings.EqualFold(repoPermissions.RoleName, "write") {
		expected["permission"] = "push"
	} else {
		expected["permission"] = repoPermissions.RoleName
	}

	b, err := json.Marshal(expected)
	if err != nil {
		h.Log.Print(err)
		w.Write([]byte(fmt.Sprint("Error: ", err)))
	}
	w.Write(b)

	w.Header().Set("Content-Type", "application/json")

	return
}
