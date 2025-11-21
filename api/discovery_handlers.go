package api

import (
	"encoding/json"
	"net/http"
)

// APIDiscoveryResponse represents the root API discovery endpoint
type APIDiscoveryResponse struct {
	Version       string                 `json:"version"`
	Documentation DocumentationLinks     `json:"documentation"`
	Endpoints     EndpointCategories     `json:"endpoints"`
	RateLimit     RateLimitInfo          `json:"rate_limit"`
	Features      []string               `json:"features"`
}

type DocumentationLinks struct {
	ComprehensiveAPI string `json:"comprehensive_api"`
	QuickReference   string `json:"quick_reference"`
	FilterCatalog    string `json:"filter_catalog"`
	OpenAPISpec      string `json:"openapi_spec"`
	SwaggerUI        string `json:"swagger_ui"`
}

type EndpointCategories struct {
	Authentication   []string `json:"authentication"`
	Resources        []string `json:"resources"`
	Transformations  []string `json:"transformations"`
	AIAssistant      []string `json:"ai_assistant"`
	Monitoring       []string `json:"monitoring"`
}

type RateLimitInfo struct {
	DefaultLimit   int    `json:"default_limit"`
	ServiceLimit   int    `json:"service_limit"`
	WindowDuration string `json:"window_duration"`
}

// GetAPIDiscovery returns the root API discovery information
func (api *API) GetAPIDiscovery() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := APIDiscoveryResponse{
			Version: "1.0.0",
			Documentation: DocumentationLinks{
				ComprehensiveAPI: "/api/v1/docs/ai-agent",
				QuickReference:   "/api/v1/docs/ai-agent/quick-reference",
				FilterCatalog:    "/api/v1/filters/catalog",
				OpenAPISpec:      "/api/v1/openapi.json",
				SwaggerUI:        "/v1/docs",
			},
			Endpoints: EndpointCategories{
				Authentication: []string{
					"/api/v1/login",
					"/api/v1/refresh",
					"/api/v1/password-reset",
				},
				Resources: []string{
					"/api/v1/accounts",
					"/api/v1/tenants",
					"/api/v1/datasets",
					"/api/v1/users",
				},
				Transformations: []string{
					"/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/schema",
					"/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/test",
					"/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/validate",
					"/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/activate",
					"/api/v1/transformations/jobs/{jobId}",
				},
				AIAssistant: []string{
					"/api/v1/ai/pipeline/generate",
					"/api/v1/ai/pipeline/validate",
					"/api/v1/filters/catalog",
				},
				Monitoring: []string{
					"/api/v1/health",
					"/api/v1/stats",
					"/api/v1/ecosystem/health",
				},
			},
			RateLimit: RateLimitInfo{
				DefaultLimit:   100,
				ServiceLimit:   1000,
				WindowDuration: "1 minute",
			},
			Features: []string{
				"async_job_processing",
				"ai_pipeline_generation",
				"real_time_transformation_testing",
				"comprehensive_filter_catalog",
				"rate_limiting",
				"jwt_authentication",
				"openapi_specification",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
