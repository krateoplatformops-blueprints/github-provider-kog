package collaborator

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

// createTestGetHandler creates a GET handler instance for testing with a mock client
func createTestGetHandler(mockClient *mockHTTPClient) *getHandler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	opts := handlers.HandlerOptions{
		Client: mockClient,
		Log:    &logger,
	}
	return GetCollaborator(opts).(*getHandler)
}

// createTestPostHandler creates a POST handler instance for testing with a mock client
func createTestPostHandler(mockClient *mockHTTPClient) *postHandler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	opts := handlers.HandlerOptions{
		Client: mockClient,
		Log:    &logger,
	}
	return PostCollaborator(opts).(*postHandler)
}

// createTestPatchHandler creates a PATCH handler instance for testing with a mock client
func createTestPatchHandler(mockClient *mockHTTPClient) *patchHandler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	opts := handlers.HandlerOptions{
		Client: mockClient,
		Log:    &logger,
	}
	return PatchCollaborator(opts).(*patchHandler)
}

// createTestDeleteHandler creates a DELETE handler instance for testing with a mock client
func createTestDeleteHandler(mockClient *mockHTTPClient) *deleteHandler {
	logger := zerolog.New(io.Discard).With().Timestamp().Logger()
	opts := handlers.HandlerOptions{
		Client: mockClient,
		Log:    &logger,
	}
	return DeleteCollaborator(opts).(*deleteHandler)
}

// Test data constants
const (
	testOwner    = "testowner"
	testRepo     = "testrepo"
	testUsername = "testuser"
	testToken    = "token test-token-123"
)

var (
	collaboratorExternalURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s", testOwner, testRepo, testUsername)
	permissionExternalURL   = fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s/permission", testOwner, testRepo, testUsername)
	invitationsExternalURL  = fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations", testOwner, testRepo)
	validPermissionResp     = `{
		"permission": "admin",
		"user": {
			"login": "testuser",
			"id": 12345,
			"html_url": "https://github.com/testuser",
			"permissions": {
				"admin": true,
				"maintain": true,
				"push": true,
				"triage": true,
				"pull": true
			}
		},
		"role_name": "admin"
	}`
	validInvitationResp = `[{
		"id": 1,
		"node_id": "MDEwOlJlcG9zaXRvcnkxMjk2MjY5",
		"repository": {
			"id": 1296269,
			"name": "Hello-World",
			"full_name": "octocat/Hello-World"
		},
		"invitee": {
			"login": "testuser",
			"id": 12345,
			"node_id": "MDQ6VXNlcjU4MzIzMQ==",
			"avatar_url": "https://github.com/images/error/testuser_happy.gif",
			"html_url": "https://github.com/testuser",
			"type": "User"
		},
		"inviter": {
			"login": "testowner",
			"id": 67890,
			"node_id": "MDQ6VXNlcjU4MzIzMQ==",
			"avatar_url": "https://github.com/images/error/testowner_happy.gif",
			"html_url": "https://github.com/testowner",
			"type": "User"
		},
		"permissions": "read",
		"created_at": "2016-06-13T14:52:50-05:00",
		"url": "https://api.github.com/user/repository_invitations/1",
		"html_url": "https://github.com/testowner/testrepo/invitations",
		"expired": false
	}]`
	emptyInvitationResp = `[]`
)

// Test handler constructors
func TestGetCollaborator(t *testing.T) {
	t.Run("returns valid handler", func(t *testing.T) {
		client := &http.Client{}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		opts := handlers.HandlerOptions{
			Client: client,
			Log:    &logger,
		}

		handlerInterface := GetCollaborator(opts)

		if handlerInterface == nil {
			t.Fatal("GetCollaborator should return a non-nil handler")
		}

		// Verify it implements the Handler interface
		var _ handlers.Handler = handlerInterface

		// Verify the handler has the correct type and options
		h, ok := handlerInterface.(*getHandler)
		if !ok {
			t.Fatal("GetCollaborator should return a *getHandler")
		}

		if h.Client != client {
			t.Error("Handler should have the provided client")
		}

		if h.Log != &logger {
			t.Error("Handler should have the provided logger")
		}
	})
}

func TestPostCollaborator(t *testing.T) {
	t.Run("returns valid handler", func(t *testing.T) {
		client := &http.Client{}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		opts := handlers.HandlerOptions{
			Client: client,
			Log:    &logger,
		}

		handlerInterface := PostCollaborator(opts)

		if handlerInterface == nil {
			t.Fatal("PostCollaborator should return a non-nil handler")
		}

		// Verify it implements the Handler interface
		var _ handlers.Handler = handlerInterface

		// Verify the handler has the correct type
		_, ok := handlerInterface.(*postHandler)
		if !ok {
			t.Fatal("PostCollaborator should return a *postHandler")
		}
	})
}

