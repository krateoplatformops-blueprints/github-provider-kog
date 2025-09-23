package main

import (
	"net/http"

	"github.com/krateoplatformops/github-provider-kog/pkg/health"
	"github.com/krateoplatformops/github-provider-kog/pkg/server"
	"github.com/krateoplatformops/github-provider-kog/pkg/handlers"
	"github.com/krateoplatformops/github-provider-kog/teamrepo-plugin/handlers"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	srv := server.New()

	opts := handlers.HandlerOptions{
		Log:    &log.Logger,
		Client: http.DefaultClient,
	}

	// TeamRepo
	srv.Mux().Handle("GET /teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamrepo.GetTeamRepo(opts))

	// Swagger UI
	srv.Mux().Handle("/swagger/", httpSwagger.WrapHandler)

	// Kubernetes health check endpoints
	srv.Mux().HandleFunc("GET /healthz", health.LivenessHandler(srv.Healthy()))
	srv.Mux().HandleFunc("GET /readyz", health.ReadinessHandler(srv.Ready(), opts.Client.(*http.Client)))

	srv.Run()
}
