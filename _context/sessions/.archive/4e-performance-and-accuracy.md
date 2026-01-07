# Session 4e: Performance and Accuracy

## Problem Context

Session 4d identified a key refinement needed for classification accuracy:

> Detected markings should contribute to classification regardless of fading/confidence. Fading/confidence should affect ACCEPT/REVIEW/REJECT recommendation, not classification itself.

Additionally, the current runtime (~50s for single-page documents) needs optimization to meet the <30s target.

## Architecture Approach

### Terminology Refinement

The current `confidence` metric conflates two concepts:
- **Detection reliability**: Can the vision model read the marking?
- **Visual quality**: Is the marking faded/washed out?

**Solution**: Rename `confidence` to `legibility` with clear semantics:
- **Legibility**: Can the text be read? (1.0 = perfectly readable, 0.0 = illegible)
- **Faded**: Visual appearance (true = washed out, but may still be legible)

A faded marking that is still readable should have high legibility. Enhancement should only trigger when markings are truly illegible (legibility < 0.4).

### Classification Logic

Detected caveats (NOFORN, ORCON, REL TO, etc.) should be included in the primary classification, not treated as alternatives. For example, if SECRET and NOFORN are both detected, the classification should be SECRET//NOFORN.

### Enhancement Decision Logic

Current logic uses `NeedsEnhancement()` method on `PageDetection`. This will be:
1. Removed from the struct
2. Inlined in `detectNode` with updated terminology

The decision remains: `legibility < 0.4 AND filter_suggestion != nil`

### Prompt Engineering

- **DetectionSystemPrompt**: Use legibility terminology, clarify faded ≠ illegible, threshold 0.4
- **ClassificationSystemPrompt**: Include caveats in primary classification, all markings contribute
- **ScoringSystemPrompt**: Rename `detection_confidence` → `detection_legibility`

## Implementation Steps

### Phase 1: Runtime Profiling Infrastructure

Add timing instrumentation to identify bottlenecks before optimization.

#### Step 1.1: Add Timing Helper

**File**: `workflows/classify/classify.go`

Add timing helper function after imports:

```go
func logNodeTiming(logger *slog.Logger, nodeName string, start time.Time) {
	logger.Info("node completed", "node", nodeName, "duration", time.Since(start).String())
}
```

Update import to include `"log/slog"` and `"time"`.

#### Step 1.2: Instrument Nodes

Update each node function to log timing. Example for `initNode`:

```go
func initNode(runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		start := time.Now()
		defer logNodeTiming(runtime.Logger(), "init", start)

		// ... existing logic ...
	})
}
```

Apply same pattern to: `detectNode`, `enhanceNode`, `classifyNode`, `scoreNode`.

### Phase 2: Terminology Refactoring

#### Step 2.1: Update Constants and Types

**File**: `workflows/classify/classify.go`

Rename constant:
```go
const DefaultLegibilityThreshold = 0.4
```

Update `MarkingInfo`:
```go
type MarkingInfo struct {
	Text       string  `json:"text"`
	Location   string  `json:"location"`
	Legibility float64 `json:"legibility"`
	Faded      bool    `json:"faded"`
}
```

Update `EnhanceOptions`:
```go
type EnhanceOptions struct {
	LegibilityThreshold float64 `json:"legibility_threshold"`
}

func DefaultEnhanceOptions() EnhanceOptions {
	return EnhanceOptions{LegibilityThreshold: DefaultLegibilityThreshold}
}
```

#### Step 2.2: Remove NeedsEnhancement Method

**File**: `workflows/classify/classify.go`

Delete this method entirely:
```go
// DELETE THIS:
func (d PageDetection) NeedsEnhancement(threshold float64) bool {
	return d.ClarityScore < threshold && d.FilterSuggestion != nil
}
```

#### Step 2.3: Update detectNode

**File**: `workflows/classify/classify.go`

Replace the enhancement decision logic (lines 275-281) with inline logic checking marking legibility:

