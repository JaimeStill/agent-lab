# Milestone 4: classify-docs Workflow Integration

## Overview

Implement document classification workflow using go-agents-orchestration with full observability and A/B testing capability. This milestone introduces two foundational components:

1. **Workflow Profiles Infrastructure** - Database-stored configurations for workflow stages
2. **classify-docs Workflow** - Multi-stage document classification using parallel processing

**Key Insight**: Workflow profiles separate agent configuration from workflow code, enabling A/B testing and prompt iteration without code changes.

## Key Decisions

### 1. Workflow Profiles (Not Workflow Configs)

Named "profiles" to avoid redundant "workflowconfigs" naming. Profiles store per-stage agent and prompt configurations.

**Rationale**:
- Clean API: `/api/profiles` instead of `/api/workflow-configs`
- Profiles is semantically correct (a named configuration set)
- Avoids "workflow" prefix redundancy

**Schema**:
```sql
CREATE TABLE profiles (
    id UUID PRIMARY KEY,
    workflow_name VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    UNIQUE(workflow_name, name)
);

CREATE TABLE profile_stages (
    id UUID PRIMARY KEY,
    profile_id UUID REFERENCES profiles(id) ON DELETE CASCADE,
    stage_name VARCHAR(255) NOT NULL,
    agent_id UUID REFERENCES agents(id),
    system_prompt TEXT,
    options JSONB,
    UNIQUE(profile_id, stage_name)
);
```

### 2. Workflow Directory Separation

Workflow definitions moved from `internal/workflows/samples/` to top-level `workflows/` directory.

**Rationale**:
- `internal/` is for service infrastructure, not business logic
- Workflows are application-specific, highly visible code
- Clear separation: `internal/workflows/` = infrastructure, `workflows/` = definitions
- `workflows/` can import from `internal/` (Go allows this)

**Directory Structure**:
```
agent-lab/
├── internal/workflows/     # Infrastructure (executor, observer, registry)
├── internal/profiles/      # Profile domain (CRUD)
└── workflows/              # Workflow definitions
    ├── summarize/
    ├── reasoning/
    └── classify/
```

### 3. Parallel Page Detection

Uses go-agents-orchestration's `ProcessParallel` for concurrent page analysis.

**Rationale**:
- Library patterns over custom goroutines
- Worker pool auto-sizing: `min(NumCPU*2, WorkerCap, len(pages))`
- Order preservation despite concurrent execution
- Built-in error handling (fail-fast or collect-all)

### 4. Numeric Clarity Scoring (0.0-1.0)

Replace categorical HIGH/MEDIUM/LOW with numeric 0.0-1.0 scale.

**Rationale**:
- Finer-grained decisions
- Enables threshold tuning without code changes
- Weighted factor aggregation for confidence scoring
- Explicit semantics: 0.7 threshold for enhancement trigger

### 5. Full Prompts in Database

System prompts stored as full text in `profile_stages.system_prompt`.

**Rationale**:
- Simpler than file references
- All configuration in one place
- Easy to compare profiles via SQL
- No file system dependencies for prompt changes

## Architecture

### Workflow Execution Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                    classify-docs Workflow                            │
└─────────────────────────────────────────────────────────────────────┘

Params: { document_id, profile_id (optional), token (optional) }

1. INIT
   ├─ Load workflow profile from DB (or default)
   ├─ Load document metadata
   └─ Render all pages to images (via images domain)

2. DETECT (Parallel - ProcessParallel)
   ├─ For each page concurrently:
   │   ├─ Vision API analyzes page image
   │   ├─ Detects all security markings
   │   ├─ Scores clarity (0.0-1.0)
   │   └─ If clarity < 0.7: suggest filter adjustments
   └─ Aggregate: collect all PageDetection results

3. ENHANCE (Conditional)
   ├─ Predicate: any page with clarity < 0.7 AND filter suggestion
   ├─ If true:
   │   ├─ Re-render with suggested filters
   │   └─ Re-run detection on enhanced images
   └─ Merge enhanced results into detection set

4. CLASSIFY
   ├─ Single agent evaluates all detections
   ├─ Determines overall classification
   ├─ Provides rationale
   └─ Lists alternatives if ambiguity exists

5. SCORE
   ├─ Confidence scoring agent evaluates metrics
   ├─ Weighted factors (see below)
   └─ Returns 0.0-1.0 score with recommendation
```

### Parallel Detection Pattern

```go
// Item type for parallel processing
type PageItem struct {
    PageNumber int
    ImageData  string  // Base64 data URI
}

// Processor function
processor := func(ctx context.Context, item PageItem) (PageDetection, error) {
    prompt := buildDetectionPrompt(item.PageNumber)
    opts := map[string]any{"system_prompt": stageConfig.SystemPrompt}

    resp, err := runtime.Agents().Vision(ctx, stageConfig.AgentID, prompt, []string{item.ImageData}, opts, token)
    if err != nil {
        return PageDetection{}, err
    }

    return parseDetectionResponse(resp.Content())
}

// Execute parallel detection
result, err := workflows.ProcessParallel(ctx, cfg, items, processor, nil)
```

### Runtime Integration

Runtime extended to include profiles system:

```go
type Runtime struct {
    agents    agents.System
    documents documents.System
    images    images.System
    profiles  profiles.System  // NEW
    logger    *slog.Logger
}

