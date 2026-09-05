package exports

// SchemaResource describes one exportable resource from GET /exports/schema.
type SchemaResource struct {
	Name    string         `json:"name"`
	Formats []string       `json:"formats"`
	Filters []SchemaFilter `json:"filters"`
}

// SchemaFilter describes one accepted filter key/type.
type SchemaFilter struct {
	Key           string   `json:"key"`
	Type          string   `json:"type"`
	Optional      bool     `json:"optional"`
	AllowedValues []string `json:"allowed_values,omitempty"`
}

// SchemaResponse is the top-level response from GET /exports/schema.json.
type SchemaResponse struct {
	Resources []SchemaResource `json:"resources"`
}

// CreateRequest is the body for POST /exports.json.
type CreateRequest struct {
	Resource   string         `json:"resource"`
	Format     string         `json:"format"`
	Filters    map[string]any `json:"filters,omitempty"`
	Inline     bool           `json:"inline,omitempty"`
	ForceAsync bool           `json:"force_async,omitempty"`
}

// CreateResponse is the body returned by POST /exports.json.
type CreateResponse struct {
	Status      string `json:"status"`
	ExportLogID int    `json:"export_log_id"`
	RowCount    int    `json:"row_count"`
	DownloadURL string `json:"download_url"`
	Format      string `json:"format"`
	Data        string `json:"data,omitempty"`
}

// DownloadPendingResponse is returned by GET /exports/:id/download when 202.
type DownloadPendingResponse struct {
	Status     string `json:"status"`
	RetryAfter int    `json:"retry_after"`
}

// DownloadFailedResponse is returned by GET /exports/:id/download when 410.
type DownloadFailedResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
