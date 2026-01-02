package classify

import "errors"

// Domain errors for the classify workflow.
var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrNoPages          = errors.New("document has no pages")
	ErrRenderFailed     = errors.New("failed to render pages")
	ErrParseResponse    = errors.New("failed to parse detection response")
	ErrDetectionFailed  = errors.New("detection failed")
)