```go
needsEnhancement := false
for _, d := range result.Results {
	if d.FilterSuggestion == nil {
		continue
	}
	for _, m := range d.MarkingsFound {
		if m.Legibility < enhanceOpts.LegibilityThreshold {
			needsEnhancement = true
			break
		}
	}
	if needsEnhancement {
		break
	}
}
```

#### Step 2.4: Update enhanceNode

**File**: `workflows/classify/classify.go`

Update the page selection logic (lines 320-325):

```go
var pagesToEnhance []PageDetection
for _, d := range detectList {
	if d.FilterSuggestion == nil {
		continue
	}
	for _, m := range d.MarkingsFound {
		if m.Legibility < enhanceOpts.LegibilityThreshold {
			pagesToEnhance = append(pagesToEnhance, d)
			break
		}
	}
}
```

#### Step 2.5: Update mergeDetections

**File**: `workflows/classify/classify.go`

Update the merge logic to use `Legibility`:

```go
for _, em := range enhanced.MarkingsFound {
	key := em.Text + "|" + em.Location
	if om, exists := markingMap[key]; exists {
		if (om.Faded || om.Legibility < threshold) && em.Legibility > om.Legibility {
			markingMap[key] = em
		}
	} else {
		markingMap[key] = em
	}
}
```

#### Step 2.6: Update buildClassificationPrompt

**File**: `workflows/classify/classify.go`

Update the marking output format:

```go
fmt.Fprintf(
	&sb,
	"  - %s [%s] (legibility: %.2f, faded: %v)\n",
	m.Text, m.Location, m.Legibility, m.Faded,
)
```

#### Step 2.7: Update buildScoringPrompt

**File**: `workflows/classify/classify.go`

Rename variable and output:

```go
var totalClarity, totalLegibility float64
// ...
for _, m := range d.MarkingsFound {
	totalLegibility += m.Legibility
	// ...
}
// ...
avgLegibility := 0.0
if markingCount > 0 {
	avgLegibility = totalLegibility / float64(markingCount)
}
// ...
fmt.Fprintf(&sb, "  Average Legibility: %.2f\n", avgLegibility)
```

#### Step 2.8: Update parse.go

**File**: `workflows/classify/parse.go`

Update `validateDetection`:

```go
func validateDetection(d PageDetection) PageDetection {
	d.ClarityScore = clamp(d.ClarityScore, 0.0, 1.0)

	for i := range d.MarkingsFound {
		d.MarkingsFound[i].Legibility = clamp(d.MarkingsFound[i].Legibility, 0.0, 1.0)
	}

	return d
}
```

### Phase 3: Prompt Engineering

#### Step 3.1: Update DetectionSystemPrompt

**File**: `workflows/classify/profile.go`

```go
const DetectionSystemPrompt = `You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"page_number": <integer>,
	"markings_found": [
		{
			"text": "<exact marking text>",
			"location": "<header|footer|margin|body>",
			"legibility": <0.0-1.0>,
			"faded": <boolean>
		}
	],
	"clarity_score": <0.0-1.0>,
	"filter_suggestion": {
		"brightness": <optional integer 0-200>,
		"contrast": <optional integer -100 to 100>,
		"saturation": <optional integer 0-200>
	} or null
}

INSTRUCTIONS:
- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON, or any other code names)
- Note the location of each marking (header, footer, margin, or body)
- LEGIBILITY measures readability: 1.0 = text is perfectly readable, 0.0 = text is illegible
- FADED indicates visual appearance: true if marking appears washed out or pale
- IMPORTANT: A faded marking can still have high legibility if the text is readable
- clarity_score reflects overall page quality for marking detection
- Only suggest filter_suggestion if legibility < 0.4 AND you believe enhancement would improve readability
- JSON response only; no preamble or dialog`
```

#### Step 3.2: Update ClassificationSystemPrompt

**File**: `workflows/classify/profile.go`

