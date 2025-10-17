package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/web"
	swgui "github.com/swaggest/swgui/v5emb"
	"go.opentelemetry.io/otel/metric"
)

type API struct {
	Services   *services.Services
	ApiMetrics map[string]metric.Int64Counter
	HttpServer *http.Server
	sync.RWMutex
	Config *config.Config
}

// NewAPI creates a new API instance
func NewAPI(services *services.Services, conf *config.Config) *API {
	return &API{
		Services:   services,
		ApiMetrics: make(map[string]metric.Int64Counter),
		Config:     conf,
	}
}

// NewRouter returns a new router serving API endpoints
func (api *API) NewRouter() *web.Service {
	service := web.NewService(openapi3.NewReflector())

	// Configure OpenAPI schema
	service.OpenAPISchema().SetTitle("ByteFreezer Control API")
	service.OpenAPISchema().SetDescription("ByteFreezer Control API for ecosystem management and monitoring")
	service.OpenAPISchema().SetVersion("v1.0.0")

	// Apply defaults for decoder factory
	service.DecoderFactory.ApplyDefaults = true

	// Add CORS middleware
	service.Use(middleware.CORSMiddleware())
	log.Info("CORS middleware enabled")

	// Add rate limiting middleware if enabled
	if api.Config.RateLimit.Enabled {
		rateLimiter := middleware.NewRateLimiter(api.Config.RateLimit)
		service.Use(rateLimiter.RateLimitMiddleware())
		log.Info("Rate limiting middleware enabled")
	}

	// Add authentication middleware if enabled (but exclude public endpoints)
	if api.Config.Auth.Enabled {
		service.Use(middleware.ConditionalAuthMiddleware(api.Config.Auth))
		log.Info("Authentication middleware enabled")
	}

	// Wrap to finalize middleware setup
	service.Wrap()

	// Health check endpoint (public)
	service.Get("/api/v1/health", api.HealthCheck())

	// Authentication endpoints (public)
	service.Post("/api/v1/login", api.Login())
	service.Post("/api/v1/refresh", api.RefreshToken())
	service.Post("/api/v1/password-reset", api.RequestPasswordReset())

	// Configuration endpoints
	service.Get("/api/v1/config", api.GetConfig())
	service.Put("/api/v1/config", api.UpdateConfig())

	// Statistics endpoint
	service.Get("/api/v1/stats", api.GetStats())

	// Ecosystem health endpoint
	service.Get("/api/v1/ecosystem/health", api.GetEcosystemHealth())

	// Service reporting endpoint (legacy)
	service.Post("/api/v1/services/report", api.ReceiveServiceReport())

	// Health monitoring endpoints
	service.Post("/api/v1/health/register", api.RegisterService())
	service.Get("/api/v1/health/status", api.GetHealthStatus())
	service.Get("/api/v1/health/summary", api.GetHealthSummary())
	service.Get("/api/v1/health/services/{serviceType}", api.GetServiceHealth())

	// Account management endpoints
	service.Get("/api/v1/accounts", api.ListAccounts())
	service.Get("/api/v1/accounts/{accountId}", api.GetAccount())
	service.Post("/api/v1/accounts", api.CreateAccount())
	service.Put("/api/v1/accounts/{accountId}", api.UpdateAccount())
	service.Delete("/api/v1/accounts/{accountId}", api.DeleteAccount())
	service.Post("/api/v1/accounts/{accountId}/assume-admin", api.AssumeAccountAdmin())

	// Tenant management endpoints (scoped to account)
	service.Get("/api/v1/accounts/{accountId}/tenants", api.ListTenants())
	service.Get("/api/v1/accounts/{accountId}/tenants/{tenantId}", api.GetTenant())
	service.Post("/api/v1/accounts/{accountId}/tenants", api.CreateTenant())
	service.Put("/api/v1/accounts/{accountId}/tenants/{tenantId}", api.UpdateTenant())
	service.Delete("/api/v1/accounts/{accountId}/tenants/{tenantId}", api.DeleteTenant())

	// Direct tenant lookup endpoint (for proxy validation - no account ID required)
	service.Get("/api/v1/tenants/{tenantId}", api.GetTenantDirect())

	// Dataset management endpoints (scoped to tenant)
	service.Get("/api/v1/tenants/{tenantId}/datasets", api.ListDatasets())
	service.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}", api.GetDataset())
	service.Post("/api/v1/tenants/{tenantId}/datasets", api.CreateDataset())
	service.Put("/api/v1/tenants/{tenantId}/datasets/{datasetId}", api.UpdateDataset())
	service.Delete("/api/v1/tenants/{tenantId}/datasets/{datasetId}", api.DeleteDataset())

	// API documentation
	service.Docs("/v1/docs", swgui.New)

	// Root redirect to documentation
	service.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v1/docs", http.StatusFound)
	})

	return service
}

// Serve serves HTTP endpoints
func (api *API) Serve(address string, router http.Handler) {
	log.Infof("Control API server started on %s", address)

	api.HttpServer = &http.Server{
		Addr:           address,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	err := api.HttpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("Control API server closed")
	} else {
		log.Errorf("Control API server failed and closed: %v", err)
	}
}

// Stop stops the server
func (api *API) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer func() {
		api.HttpServer = nil
		cancel()
	}()

	if api.HttpServer != nil {
		if err := api.HttpServer.Shutdown(ctx); err != nil {
			log.Errorf("Error shutting down Control API server: %v", err)
		}
	}

	log.Info("Control API server shut down gracefully")
}
