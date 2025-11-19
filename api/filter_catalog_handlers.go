package api

import (
	"context"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// ====== Request/Response Types ======

// ListFiltersRequest represents a request to list filters
type ListFiltersRequest struct {
	Category string `query:"category" description:"Optional category filter"`
}

// ListFiltersResponse represents the response for listing filters
type ListFiltersResponse struct {
	Filters []*storage.PiperFilter `json:"filters"`
	Total   int                    `json:"total"`
}

// GetFilterRequest represents a request to get a single filter
type GetFilterRequest struct {
	FilterType string `path:"filterType" required:"true" description:"Filter type identifier"`
}

// GetFilterResponse represents the response for getting a single filter
type GetFilterResponse struct {
	Filter *storage.PiperFilter `json:"filter"`
}

// UpsertFilterRequest represents a request to upsert a filter
type UpsertFilterRequest struct {
	FilterType  string                       `json:"filter_type" required:"true"`
	DisplayName string                       `json:"display_name" required:"true"`
	Category    string                       `json:"category" required:"true"`
	Purpose     string                       `json:"purpose" required:"true"`
	Parameters  []storage.FilterParameter    `json:"parameters"`
	Examples    []storage.FilterExample      `json:"examples"`
	Version     string                       `json:"version,omitempty"`
}

// UpsertFilterResponse represents the response for upserting a filter
type UpsertFilterResponse struct{
	Success bool                  `json:"success"`
	Message string                `json:"message"`
	Filter  *storage.PiperFilter `json:"filter"`
}

// DeleteFilterRequest represents a request to delete a filter
type DeleteFilterRequest struct {
	FilterType string `path:"filterType" required:"true" description:"Filter type identifier"`
}

// DeleteFilterResponse represents the response for deleting a filter
type DeleteFilterResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetFilterCatalogRequest represents a request to get the filter catalog
type GetFilterCatalogRequest struct {
	Format string `query:"format" enum:"ui,ai,full" description:"Catalog format: ui (grouped by category), ai (AI training), or full (default)"`
}

// GetFilterCatalogResponse represents the response for the filter catalog
type GetFilterCatalogResponse struct {
	Data interface{} `json:"data"`
}

// ====== Handler Implementations ======

// ListFilters lists all piper filters, optionally filtered by category
func (api *API) ListFilters() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ListFiltersRequest, output *ListFiltersResponse) error {
		api.Services.IncrementAPIRequests()

		var filters []*storage.PiperFilter
		var err error

		if input.Category != "" {
			filters, err = api.Services.Storage.ListPiperFiltersByCategory(ctx, input.Category)
			if err != nil {
				log.Errorf("Failed to list filters by category %s: %v", input.Category, err)
				return usecaseStatus.Wrap(fmt.Errorf("failed to list filters: %w", err), usecaseStatus.Internal)
			}
		} else {
			filters, err = api.Services.Storage.ListPiperFilters(ctx)
			if err != nil {
				log.Errorf("Failed to list all filters: %v", err)
				return usecaseStatus.Wrap(fmt.Errorf("failed to list filters: %w", err), usecaseStatus.Internal)
			}
		}

		output.Filters = filters
		output.Total = len(filters)

		log.Infof("Listed %d piper filters (category: %s)", len(filters), input.Category)
		return nil
	})

	u.SetTitle("List Piper Filters")
	u.SetDescription("Retrieves all available piper transformation filters, optionally filtered by category")
	u.SetTags("filters", "piper")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}

// GetFilter retrieves a specific piper filter by type
func (api *API) GetFilter() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input GetFilterRequest, output *GetFilterResponse) error {
		api.Services.IncrementAPIRequests()

		filter, err := api.Services.Storage.GetPiperFilter(ctx, input.FilterType)
		if err != nil {
			log.Errorf("Failed to get filter %s: %v", input.FilterType, err)
			return usecaseStatus.Wrap(fmt.Errorf("filter not found"), usecaseStatus.NotFound)
		}

		output.Filter = filter

		log.Infof("Retrieved piper filter: %s", input.FilterType)
		return nil
	})

	u.SetTitle("Get Piper Filter")
	u.SetDescription("Retrieves details for a specific piper transformation filter")
	u.SetTags("filters", "piper")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.Internal)

	return u
}

// UpsertFilter creates or updates a piper filter
func (api *API) UpsertFilter() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input UpsertFilterRequest, output *UpsertFilterResponse) error {
		api.Services.IncrementAPIRequests()

		// Validate input
		if input.FilterType == "" {
			return usecaseStatus.Wrap(fmt.Errorf("filter_type is required"), usecaseStatus.InvalidArgument)
		}
		if input.DisplayName == "" {
			return usecaseStatus.Wrap(fmt.Errorf("display_name is required"), usecaseStatus.InvalidArgument)
		}
		if input.Category == "" {
			return usecaseStatus.Wrap(fmt.Errorf("category is required"), usecaseStatus.InvalidArgument)
		}
		if input.Purpose == "" {
			return usecaseStatus.Wrap(fmt.Errorf("purpose is required"), usecaseStatus.InvalidArgument)
		}

		filter := &storage.PiperFilter{
			FilterType:  input.FilterType,
			DisplayName: input.DisplayName,
			Category:    input.Category,
			Purpose:     input.Purpose,
			Parameters:  input.Parameters,
			Examples:    input.Examples,
			Version:     input.Version,
		}

		err := api.Services.Storage.UpsertPiperFilter(ctx, filter)
		if err != nil {
			log.Errorf("Failed to upsert filter %s: %v", input.FilterType, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to upsert filter: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Filter %s upserted successfully", input.FilterType)
		output.Filter = filter

		log.Infof("Upserted piper filter: %s", input.FilterType)
		return nil
	})

	u.SetTitle("Upsert Piper Filter")
	u.SetDescription("Creates or updates a piper transformation filter in the catalog")
	u.SetTags("filters", "piper", "admin")
	u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.Internal)

	return u
}

// DeleteFilter deletes a piper filter
func (api *API) DeleteFilter() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input DeleteFilterRequest, output *DeleteFilterResponse) error {
		api.Services.IncrementAPIRequests()

		err := api.Services.Storage.DeletePiperFilter(ctx, input.FilterType)
		if err != nil {
			log.Errorf("Failed to delete filter %s: %v", input.FilterType, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to delete filter: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Filter %s deleted successfully", input.FilterType)

		log.Infof("Deleted piper filter: %s", input.FilterType)
		return nil
	})

	u.SetTitle("Delete Piper Filter")
	u.SetDescription("Deletes a piper transformation filter from the catalog")
	u.SetTags("filters", "piper", "admin")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.Internal)

	return u
}

// GetFilterCatalog retrieves the complete filter catalog in various formats
func (api *API) GetFilterCatalog() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input GetFilterCatalogRequest, output *GetFilterCatalogResponse) error {
		api.Services.IncrementAPIRequests()

		catalog, err := api.Services.Storage.GetPiperFilterCatalog(ctx, input.Format)
		if err != nil {
			log.Errorf("Failed to get filter catalog (format: %s): %v", input.Format, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to get filter catalog: %w", err), usecaseStatus.Internal)
		}

		output.Data = catalog

		log.Infof("Retrieved filter catalog (format: %s)", input.Format)
		return nil
	})

	u.SetTitle("Get Piper Filter Catalog")
	u.SetDescription("Retrieves the complete filter catalog optimized for different consumers (UI, AI, or full)")
	u.SetTags("filters", "piper", "catalog")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}
