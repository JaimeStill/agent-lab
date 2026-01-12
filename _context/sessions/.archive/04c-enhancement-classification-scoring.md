# Session 4c: Enhancement, Classification, and Scoring

## Overview

Extend the classify-docs workflow from `[init] → [detect]` to the complete graph:

```
[init] → [detect] → [enhance]? → [classify] → [score]
                        │
              (conditional: skip if no pages need enhancement)
```

## Phase 1: Errors

**File: `workflows/classify/errors.go`**

Add three new errors after `ErrDetectionFailed`:

```go
ErrEnhancementFailed    = errors.New("enhancement failed")
ErrClassificationFailed = errors.New("classification failed")
ErrScoringFailed        = errors.New("scoring failed")
```

## Phase 2: Types

**File: `workflows/classify/classify.go`**

### 2.1 Update PageDetection

Replace the existing `PageDetection` struct with:

```go
type PageDetection struct {
	PageNumber       int               `json:"page_number"`
	OriginalImageID  uuid.UUID         `json:"original_image_id"`
	EnhancedImageID  *uuid.UUID        `json:"enhanced_image_id,omitempty"`
	MarkingsFound    []MarkingInfo     `json:"markings_found"`
	ClarityScore     float64           `json:"clarity_score"`
	FilterSuggestion *FilterSuggestion `json:"filter_suggestion,omitempty"`
}
```

### 2.2 Add Constant and New Types

Add this constant before the type definitions (after imports):

```go
const DefaultClarityThreshold = 0.7
```

Add these types after `FilterSuggestion`:

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

type ConfidenceAssessment struct {
	OverallScore   float64            `json:"overall_score"`
	Factors        []ConfidenceFactor `json:"factors"`
	Recommendation string             `json:"recommendation"`
}

type ConfidenceFactor struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

type EnhanceOptions struct {
	ClarityThreshold float64 `json:"clarity_threshold"`
}

func DefaultEnhanceOptions() EnhanceOptions {
	return EnhanceOptions{ClarityThreshold: DefaultClarityThreshold}
}
```

## Phase 3: Profile Updates

**File: `workflows/classify/profile.go`**

### 3.1 Add System Prompts

Add after `DetectionSystemPrompt`:

```go
const ClassificationSystemPrompt = `You are a document classification specialist. Analyze security marking detections across all pages to determine the overall document classification.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"classification": "<overall classification level>",
	"alternative_readings": [
		{
			"classification": "<alternative classification>",
			"probability": <0.0-1.0>,
			"reason": "<why this could be the correct classification>"
		}
	],
	"marking_summary": ["<list of unique markings found>"],
	"rationale": "<explanation of classification decision>"
}

INSTRUCTIONS:
- Analyze all marking detections provided
- Determine the HIGHEST classification level present
- If markings are inconsistent or ambiguous, list alternative readings
- marking_summary should list unique marking texts (deduplicated)
- rationale should explain how you determined the classification
- Consider marking confidence and consistency across pages
- JSON response only; no preamble or dialog`

const ScoringSystemPrompt = `You are a confidence scoring specialist. Evaluate the quality and reliability of document classification results.

OUTPUT FORMAT: Respond with ONLY a JSON object matching this exact schema:
{
	"overall_score": <0.0-1.0>,
	"factors": [
		{
			"name": "<factor name>",
			"score": <0.0-1.0>,
			"weight": <weight>,
			"description": "<explanation of score>"
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
- detection_confidence (weight: 0.10): Average marking confidence

THRESHOLDS:
- >= 0.90: ACCEPT - Classification is reliable
- 0.70-0.89: REVIEW - Human verification recommended
- < 0.70: REJECT - Insufficient confidence

JSON response only; no preamble or dialog`
```

### 3.2 Update DefaultProfile

Add import for `encoding/json` and replace `DefaultProfile`:

```go
func DefaultProfile() *profiles.ProfileWithStages {
	initPrompt := ""
	detectPrompt := DetectionSystemPrompt
	enhancePrompt := DetectionSystemPrompt
	classifyPrompt := ClassificationSystemPrompt
	scorePrompt := ScoringSystemPrompt

	enhanceOpts, _ := json.Marshal(DefaultEnhanceOptions())

	return profiles.NewProfileWithStages(
		profiles.ProfileStage{StageName: "init", SystemPrompt: &initPrompt},
		profiles.ProfileStage{StageName: "detect", SystemPrompt: &detectPrompt},
		profiles.ProfileStage{StageName: "enhance", SystemPrompt: &enhancePrompt, Options: enhanceOpts},
		profiles.ProfileStage{StageName: "classify", SystemPrompt: &classifyPrompt},
		profiles.ProfileStage{StageName: "score", SystemPrompt: &scorePrompt},
	)
}
```

## Phase 4: Parser Functions

**File: `workflows/classify/parse.go`**

### 4.1 Refactor to Generic parseResponse

Replace the existing `ParseDetectionResponse` and `validateDetection` functions with a generic approach. Add a `clamp` helper for readability:

```go
func clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func parseResponse[T any](content string, validate func(T) T, errMsg string) (T, error) {
	var result T
	content = strings.TrimSpace(content)
	if err := json.Unmarshal([]byte(content), &result); err == nil {
		return validate(result), nil
	}

	matches := jsonBlockRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		cleaned := strings.TrimSpace(matches[1])
		if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
			return validate(result), nil
		}
	}

	return result, fmt.Errorf("%w: %s", ErrParseResponse, errMsg)
}

