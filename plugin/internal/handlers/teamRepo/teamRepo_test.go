package teamrepo

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/krateoplatformops/github-rest-dynamic-controller-plugin/internal/handlers"
	"github.com/rs/zerolog"
)

// mockHTTPClient implements http.Client's Do method for testing
// this is needed for external API calls in the handler
// it allows us to simulate responses and errors without making real HTTP requests (e.g., for GitHub API calls).
type mockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
	requests  []*http.Request
}

// newMockHTTPClient creates a new instance of mockHTTPClient
// with empty maps for responses and errors
// and an empty slice for requests.
func newMockHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
		requests:  make([]*http.Request, 0),
	}
}

// Do implements the http.Client Do method for mockHTTPClient.
// It simulates sending an HTTP request and returns a response or an error
func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Store the request for verification
	m.requests = append(m.requests, req)

	key := req.URL.String()

	// Check if there's an error configured for this URL
	if err, exists := m.errors[key]; exists {
		return nil, err
	}

	// Return configured response or default 404
	if resp, exists := m.responses[key]; exists {
		return resp, nil
	}

	// Default response
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{"message": "Not Found"}`)),
		Header:     make(http.Header),
	}, nil
}

// setResponse allows setting a predefined response for a specific URL
func (m *mockHTTPClient) setResponse(url string, statusCode int, body string) {
	m.responses[url] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func (m *mockHTTPClient) setError(url string, err error) {
	m.errors[url] = err
}

func (m *mockHTTPClient) getLastRequest() *http.Request {
	if len(m.requests) == 0 {
		return nil
	}
	return m.requests[len(m.requests)-1]
}

func (m *mockHTTPClient) getRequestCount() int {
	return len(m.requests)
}

func (m *mockHTTPClient) reset() {
	m.responses = make(map[string]*http.Response)
	m.errors = make(map[string]error)
	m.requests = make([]*http.Request, 0)
}

// createTestHandler creates a handler instance for testing with a mock client
func createTestHandler(mockClient *mockHTTPClient) *handler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	h := &handler{
		HandlerOptions: handlers.HandlerOptions{
			Client: mockClient, // Use the mock client directly
			Log:    &logger,
		},
	}
	return h
}

// createTestHandlerWithSilentLog creates a handler with discarded logs
func createTestHandlerWithSilentLog(mockClient *mockHTTPClient) *handler {
	return createTestHandler(mockClient)
}

// Test data constants
const (
	testOrg      = "testorg"
	testTeamSlug = "test-team"
	testOwner    = "testowner"
	testRepo     = "testrepo"
	testToken    = "token test-token-123"
)

var (
	teamRepoExternalURL = fmt.Sprintf("https://api.github.com/orgs/%s/teams/%s/repos/%s/%s", testOrg, testTeamSlug, testOwner, testRepo)
	validAdminResp      = `{
		"id": 12345,
		"name": "testrepo",
		"full_name": "testowner/testrepo",
		"owner": {
			"login": "originalowner",
			"id": 67890
		},
		"permissions": {
			"admin": true,
			"maintain": true,
			"push": true,
			"triage": true,
			"pull": true
		},
		"role_name": "admin"
	}`
	validReadResp = `{
		"id": 12345,
		"name": "testrepo",
		"full_name": "testowner/testrepo",
		"owner": {
			"login": "originalowner",
			"id": 67890
		},
		"permissions": {
			"admin": false,
			"maintain": false,
			"push": false,
			"triage": true,
			"pull": true
		},
		"role_name": "read"
	}`
	validWriteResp = `{
		"id": 12345,
		"name": "testrepo",
		"full_name": "testowner/testrepo",
		"owner": {
			"login": "originalowner",
			"id": 67890
		},
		"permissions": {
			"admin": false,
			"maintain": false,
			"push": true,
			"triage": true,
			"pull": true
		},
		"role_name": "write"
	}`
	validMaintainResp = `{
		"id": 12345,
		"name": "testrepo",
		"full_name": "testowner/testrepo",
		"owner": {
			"login": "originalowner",
			"id": 67890
		},
		"permissions": {
			"admin": false,
			"maintain": true,
			"push": true,
			"triage": true,
			"pull": true
		},
		"role_name": "maintain"
	}`
	validTriageResp = `{
		"id": 12345,
		"name": "testrepo",
		"full_name": "testowner/testrepo",
		"owner": {
			"login": "originalowner",
			"id": 67890
		},
		"permissions": {
			"admin": false,
			"maintain": false,
			"push": false,
			"triage": true,
			"pull": true
		},
		"role_name": "triage"
	}`
)

// GetTeamRepo returns a new handler for team repository permissions
func TestGetTeamRepo(t *testing.T) {
	t.Run("returns valid handler", func(t *testing.T) {
		client := &http.Client{}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		opts := handlers.HandlerOptions{
			Client: client, // Use the real http.Client for this test
			Log:    &logger,
		}

		handlerInterface := GetTeamRepo(opts)

		if handlerInterface == nil {
			t.Fatal("GetTeamRepo should return a non-nil handler")
		}

		// Verify it implements the Handler interface
		var _ handlers.Handler = handlerInterface

		// Verify the handler has the correct type and options
		h, ok := handlerInterface.(*handler)
		if !ok {
			t.Fatal("GetTeamRepo should return a *handler")
		}

		if h.Client != client {
			t.Error("Handler should have the provided client")
		}

		if h.Log != &logger {
			t.Error("Handler should have the provided logger")
		}
	})
}

func TestHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		org                  string
		teamSlug             string
		owner                string
		repo                 string
		authHeader           string
		setupMock            func(*mockHTTPClient)
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
		expectedRequestCount int
		verifyRequests       func(t *testing.T, mockClient *mockHTTPClient) // optional function to verify external requests
	}{
		{
			name:       "successful team permission check with admin role",
			org:        testOrg,
			teamSlug:   testTeamSlug,
			owner:      testOwner,
			repo:       testRepo,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(teamRepoExternalURL, http.StatusOK, validAdminResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"admin"`,
			expectedRequestCount: 1,
			verifyRequests: func(t *testing.T, mockClient *mockHTTPClient) {
				if mockClient.getRequestCount() != 1 { // Verify that exactly one request to GitHub API was made
					t.Errorf("Expected 1 request, got %d", mockClient.getRequestCount())
				}

				req := mockClient.getLastRequest()
				if req.URL.String() != teamRepoExternalURL {
					t.Errorf("Request URL = %s, want %s", req.URL.String(), teamRepoExternalURL)
				}
				if req.Header.Get("Authorization") != testToken {
					t.Errorf("Request Authorization header = %s, want %s", req.Header.Get("Authorization"), testToken)
				}
				if req.Header.Get("Accept") != "application/vnd.github.v3.repository+json" { // Check if Accept header is set correctly by the handler
					t.Errorf("Request Accept header = %s, want %s", req.Header.Get("Accept"), "application/vnd.github.v3.repository+json")
				}
			},
		},
		{
			name:       "successful team permission check with read role (corrected to pull)",
			org:        testOrg,
			teamSlug:   testTeamSlug,
			owner:      testOwner,
			repo:       testRepo,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(teamRepoExternalURL, http.StatusOK, validReadResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"pull"`, // we expect the permission to be corrected to `pull`
			expectedRequestCount: 1,
		},
		{
			name:       "successful team permission check with write role (corrected to push)",
			org:        testOrg,
			teamSlug:   testTeamSlug,
			owner:      testOwner,
			repo:       testRepo,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(teamRepoExternalURL, http.StatusOK, validWriteResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"push"`, // we expect the permission to be corrected to `push`
			expectedRequestCount: 1,
		},
		{
			name:       "successful team permission check with maintain role",
			org:        testOrg,
			teamSlug:   testTeamSlug,
			owner:      testOwner,
			repo:       testRepo,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(teamRepoExternalURL, http.StatusOK, validMaintainResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"maintain"`,
			expectedRequestCount: 1,
		},
		{
			name:       "successful team permission check with triage role",
			org:        testOrg,
			teamSlug:   testTeamSlug,
			owner:      testOwner,
			repo:       testRepo,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(teamRepoExternalURL, http.StatusOK, validTriageResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"triage"`,
			expectedRequestCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			handler := createTestHandlerWithSilentLog(mockClient)

			// Create a proper ServeMux with the pattern matching
			mux := http.NewServeMux()
			mux.Handle("GET /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", handler)

			// Create request with the proper path
			path := fmt.Sprintf("/teamrepository/orgs/%s/teams/%s/repos/%s/%s", tt.org, tt.teamSlug, tt.owner, tt.repo)
			req := httptest.NewRequest("GET", path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute through the mux so path values are properly set
			mux.ServeHTTP(rr, req)

			// Verify status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, tt.expectedStatus)
			}

			// Verify content type if expected
			if tt.expectedContentType != "" {
				contentType := rr.Header().Get("Content-Type")
				if contentType != tt.expectedContentType {
					t.Errorf("handler returned wrong content type: got %v want %v", contentType, tt.expectedContentType)
				}
			}

			// Verify response body contains expected content
			if tt.expectedBodyContains != "" {
				body := rr.Body.String()
				if !strings.Contains(body, tt.expectedBodyContains) {
					t.Errorf("handler response body does not contain expected content.\nGot: %s\nWant to contain: %s", body, tt.expectedBodyContains)
				}
			}

			// Verify request count
			if mockClient.getRequestCount() != tt.expectedRequestCount {
				t.Errorf("expected %d requests, got %d", tt.expectedRequestCount, mockClient.getRequestCount())
			}

			// Run custom request verification if provided
			if tt.verifyRequests != nil {
				tt.verifyRequests(t, mockClient)
			}
		})
	}
}
