# Session 4b: classify-docs Types and Detection Stage

## Summary

Implemented the foundational types and first two nodes (init, detect) of the classify-docs workflow for document security marking detection using parallel Vision API analysis.

## What Was Implemented

### Classify Workflow (`workflows/classify/`)

- **errors.go** - Domain errors (ErrDocumentNotFound, ErrNoPages, ErrRenderFailed, ErrParseResponse, ErrDetectionFailed)
- **parse.go** - JSON response parsing with markdown code block fallback and value clamping
- **profile.go** - DefaultProfile with detection system prompt for structured JSON output
- **classify.go** - Types (PageImage, PageDetection, MarkingInfo, FilterSuggestion) and workflow factory with init/detect nodes

### Workflow Nodes

**Init Node:**
- Parses document_id from params
- Loads document metadata via runtime.Documents()
- Validates document has pages
- Renders all pages to images via runtime.Images()
- Stores lightweight PageImage array (ImageID only, no data URIs) in state

**Detect Node:**
- Uses ProcessParallel for concurrent page analysis
- Fetches image data on-demand for each page
- Builds base64 data URI and calls Vision API
- Parses JSON response with fallback for markdown code blocks
- Stores PageDetection array in state

### Security Enhancement

Added secure token handling to workflow execution:
- `ExecuteRequest.Token` field (not persisted to database)
- Token injected into workflow state at runtime
- Token NOT stored in `workflow_runs.params`
- Token NOT returned in API responses
- OpenAPI spec updated to document token field

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Lightweight PageImage | Avoid massive payloads - store only ImageID, fetch data on-demand |
| JSON output from LLM | Reliable structured parsing vs natural language extraction |
| Markdown fallback parsing | LLMs often wrap JSON in code blocks |
| Value clamping | Defensive handling of out-of-range LLM outputs |
| Token separation | Security - JWT tokens should not be persisted |
| WorkerCap=4 | Conservative for Vision API rate limits |

## Files Changed

### New Files
- `workflows/classify/errors.go`
- `workflows/classify/parse.go`
- `workflows/classify/profile.go`
- `workflows/classify/classify.go`
- `tests/workflows_classify/errors_test.go`
- `tests/workflows_classify/parse_test.go`
- `tests/workflows_classify/classify_test.go`

### Modified Files
- `workflows/init.go` - Added classify import
- `internal/workflows/handler.go` - Added Token field to ExecuteRequest
- `internal/workflows/system.go` - Added token parameter to Execute/ExecuteStream
- `internal/workflows/executor.go` - Token injection into state (not persisted)
- `internal/workflows/openapi.go` - Added token to ExecuteRequest schema
- `tests/internal_workflows/system_test.go` - Updated interface test

## Validation Results

- Workflow registered as "classify-docs"
- Init node loads document and renders images
- Detect node processes pages in parallel via ProcessParallel
- JSON parsing handles direct JSON and markdown-wrapped responses
- Value clamping works for out-of-range clarity/confidence scores
- Token not persisted in database
- All tests passing

## Notes for Future Sessions

- Detection accuracy depends heavily on LLM capability (local Gemma 3 4B showed hallucinations)
- Production use with GPT-5-mini on Azure Government IL6 expected to perform significantly better
- Session 4c will add enhance, classify, and score nodes
- Detection prompt may need refinement based on production testing
