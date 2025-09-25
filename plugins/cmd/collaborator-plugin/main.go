package main

import (
	"net/http"

	collaborator "github.com/krateoplatformops/github-provider-kog/collaborator-plugin/handlers"
	"github.com/krateoplatformops/github-provider-kog/pkg/handlers"
	"github.com/krateoplatformops/github-provider-kog/pkg/health"
	"github.com/krateoplatformops/github-provider-kog/pkg/server"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           GitHub Collaborator Plugin API for Krateo Operator Generator (KOG)
// @version         1.0
// @description     Simple wrapper around GitHub API to provide consisentency of API response for Krateo Operator Generator (KOG)
// @termsOfService  http://swagger.io/terms/
// @contact.name    Krateo Support
// @contact.url     https://krateo.io
// @contact.email   contact@krateoplatformops.io
// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html
// @host            localhost:8080
// @BasePath        /
// @schemes         http
func main() {
	srv := server.New()

	opts := handlers.HandlerOptions{
		Log:    &log.Logger,
		Client: http.DefaultClient,
	}

	// Collaborator
	srv.Mux().Handle("GET /repository/{owner}/{repo}/collaborators/{username}/permission", collaborator.GetCollaborator(opts))
	srv.Mux().Handle("POST /repository/{owner}/{repo}/collaborators/{username}", collaborator.PostCollaborator(opts))
	srv.Mux().Handle("PATCH /repository/{owner}/{repo}/collaborators/{username}", collaborator.PatchCollaborator(opts))
	srv.Mux().Handle("DELETE /repository/{owner}/{repo}/collaborators/{username}", collaborator.DeleteCollaborator(opts))

	// Swagger UI
	srv.Mux().Handle("/swagger/", httpSwagger.WrapHandler)

	// Kubernetes health check endpoints
	srv.Mux().HandleFunc("GET /healthz", health.LivenessHandler(srv.Healthy()))
	srv.Mux().HandleFunc("GET /readyz", health.ReadinessHandler(srv.Ready(), opts.Client.(*http.Client)))

	srv.Run()
}

// test
