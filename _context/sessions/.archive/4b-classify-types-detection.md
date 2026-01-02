# Session 4b: classify-docs Types and Detection Stage

## Problem Context

The classify-docs workflow needs to analyze document images for security classification markings. This session establishes the type definitions and implements two core nodes:

1. **Init Node** - Loads document metadata and renders all pages to images
2. **Detect Node** - Processes pages in parallel using Vision API to identify markings

The workflow uses `ProcessParallel` from go-agents-orchestration for concurrent page analysis.

## Architecture Approach

### State Flow

```
params: { document_id, profile_id?, agent_id?, token? }
         ↓
      [init]
         ↓
state: { document, page_images: []PageImage (lightweight - no data URIs) }
         ↓
      [detect] (fetches image data on-demand for Vision API)
         ↓
state: { document, page_images, detections: []PageDetection }
```

### File Structure

```
workflows/classify/
    errors.go      # Domain errors
    parse.go       # JSON response parsing
    profile.go     # DefaultProfile with stage prompts
    classify.go    # Types, workflow registration, and nodes
```

## Implementation Steps

### Phase 1: Errors

Create `workflows/classify/errors.go`:

```go
package classify

import "errors"

var (
	ErrDocumentNotFound = errors.New("document not found")
	ErrNoPages          = errors.New("document has no pages")
	ErrRenderFailed     = errors.New("failed to render pages")
	ErrParseResponse    = errors.New("failed to parse detection response")
	ErrDetectionFailed  = errors.New("detection failed")
)
```

### Phase 2: Response Parsing

Create `workflows/classify/parse.go`:

```go
package classify

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var jsonBlockRegex = regexp.MustCompile(`(?s)` + "```" + `(?:json)?\s*\n?(.*?)\n?` + "```")

func parseDetectionResponse(content string) (PageDetection, error) {
	var detection PageDetection

	content = strings.TrimSpace(content)
	if err := json.Unmarshal([]byte(content), &detection); err == nil {
		return validateDetection(detection)
	}

	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		cleaned := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(cleaned), &detection); err == nil {
			return validateDetection(detection)
		}
	}

	return PageDetection{}, fmt.Errorf("%w: could not parse JSON from response", ErrParseResponse)
}

func validateDetection(d PageDetection) (PageDetection, error) {
	if d.ClarityScore < 0.0 {
		d.ClarityScore = 0.0
	}
	if d.ClarityScore > 1.0 {
		d.ClarityScore = 1.0
	}

	for i := range d.MarkingsFound {
		if d.MarkingsFound[i].Confidence < 0.0 {
			d.MarkingsFound[i].Confidence = 0.0
		}
		if d.MarkingsFound[i].Confidence > 1.0 {
			d.MarkingsFound[i].Confidence = 1.0
		}
	}

	return d, nil
}
```

### Phase 3: Profile Definition

Create `workflows/classify/profile.go`:

