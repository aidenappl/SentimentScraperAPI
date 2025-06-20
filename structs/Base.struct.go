package structs

type BaseListRequest struct {
	// Pagination parameters
	Limit  *int `json:"limit"`  // Maximum number of items to return
	Offset *int `json:"offset"` // Number of items to skip before starting to collect the result set
	// Sorting parameters
	SortBy    *string `json:"sort_by"`    // Field to sort by
	SortOrder *string `json:"sort_order"` // Order of sorting, e.g., "asc" or "desc"
	// Search parameters
	SearchQuery *string `json:"search_query"` // Query string for searching
}

const (
	SortOrderAsc     = "asc"
	SortOrderDesc    = "desc"
	DefaultListLimit = 25
	MaximumListLimit = 500
)

type GeneralNSN struct {
	ID        *int    `json:"id"`         // Unique identifier for the status
	Name      *string `json:"name"`       // Name of the status
	ShortName *string `json:"short_name"` // Short name or code for the status
}
