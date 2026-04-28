package branchprotection

import (
	"encoding/json"
	"testing"
)

// ---- normalizeGetResponse tests ----

func TestNormalizeGetResponse_UnwrapsBooleanFields(t *testing.T) {
	input := `{
		"url": "https://api.github.com/repos/owner/repo/branches/main/protection",
		"enforce_admins":                  { "url": "https://...", "enabled": true },
		"required_linear_history":          { "enabled": false },
		"allow_force_pushes":               { "enabled": false },
		"allow_deletions":                  { "enabled": false },
		"block_creations":                  { "enabled": false },
		"required_conversation_resolution": { "enabled": false },
		"lock_branch":                      { "enabled": false },
		"allow_fork_syncing":               { "enabled": false },
		"required_signatures":              { "url": "https://...", "enabled": false }
	}`

	result, err := normalizeGetResponse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	boolFields := map[string]bool{
		"enforce_admins":                  true,
		"required_linear_history":          false,
		"allow_force_pushes":               false,
		"allow_deletions":                  false,
		"block_creations":                  false,
		"required_conversation_resolution": false,
		"lock_branch":                      false,
		"allow_fork_syncing":               false,
	}
	for field, want := range boolFields {
		got, ok := data[field].(bool)
		if !ok {
			t.Errorf("field %q: expected bool, got %T (%v)", field, data[field], data[field])
			continue
		}
		if got != want {
			t.Errorf("field %q: expected %v, got %v", field, want, got)
		}
	}

	// required_signatures must be stripped
	if _, exists := data["required_signatures"]; exists {
		t.Error("required_signatures should have been removed")
	}

	// root url must be preserved (additionalStatusField)
	if _, exists := data["url"]; !exists {
		t.Error("root url should be preserved")
	}
}

func TestNormalizeGetResponse_NormalizesPullRequestReviews(t *testing.T) {
	input := `{
		"url": "https://...",
		"required_pull_request_reviews": {
			"url": "https://api.github.com/.../required_pull_request_reviews",
			"dismiss_stale_reviews": true,
			"require_code_owner_reviews": false,
			"require_last_push_approval": false,
			"required_approving_review_count": 2,
			"dismissal_restrictions": {
				"url": "https://...",
				"users_url": "https://...",
				"teams_url": "https://...",
				"users": [{"login": "alice", "id": 1}, {"login": "bob", "id": 2}],
				"teams": [{"slug": "justice-league", "id": 3}],
				"apps":  []
			},
			"bypass_pull_request_allowances": {
				"users": [{"login": "charlie", "id": 4}],
				"teams": [],
				"apps":  []
			}
		}
	}`

	result, err := normalizeGetResponse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	prr, ok := data["required_pull_request_reviews"].(map[string]interface{})
	if !ok {
		t.Fatal("expected required_pull_request_reviews to be an object")
	}

	if _, exists := prr["url"]; exists {
		t.Error("url should be removed from required_pull_request_reviews")
	}

	dr, ok := prr["dismissal_restrictions"].(map[string]interface{})
	if !ok {
		t.Fatal("expected dismissal_restrictions to be an object")
	}
	for _, urlField := range []string{"url", "users_url", "teams_url"} {
		if _, exists := dr[urlField]; exists {
			t.Errorf("%q should be removed from dismissal_restrictions", urlField)
		}
	}

	users := toStringSlice(t, dr["users"])
	if len(users) != 2 || users[0] != "alice" || users[1] != "bob" {
		t.Errorf("unexpected users: %v", users)
	}

	teams := toStringSlice(t, dr["teams"])
	if len(teams) != 1 || teams[0] != "justice-league" {
		t.Errorf("unexpected teams: %v", teams)
	}

	bpa, ok := prr["bypass_pull_request_allowances"].(map[string]interface{})
	if !ok {
		t.Fatal("expected bypass_pull_request_allowances to be an object")
	}
	bpaUsers := toStringSlice(t, bpa["users"])
	if len(bpaUsers) != 1 || bpaUsers[0] != "charlie" {
		t.Errorf("unexpected bypass users: %v", bpaUsers)
	}
}

