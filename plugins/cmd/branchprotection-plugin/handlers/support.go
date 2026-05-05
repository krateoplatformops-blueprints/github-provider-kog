package branchprotection

import (
	"encoding/json"
	"fmt"
)

// ensureGitHubRequiredFields adds the four fields that the GitHub PUT endpoint
// requires (required_status_checks, enforce_admins, required_pull_request_reviews,
// restrictions) with null/false defaults when they are absent from the incoming body.
func ensureGitHubRequiredFields(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	if _, ok := data["required_status_checks"]; !ok {
		data["required_status_checks"] = nil
	}
	if _, ok := data["enforce_admins"]; !ok {
		data["enforce_admins"] = false
	}
	if _, ok := data["required_pull_request_reviews"]; !ok {
		data["required_pull_request_reviews"] = nil
	}
	if _, ok := data["restrictions"]; !ok {
		data["restrictions"] = nil
	}

	// Make sure that if `required_status_checks` is present then both `strict` and `checks` fields are present,
	// otherwise return errors
	if rsc, ok := data["required_status_checks"].(map[string]interface{}); ok {
		if _, ok := rsc["strict"]; !ok {
			return nil, fmt.Errorf("required_status_checks is present but missing required field 'strict'")
		}
		if _, ok := rsc["checks"]; !ok {
			return nil, fmt.Errorf("required_status_checks is present but missing required field 'checks'")
		}
	}

	// Make sure that if `restrictions` is present then both `teams` and `users` fields are present,
	// otherwise return errors
	if res, ok := data["restrictions"].(map[string]interface{}); ok {
		if _, ok := res["teams"]; !ok {
			return nil, fmt.Errorf("restrictions is present but missing required field 'teams'")
		}
		if _, ok := res["users"]; !ok {
			return nil, fmt.Errorf("restrictions is present but missing required field 'users'")
		}
	}

	return json.Marshal(data)
}

// normalizeGetResponse transforms the raw GitHub branch protection GET/PUT response
// into the structure that matches the plugin request body schema, so that the
// rest-dynamic-controller can perform correct drift detection.
//
// Transformations applied:
//   - boolean-in-object fields (e.g. {"enabled": true}) are unwrapped to plain booleans
//   - server-only fields (required_signatures, url sub-fields) are stripped
//   - user/team/app arrays with full objects are flattened to login/slug strings
func normalizeGetResponse(body []byte) ([]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GitHub response: %w", err)
	}

	// Unwrap all boolean-in-object fields to plain booleans.
	for _, field := range []string{
		"enforce_admins",
		"required_linear_history",
		"allow_force_pushes",
		"allow_deletions",
		"block_creations",
		"required_conversation_resolution",
		"lock_branch",
		"allow_fork_syncing",
	} {
		unwrapEnabled(data, field)
	}

	// required_signatures is a server-managed field not present in request body.
	delete(data, "required_signatures")

	// Normalize required_pull_request_reviews sub-object.
	if prr, ok := data["required_pull_request_reviews"].(map[string]interface{}); ok {
		delete(prr, "url")
		normalizeUserTeamAppObj(prr, "dismissal_restrictions")
		normalizeUserTeamAppObj(prr, "bypass_pull_request_allowances")
	}

	// Normalize required_status_checks: strip server-only URL fields and fix
	// app_id representation. GitHub stores app_id=-1 ("any app") as null in
	// its GET response; convert null back to -1 to match the desired CR state.
	// Otherswise, the presence of app_id in the response (even as null) would cause
	// perpetual drift when the user sets app_id=-1 in their CR.
	if rsc, ok := data["required_status_checks"].(map[string]interface{}); ok {
		delete(rsc, "url")
		delete(rsc, "contexts_url")
		if checks, ok := rsc["checks"].([]interface{}); ok {
			for _, item := range checks {
				if c, ok := item.(map[string]interface{}); ok && c["app_id"] == nil {
					c["app_id"] = float64(-1)
				}
			}
		}

		// Delete `contexts` field if present, since we use `checks` instead and contexts is `deprecated` by GitHub.
		delete(rsc, "contexts")
	}

	// Normalize restrictions: full user/team/app objects -> login/slug strings.
	normalizeUserTeamAppObj(data, "restrictions")

	return json.Marshal(data)
}

// unwrapEnabled converts {"enabled": bool, ...} to just bool in data[field].
// If the field is not an object with an "enabled" key, it is left unchanged.
func unwrapEnabled(data map[string]interface{}, field string) {
	obj, ok := data[field].(map[string]interface{})
	if !ok {
		return
	}
	enabled, ok := obj["enabled"].(bool)
	if !ok {
		return
	}
	data[field] = enabled
}

// normalizeUserTeamAppObj flattens the users/teams/apps arrays inside data[field]
// from full GitHub API objects to plain login/slug strings, and strips server-only
// URL sub-fields.
func normalizeUserTeamAppObj(data map[string]interface{}, field string) {
	obj, ok := data[field].(map[string]interface{})
	if !ok {
		return
	}

	// Strip server-only URL fields.
	delete(obj, "url")
	delete(obj, "users_url")
	delete(obj, "teams_url")
	delete(obj, "apps_url")

	// users: [{login: "x", ...}] -> ["x"]
	if arr, ok := obj["users"].([]interface{}); ok {
		obj["users"] = extractField(arr, "login")
	}
	// teams: [{slug: "x", ...}] -> ["x"]
	if arr, ok := obj["teams"].([]interface{}); ok {
		obj["teams"] = extractField(arr, "slug")
	}
	// apps: [{slug: "x", ...}] -> ["x"]
	if arr, ok := obj["apps"].([]interface{}); ok {
		obj["apps"] = extractField(arr, "slug")
	}
}

// extractField extracts the named string field from each element of an interface slice.
// Elements that are already strings are passed through as-is.
func extractField(arr []interface{}, field string) []string {
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		switch v := item.(type) {
		case string:
			result = append(result, v)
		case map[string]interface{}:
			if s, ok := v[field].(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}