**Key changes**:
- Token constraints for brevity
- Include caveats in primary classification
- Only use alternatives for genuine ambiguity

```go
const ClassificationSystemPrompt = `You are a document classification specialist. Analyze security marking detections across all pages to determine the overall document classification.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"classification": "<overall classification level>",
	"alternative_readings": [
		{
			"classification": "<alternative classification>",
			"probability": <0.0-1.0>,
			"reason": "<brief phrase, max 15 words>"
		}
	],
	"marking_summary": ["<list of unique markings found>"],
	"rationale": "<1-2 sentences, max 40 words>"
}

INSTRUCTIONS:
- Analyze all marking detections provided
- Determine the HIGHEST classification level present
- IMPORTANT: ALL detected markings contribute to classification regardless of legibility or fading
- A faded or low-legibility marking is still a valid marking - include it in your classification decision
- Include detected caveats (NOFORN, ORCON, REL TO, etc.) in the primary classification (e.g., SECRET//NOFORN)
- Legibility and fading affect confidence scoring, NOT the classification itself
- Only list alternative readings if there is genuine ambiguity about what marking text says
- marking_summary should list unique marking texts (deduplicated)
- Keep rationale brief: 1-2 sentences explaining the key deciding factor
- JSON response only; no preamble or dialog`
```

#### Step 3.3: Update ScoringSystemPrompt

**File**: `workflows/classify/profile.go`

**Key change**: Added token constraint for factor descriptions - limited to brief phrases (max 10 words).

```go
const ScoringSystemPrompt = `You are a confidence scoring specialist. Evaluate the quality and reliability of document classification results.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"overall_score": <0.0-1.0>,
	"factors": [
		{
			"name": "<factor name>",
			"score": <0.0-1.0>,
			"weight": <weight>,
			"description": "<brief phrase, max 10 words>"
		}
	],
	"recommendation": "<ACCEPT|REVIEW|REJECT>"
}

FACTORS TO EVALUATE:
- marking_clarity (weight: 0.30): Average clarity across pages
- marking_consistency (weight: 0.25): Marking agreement across pages
- spatial_coverage (weight: 0.15): Markings in expected locations (header/footer)
- enhancement_impact (weight: 0.10): Value added by enhancement (if applied)
- alternative_count (weight: 0.10): Fewer alternatives = higher confidence
- detection_legibility (weight: 0.10): Average marking legibility (low legibility reduces confidence)

THRESHOLDS:
- >= 0.90: ACCEPT - Classification is reliable
- 0.70-0.89: REVIEW - Human verification recommended
- < 0.70: REJECT - Insufficient confidence

Keep factor descriptions brief (max 10 words each). JSON response only; no preamble or dialog`
```

### Phase 4: Update Seed Data

#### Step 4.1: Update classify_profiles.json

**File**: `cmd/seed/seeds/classify_profiles.json`

Update the default profile with:
- `legibility` instead of `confidence` in detect/enhance prompts
- Token constraints in classify/score prompts
- `legibility_threshold: 0.4` in options
- Caveats included in primary classification

