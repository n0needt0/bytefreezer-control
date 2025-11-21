package api

import (
	"encoding/json"
	"net/http"
)

// EnhancedError represents an AI-friendly error response
type EnhancedError struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code            string                 `json:"code"`
	Message         string                 `json:"message"`
	Details         map[string]interface{} `json:"details,omitempty"`
	Suggestion      string                 `json:"suggestion,omitempty"`
	DocumentationURL string                `json:"documentation_url,omitempty"`
	RequestID       string                 `json:"request_id,omitempty"`
}

// RespondWithEnhancedError sends an enhanced error response
func RespondWithEnhancedError(w http.ResponseWriter, statusCode int, code, message string, details map[string]interface{}, suggestion, docURL string) {
	errorResponse := EnhancedError{
		Error: ErrorDetail{
			Code:            code,
			Message:         message,
			Details:         details,
			Suggestion:      suggestion,
			DocumentationURL: docURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// Common error codes
const (
	ErrCodeInvalidRequest      = "INVALID_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	ErrCodeInternalError       = "INTERNAL_ERROR"
	ErrCodeInvalidFilterType   = "INVALID_FILTER_TYPE"
	ErrCodeInvalidJSON         = "INVALID_JSON"
	ErrCodeMissingParameter    = "MISSING_PARAMETER"
	ErrCodeInvalidParameter    = "INVALID_PARAMETER"
	ErrCodeJobNotFound         = "JOB_NOT_FOUND"
	ErrCodeJobFailed           = "JOB_FAILED"
	ErrCodeResourceNotFound    = "RESOURCE_NOT_FOUND"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
)

// StandardResponse wraps API responses with metadata
type StandardResponse struct {
	Data interface{}      `json:"data"`
	Meta ResponseMetadata `json:"meta"`
}

type ResponseMetadata struct {
	Endpoint   string                 `json:"endpoint"`
	Method     string                 `json:"method"`
	Version    string                 `json:"version"`
	RequestID  string                 `json:"request_id,omitempty"`
	Pagination *PaginationMeta        `json:"pagination,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

type PaginationMeta struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalItems int    `json:"total_items"`
	TotalPages int    `json:"total_pages"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
	NextURL    string `json:"next_url,omitempty"`
	PrevURL    string `json:"prev_url,omitempty"`
}

// RespondWithStandardResponse sends a standardized response with metadata
func RespondWithStandardResponse(w http.ResponseWriter, r *http.Request, data interface{}, pagination *PaginationMeta) {
	response := StandardResponse{
		Data: data,
		Meta: ResponseMetadata{
			Endpoint: r.URL.Path,
			Method:   r.Method,
			Version:  "1.0.0",
			Pagination: pagination,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
