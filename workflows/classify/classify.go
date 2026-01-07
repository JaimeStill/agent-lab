// Package classify implements a document security marking classification workflow.
// It uses parallel vision analysis to detect security markings across document pages.
package classify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/JaimeStill/agent-lab/internal/documents"
	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/profiles"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	wf "github.com/JaimeStill/go-agents-orchestration/pkg/workflows"
)

// DefaultLegibilityThreshold is the legibility score below which markings are
// candidates for enhancement. Pages with any marking below this threshold and
// a valid FilterSuggestion will be re-rendered with suggested filters.
const DefaultLegibilityThreshold = 0.4

func logNodeTiming(logger *slog.Logger, nodeName string, start time.Time) {
	logger.Info("node completed", "node", nodeName, "duration", time.Since(start).String())
}

// PageImage represents a rendered document page ready for vision analysis.
type PageImage struct {
	PageNumber int       `json:"page_number"`
	ImageID    uuid.UUID `json:"image_id"`
}

// PageDetection contains detection results for a single document page.
type PageDetection struct {
	PageNumber       int               `json:"page_number"`
	OriginalImageID  uuid.UUID         `json:"original_image_id"`
	EnhancedImageID  *uuid.UUID        `json:"enhanced_image_id,omitempty"`
	MarkingsFound    []MarkingInfo     `json:"markings_found"`
	ClarityScore     float64           `json:"clarity_score"`
	FilterSuggestion *FilterSuggestion `json:"filter_suggestion,omitempty"`
}

// MarkingInfo describes a detected security marking on a document page.
type MarkingInfo struct {
	Text       string  `json:"text"`
	Location   string  `json:"location"`
	Legibility float64 `json:"legibility"`
	Faded      bool    `json:"faded"`
}

// FilterSuggestion recommends image enhancement settings for improved detection.
type FilterSuggestion struct {
	Brightness *int `json:"brightness,omitempty"`
	Contrast   *int `json:"contrast,omitempty"`
	Saturation *int `json:"saturation,omitempty"`
}

// ClassificationResult contains the overall document classification determined
// by analyzing all detected markings across pages.
type ClassificationResult struct {
	Classification      string               `json:"classification"`
	AlternativeReadings []AlternativeReading `json:"alternative_readings,omitempty"`
	MarkingSummary      []string             `json:"marking_summary"`
	Rationale           string               `json:"rationale"`
}

// AlternativeReading represents a possible alternative classification when
// markings are ambiguous or inconsistent.
type AlternativeReading struct {
	Classification string  `json:"classification"`
	Probability    float64 `json:"probability"`
	Reason         string  `json:"reason"`
}

// ConfidenceAssessment evaluates the reliability of the classification result
// based on weighted scoring factors and provides a recommendation.
type ConfidenceAssessment struct {
	OverallScore   float64            `json:"overall_score"`
	Factors        []ConfidenceFactor `json:"factors"`
	Recommendation string             `json:"recommendation"`
}