func TestPatchCollaborator(t *testing.T) {
	t.Run("returns valid handler", func(t *testing.T) {
		client := &http.Client{}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		opts := handlers.HandlerOptions{
			Client: client,
			Log:    &logger,
		}

		handlerInterface := PatchCollaborator(opts)

		if handlerInterface == nil {
			t.Fatal("PatchCollaborator should return a non-nil handler")
		}

		// Verify it implements the Handler interface
		var _ handlers.Handler = handlerInterface

		// Verify the handler has the correct type
		_, ok := handlerInterface.(*patchHandler)
		if !ok {
			t.Fatal("PatchCollaborator should return a *patchHandler")
		}
	})
}

func TestDeleteCollaborator(t *testing.T) {
	t.Run("returns valid handler", func(t *testing.T) {
		client := &http.Client{}
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		opts := handlers.HandlerOptions{
			Client: client,
			Log:    &logger,
		}

		handlerInterface := DeleteCollaborator(opts)

		if handlerInterface == nil {
			t.Fatal("DeleteCollaborator should return a non-nil handler")
		}

		// Verify it implements the Handler interface
		var _ handlers.Handler = handlerInterface

		// Verify the handler has the correct type
		_, ok := handlerInterface.(*deleteHandler)
		if !ok {
			t.Fatal("DeleteCollaborator should return a *deleteHandler")
		}
	})
}

