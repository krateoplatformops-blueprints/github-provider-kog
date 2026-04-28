package main

import (
	"net/http"

	branchprotection "github.com/krateoplatformops/github-provider-kog/branchprotection-plugin/handlers"
	"github.com/krateoplatformops/github-provider-kog/pkg/handlers"
	"github.com/krateoplatformops/github-provider-kog/pkg/health"
	"github.com/krateoplatformops/github-provider-kog/pkg/server"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           GitHub BranchProtection Plugin API for Krateo Operator Generator (KOG)
// @version         1.0
// @description     Wrapper around the GitHub Branch Protection API that normalizes response fields for consistent reconciliation in KOG.
// @termsOfService  http://swagger.io/terms/
// @contact.name    Krateo Support
// @contact.url     https://krateo.io
// @contact.email   contact@krateoplatformops.io
// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html
// @host            api.github.com
// @BasePath        /
// @schemes         http
func main() {
	srv := server.New()

	opts := handlers.HandlerOptions{
		Log:    &log.Logger,
		Client: http.DefaultClient,
	}

	// BranchProtection — uses PUT for both create and update (GitHub is idempotent).
	srv.Mux().Handle("GET /api/{owner}/{repo}/branches/{branch}/protection", branchprotection.GetBranchProtection(opts))
	srv.Mux().Handle("PUT /api/{owner}/{repo}/branches/{branch}/protection", branchprotection.PutBranchProtection(opts))
	srv.Mux().Handle("DELETE /api/{owner}/{repo}/branches/{branch}/protection", branchprotection.DeleteBranchProtection(opts))

	// Swagger UI
	srv.Mux().Handle("/swagger/", httpSwagger.WrapHandler)

	// Kubernetes health check endpoints
	srv.Mux().HandleFunc("GET /healthz", health.LivenessHandler(srv.Healthy()))
	srv.Mux().HandleFunc("GET /readyz", health.ReadinessHandler(srv.Ready(), opts.Client.(*http.Client)))

	srv.Run()
}
