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

	// Add authentication middleware (always enabled, but exclude public endpoints)
	service.Use(middleware.ConditionalJWTAuthMiddleware(api.Config.Auth, api.Services.Auth))
	log.Info("JWT authentication middleware enabled")

	// Add rate limiting middleware if enabled (must be AFTER auth middleware)
	if api.Config.RateLimit.Enabled {
		rateLimiter := middleware.NewRateLimiter(api.Config.RateLimit, api.Services.Auth)
		service.Use(rateLimiter.RateLimitMiddleware())
		log.Info("Token-based rate limiting middleware enabled")
	}

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

	// Account-scoped health endpoints (for proxy services)
	service.Post("/api/v1/accounts/{accountId}/services/register", api.RegisterAccountService())
	service.Post("/api/v1/accounts/{accountId}/services/report", api.ReceiveAccountServiceReport())
	service.Get("/api/v1/accounts/{accountId}/services", api.ListAccountServices())

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

	// Transformation job endpoints (proxy to piper)
	service.Post("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/test", api.CreateTransformationTest())
	service.Post("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/validate", api.CreateTransformationValidate())
	service.Post("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/activate", api.CreateTransformationActivate())
	service.Get("/api/v1/transformations/jobs/{jobId}", api.GetTransformationJobStatus())
	service.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/jobs", api.ListTransformationJobs())

	// Transformation read-only endpoints (proxy to piper - protected with auth and CORS)
	corsMiddleware := middleware.CORSMiddleware()
	authMiddleware := middleware.ConditionalJWTAuthMiddleware(api.Config.Auth, api.Services.Auth)

	// Wrap handlers with both CORS and auth middleware
	service.Router.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/schema", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(authMiddleware(api.GetTransformationSchema())).ServeHTTP(w, r)
	})
	service.Router.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/stats", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(authMiddleware(api.GetTransformationStats())).ServeHTTP(w, r)
	})
	service.Router.Get("/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/preview", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware(authMiddleware(api.GetTransformationPreview())).ServeHTTP(w, r)
	})

	// Piper submits schema and samples (service-to-service, requires API key auth)
	service.Post("/api/v1/tenants/{tenantId}/datasets/{datasetId}/schema", api.SubmitDatasetSchema())

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

	// Error reporting endpoints (system services)
	service.Post("/api/v1/errors", api.ReportError())
	service.Get("/api/v1/errors", api.ListErrors())
	service.Get("/api/v1/errors/stats", api.GetErrorStats())
	service.Patch("/api/v1/errors/{errorId}", api.UpdateErrorStatus())

	// Error reporting endpoints (account-scoped)
	service.Post("/api/v1/accounts/{accountId}/errors", api.ReportAccountError())
	service.Get("/api/v1/accounts/{accountId}/errors", api.ListAccountErrors())
	service.Get("/api/v1/accounts/{accountId}/errors/stats", api.GetAccountErrorStats())
	service.Get("/api/v1/accounts/{accountId}/datasets/{datasetId}/errors", api.ListDatasetErrors())

	// Plugin schema endpoints
	service.Get("/api/v1/plugins", api.GetPluginSchemas())

	// Account proxy health endpoints
	service.Get("/api/v1/accounts/{account_id}/proxies", api.GetAccountProxies())

	// Proxy health endpoints (for system admins - returns health data)
	service.Get("/api/v1/proxies/health", api.ListAllProxies())

	// Proxy configuration management endpoints
	service.Get("/api/v1/proxies", api.ListProxyInstances())
	service.Get("/api/v1/proxies/{instanceId}/config", api.GetProxyConfig())
	service.Put("/api/v1/proxies/{instanceId}/config", api.UpsertProxyConfig())
	service.Post("/api/v1/proxies/{instanceId}/config/applied", api.MarkProxyConfigApplied())
	service.Get("/api/v1/proxies/{instanceId}/config/history", api.GetProxyConfigHistory())
	service.Delete("/api/v1/proxies/{instanceId}", api.DeleteProxyInstance())

	// Proxy configuration polling endpoint (returns tenants + datasets for account)
	service.Get("/api/v1/proxy/config", api.GetProxyConfiguration())

	// ====================================================================================
	// PIPER API ENDPOINTS (for piper service state management via control API)
	// ====================================================================================

	// Piper File Lock operations
	service.Post("/api/v1/piper/locks/files", api.AcquireFileLock())
	service.Post("/api/v1/piper/locks/files/release", api.ReleaseFileLock())
	service.Delete("/api/v1/piper/locks/files", api.ReleaseFileLock()) // Legacy endpoint
	service.Get("/api/v1/piper/locks/files/{tenant_id}/{dataset_id}/{file_key}", api.CheckFileLock())
	service.Delete("/api/v1/piper/locks/files/cleanup/expired", api.CleanupExpiredFileLocks())
	service.Delete("/api/v1/piper/locks/files/cleanup/stale", api.CleanupStaleFileLocks())
	service.Delete("/api/v1/piper/locks/files/cleanup/instance", api.CleanupInstanceFileLocks())

	// Piper Job Record operations
	service.Post("/api/v1/piper/jobs", api.CreatePiperJob())
	service.Put("/api/v1/piper/jobs/{job_id}/status", api.UpdatePiperJobStatus())
	service.Get("/api/v1/piper/jobs/{job_id}", api.GetPiperJob())
	service.Get("/api/v1/piper/jobs", api.ListPiperJobsByStatus())
	service.Get("/api/v1/piper/jobs/tenant/{tenant_id}", api.ListPiperJobsForTenant())
	service.Delete("/api/v1/piper/jobs/cleanup/old", api.CleanupOldPiperJobs())

	// Piper Pipeline Configuration Cache operations
	service.Post("/api/v1/piper/cache/pipelines", api.CachePipelineConfiguration())
	service.Get("/api/v1/piper/cache/pipelines/{tenant_id}/{dataset_id}", api.GetCachedPipelineConfiguration())
	service.Delete("/api/v1/piper/cache/pipelines/{tenant_id}/{dataset_id}", api.InvalidatePipelineConfiguration())
	service.Get("/api/v1/piper/cache/pipelines", api.ListCachedPipelines())
	service.Delete("/api/v1/piper/cache/pipelines/cleanup/expired", api.CleanupExpiredPipelineCache())

	// Piper Tenant Cache operations
	service.Post("/api/v1/piper/cache/tenants", api.CacheTenant())
	service.Get("/api/v1/piper/cache/tenants", api.GetCachedTenants())
	service.Delete("/api/v1/piper/cache/tenants", api.InvalidateTenantCache())
	service.Delete("/api/v1/piper/cache/tenants/cleanup/expired", api.CleanupExpiredTenantCache())

	// Piper Transformation Job operations
	service.Post("/api/v1/piper/transformation-jobs", api.CreatePiperTransformationJob())
	service.Post("/api/v1/piper/transformation-jobs/claim", api.ClaimPiperTransformationJob())
	service.Put("/api/v1/piper/transformation-jobs/{job_id}", api.UpdatePiperTransformationJob())
	service.Get("/api/v1/piper/transformation-jobs/{job_id}", api.GetPiperTransformationJob())
	service.Get("/api/v1/piper/transformation-jobs/pending", api.ListPendingPiperTransformationJobs())
	service.Delete("/api/v1/piper/transformation-jobs/cleanup/expired", api.CleanupExpiredPiperTransformationJobs())

	// ====================================================================================
	// PACKER API ENDPOINTS (for packer service state management via control API)
	// ====================================================================================

	// Packer Tenant Lock operations
	service.Post("/api/v1/packer/locks/tenants", api.AcquireTenantLock())
	service.Post("/api/v1/packer/locks/tenants/{tenant_id}/release", api.ReleaseTenantLock())
	service.Put("/api/v1/packer/locks/tenants/{tenant_id}/heartbeat", api.UpdateTenantLockHeartbeat())
	service.Get("/api/v1/packer/locks/tenants/{tenant_id}", api.CheckTenantLock())
	service.Delete("/api/v1/packer/locks/tenants/cleanup/expired", api.CleanupExpiredTenantLocks())
	service.Delete("/api/v1/packer/locks/tenants/cleanup/all", api.ClearAllTenantLocks())
	service.Delete("/api/v1/packer/locks/tenants/cleanup/stale", api.CleanupStaleTenantLocks())
	service.Delete("/api/v1/packer/locks/tenants/cleanup/instance", api.CleanupInstanceTenantLocks())

	// Packer Parquet Metadata operations
	service.Post("/api/v1/packer/metadata/files", api.UpsertParquetFileMetadata())
	service.Get("/api/v1/packer/metadata/files/{tenant_id}/{dataset_id}", api.GetParquetFileMetadataByPartition())
	service.Get("/api/v1/packer/metadata/files/{tenant_id}/{dataset_id}/all", api.GetAllParquetFileMetadata())
	service.Delete("/api/v1/packer/metadata/files/{tenant_id}/{dataset_id}", api.DeleteParquetFileMetadata())
	service.Post("/api/v1/packer/metadata/files/cleanup/orphaned", api.CleanupOrphanedParquetMetadata())
	service.Delete("/api/v1/packer/metadata/files/cleanup/expired", api.CleanupExpiredParquetMetadata())

	// Packer Metadata Generation Status operations
	service.Post("/api/v1/packer/metadata/generation/status", api.UpdateMetadataGenerationStatus())
	service.Get("/api/v1/packer/metadata/generation/status/{tenant_id}/{dataset_id}/{partition_path}", api.GetMetadataGenerationStatus())

	// Packer Metadata Summary operations
	service.Get("/api/v1/packer/metadata/summary/{tenant_id}/{dataset_id}/{partition_path}", api.GetParquetMetadataSummary())

	// ====================================================================================
	// ACTIVITY TRACKING ENDPOINTS (for real-time operation monitoring)
	// ====================================================================================

	// Activity tracking endpoints (services report here)
	service.Post("/api/v1/activity/operations", api.UpsertServiceOperation())

	// Activity retrieval endpoints (UI queries here)
	service.Get("/api/v1/activity/operations/active", api.GetActiveOperations())
	service.Get("/api/v1/activity/operations/recent", api.GetRecentOperations())
	service.Get("/api/v1/activity/summary", api.GetActivitySummary())

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
