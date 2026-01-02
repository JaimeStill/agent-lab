// Package classify implements a document security marking classification workflow.
// It uses parallel vision analysis to detect security markings across document pages.
package classify

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"

	"github.com/JaimeStill/agent-lab/internal/images"
	"github.com/JaimeStill/agent-lab/internal/workflows"
	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
	wf "github.com/JaimeStill/go-agents-orchestration/pkg/workflows"
)

// PageImage represents a rendered document page ready for vision analysis.
type PageImage struct {
	PageNumber int       `json:"page_number"`
	ImageID    uuid.UUID `json:"image_id"`
}

// PageDetection contains detection results for a single document page.
type PageDetection struct {
	PageNumber       int               `json:"page_number"`
	MarkingsFound    []MarkingInfo     `json:"markings_found"`
	ClarityScore     float64           `json:"clarity_score"`
	FilterSuggestion *FilterSuggestion `json:"filter_suggestion,omitempty"`
}

// MarkingInfo describes a detected security marking on a document page.
type MarkingInfo struct {
	Text       string  `json:"text"`
	Location   string  `json:"location"`
	Confidence float64 `json:"confidence"`
	Faded      bool    `json:"faded"`
}

// FilterSuggestion recommends image enhancement settings for improved detection.
type FilterSuggestion struct {
	Brightness *int `json:"brightness,omitempty"`
	Contrast   *int `json:"contrast,omitempty"`
	Saturation *int `json:"saturation,omitempty"`
}

// NeedsEnhancement returns true if the page clarity is below the threshold
// and a filter suggestion is available.
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

		pages, ok := s.Get("page_images")
		if !ok {
			return s, fmt.Errorf("page-images not found in state")
		}
		pageImages := pages.([]PageImage)

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

			detection, err := ParseDetectionResponse(resp.Content())
			if err != nil {
				return PageDetection{}, err
			}

			detection.PageNumber = img.PageNumber

			return detection, nil
		}

		cfg := detectionParallelConfig()
		result, err := wf.ProcessParallel(ctx, cfg, pageImages, processor, nil)
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
