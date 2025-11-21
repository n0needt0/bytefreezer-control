package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/n0needt0/go-goodies/log"
)

// GetAPIDocsForAI returns the comprehensive API documentation for AI agents
func (api *API) GetAPIDocsForAI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the AI agent API documentation file
		docsPath := filepath.Join("docs", "AI_AGENT_API.md")
		content, err := os.ReadFile(docsPath)
		if err != nil {
			log.Errorf("Failed to read API docs: %v", err)
			http.Error(w, fmt.Sprintf("Failed to read documentation: %v", err), http.StatusInternalServerError)
			return
		}

		// Set content type to markdown
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

// GetAPIQuickReference returns the quick reference guide for AI agents
func (api *API) GetAPIQuickReference() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the quick reference file
		docsPath := filepath.Join("docs", "AI_AGENT_QUICK_REFERENCE.md")
		content, err := os.ReadFile(docsPath)
		if err != nil {
			log.Errorf("Failed to read quick reference: %v", err)
			http.Error(w, fmt.Sprintf("Failed to read documentation: %v", err), http.StatusInternalServerError)
			return
		}

		// Set content type to markdown
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}
}

// GetAPIDocsIndex returns an index of available documentation
func (api *API) GetAPIDocsIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		index := map[string]interface{}{
			"documentation": map[string]string{
				"comprehensive_api": "/api/v1/docs/ai-agent",
				"quick_reference":   "/api/v1/docs/ai-agent/quick-reference",
				"filter_catalog":    "/api/v1/filters/catalog",
				"openapi_spec":      "/api/v1/openapi.json",
				"swagger_ui":        "/v1/docs",
			},
			"description": "ByteFreezer API Documentation for AI Agents",
			"version":     "v1.0.0",
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)

		// Simple JSON encoding
		fmt.Fprintf(w, `{
  "documentation": {
    "comprehensive_api": "%s",
    "quick_reference": "%s",
    "filter_catalog": "%s",
    "openapi_spec": "%s",
    "swagger_ui": "%s"
  },
  "description": "%s",
  "version": "%s"
}`,
			index["documentation"].(map[string]string)["comprehensive_api"],
			index["documentation"].(map[string]string)["quick_reference"],
			index["documentation"].(map[string]string)["filter_catalog"],
			index["documentation"].(map[string]string)["openapi_spec"],
			index["documentation"].(map[string]string)["swagger_ui"],
			index["description"],
			index["version"],
		)
	}
}