func TestNormalizeGetResponse_NormalizesRestrictions(t *testing.T) {
	input := `{
		"url": "https://...",
		"restrictions": {
			"url": "https://...",
			"users_url": "https://...",
			"teams_url": "https://...",
			"users": [{"login": "alice", "id": 1}],
			"teams": [{"slug": "ops-team", "id": 2}],
			"apps":  [{"slug": "my-app",  "id": 3}]
		}
	}`

	result, err := normalizeGetResponse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	r, ok := data["restrictions"].(map[string]interface{})
	if !ok {
		t.Fatal("expected restrictions to be an object")
	}

	if toStringSlice(t, r["users"])[0] != "alice" {
		t.Error("expected user login alice")
	}
	if toStringSlice(t, r["teams"])[0] != "ops-team" {
		t.Error("expected team slug ops-team")
	}
	if toStringSlice(t, r["apps"])[0] != "my-app" {
		t.Error("expected app slug my-app")
	}
}

func TestNormalizeGetResponse_StripsStatusChecksURLFields(t *testing.T) {
	input := `{
		"url": "https://...",
		"required_status_checks": {
			"url": "https://...",
			"contexts_url": "https://...",
			"strict": true,
			"contexts": ["ci/test"],
			"checks": [{"context": "ci/test", "app_id": null}]
		}
	}`

	result, err := normalizeGetResponse([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	rsc, ok := data["required_status_checks"].(map[string]interface{})
	if !ok {
		t.Fatal("expected required_status_checks to be present")
	}
	if _, exists := rsc["url"]; exists {
		t.Error("url should be removed from required_status_checks")
	}
	if _, exists := rsc["contexts_url"]; exists {
		t.Error("contexts_url should be removed from required_status_checks")
	}
	if strict, _ := rsc["strict"].(bool); !strict {
		t.Error("strict should be preserved as true")
	}
}

// ---- ensureGitHubRequiredFields tests ----

func TestEnsureGitHubRequiredFields_AddsMissingDefaults(t *testing.T) {
	input := `{
		"required_pull_request_reviews": {
			"required_approving_review_count": 2,
			"dismiss_stale_reviews": true
		}
	}`

	result, err := ensureGitHubRequiredFields([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := data["required_status_checks"]; !ok {
		t.Error("expected required_status_checks to be added")
	}
	if v, ok := data["enforce_admins"].(bool); !ok || v {
		t.Errorf("expected enforce_admins=false, got %v", data["enforce_admins"])
	}
	if _, ok := data["restrictions"]; !ok {
		t.Error("expected restrictions to be added")
	}
}

func TestEnsureGitHubRequiredFields_PreservesExistingValues(t *testing.T) {
	input := `{
		"enforce_admins": true,
		"required_status_checks": null,
		"required_pull_request_reviews": null,
		"restrictions": null,
		"required_linear_history": true
	}`

	result, err := ensureGitHubRequiredFields([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if v, ok := data["enforce_admins"].(bool); !ok || !v {
		t.Errorf("expected enforce_admins=true, got %v", data["enforce_admins"])
	}
	if v, ok := data["required_linear_history"].(bool); !ok || !v {
		t.Errorf("expected required_linear_history=true, got %v", data["required_linear_history"])
	}
}

// ---- extractField tests ----

func TestExtractField_HandlesStringPassthrough(t *testing.T) {
	arr := []interface{}{"alice", "bob"}
	got := extractField(arr, "login")
	if len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestExtractField_ExtractsFromObjects(t *testing.T) {
	arr := []interface{}{
		map[string]interface{}{"login": "alice", "id": float64(1)},
		map[string]interface{}{"login": "bob", "id": float64(2)},
	}
	got := extractField(arr, "login")
	if len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestExtractField_EmptyArray(t *testing.T) {
	got := extractField([]interface{}{}, "login")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

// ---- helpers ----

func toStringSlice(t *testing.T, v interface{}) []string {
	t.Helper()
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T (%v)", v, v)
	}
	result := make([]string, len(arr))
	for i, item := range arr {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("item %d: expected string, got %T (%v)", i, item, item)
		}
		result[i] = s
	}
	return result
}