func ParseDetectionResponse(content string) (PageDetection, error) {
	return parseResponse(content, validateDetection, "could not parse detection JSON")
}

func validateDetection(d PageDetection) PageDetection {
	d.ClarityScore = clamp(d.ClarityScore, 0.0, 1.0)

	for i := range d.MarkingsFound {
		d.MarkingsFound[i].Confidence = clamp(d.MarkingsFound[i].Confidence, 0.0, 1.0)
	}

	return d
}
```

### 4.2 Add ParseClassificationResponse

```go
func ParseClassificationResponse(content string) (ClassificationResult, error) {
	return parseResponse(content, validateClassification, "could not parse classification JSON")
}

func validateClassification(c ClassificationResult) ClassificationResult {
	for i := range c.AlternativeReadings {
		c.AlternativeReadings[i].Probability = clamp(c.AlternativeReadings[i].Probability, 0.0, 1.0)
	}
	return c
}
```

### 4.3 Add ParseScoringResponse

```go
func ParseScoringResponse(content string) (ConfidenceAssessment, error) {
	return parseResponse(content, validateScoring, "could not parse scoring JSON")
}

func validateScoring(a ConfidenceAssessment) ConfidenceAssessment {
	a.OverallScore = clamp(a.OverallScore, 0.0, 1.0)

	for i := range a.Factors {
		a.Factors[i].Score = clamp(a.Factors[i].Score, 0.0, 1.0)
		a.Factors[i].Weight = clamp(a.Factors[i].Weight, 0.0, 1.0)
	}

	switch a.Recommendation {
	case "ACCEPT", "REVIEW", "REJECT":
	default:
		a.Recommendation = computeRecommendation(a.OverallScore)
	}

	return a
}

func computeRecommendation(score float64) string {
	switch {
	case score >= 0.90:
		return "ACCEPT"
	case score >= 0.70:
		return "REVIEW"
	default:
		return "REJECT"
	}
}
```

## Phase 5: Helper Functions

**File: `workflows/classify/classify.go`**

Add these helper functions after the type definitions:

```go
func extractEnhanceOptions(stage *profiles.ProfileStage) EnhanceOptions {
	opts := DefaultEnhanceOptions()
	if stage == nil || len(stage.Options) == 0 {
		return opts
	}
	json.Unmarshal(stage.Options, &opts)
	opts.ClarityThreshold = clamp(opts.ClarityThreshold, 0.0, 1.0)
	return opts
}

func mergeDetections(original, enhanced PageDetection, threshold float64) PageDetection {
	result := PageDetection{
		PageNumber:      original.PageNumber,
		OriginalImageID: original.OriginalImageID,
		EnhancedImageID: &enhanced.OriginalImageID,
		ClarityScore:    math.Max(original.ClarityScore, enhanced.ClarityScore),
	}

	markingMap := make(map[string]MarkingInfo)
	for _, m := range original.MarkingsFound {
		key := m.Text + "|" + m.Location
		markingMap[key] = m
	}

	for _, em := range enhanced.MarkingsFound {
		key := em.Text + "|" + em.Location
		if om, exists := markingMap[key]; exists {
			if (om.Faded || om.Confidence < threshold) && em.Confidence > om.Confidence {
				markingMap[key] = em
			}
		} else {
			markingMap[key] = em
		}
	}

	result.MarkingsFound = make([]MarkingInfo, 0, len(markingMap))
	for _, m := range markingMap {
		result.MarkingsFound = append(result.MarkingsFound, m)
	}

	return result
}