```json
{
  "profiles": [
    {
      "workflow_name": "classify-docs",
      "name": "default",
      "description": "Default classify workflow profile with standard legibility threshold (0.4)",
      "stages": [
        {
          "stage_name": "init",
          "system_prompt": ""
        },
        {
          "stage_name": "detect",
          "system_prompt": "You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"page_number\": <integer>,\n\t\"markings_found\": [\n\t\t{\n\t\t\t\"text\": \"<exact marking text>\",\n\t\t\t\"location\": \"<header|footer|margin|body>\",\n\t\t\t\"legibility\": <0.0-1.0>,\n\t\t\t\"faded\": <boolean>\n\t\t}\n\t],\n\t\"clarity_score\": <0.0-1.0>,\n\t\"filter_suggestion\": {\n\t\t\"brightness\": <optional integer 0-200>,\n\t\t\"contrast\": <optional integer -100 to 100>,\n\t\t\"saturation\": <optional integer 0-200>\n\t} or null\n}\n\nINSTRUCTIONS:\n- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON, or any other code names)\n- Note the location of each marking (header, footer, margin, or body)\n- LEGIBILITY measures readability: 1.0 = text is perfectly readable, 0.0 = text is illegible\n- FADED indicates visual appearance: true if marking appears washed out or pale\n- IMPORTANT: A faded marking can still have high legibility if the text is readable\n- clarity_score reflects overall page quality for marking detection\n- Only suggest filter_suggestion if legibility < 0.4 AND you believe enhancement would improve readability\n- JSON response only; no preamble or dialog"
        },
        {
          "stage_name": "enhance",
          "system_prompt": "You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"page_number\": <integer>,\n\t\"markings_found\": [\n\t\t{\n\t\t\t\"text\": \"<exact marking text>\",\n\t\t\t\"location\": \"<header|footer|margin|body>\",\n\t\t\t\"legibility\": <0.0-1.0>,\n\t\t\t\"faded\": <boolean>\n\t\t}\n\t],\n\t\"clarity_score\": <0.0-1.0>,\n\t\"filter_suggestion\": {\n\t\t\"brightness\": <optional integer 0-200>,\n\t\t\"contrast\": <optional integer -100 to 100>,\n\t\t\"saturation\": <optional integer 0-200>\n\t} or null\n}\n\nINSTRUCTIONS:\n- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON, or any other code names)\n- Note the location of each marking (header, footer, margin, or body)\n- LEGIBILITY measures readability: 1.0 = text is perfectly readable, 0.0 = text is illegible\n- FADED indicates visual appearance: true if marking appears washed out or pale\n- IMPORTANT: A faded marking can still have high legibility if the text is readable\n- clarity_score reflects overall page quality for marking detection\n- Only suggest filter_suggestion if legibility < 0.4 AND you believe enhancement would improve readability\n- JSON response only; no preamble or dialog",
          "options": {"legibility_threshold": 0.4}
        },
        {
          "stage_name": "classify",
          "system_prompt": "You are a document classification specialist. Analyze security marking detections across all pages to determine the overall document classification.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"classification\": \"<overall classification level>\",\n\t\"alternative_readings\": [\n\t\t{\n\t\t\t\"classification\": \"<alternative classification>\",\n\t\t\t\"probability\": <0.0-1.0>,\n\t\t\t\"reason\": \"<brief phrase, max 15 words>\"\n\t\t}\n\t],\n\t\"marking_summary\": [\"<list of unique markings found>\"],\n\t\"rationale\": \"<1-2 sentences, max 40 words>\"\n}\n\nINSTRUCTIONS:\n- Analyze all marking detections provided\n- Determine the HIGHEST classification level present\n- IMPORTANT: ALL detected markings contribute to classification regardless of legibility or fading\n- A faded or low-legibility marking is still a valid marking - include it in your classification decision\n- Include detected caveats (NOFORN, ORCON, REL TO, etc.) in the primary classification (e.g., SECRET//NOFORN)\n- Legibility and fading affect confidence scoring, NOT the classification itself\n- Only list alternative readings if there is genuine ambiguity about what marking text says\n- marking_summary should list unique marking texts (deduplicated)\n- Keep rationale brief: 1-2 sentences explaining the key deciding factor\n- JSON response only; no preamble or dialog"
        },
        {
          "stage_name": "score",
          "system_prompt": "You are a confidence scoring specialist. Evaluate the quality and reliability of document classification results.\n\nOUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:\n{\n\t\"overall_score\": <0.0-1.0>,\n\t\"factors\": [\n\t\t{\n\t\t\t\"name\": \"<factor name>\",\n\t\t\t\"score\": <0.0-1.0>,\n\t\t\t\"weight\": <weight>,\n\t\t\t\"description\": \"<brief phrase, max 10 words>\"\n\t\t}\n\t],\n\t\"recommendation\": \"<ACCEPT|REVIEW|REJECT>\"\n}\n\nFACTORS TO EVALUATE:\n- marking_clarity (weight: 0.30): Average clarity across pages\n- marking_consistency (weight: 0.25): Marking agreement across pages\n- spatial_coverage (weight: 0.15): Markings in expected locations (header/footer)\n- enhancement_impact (weight: 0.10): Value added by enhancement (if applied)\n- alternative_count (weight: 0.10): Fewer alternatives = higher confidence\n- detection_legibility (weight: 0.10): Average marking legibility (low legibility reduces confidence)\n\nTHRESHOLDS:\n- >= 0.90: ACCEPT - Classification is reliable\n- 0.70-0.89: REVIEW - Human verification recommended\n- < 0.70: REJECT - Insufficient confidence\n\nKeep factor descriptions brief (max 10 words each). JSON response only; no preamble or dialog"
        }
      ]
    },
    {
      "workflow_name": "classify-docs",
      "name": "aggressive-enhancement",
      "description": "Lower legibility threshold (0.3) for more aggressive enhancement triggering",
      "stages": [
        {
          "stage_name": "enhance",
          "options": {"legibility_threshold": 0.3}
        }
      ]
    }
  ]
}
```