```go
package classify

import "github.com/JaimeStill/agent-lab/internal/profiles"

const DetectionSystemPrompt = `You are a document security marking detection specialist. Analyze the provided document page image and identify all security classification markings.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
  "page_number": <integer>,
  "markings_found": [
    {
      "text": "<exact marking text>",
      "location": "<header|footer|margin|body>",
      "confidence": <0.0-1.0>,
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
- Identify ALL security markings (e.g., UNCLASSIFIED, CONFIDENTIAL, SECRET, TOP SECRET, caveats like NOFORN, ORCON)
- Note the location of each marking (header, footer, margin, or body)
- Assess confidence based on readability (1.0 = perfectly clear, 0.0 = illegible)
- Set faded=true if the marking appears washed out or hard to read
- clarity_score reflects overall page quality for marking detection
- If clarity_score < 0.7, suggest filter adjustments that might improve readability
- Do NOT include any text outside the JSON object`

func DefaultProfile() *profiles.ProfileWithStages {
	initPrompt := ""
	detectPrompt := DetectionSystemPrompt

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "init", SystemPrompt: &initPrompt},
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &detectPrompt},
	)
}
```

### Phase 4: Workflow Implementation

Create `workflows/classify/classify.go`:

```go
package classify

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	gowf "github.com/JaimeStill/go-agents-orchestration/pkg/workflows"
	"github.com/google/uuid"
)

type PageImage struct {
	PageNumber int       `json:"page_number"`
	ImageID    uuid.UUID `json:"image_id"`
}

type PageDetection struct {
	PageNumber       int               `json:"page_number"`
	MarkingsFound    []MarkingInfo     `json:"markings_found"`
	ClarityScore     float64           `json:"clarity_score"`
	FilterSuggestion *FilterSuggestion `json:"filter_suggestion,omitempty"`
}

type MarkingInfo struct {
	Text       string  `json:"text"`
	Location   string  `json:"location"`
	Confidence float64 `json:"confidence"`
	Faded      bool    `json:"faded"`
}

type FilterSuggestion struct {
	Brightness *int `json:"brightness,omitempty"`
	Contrast   *int `json:"contrast,omitempty"`
	Saturation *int `json:"saturation,omitempty"`
}

func (d PageDetection) NeedsEnhancement(threshold float64) bool {
	return d.ClarityScore < threshold && d.FilterSuggestion != nil
}

func init() {
	workflows.Register("classify-docs", factory, "Classifies document security markings using vision analysis")
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	initNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		docIDStr, ok := s.Get("document_id")
		if !ok {
			return s, fmt.Errorf("document_id is required")
		}

		docID, err := uuid.Parse(docIDStr.(string))
		if err != nil {
			return s, fmt.Errorf("invalid document_id: %w", err)
		}

		doc, err := runtime.Documents().Find(ctx, docID)
		if err != nil {
			return s, fmt.Errorf("%w: %v", ErrDocumentNotFound, err)
		}

		if doc.PageCount == nil || *doc.PageCount < 1 {
			return s, ErrNoPages
		}

		renderOpts := images.RenderOptions{
			Pages:  "",
			Format: "png",
			DPI:    300,
		}

		renderedImages, err := runtime.Images().Render(ctx, docID, renderOpts)
		if err != nil {
			return s, fmt.Errorf("%w: %v", ErrRenderFailed, err)
		}

		pageImages := make([]PageImage, len(renderedImages))
		for i, img := range renderedImages {
			pageImages[i] = PageImage{
				PageNumber: img.PageNumber,
				ImageID:    img.ID,
			}
		}

		s = s.Set("document", doc)
		s = s.Set("page_images", pageImages)

		return s, nil
	})

	detectNode := state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("detect")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		pageImagesAny, ok := s.Get("page_images")
		if !ok {
			return s, fmt.Errorf("page_images not found in state")
		}
		pageImages := pageImagesAny.([]PageImage)

		opts := map[string]any{}
		if stage.SystemPrompt != nil {
			opts["system_prompt"] = *stage.SystemPrompt
		}

		processor := func(ctx context.Context, img PageImage) (PageDetection, error) {
			data, contentType, err := runtime.Images().Data(ctx, img.ImageID)
			if err != nil {
				return PageDetection{}, fmt.Errorf("%w: failed to retrieve image data: %v", ErrDetectionFailed, err)
			}

			dataURI := buildDataURI(data, contentType)
			prompt := fmt.Sprintf("Analyze page %d of this document for security classification markings.", img.PageNumber)

			resp, err := runtime.Agents().Vision(ctx, agentID, prompt, []string{dataURI}, opts, token)
			if err != nil {
				return PageDetection{}, fmt.Errorf("%w: %v", ErrDetectionFailed, err)
			}

			detection, err := parseDetectionResponse(resp.Content())
			if err != nil {
				return PageDetection{}, err
			}

			detection.PageNumber = img.PageNumber

			return detection, nil
		}

		cfg := detectionParallelConfig()
		result, err := gowf.ProcessParallel(ctx, cfg, pageImages, processor, nil)
		if err != nil {
			return s, fmt.Errorf("parallel detection failed: %w", err)
		}

		s = s.Set("detections", result.Results)

		return s, nil
	})

	if err := graph.AddNode("init", initNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("detect", detectNode); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("init", "detect", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("init"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("detect"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}

func detectionParallelConfig() config.ParallelConfig {
	failFast := true
	return config.ParallelConfig{
		MaxWorkers:  0,
		WorkerCap:   4,
		FailFastNil: &failFast,
		Observer:    "noop",
	}
}

func buildDataURI(data []byte, contentType string) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}
```

### Phase 5: Register Workflow

Update `workflows/init.go` to add the import:

```go
package workflows

import (
	_ "github.com/JaimeStill/agent-lab/workflows/classify"
	_ "github.com/JaimeStill/agent-lab/workflows/reasoning"
	_ "github.com/JaimeStill/agent-lab/workflows/summarize"
)
```

### Phase 6: Validation

After implementation, validate:

1. **Build check**:
   ```bash
   go vet ./...
   ```

2. **Workflow registration** - verify `classify-docs` appears in:
   ```
   GET /api/workflows
   ```

3. **Execution test** - execute with a document:
   ```
   POST /api/workflows/classify-docs/execute
   {
     "params": {
       "document_id": "38efe005-9aeb-461d-8b1e-7852e96db45c",
       "agent_id": "cc54e535-32cb-4c4d-958e-fd1d686ee5d0"
     }
   }
   ```

4. **State verification** - confirm response contains:
   - `document` - loaded document metadata
   - `page_images` - array of PageImage (lightweight, ImageID only)
   - `detections` - array of PageDetection results

## Dependencies

**Internal**:
- `internal/profiles` - ProfileWithStages, ProfileStage
- `internal/workflows` - Register, LoadProfile, ExtractAgentParams, Runtime
- `internal/images` - RenderOptions, Image
- `internal/documents` - Document

**External**:
- `go-agents-orchestration/pkg/state` - StateGraph, State, NewFunctionNode
- `go-agents-orchestration/pkg/config` - ParallelConfig
- `go-agents-orchestration/pkg/workflows` - ProcessParallel