func buildClassificationPrompt(docName string, detections []PageDetection) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Document: %s\n", docName))
	sb.WriteString(fmt.Sprintf("Total Pages: %d\n\n", len(detections)))
	sb.WriteString("Detections by Page:\n")

	for _, d := range detections {
		sb.WriteString(fmt.Sprintf("\nPage %d (clarity: %.2f):\n", d.PageNumber, d.ClarityScore))
		if len(d.MarkingsFound) == 0 {
			sb.WriteString("  No markings detected\n")
		} else {
			for _, m := range d.MarkingsFound {
				sb.WriteString(fmt.Sprintf("  - %s [%s] (confidence: %.2f, faded: %v)\n",
					m.Text, m.Location, m.Confidence, m.Faded))
			}
		}
	}

	sb.WriteString("\nDetermine the overall document classification.")
	return sb.String()
}

func buildScoringPrompt(detections []PageDetection, classification ClassificationResult, enhancementApplied bool) string {
	var sb strings.Builder
	sb.WriteString("Classification Result:\n")
	sb.WriteString(fmt.Sprintf("  Classification: %s\n", classification.Classification))
	sb.WriteString(fmt.Sprintf("  Alternatives: %d\n", len(classification.AlternativeReadings)))
	sb.WriteString(fmt.Sprintf("  Rationale: %s\n\n", classification.Rationale))

	sb.WriteString(fmt.Sprintf("Enhancement Applied: %v\n\n", enhancementApplied))

	sb.WriteString("Detection Summary:\n")
	var totalClarity, totalConfidence float64
	var markingCount int
	markings := make(map[string]int)
	headerFooterPages := 0

	for _, d := range detections {
		totalClarity += d.ClarityScore
		hasHeaderFooter := false
		for _, m := range d.MarkingsFound {
			totalConfidence += m.Confidence
			markingCount++
			markings[m.Text]++
			if m.Location == "header" || m.Location == "footer" {
				hasHeaderFooter = true
			}
		}
		if hasHeaderFooter {
			headerFooterPages++
		}
	}

	avgClarity := 0.0
	if len(detections) > 0 {
		avgClarity = totalClarity / float64(len(detections))
	}
	avgConfidence := 0.0
	if markingCount > 0 {
		avgConfidence = totalConfidence / float64(markingCount)
	}
	spatialCoverage := 0.0
	if len(detections) > 0 {
		spatialCoverage = float64(headerFooterPages) / float64(len(detections))
	}

	sb.WriteString(fmt.Sprintf("  Pages: %d\n", len(detections)))
	sb.WriteString(fmt.Sprintf("  Average Clarity: %.2f\n", avgClarity))
	sb.WriteString(fmt.Sprintf("  Average Confidence: %.2f\n", avgConfidence))
	sb.WriteString(fmt.Sprintf("  Spatial Coverage: %.2f\n", spatialCoverage))
	sb.WriteString(fmt.Sprintf("  Unique Markings: %d\n", len(markings)))

	sb.WriteString("\nEvaluate confidence and provide recommendation.")
	return sb.String()
}
```

## Phase 6: Node Implementations

**File: `workflows/classify/classify.go`**

Extract node definitions as standalone functions that return `state.StateNode`. This keeps the factory focused on workflow structure while node logic lives in dedicated functions.

### 6.1 initNode Function

Add after helper functions (before `func init()`):

```go
func initNode(runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
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
}
```

### 6.2 detectNode Function

```go
func detectNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("detect")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		pages, ok := s.Get("page_images")
		if !ok {
			return s, fmt.Errorf("page_images not found in state")
		}
		pageImages := pages.([]PageImage)

		opts := map[string]any{}
		if stage.SystemPrompt != nil {
			opts["system_prompt"] = *stage.SystemPrompt
		}

		enhanceOpts := extractEnhanceOptions(profile.Stage("enhance"))

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

			detection, err := ParseDetectionResponse(resp.Content())
			if err != nil {
				return PageDetection{}, err
			}

			detection.PageNumber = img.PageNumber
			detection.OriginalImageID = img.ImageID

			return detection, nil
		}

		cfg := detectionParallelConfig()
		result, err := wf.ProcessParallel(ctx, cfg, pageImages, processor, nil)
		if err != nil {
			return s, fmt.Errorf("parallel detection failed: %w", err)
		}

		needsEnhancement := false
		for _, d := range result.Results {
			if d.NeedsEnhancement(enhanceOpts.ClarityThreshold) {
				needsEnhancement = true
				break
			}
		}

		s = s.Set("detections", result.Results)
		s = s.Set("needs_enhancement", needsEnhancement)

		return s, nil
	})
}
```

### 6.3 enhanceNode Function

```go
func enhanceNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("enhance")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		detections, _ := s.Get("detections")
		detectList := detections.([]PageDetection)

		docVal, _ := s.Get("document")
		doc := docVal.(*documents.Document)

		opts := map[string]any{}
		if stage.SystemPrompt != nil {
			opts["system_prompt"] = *stage.SystemPrompt
		}

		enhanceOpts := extractEnhanceOptions(stage)

		var pagesToEnhance []PageDetection
		for _, d := range detectList {
			if d.NeedsEnhancement(enhanceOpts.ClarityThreshold) {
				pagesToEnhance = append(pagesToEnhance, d)
			}
		}

		if len(pagesToEnhance) == 0 {
			s = s.Set("enhancement_applied", false)
			return s, nil
		}

		processor := func(ctx context.Context, original PageDetection) (PageDetection, error) {
			renderOpts := images.RenderOptions{
				Pages:      fmt.Sprintf("%d", original.PageNumber),
				Format:     "png",
				DPI:        300,
				Brightness: original.FilterSuggestion.Brightness,
				Contrast:   original.FilterSuggestion.Contrast,
				Saturation: original.FilterSuggestion.Saturation,
				Force:      true,
			}

			rendered, err := runtime.Images().Render(ctx, doc.ID, renderOpts)
			if err != nil {
				return PageDetection{}, fmt.Errorf("%w: re-render failed: %v", ErrEnhancementFailed, err)
			}

			if len(rendered) == 0 {
				return PageDetection{}, fmt.Errorf("%w: no image returned from re-render", ErrEnhancementFailed)
			}

			enhancedImg := rendered[0]

			data, contentType, err := runtime.Images().Data(ctx, enhancedImg.ID)
			if err != nil {
				return PageDetection{}, fmt.Errorf("%w: failed to retrieve enhanced image data: %v", ErrEnhancementFailed, err)
			}

			dataURI := buildDataURI(data, contentType)
			prompt := fmt.Sprintf("Analyze page %d of this document for security classification markings.", original.PageNumber)

			resp, err := runtime.Agents().Vision(ctx, agentID, prompt, []string{dataURI}, opts, token)
			if err != nil {
				return PageDetection{}, fmt.Errorf("%w: %v", ErrEnhancementFailed, err)
			}

			enhanced, err := ParseDetectionResponse(resp.Content())
			if err != nil {
				return PageDetection{}, err
			}

			enhanced.PageNumber = original.PageNumber
			enhanced.OriginalImageID = enhancedImg.ID

			return mergeDetections(original, enhanced, enhanceOpts.ClarityThreshold), nil
		}

		cfg := detectionParallelConfig()
		result, err := wf.ProcessParallel(ctx, cfg, pagesToEnhance, processor, nil)
		if err != nil {
			return s, fmt.Errorf("parallel enhancement failed: %w", err)
		}

		enhancedMap := make(map[int]PageDetection)
		for _, d := range result.Results {
			enhancedMap[d.PageNumber] = d
		}

		updatedDetections := make([]PageDetection, len(detectList))
		for i, d := range detectList {
			if enhanced, ok := enhancedMap[d.PageNumber]; ok {
				updatedDetections[i] = enhanced
			} else {
				updatedDetections[i] = d
			}
		}

		s = s.Set("detections", updatedDetections)
		s = s.Set("enhancement_applied", true)

		return s, nil
	})
}
```

### 6.4 classifyNode Function

```go
func classifyNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("classify")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		detections, _ := s.Get("detections")
		detectList := detections.([]PageDetection)

		docVal, _ := s.Get("document")
		doc := docVal.(*documents.Document)

		prompt := buildClassificationPrompt(doc.Name, detectList)

		opts := map[string]any{}
		if stage.SystemPrompt != nil {
			opts["system_prompt"] = *stage.SystemPrompt
		}

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("%w: %v", ErrClassificationFailed, err)
		}

		classification, err := ParseClassificationResponse(resp.Content())
		if err != nil {
			return s, err
		}

		s = s.Set("classification", classification)

		return s, nil
	})
}
```

### 6.5 scoreNode Function

```go
func scoreNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("score")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		detections, _ := s.Get("detections")
		detectList := detections.([]PageDetection)

		classificationVal, _ := s.Get("classification")
		classification := classificationVal.(ClassificationResult)

		enhancementApplied := false
		if ea, ok := s.Get("enhancement_applied"); ok {
			enhancementApplied = ea.(bool)
		}

		prompt := buildScoringPrompt(detectList, classification, enhancementApplied)

		opts := map[string]any{}
		if stage.SystemPrompt != nil {
			opts["system_prompt"] = *stage.SystemPrompt
		}

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("%w: %v", ErrScoringFailed, err)
		}

		assessment, err := ParseScoringResponse(resp.Content())
		if err != nil {
			return s, err
		}

		s = s.Set("confidence", assessment)

		return s, nil
	})
}
```

## Phase 7: Factory Refactor

**File: `workflows/classify/classify.go`**

Replace the entire `factory` function with this clean, declarative version:

```go
func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("init", initNode(runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("detect", detectNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("enhance", enhanceNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("classify", classifyNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("score", scoreNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("init", "detect", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("detect", "enhance", state.KeyEquals("needs_enhancement", true)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("detect", "classify", state.KeyEquals("needs_enhancement", false)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("enhance", "classify", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("classify", "score", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("init"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("score"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}
```

## Phase 8: Import Updates

**File: `workflows/classify/classify.go`**

Update imports to include:

```go
import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"

	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	wf "github.com/JaimeStill/go-agents-orchestration/pkg/workflows"
)
```

**File: `workflows/classify/profile.go`**

Update imports to include:

```go
import (
	"encoding/json"

	"github.com/JaimeStill/agent-lab/internal/profiles"
)
```

## Phase 9: Refactor Existing Workflows

Apply the same node extraction pattern to the summarize and reasoning workflows for consistency.

### 9.1 Summarize Workflow

**File: `workflows/summarize/summarize.go`**

```go
package summarize

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func init() {
	workflows.Register("summarize", factory, "Summarizes input text using an AI agent")
}

func summarizeNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("summarize")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		text, ok := s.Get("text")
		if !ok {
			return s, fmt.Errorf("text is required")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Please summarize the following text:\n\n%s", text)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("chat failed: %w", err)
		}

		return s.Set("summary", resp.Content()), nil
	})
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("summarize", summarizeNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("summarize"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("summarize"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}
```

### 9.2 Reasoning Workflow

**File: `workflows/reasoning/reasoning.go`**

```go
package reasoning

import (
	"context"
	"fmt"

	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

func init() {
	workflows.Register("reasoning", factory, "Multi-step reasoning workflow that analyzes problems")
}

func analyzeNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("analyze")
		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		problem, ok := s.Get("problem")
		if !ok {
			return s, fmt.Errorf("problem is required")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Analyze this problem and identify its key components:\n\n%s", problem)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("analyze failed: %w", err)
		}

		return s.Set("analysis", resp.Content()), nil
	})
}

func reasonNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("reason")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		analysis, ok := s.Get("analysis")
		if !ok {
			return s, fmt.Errorf("analysis not found in state")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Given this analysis:\n\n%s\n\nWhat are the logical steps to solve this problem?", analysis)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("reason failed: %w", err)
		}

		return s.Set("reasoning", resp.Content()), nil
	})
}

func concludeNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		stage := profile.Stage("conclude")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		reasoning, ok := s.Get("reasoning")
		if !ok {
			return s, fmt.Errorf("reasoning not found in state")
		}

		opts := map[string]any{
			"system_prompt": *stage.SystemPrompt,
		}

		prompt := fmt.Sprintf("Based on this reasoning:\n\n%s\n\nWhat is the conclusion?", reasoning)

		resp, err := runtime.Agents().Chat(ctx, agentID, prompt, opts, token)
		if err != nil {
			return s, fmt.Errorf("conclude failed: %w", err)
		}

		return s.Set("conclusion", resp.Content()), nil
	})
}

func factory(ctx context.Context, graph state.StateGraph, runtime *workflows.Runtime, params map[string]any) (state.State, error) {
	profile, err := workflows.LoadProfile(ctx, runtime, params, DefaultProfile())
	if err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("analyze", analyzeNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("reason", reasonNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddNode("conclude", concludeNode(profile, runtime)); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("analyze", "reason", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.AddEdge("reason", "conclude", nil); err != nil {
		return state.State{}, err
	}

	if err := graph.SetEntryPoint("analyze"); err != nil {
		return state.State{}, err
	}

	if err := graph.SetExitPoint("conclude"); err != nil {
		return state.State{}, err
	}

	initialState := state.New(nil)
	for k, v := range params {
		initialState = initialState.Set(k, v)
	}

	return initialState, nil
}
```
