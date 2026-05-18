package handlers

import (
	"net/http"
)

// HTTPClient interface allows mocking of HTTP client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Logger interface allows mocking of logger
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type HandlerOptions struct {
	Client  HTTPClient // HTTPClient interface
	Log     Logger     // Logger interface
	BaseURL string     // GitHub API base URL, e.g. "https://api.github.com" or "https://ghe.corp.com/api/v3"
}

// Handler interface
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