### Phase 5: Testing & Validation

#### Step 5.1: Run Profiling Tests

Execute workflow against:
1. Single-page test PDF
2. 20-page test PDF

Observe timing logs to identify bottlenecks.

#### Step 5.2: Analyze Results

Document timing breakdown:
- init (rendering)
- detect (parallel vision calls)
- enhance (if triggered)
- classify (single LLM call)
- score (single LLM call)

#### Step 5.3: Multi-page Validation

Verify:
- Parallel detection works correctly across 20 pages
- Enhancement only triggers for pages with illegible markings
- Classification synthesizes all page detections correctly
- Scoring reflects aggregate metrics

## Test Updates Required

The AI will update tests during validation phase:

1. **classify_test.go**:
   - Remove `TestPageDetection_NeedsEnhancement` (method removed)
   - Update `TestMarkingInfo_Fields` to use `Legibility` instead of `Confidence`

2. **parse_test.go**:
   - Update all JSON test data to use `"legibility"` instead of `"confidence"`
   - Update test assertions to check `Legibility` field

### Phase 6: Multi-Page PDF Optimization

Testing with a 27-page PDF revealed critical bottlenecks:

| Stage | Duration | Issue |
|-------|----------|-------|
| init | 1m28s | Sequential PDF rendering |
| detect | 1m13s | Worker cap limited to 4 |
| Total | 3m19s | Exceeded 3m WriteTimeout |

#### Step 6.1: Increase Server WriteTimeout

**File**: `internal/config/server.go`

Change default from 3m to 15m to accommodate large documents:

```go
if c.WriteTimeout == "" {
	c.WriteTimeout = "15m"
}
```

**Rationale**: 15m accommodates 100+ page documents with buffer for LLM latency variance. Users can override via `SERVER_WRITE_TIMEOUT` env var.

#### Step 6.2: Parallel PDF Rendering

**File**: `internal/images/repository.go`

Add imports for parallel processing:

```go
import (
	// ... existing imports ...
	"runtime"
	"sync"
)
```

Add parallel rendering types and helper:

```go
type renderTask struct {
	pageNum int
	result  *Image
	err     error
}

func renderWorkerCount(pageCount int) int {
	workers := runtime.NumCPU()
	if workers > pageCount {
		workers = pageCount
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}
```

Replace sequential rendering in `Render` method:

```go
func (r *repo) Render(ctx context.Context, documentID uuid.UUID, opts RenderOptions) ([]Image, error) {
	doc, err := r.documents.Find(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if !document.IsSupported(doc.ContentType) {
		return nil, ErrUnsupportedFormat
	}

	if doc.PageCount == nil || *doc.PageCount < 1 {
		return nil, fmt.Errorf("%w: document has no pages to render", ErrRenderFailed)
	}

	pageExpr := opts.Pages
	if pageExpr == "" {
		pageExpr = fmt.Sprintf("1-%d", *doc.PageCount)
	}

	pages, err := ParsePageRange(pageExpr, *doc.PageCount)
	if err != nil {
		return nil, err
	}

	docPath, err := r.storage.Path(ctx, doc.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRenderFailed, err)
	}

	workerCount := renderWorkerCount(len(pages))
	tasks := make(chan int, len(pages))
	results := make(chan renderTask, len(pages))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.renderWorker(ctx, documentID, docPath, doc.ContentType, opts, tasks, results)
		}()
	}

	for _, pageNum := range pages {
		tasks <- pageNum
	}
	close(tasks)

	go func() {
		wg.Wait()
		close(results)
	}()

	resultMap := make(map[int]*Image)
	for task := range results {
		if task.err != nil {
			return nil, task.err
		}
		resultMap[task.pageNum] = task.result
	}

	images := make([]Image, 0, len(pages))
	for _, pageNum := range pages {
		if img, ok := resultMap[pageNum]; ok {
			images = append(images, *img)
		}
	}

	return images, nil
}
```

Add worker function:

```go
func (r *repo) renderWorker(
	ctx context.Context,
	documentID uuid.UUID,
	docPath string,
	contentType string,
	opts RenderOptions,
	tasks <-chan int,
	results chan<- renderTask,
) {
	openDoc, err := document.Open(docPath, contentType)
	if err != nil {
		for pageNum := range tasks {
			results <- renderTask{pageNum: pageNum, err: fmt.Errorf("%w: %v", ErrRenderFailed, err)}
		}
		return
	}
	defer openDoc.Close()

	renderer, err := image.NewImageMagickRenderer(opts.ToImageConfig())
	if err != nil {
		for pageNum := range tasks {
			results <- renderTask{pageNum: pageNum, err: fmt.Errorf("%w: %v", ErrRenderFailed, err)}
		}
		return
	}

	for pageNum := range tasks {
		select {
		case <-ctx.Done():
			results <- renderTask{pageNum: pageNum, err: ctx.Err()}
			return
		default:
		}

		img, err := r.renderPage(ctx, documentID, openDoc, renderer, pageNum, opts)
		results <- renderTask{pageNum: pageNum, result: img, err: err}
	}
}
```

#### Step 6.3: Dynamic Worker Detection for LLM Calls

**File**: `workflows/classify/classify.go`

Replace hardcoded parallel config with library defaults:

```go
func detectionParallelConfig() config.ParallelConfig {
	cfg := config.DefaultParallelConfig()
	cfg.Observer = "noop"
	return cfg
}
```

This uses `min(NumCPU*2, 16, itemCount)` automatically instead of hardcoded `WorkerCap: 4`.

#### Step 6.4: Consolidate to Single Streaming Execute with Lifecycle Context

Given the complexity of workflow execution (10+ minutes possible), synchronous execution without progress reporting doesn't make sense. Consolidate to a single streaming-based Execute that:
- Survives HTTP disconnection (client can reconnect, poll status)
- Respects server shutdown (derives from lifecycle context)
- Supports explicit cancellation via `/cancel` endpoint

**File**: `internal/workflows/runtime.go`

Add lifecycle coordinator to the workflow Runtime so executor can access `e.runtime.Lifecycle().Context()`:

```go
import (
	"log/slog"

	"github.com/JaimeStill/agent-lab/internal/agents"
	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/lifecycle"
	"github.com/JaimeStill/agent-lab/internal/profiles"
)

type Runtime struct {
	agents    agents.System
	documents documents.System
	images    images.System
	profiles  profiles.System
	lifecycle *lifecycle.Coordinator
	logger    *slog.Logger
}

func NewRuntime(
	agents agents.System,
	documents documents.System,
	images images.System,
	profiles profiles.System,
	lifecycle *lifecycle.Coordinator,
	logger *slog.Logger,
) *Runtime {
	return &Runtime{
		agents:    agents,
		documents: documents,
		images:    images,
		profiles:  profiles,
		lifecycle: lifecycle,
		logger:    logger,
	}
}

func (r *Runtime) Lifecycle() *lifecycle.Coordinator { return r.lifecycle }
```

