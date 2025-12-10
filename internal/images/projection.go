package images

import "github.com/JaimeStill/agent-lab/pkg/query"

// projection maps database columns to Image struct fields for query building.
var projection = query.NewProjectionMap("public", "images", "i").
	Project("id", "ID").
	Project("document_id", "DocumentID").
	Project("page_number", "PageNumber").
	Project("format", "Format").
	Project("dpi", "DPI").
	Project("quality", "Quality").
	Project("brightness", "Brightness").
	Project("contrast", "Contrast").
	Project("saturation", "Saturation").
	Project("rotation", "Rotation").
	Project("background", "Background").
	Project("storage_key", "StorageKey").
	Project("size_bytes", "SizeBytes").
	Project("created_at", "CreatedAt")

// defaultSort orders images by creation time, newest first.
var defaultSort = query.SortField{Field: "CreatedAt", Descending: true}