func (r *Runtime) Profiles() profiles.System
```

## Type Definitions

### Detection Stage Output

```go
type PageDetection struct {
    PageNumber       int               `json:"page_number"`
    MarkingsFound    []MarkingInfo     `json:"markings_found"`
    ClarityScore     float64           `json:"clarity_score"`     // 0.0-1.0
    FilterSuggestion *FilterSuggestion `json:"filter_suggestion,omitempty"`
}

type MarkingInfo struct {
    Text       string  `json:"text"`        // e.g., "SECRET//NOFORN"
    Location   string  `json:"location"`    // header/footer/margin/body
    Confidence float64 `json:"confidence"`  // 0.0-1.0
    Faded      bool    `json:"faded"`
}

type FilterSuggestion struct {
    Brightness *int `json:"brightness,omitempty"`
    Contrast   *int `json:"contrast,omitempty"`
    Saturation *int `json:"saturation,omitempty"`
}
```

### Classification Stage Output

```go
type ClassificationResult struct {
    Classification      string               `json:"classification"`
    AlternativeReadings []AlternativeReading `json:"alternative_readings,omitempty"`
    MarkingSummary      []string             `json:"marking_summary"`
    Rationale           string               `json:"rationale"`
}

type AlternativeReading struct {
    Classification string  `json:"classification"`
    Probability    float64 `json:"probability"`
    Reason         string  `json:"reason"`
}
```

### Confidence Assessment

```go
type ConfidenceAssessment struct {
    OverallScore   float64            `json:"overall_score"`   // 0.0-1.0
    Factors        []ConfidenceFactor `json:"factors"`
    Recommendation string             `json:"recommendation"`  // ACCEPT/REVIEW/REJECT
}

type ConfidenceFactor struct {
    Name        string  `json:"name"`
    Score       float64 `json:"score"`
    Weight      float64 `json:"weight"`
    Description string  `json:"description"`
}
```

## Confidence Scoring Factors

| Factor | Weight | Description |
|--------|--------|-------------|
| marking_clarity | 0.30 | Average clarity score across pages |
| marking_consistency | 0.25 | Same markings on all pages |
| spatial_coverage | 0.15 | Markings in expected locations |
| enhancement_impact | 0.10 | Enhancement revealed vs confirmed |
| alternative_count | 0.10 | Fewer alternatives = higher confidence |
| detection_confidence | 0.10 | Average per-marking confidence |

**Thresholds**:
- >= 0.90: **ACCEPT** - Classification reliable
- 0.70-0.89: **REVIEW** - Human verification recommended
- < 0.70: **REJECT** - Insufficient confidence

## API Endpoints

### Profiles API (New)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/profiles` | Create workflow profile |
| GET | `/api/profiles` | List profiles (filter by workflow_name) |
| GET | `/api/profiles/{id}` | Get profile with stages |
| PUT | `/api/profiles/{id}` | Update profile metadata |
| DELETE | `/api/profiles/{id}` | Delete profile and stages |
| POST | `/api/profiles/{id}/stages` | Add/update stage config |
| DELETE | `/api/profiles/{id}/stages/{stage}` | Remove stage config |

### Workflow Execution (Existing)

```
POST /api/workflows/classify-docs/execute
POST /api/workflows/classify-docs/execute/stream
```

Request:
```json
{
  "params": {
    "document_id": "uuid",
    "profile_id": "uuid (optional)",
    "token": "optional"
  }
}
```

## Session Breakdown

### Session 4a: Profiles Infrastructure & Workflow Migration ✓

- Database migration for profiles and profile_stages tables
- Profile domain: types, repository, handler, OpenAPI
- Move workflows to `workflows/` directory
- Update Runtime with profiles system access
- Profile resolution helpers (`LoadProfile`, `ExtractAgentParams`)
- Stage AgentID override support

### Session 4b: classify-docs Types and Detection Stage

- Type definitions (PageDetection, MarkingInfo, etc.)
- Detection system prompt and response parser
- Init node (load profile, document, render images)
- Detect node using ProcessParallel

### Session 4c: Enhancement, Classification, and Scoring

- Enhancement conditional node
- Classification node and prompt
- Scoring node with weighted factors
- Complete workflow graph assembly

### Session 4d: Testing and Refinement

- Unit tests for parsers and helpers
- Integration tests with real LLM calls
- Default profile seeding
- Validation against test document set

## Dependencies

No external library updates required:
- go-agents-orchestration v0.3.1 (ProcessParallel, StateGraph)
- go-agents v0.3.0 (Vision API)
- document-context v0.1.1 (Image rendering)

## Success Criteria

1. Workflow profiles CRUD via API
2. Existing workflows moved to `workflows/` and functioning
3. classify-docs workflow registered and executable
4. Parallel page detection with ProcessParallel
5. Conditional enhancement for low-clarity pages
6. Classification with alternatives when ambiguous
7. Confidence scoring with weighted factors
8. A/B testing capability via profile_id parameter
9. Baseline accuracy on test documents (target: match 96.3% prototype)