**File**: `cmd/server/domain.go`

Update `NewRuntime` call to include lifecycle:

```go
workflowRuntime := workflows.NewRuntime(
	agentsSys,
	documentsSys,
	imagesSys,
	profilesSys,
	runtime.Lifecycle,
	runtime.Logger,
)
```

**File**: `internal/workflows/system.go`

Update System interface - remove synchronous Execute, rename ExecuteStream to Execute, remove ctx from Execute signature:

```go
type System interface {
	ListWorkflows() []WorkflowInfo
	Execute(name string, params map[string]any, token string) (<-chan ExecutionEvent, *Run, error)
	ListRuns(ctx context.Context, page pagination.PageRequest, filters RunFilters) (*pagination.PageResult[Run], error)
	FindRun(ctx context.Context, id uuid.UUID) (*Run, error)
	GetStages(ctx context.Context, runID uuid.UUID) ([]Stage, error)
	GetDecisions(ctx context.Context, runID uuid.UUID) ([]Decision, error)
	DeleteRun(ctx context.Context, id uuid.UUID) error
	Cancel(ctx context.Context, runID uuid.UUID) error
	Resume(ctx context.Context, runID uuid.UUID) (*Run, error)
}
```

Note: `Execute` no longer receives `ctx` - it uses `e.runtime.Lifecycle().Context()` internally. Other methods retain `ctx context.Context` in their signatures, but handlers pass the lifecycle context obtained from calling `Execute`, not the HTTP request context.

**File**: `internal/workflows/executor.go`

Remove synchronous `Execute` method entirely.

Rename `ExecuteStream` to `Execute` and `executeStreamAsync` to `executeAsync`.

Update `Execute` to use lifecycle context instead of receiving HTTP context:

```go
func (e *executor) Execute(name string, params map[string]any, token string) (<-chan ExecutionEvent, *Run, error) {
	factory, exists := Get(name)
	if !exists {
		return nil, nil, ErrWorkflowNotFound
	}

	ctx := e.runtime.Lifecycle().Context()

	run, err := e.repo.CreateRun(ctx, name, params)
	if err != nil {
		return nil, nil, fmt.Errorf("create run: %w", err)
	}

	streamingObs := NewStreamingObserver(defaultStreamBufferSize)

	go e.executeAsync(ctx, run.ID, factory, params, token, streamingObs)

	return streamingObs.Events(), run, nil
}

func (e *executor) executeAsync(ctx context.Context, runID uuid.UUID, factory WorkflowFactory, params map[string]any, token string, streamingObs *StreamingObserver) {
	defer streamingObs.Close()

	// Wrap lifecycle context to support explicit cancellation via Cancel endpoint
	execCtx, cancel := context.WithCancel(ctx)
	e.trackRun(runID, cancel)
	defer e.untrackRun(runID)

	_, err := e.repo.UpdateRunStarted(execCtx, runID)
	if err != nil {
		streamingObs.SendError(err, "")
		e.finalizeRun(execCtx, runID, StatusFailed, nil, err)
		return
	}

	postgresObs := NewPostgresObserver(e.db, runID, e.logger)
	multiObs := observability.NewMultiObserver(postgresObs, streamingObs)
	checkpointStore := NewPostgresCheckpointStore(e.db, e.logger)

	cfg := workflowGraphConfig(runID.String())

	graph, err := state.NewGraphWithDeps(cfg, multiObs, checkpointStore)
	if err != nil {
		streamingObs.SendError(err, "")
		e.finalizeRun(execCtx, runID, StatusFailed, nil, err)
		return
	}

	initialState, err := factory(execCtx, graph, e.runtime, params)
	if err != nil {
		streamingObs.SendError(err, "")
		e.finalizeRun(execCtx, runID, StatusFailed, nil, err)
		return
	}

	initialState.RunID = runID.String()
	if token != "" {
		initialState = initialState.SetSecret("token", token)
	}

	finalState, err := graph.Execute(execCtx, initialState)
	if err != nil {
		if execCtx.Err() != nil {
			errMsg := "execution cancelled"
			streamingObs.SendError(fmt.Errorf("%s", errMsg), "")
			e.repo.UpdateRunCompleted(execCtx, runID, StatusCancelled, nil, &errMsg)
			return
		}
		streamingObs.SendError(err, "")
		errMsg := err.Error()
		e.repo.UpdateRunCompleted(execCtx, runID, StatusFailed, nil, &errMsg)
		return
	}

	streamingObs.SendComplete(finalState.Data)
	e.repo.UpdateRunCompleted(execCtx, runID, StatusCompleted, finalState.Data, nil)
}
```

