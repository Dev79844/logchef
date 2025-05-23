package models

import "time"

// APIQueryRequest represents the request payload for the standard log querying endpoint.
type APIQueryRequest struct {
	Limit  int    `json:"limit"`
	RawSQL string `json:"raw_sql"`
	// Sort and other general query params could be added here if needed later.
}

// APIHistogramRequest represents the request payload for the histogram endpoint.
type APIHistogramRequest struct {
	StartTimestamp int64  `json:"start_timestamp,omitempty"` // Kept for histogram, optional otherwise
	EndTimestamp   int64  `json:"end_timestamp,omitempty"`   // Kept for histogram, optional otherwise
	Limit          int    `json:"limit"`                     // Limit might influence histogram sampling/performance
	RawSQL         string `json:"raw_sql"`                   // Contains non-time filters
	Window         string `json:"window,omitempty"`          // For histogram queries: time window size like "1m", "5m", "1h"
	GroupBy        string `json:"group_by,omitempty"`        // For histogram queries: field to group by
	Timezone       string `json:"timezone,omitempty"`        // Kept for histogram, optional otherwise
}

// LogQueryResult represents the result of a log query
type LogQueryResult struct {
	Data    []map[string]interface{} `json:"data"`
	Stats   QueryStats               `json:"stats"`
	Columns []ColumnInfo             `json:"columns"`
}

// LogContextRequest represents a request to get temporal context around a log
type LogContextRequest struct {
	SourceID    SourceID `json:"source_id"`
	Timestamp   int64    `json:"timestamp"`    // Target timestamp in milliseconds
	BeforeLimit int      `json:"before_limit"` // Optional, defaults to 5
	AfterLimit  int      `json:"after_limit"`  // Optional, defaults to 5
}

// LogContextResponse represents temporal context query results
type LogContextResponse struct {
	TargetTimestamp int64                    `json:"target_timestamp"`
	BeforeLogs      []map[string]interface{} `json:"before_logs"`
	TargetLogs      []map[string]interface{} `json:"target_logs"` // Multiple logs might have the same timestamp
	AfterLogs       []map[string]interface{} `json:"after_logs"`
	Stats           QueryStats               `json:"stats"`
}

// SavedQueryTab represents the active tab in the explorer
type SavedQueryTab string

const (
	// SavedQueryTabFilters represents the filters tab
	SavedQueryTabFilters SavedQueryTab = "filters"

	// SavedQueryTabRawSQL represents the raw SQL tab
	SavedQueryTabRawSQL SavedQueryTab = "raw_sql"
)

// SavedQueryTimeRange represents a time range for a saved query
type SavedQueryTimeRange struct {
	Absolute struct {
		Start int64 `json:"start"` // Unix timestamp in milliseconds
		End   int64 `json:"end"`   // Unix timestamp in milliseconds
	} `json:"absolute"`
}

// SavedQueryType represents the type of saved query
type SavedQueryType string

const (
	// SavedQueryTypeLogchefQL represents a query saved in LogchefQL format
	SavedQueryTypeLogchefQL SavedQueryType = "logchefql"

	// SavedQueryTypeSQL represents a query saved in SQL format
	SavedQueryTypeSQL SavedQueryType = "sql"
)

// SavedQueryContent represents the content of a saved query
type SavedQueryContent struct {
	Version   int                 `json:"version"`
	SourceID  SourceID            `json:"sourceId"`
	TimeRange SavedQueryTimeRange `json:"timeRange"`
	Limit     int                 `json:"limit"`
	Content   string              `json:"content"` // Query content (SQL or LogchefQL)
}

// SavedTeamQuery represents a saved query associated with a team
type SavedTeamQuery struct {
	ID           int            `json:"id" db:"id"`
	TeamID       TeamID         `json:"team_id" db:"team_id"`
	SourceID     SourceID       `json:"source_id" db:"source_id"`
	Name         string         `json:"name" db:"name"`
	Description  string         `json:"description" db:"description"`
	QueryType    SavedQueryType `json:"query_type" db:"query_type"`
	QueryContent string         `json:"query_content" db:"query_content"` // JSON string of SavedQueryContent
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// CreateTeamQueryRequest represents a request to create a team query
type CreateTeamQueryRequest struct {
	Name         string         `json:"name" validate:"required"`
	Description  string         `json:"description"`
	SourceID     SourceID       `json:"source_id" validate:"required"`
	QueryType    SavedQueryType `json:"query_type" validate:"required"`
	QueryContent string         `json:"query_content" validate:"required"`
}

// UpdateTeamQueryRequest represents a request to update a team query
type UpdateTeamQueryRequest struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	SourceID     SourceID       `json:"source_id"`
	QueryType    SavedQueryType `json:"query_type"`
	QueryContent string         `json:"query_content"`
}

// SavedQuery represents a generic saved query
type SavedQuery struct {
	ID          int       `json:"id" db:"id"`
	TeamID      string    `json:"team_id" db:"team_id"`
	SourceID    string    `json:"source_id" db:"source_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	QuerySQL    string    `json:"query_sql" db:"query_sql"`
	CreatedBy   UserID    `json:"created_by" db:"created_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
