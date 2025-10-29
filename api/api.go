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

	// Add authentication middleware (always enabled, but exclude public endpoints)
	service.Use(middleware.ConditionalJWTAuthMiddleware(api.Config.Auth, api.Services.Auth))
	log.Info("JWT authentication middleware enabled")

	// Add audit middleware to extract user info and IP for audit logging
	// This must come AFTER auth middleware so JWT claims are available in context
	service.Use(middleware.AuditMiddleware())
	log.Info("Audit middleware enabled")

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

	// Flat list endpoints (for UI convenience) - MUST come before parametrized routes
	service.Get("/api/v1/tenants", api.ListAllTenants())
	service.Get("/api/v1/datasets", api.ListAllDatasets())

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
	service.Post("/api/v1/tenants/{tenantId}/datasets/{datasetId}/test", api.TestDataset())

	// Dataset metrics endpoints
	service.Post("/api/v1/tenants/{tenantId}/datasets/{datasetId}/metrics", api.RecordDatasetMetric())
	service.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}/metrics", api.QueryDatasetMetrics())
	service.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}/metrics/aggregated", api.GetAggregatedMetrics())

	// User management endpoints
	service.Get("/api/v1/users", api.ListUsers())
	service.Get("/api/v1/users/{userId}", api.GetUser())
	service.Post("/api/v1/users", api.CreateUser())
	service.Put("/api/v1/users/{userId}", api.UpdateUser())
	service.Delete("/api/v1/users/{userId}", api.DeleteUser())
	service.Post("/api/v1/users/{userId}/toggle-active", api.ToggleUserActive())

	// Audit log endpoints
	service.Get("/api/v1/audit-logs", api.ListAuditLogs())
	service.Get("/api/v1/audit-logs/{logId}", api.GetAuditLog())

	// Plugin schema endpoints
	service.Get("/api/v1/plugins", api.GetPluginSchemas())

	// Account proxy health endpoints
	service.Get("/api/v1/accounts/{account_id}/proxies", api.GetAccountProxies())

	// Proxy configuration management endpoints
	service.Get("/api/v1/proxies", api.ListProxyInstances())
	service.Get("/api/v1/proxies/{instanceId}/config", api.GetProxyConfig())
	service.Put("/api/v1/proxies/{instanceId}/config", api.UpsertProxyConfig())
	service.Post("/api/v1/proxies/{instanceId}/config/applied", api.MarkProxyConfigApplied())
	service.Get("/api/v1/proxies/{instanceId}/config/history", api.GetProxyConfigHistory())
	service.Delete("/api/v1/proxies/{instanceId}", api.DeleteProxyInstance())

	// Proxy configuration polling endpoint (returns tenants + datasets for account)
	service.Get("/api/v1/proxy/config", api.GetProxyConfiguration())

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
