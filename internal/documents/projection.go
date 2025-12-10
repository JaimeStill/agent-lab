package documents

import "github.com/JaimeStill/agent-lab/pkg/query"

var projection = query.NewProjectionMap("public", "documents", "d").
	Project("id", "Id").
	Project("name", "Name").
	Project("filename", "Filename").
	Project("content_type", "ContentType").
	Project("size_bytes", "SizeBytes").
	Project("page_count", "PageCount").
	Project("storage_key", "StorageKey").
	Project("created_at", "CreatedAt").
	Project("updated_at", "UpdatedAt")

var defaultSort = query.SortField{Field: "CreatedAt", Descending: true}
