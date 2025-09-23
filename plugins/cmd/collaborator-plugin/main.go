package main

import (
	"net/http"

	"github.com/krateoplatformops/github-provider-kog/pkg/handlers"
	"github.com/krateoplatformops/github-provider-kog/collaborator-plugin/handlers"
	"github.com/krateoplatformops/github-provider-kog/pkg/health"
	"github.com/krateoplatformops/github-provider-kog/pkg/server"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

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