// ConfidenceFactor represents a weighted component of the confidence assessment.
type ConfidenceFactor struct {
	Name        string  `json:"name"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// EnhanceOptions configures the enhancement stage behavior.
type EnhanceOptions struct {
	LegibilityThreshold float64 `json:"legibility_threshold"`
}

// DefaultEnhanceOptions returns the default enhancement configuration.
func DefaultEnhanceOptions() EnhanceOptions {
	return EnhanceOptions{LegibilityThreshold: DefaultLegibilityThreshold}
}

func init() {
	workflows.Register("classify-docs", factory, "Classifies document security markings using vision analysis")
}

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

func initNode(runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		start := time.Now()
		defer logNodeTiming(runtime.Logger(), "init", start)

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

func detectNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		start := time.Now()
		defer logNodeTiming(runtime.Logger(), "detect", start)

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

		s = s.Set("detections", result.Results)
		s = s.Set("needs_enhancement", needsEnhancement)

		return s, nil
	})
}

func enhanceNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		start := time.Now()
		defer logNodeTiming(runtime.Logger(), "enhance", start)

		stage := profile.Stage("enhance")

		agentID, token, err := workflows.ExtractAgentParams(s, stage)
		if err != nil {
			return s, err
		}

		detections, _ := s.Get("detections")
		detectList := detections.([]PageDetection)

		pageImagesVal, _ := s.Get("page_images")
		pageImages := pageImagesVal.([]PageImage)

		docVal, _ := s.Get("document")
		doc := docVal.(*documents.Document)

		opts := map[string]any{}
		if stage.SystemPrompt != nil {
			opts["system_prompt"] = *stage.SystemPrompt
		}

		enhanceOpts := extractEnhanceOptions(stage)

		pageImageMap := make(map[int]PageImage)
		for _, pi := range pageImages {
			pageImageMap[pi.PageNumber] = pi
		}

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

			return mergeDetections(original, enhanced, enhanceOpts.LegibilityThreshold), nil
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

func classifyNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		start := time.Now()
		defer logNodeTiming(runtime.Logger(), "classify", start)

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

func scoreNode(profile *profiles.ProfileWithStages, runtime *workflows.Runtime) state.StateNode {
	return state.NewFunctionNode(func(ctx context.Context, s state.State) (state.State, error) {
		start := time.Now()
		defer logNodeTiming(runtime.Logger(), "score", start)

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

func buildClassificationPrompt(docName string, detections []PageDetection) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Document: %s\n", docName)
	fmt.Fprintf(&sb, "Total Pages: %d\n\n", len(detections))
	sb.WriteString("Detections by Page:\n")

	for _, d := range detections {
		fmt.Fprintf(&sb, "\nPage %d (clarity: %.2f):\n", d.PageNumber, d.ClarityScore)
		if len(d.MarkingsFound) == 0 {
			sb.WriteString("  No markings detected\n")
		} else {
			for _, m := range d.MarkingsFound {
				fmt.Fprintf(
					&sb,
					"  - %s [%s] (legibility: %.2f, faded: %v)\n",
					m.Text, m.Location, m.Legibility, m.Faded,
				)
			}
		}
	}

	sb.WriteString("\nDetermine the overall document classification.")
	return sb.String()
}

func buildDataURI(data []byte, contentType string) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded)
}

func buildScoringPrompt(detections []PageDetection, classification ClassificationResult, enhancementApplied bool) string {
	var sb strings.Builder
	sb.WriteString("Classification Result:\n")
	fmt.Fprintf(&sb, "  Classification: %s\n", classification.Classification)
	fmt.Fprintf(&sb, "  Alternatives: %d\n", len(classification.AlternativeReadings))
	fmt.Fprintf(&sb, "  Rationale: %s\n\n", classification.Rationale)

	fmt.Fprintf(&sb, "Enhancement Applied: %v\n\n", enhancementApplied)

	sb.WriteString("Detection Summary:\n")
	var totalClarity, totalLegibility float64
	var markingCount int
	markings := make(map[string]int)
	headerFooterPages := 0

	for _, d := range detections {
		totalClarity += d.ClarityScore
		hasHeaderFooter := false
		for _, m := range d.MarkingsFound {
			totalLegibility += m.Legibility
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
	spatialCoverage := 0.0
	if len(detections) > 0 {
		avgClarity = totalClarity / float64(len(detections))
		spatialCoverage = float64(headerFooterPages) / float64(len(detections))
	}
	avgLegibility := 0.0
	if markingCount > 0 {
		avgLegibility = totalLegibility / float64(markingCount)
	}

	fmt.Fprintf(&sb, "  Pages: %d\n", len(detections))
	fmt.Fprintf(&sb, "  Average Clarity: %.2f\n", avgClarity)
	fmt.Fprintf(&sb, "  Average Legibility: %.2f\n", avgLegibility)
	fmt.Fprintf(&sb, "  Spatial Coverage: %.2f\n", spatialCoverage)
	fmt.Fprintf(&sb, "  Unique Markings: %d\n", len(markings))

	sb.WriteString("\nEvaluate confidence and provide recommendation")
	return sb.String()
}

func detectionParallelConfig() config.ParallelConfig {
	cfg := config.DefaultParallelConfig()
	cfg.Observer = "noop"
	return cfg
}

func extractEnhanceOptions(stage *profiles.ProfileStage) EnhanceOptions {
	opts := DefaultEnhanceOptions()
	if stage == nil || len(stage.Options) == 0 {
		return opts
	}
	json.Unmarshal(stage.Options, &opts)
	opts.LegibilityThreshold = clamp(opts.LegibilityThreshold, 0.0, 1.0)
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
			if (om.Faded || om.Legibility < threshold) && em.Legibility > om.Legibility {
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