**File**: `internal/workflows/handler.go`

Remove synchronous `Execute` handler. Rename `ExecuteStream` to `Execute`:

```go
func (h *Handler) Routes() routes.Group {
	return routes.Group{
		Prefix:      "/api/workflows",
		Tags:        []string{"Workflows"},
		Description: "Workflow execution and management",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: h.ListWorkflows, OpenAPI: Spec.ListWorkflows},
			{Method: "POST", Pattern: "/{name}/execute", Handler: h.Execute, OpenAPI: Spec.Execute},
		},
		// ... children unchanged
	}
}

func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handlers.RespondError(w, h.logger, http.StatusBadRequest, err)
		return
	}

	events, run, err := h.sys.Execute(r.Context(), name, req.Params, req.Token)
	if err != nil {
		handlers.RespondError(w, h.logger, MapHTTPStatus(err), err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Run-ID", run.ID.String())
	w.WriteHeader(http.StatusOK)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	for event := range events {
		select {
		case <-r.Context().Done():
			return  // Client disconnected, workflow continues via lifecycle context
		default:
		}

		data, err := json.Marshal(event)
		if err != nil {
			h.logger.Error("failed to marshal event", "error", err)
			continue
		}

		fmt.Fprintf(w, "event: %s\n", event.Type)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}
```

**File**: `internal/workflows/openapi.go`

Remove `ExecuteStream` operation. Update `Execute` operation to document SSE response.

**Behavior Summary**:

| Event | Workflow Behavior |
|-------|-------------------|
| HTTP client disconnects | Continues running (lifecycle context) |
| `/cancel` endpoint called | Cancels via tracked cancel func |
| Server shutdown initiated | Lifecycle context cancelled → graceful cancel |
| Workflow completes | Normal completion, status updated |

**Pattern established**: Any HTTP-initiated long-running process should derive its context from `runtime.Lifecycle.Context()` to ensure graceful shutdown support while remaining independent of HTTP connection lifecycle.

#### Expected Performance After Phase 6

| Stage | Before | After (estimated) |
|-------|--------|-------------------|
| init (render) | 1m28s | ~15-20s |
| detect | 1m13s | ~30-40s |
| enhance | 13s | ~13s |
| classify + score | 40s | ~40s |
| **Total** | **3m19s+** | **~1.5-2m** |

## Validation Checklist

- [ ] `go vet ./...` passes
- [ ] `go test ./tests/...` passes
- [ ] Single-page workflow executes successfully
- [ ] Multi-page workflow executes with parallel detection
- [ ] Node timing logged correctly
- [ ] Enhancement only triggers for illegible markings
- [ ] Classification includes all markings regardless of fading
- [ ] Scoring uses `detection_legibility` factor
- [ ] 27-page PDF completes without timeout
- [ ] Parallel rendering reduces init time
- [ ] Dynamic workers utilize available CPU cores