// GET Handler Tests
func TestGetHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		owner                string
		repo                 string
		username             string
		authHeader           string
		setupMock            func(*mockHTTPClient)
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
		expectedRequestCount int
		verifyRequests       func(t *testing.T, mockClient *mockHTTPClient)
	}{
		{
			name:       "successful permission check with admin role",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// First call: check if user is collaborator (GitHub returns 204)
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				// Second call: get permission (returns permission data)
				mockClient.setResponse(permissionExternalURL, http.StatusOK, validPermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"admin"`,
			expectedRequestCount: 2,
			verifyRequests: func(t *testing.T, mockClient *mockHTTPClient) {
				if mockClient.getRequestCount() != 2 {
					t.Errorf("Expected 2 requests, got %d", mockClient.getRequestCount())
				}

				// Verify first request (collaborator check)
				req1 := mockClient.requests[0]
				if req1.URL.String() != collaboratorExternalURL {
					t.Errorf("First request URL = %s, want %s", req1.URL.String(), collaboratorExternalURL)
				}
				if req1.Header.Get("Authorization") != testToken {
					t.Errorf("First request Authorization header = %s, want %s", req1.Header.Get("Authorization"), testToken)
				}

				// Verify second request (permission check)
				req2 := mockClient.requests[1]
				if req2.URL.String() != permissionExternalURL {
					t.Errorf("Second request URL = %s, want %s", req2.URL.String(), permissionExternalURL)
				}
				if req2.Header.Get("Authorization") != testToken {
					t.Errorf("Second request Authorization header = %s, want %s", req2.Header.Get("Authorization"), testToken)
				}
			},
		},
		{
			name:       "successful permission check with read permission corrected to pull",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				readPermissionResp := `{
					"permission": "read",
					"user": {
						"login": "testuser",
						"id": 12345,
						"html_url": "https://github.com/testuser",
						"permissions": {
							"admin": false,
							"maintain": false,
							"push": false,
							"triage": true,
							"pull": true
						}
					},
					"role_name": "read"
				}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, readPermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"pull"`,
			expectedRequestCount: 2,
		},
		{
			name:       "successful permission check with write permission corrected to push",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				writePermissionResp := `{
					"permission": "write",
					"user": {
						"login": "testuser",
						"id": 12345,
						"html_url": "https://github.com/testuser",
						"permissions": {
							"admin": false,
							"maintain": false,
							"push": true,
							"triage": true,
							"pull": true
						}
					},
					"role_name": "write"
				}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, writePermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"push"`,
			expectedRequestCount: 2,
		},
		{
			name:       "successful permission check with maintain role",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				maintainPermissionResp := `{
					"permission": "write",
					"user": {
						"login": "testuser",
						"id": 12345,
						"html_url": "https://github.com/testuser",
						"permissions": {
							"admin": false,
							"maintain": true,
							"push": true,
							"triage": true,
							"pull": true
						}
					},
					"role_name": "maintain"
				}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, maintainPermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"maintain"`,
			expectedRequestCount: 2,
		},
		{
			name:       "successful permission check with triage role",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				triagePermissionResp := `{
					"permission": "read",
					"user": {
						"login": "testuser",
						"id": 12345,
						"html_url": "https://github.com/testuser",
						"permissions": {
							"admin": false,
							"maintain": false,
							"push": false,
							"triage": true,
							"pull": true
						}
					},
					"role_name": "triage"
				}`
				mockClient.setResponse(permissionExternalURL, http.StatusOK, triagePermissionResp)
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: `"permission":"triage"`,
			expectedRequestCount: 2,
		},
		{
			name:       "user is not a collaborator",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// GitHub returns 404 when user is not a collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
			},
			expectedStatus:       http.StatusNotFound,
			expectedContentType:  "",
			expectedBodyContains: "User is not a collaborator of the repository or the user does not exist",
			expectedRequestCount: 1,
		},
		{
			name:       "error checking collaborator status",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setError(collaboratorExternalURL, fmt.Errorf("network error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error checking collaborator status",
			expectedRequestCount: 1,
		},
		{
			name:       "error getting user permission",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				mockClient.setError(permissionExternalURL, fmt.Errorf("permission API error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error getting user permission",
			expectedRequestCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			handler := createTestGetHandler(mockClient)

			// Create a proper ServeMux with the pattern matching
			mux := http.NewServeMux()
			mux.Handle("GET /repository/{owner}/{repo}/collaborators/{username}/permission", handler)

			// Create request with the proper path
			path := fmt.Sprintf("/repository/%s/%s/collaborators/%s/permission", tt.owner, tt.repo, tt.username)
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

// POST Handler Tests
func TestPostHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		owner                string
		repo                 string
		username             string
		authHeader           string
		requestBody          string
		setupMock            func(*mockHTTPClient)
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
		expectedRequestCount int
	}{
		{
			name:        "successful invitation sent",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// GitHub returns 201 when invitation is sent
				mockClient.setResponse(collaboratorExternalURL, http.StatusCreated, `{}`)
			},
			expectedStatus:       http.StatusAccepted,
			expectedContentType:  "application/json",
			expectedBodyContains: "Invitation sent to user",
			expectedRequestCount: 1,
		},
		{
			name:        "user already collaborator",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// GitHub returns 204 when user is already a collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
			},
			expectedStatus:       http.StatusNoContent,
			expectedContentType:  "",
			expectedBodyContains: "",
			expectedRequestCount: 1,
		},
		{
			name:        "invalid request body",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `invalid json`,
			setupMock: func(mockClient *mockHTTPClient) {
				// No mock setup needed as it should fail before making requests
			},
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "",
			expectedBodyContains: "Error reading permission from request body",
			expectedRequestCount: 0,
		},
		{
			name:        "missing permission field",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"other_field": "value"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// No mock setup needed as it should fail before making requests
			},
			expectedStatus:       http.StatusBadRequest,
			expectedContentType:  "",
			expectedBodyContains: "Error reading permission from request body",
			expectedRequestCount: 0,
		},
		{
			name:        "github api error",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setError(collaboratorExternalURL, fmt.Errorf("github api error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error adding collaborator",
			expectedRequestCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			handler := createTestPostHandler(mockClient)

			// Create a proper ServeMux with the pattern matching
			mux := http.NewServeMux()
			mux.Handle("POST /repository/{owner}/{repo}/collaborators/{username}", handler)

			// Create request with the proper path
			path := fmt.Sprintf("/repository/%s/%s/collaborators/%s", tt.owner, tt.repo, tt.username)
			req := httptest.NewRequest("POST", path, strings.NewReader(tt.requestBody))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

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
		})
	}
}

// PATCH Handler Tests
func TestPatchHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		owner                string
		repo                 string
		username             string
		authHeader           string
		requestBody          string
		setupMock            func(*mockHTTPClient)
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
		expectedRequestCount int
	}{
		{
			name:        "successful update existing collaborator permission",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "admin"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				// Update permission succeeds
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: "Permission updated successfully",
			expectedRequestCount: 2,
		},
		{
			name:        "successful update invitation permission",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// Has pending invitation
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, validInvitationResp)
				// Update invitation succeeds
				invitationUpdateURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/1", testOwner, testRepo)
				mockClient.setResponse(invitationUpdateURL, http.StatusOK, `{}`)
			},
			expectedStatus:       http.StatusAccepted,
			expectedContentType:  "application/json",
			expectedBodyContains: "Invitation permission updated successfully",
			expectedRequestCount: 3,
		},
		{
			name:        "user not found as collaborator or invitee",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// No pending invitations
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, emptyInvitationResp)
			},
			expectedStatus:       http.StatusNotFound,
			expectedContentType:  "application/json",
			expectedBodyContains: "is not a collaborator and has no pending invitation",
			expectedRequestCount: 2,
		},
		{
			name:        "invalid request body",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `invalid json`,
			setupMock: func(mockClient *mockHTTPClient) {
				// No mock setup needed as it should fail before making requests
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error reading permission from request body",
			expectedRequestCount: 0,
		},
		{
			name:        "missing permission field",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"other_field": "value"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// No mock setup needed as it should fail before making requests
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error reading permission from request body",
			expectedRequestCount: 0,
		},
		{
			name:        "error checking collaborator status",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setError(collaboratorExternalURL, fmt.Errorf("network error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error checking collaborator status",
			expectedRequestCount: 1,
		},
		{
			name:        "error checking invitations",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// Error getting invitations
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setError(invitationsURL, fmt.Errorf("invitations API error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error updating permission",
			expectedRequestCount: 2,
		},
		{
			name:        "github api error when updating invitation",
			owner:       testOwner,
			repo:        testRepo,
			username:    testUsername,
			authHeader:  testToken,
			requestBody: `{"permission": "push"}`,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// Has pending invitation
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, validInvitationResp)
				// GitHub API error on invitation update
				invitationUpdateURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/1", testOwner, testRepo)
				mockClient.setResponse(invitationUpdateURL, http.StatusForbidden, `{"message": "Permission denied"}`)
			},
			expectedStatus:       http.StatusForbidden,
			expectedContentType:  "application/json",
			expectedBodyContains: "",
			expectedRequestCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			handler := createTestPatchHandler(mockClient)

			// Create a proper ServeMux with the pattern matching
			mux := http.NewServeMux()
			mux.Handle("PATCH /repository/{owner}/{repo}/collaborators/{username}", handler)

			// Create request with the proper path
			path := fmt.Sprintf("/repository/%s/%s/collaborators/%s", tt.owner, tt.repo, tt.username)
			req := httptest.NewRequest("PATCH", path, strings.NewReader(tt.requestBody))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

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
		})
	}
}

// DELETE Handler Tests
func TestDeleteHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name                 string
		owner                string
		repo                 string
		username             string
		authHeader           string
		setupMock            func(*mockHTTPClient)
		expectedStatus       int
		expectedContentType  string
		expectedBodyContains string
		expectedRequestCount int
	}{
		{
			name:       "successful collaborator removal",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				// Remove collaborator succeeds
				mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
			},
			expectedStatus:       http.StatusOK,
			expectedContentType:  "application/json",
			expectedBodyContains: "Collaborator testuser removed successfully",
			expectedRequestCount: 2,
		},
		{
			name:       "successful invitation cancellation",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// Has pending invitation
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, validInvitationResp)
				// Cancel invitation succeeds
				invitationDeleteURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/1", testOwner, testRepo)
				mockClient.setResponse(invitationDeleteURL, http.StatusNoContent, "")
			},
			expectedStatus:       http.StatusAccepted,
			expectedContentType:  "application/json",
			expectedBodyContains: "Invitation cancelled successfully",
			expectedRequestCount: 3,
		},
		{
			name:       "user not found as collaborator or invitee",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// No pending invitations
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, emptyInvitationResp)
			},
			expectedStatus:       http.StatusNotFound,
			expectedContentType:  "application/json",
			expectedBodyContains: "is not a collaborator and has no pending invitation",
			expectedRequestCount: 2,
		},
		{
			name:       "error checking collaborator status",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				mockClient.setError(collaboratorExternalURL, fmt.Errorf("network error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error checking collaborator status",
			expectedRequestCount: 1,
		},
		{
			name:       "error checking invitations",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// Error getting invitations
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setError(invitationsURL, fmt.Errorf("invitations API error"))
			},
			expectedStatus:       http.StatusInternalServerError,
			expectedContentType:  "",
			expectedBodyContains: "Error removing user",
			expectedRequestCount: 2,
		},
		{
			name:       "github api error when cancelling invitation",
			owner:      testOwner,
			repo:       testRepo,
			username:   testUsername,
			authHeader: testToken,
			setupMock: func(mockClient *mockHTTPClient) {
				// User is not collaborator
				mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				// Has pending invitation
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, validInvitationResp)
				// GitHub API error on invitation cancellation
				invitationDeleteURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/invitations/1", testOwner, testRepo)
				mockClient.setResponse(invitationDeleteURL, http.StatusForbidden, `{"message": "Permission denied"}`)
			},
			expectedStatus:       http.StatusForbidden,
			expectedContentType:  "application/json",
			expectedBodyContains: "",
			expectedRequestCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			handler := createTestDeleteHandler(mockClient)

			// Create a proper ServeMux with the pattern matching
			mux := http.NewServeMux()
			mux.Handle("DELETE /repository/{owner}/{repo}/collaborators/{username}", handler)

			// Create request with the proper path
			path := fmt.Sprintf("/repository/%s/%s/collaborators/%s", tt.owner, tt.repo, tt.username)
			req := httptest.NewRequest("DELETE", path, nil)
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
		})
	}
}

// Test helper methods that may be missing
func TestBaseHandler_Methods(t *testing.T) {
	t.Run("checkCollaboratorStatus", func(t *testing.T) {
		tests := []struct {
			name           string
			setupMock      func(*mockHTTPClient)
			expectedStatus CollaboratorStatus
			expectError    bool
		}{
			{
				name: "user is collaborator",
				setupMock: func(mockClient *mockHTTPClient) {
					mockClient.setResponse(collaboratorExternalURL, http.StatusNoContent, "")
				},
				expectedStatus: StatusCollaborator,
				expectError:    false,
			},
			{
				name: "user is not collaborator",
				setupMock: func(mockClient *mockHTTPClient) {
					mockClient.setResponse(collaboratorExternalURL, http.StatusNotFound, `{"message": "Not Found"}`)
				},
				expectedStatus: StatusNotCollaborator,
				expectError:    false,
			},
			{
				name: "unexpected status code",
				setupMock: func(mockClient *mockHTTPClient) {
					mockClient.setResponse(collaboratorExternalURL, http.StatusInternalServerError, `{"message": "Server Error"}`)
				},
				expectedStatus: StatusNotCollaborator,
				expectError:    true,
			},
			{
				name: "network error",
				setupMock: func(mockClient *mockHTTPClient) {
					mockClient.setError(collaboratorExternalURL, fmt.Errorf("network error"))
				},
				expectedStatus: StatusNotCollaborator,
				expectError:    true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockClient := newMockHTTPClient()
				tt.setupMock(mockClient)

				handler := createTestGetHandler(mockClient)

				status, err := handler.checkCollaboratorStatus(testOwner, testRepo, testUsername, testToken)

				if tt.expectError && err == nil {
					t.Error("expected error but got nil")
				}
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if status != tt.expectedStatus {
					t.Errorf("expected status %v, got %v", tt.expectedStatus, status)
				}
			})
		}
	})
}

// Test findUserInvitationHelper function
func TestFindUserInvitationHelper(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mockHTTPClient)
		expectedFound bool
		expectError   bool
	}{
		{
			name: "user has pending invitation",
			setupMock: func(mockClient *mockHTTPClient) {
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, validInvitationResp)
			},
			expectedFound: true,
			expectError:   false,
		},
		{
			name: "user has no pending invitation",
			setupMock: func(mockClient *mockHTTPClient) {
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusOK, emptyInvitationResp)
			},
			expectedFound: false,
			expectError:   false,
		},
		{
			name: "failed to get invitations - permission denied",
			setupMock: func(mockClient *mockHTTPClient) {
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setResponse(invitationsURL, http.StatusForbidden, `{"message": "Permission denied"}`)
			},
			expectedFound: false,
			expectError:   false,
		},
		{
			name: "network error getting invitations",
			setupMock: func(mockClient *mockHTTPClient) {
				invitationsURL := fmt.Sprintf("%s?per_page=30&page=1", invitationsExternalURL)
				mockClient.setError(invitationsURL, fmt.Errorf("network error"))
			},
			expectedFound: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockHTTPClient()
			tt.setupMock(mockClient)

			logger := zerolog.New(io.Discard).With().Timestamp().Logger()

			invitation, found, err := findUserInvitationHelper(mockClient, &logger, testOwner, testRepo, testUsername, testToken)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if found != tt.expectedFound {
				t.Errorf("expected found %v, got %v", tt.expectedFound, found)
			}
			if tt.expectedFound && invitation == nil {
				t.Error("expected invitation but got nil")
			}
			if !tt.expectedFound && invitation != nil {
				t.Error("expected no invitation but got one")
			}
		})
	}
}
