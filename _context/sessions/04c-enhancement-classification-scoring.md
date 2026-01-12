# Session 4c: Enhancement, Classification, and Scoring

## Summary

Extended the classify-docs workflow from `[init] → [detect]` to the complete graph with conditional enhancement routing:

```
[init] → [detect] → [enhance]? → [classify] → [score]
                        │
              (conditional: skip if no pages need enhancement)
```

## What Was Implemented

### New Types (`workflows/classify/classify.go`)

- **ClassificationResult** - Overall document classification with alternatives
- **AlternativeReading** - Possible alternative classification with probability
- **ConfidenceAssessment** - Weighted scoring with recommendation (ACCEPT/REVIEW/REJECT)
- **ConfidenceFactor** - Individual scoring factor (name, score, weight, description)
- **EnhanceOptions** - Enhancement stage configuration (clarity_threshold)
- **DefaultClarityThreshold** constant (0.7)

### Updated Types

- **PageDetection** - Added `OriginalImageID` and `EnhancedImageID` fields for tracking both original and enhanced image references

### New Nodes

**enhanceNode:**
- Filters pages needing enhancement (clarity < threshold AND filter suggestion available)
- Re-renders pages with suggested filter adjustments
- Re-detects markings on enhanced images
- Intelligent merge: replaces markings only if (faded OR low confidence) AND enhanced.confidence > original.confidence
- Tracks both original and enhanced image IDs

**classifyNode:**
- Builds classification prompt from all detections
- Uses text chat (not vision) for classification
- Parses ClassificationResult with alternatives

**scoreNode:**
- Builds scoring prompt with detections, classification, enhancement status
- Evaluates 6 weighted factors (marking_clarity, marking_consistency, spatial_coverage, enhancement_impact, alternative_count, detection_confidence)
- Returns recommendation: ACCEPT (>=0.90), REVIEW (0.70-0.89), REJECT (<0.70)

### Conditional Routing

```go
graph.AddEdge("detect", "enhance", state.KeyEquals("needs_enhancement", true))
graph.AddEdge("detect", "classify", state.KeyEquals("needs_enhancement", false))
```

### Generic Parsing (`workflows/classify/parse.go`)

- `parseResponse[T any]` - Generic parser with validation callback
- `clamp` helper for value range enforcement
- `ParseClassificationResponse` and `ParseScoringResponse`
- `computeRecommendation` for fallback when LLM returns invalid recommendation

### Profile Updates (`workflows/classify/profile.go`)

- Added `ClassificationSystemPrompt` and `ScoringSystemPrompt`
- `DefaultProfile()` now includes 5 stages: init, detect, enhance, classify, score
- Enhance stage includes JSONB options with default clarity threshold

### Workflow Refactoring

Extracted node definitions as standalone functions returning `state.StateNode`:
- `workflows/classify/classify.go` - All nodes extracted
- `workflows/summarize/summarize.go` - Refactored to match pattern
- `workflows/reasoning/reasoning.go` - Refactored to match pattern

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Conditional edge routing | Use `state.KeyEquals` predicates for clean graph wiring |
| Single PageDetection per page | Track both OriginalImageID and EnhancedImageID in same struct |
| Intelligent merge | Replace only if (faded OR low confidence) AND enhanced > original |
| Profile-configurable threshold | JSONB options allow per-profile clarity threshold |
| Text chat for classify/score | Classification and scoring don't need vision |
| Generic parseResponse | Reduces duplication across parse functions |

## Files Changed

### Modified Files
- `workflows/classify/classify.go` - Types, nodes, factory, helpers
- `workflows/classify/profile.go` - System prompts, DefaultProfile
- `workflows/classify/parse.go` - Generic parser, new parse functions
- `workflows/classify/errors.go` - New error types
- `workflows/summarize/summarize.go` - Refactored to node pattern
- `workflows/reasoning/reasoning.go` - Refactored to node pattern

### New Test Files
- `tests/workflows_classify/parse_test.go` - Tests for all parse functions

## Validation Results

Tested with Azure GPT-5-mini against a challenging document (marked-documents_19.pdf) with faded NOFORN stamp:

**Run 1** (clarity 0.68, enhancement triggered):
- Classification: SECRET with alternative SECRET//NOFORN (55%)
- NOFORN detected (60% confidence, faded)
- Recommendation: REVIEW (0.765)

**Run 2** (clarity 0.78, no enhancement):
- Classification: SECRET
- NOFORN not detected
- Recommendation: REVIEW (0.836)

Key observation: LLM clarity assessment is non-deterministic. Enhancement improves detection when triggered.

## Notes for Session 4d

**Runtime Optimization Required:**
- ~50 seconds for 1-page document is too long
- Investigate Vision API latency, parallel processing efficiency
- Consider caching strategies

**Detection Accuracy Improvements:**
- Lower default clarity threshold (e.g., 0.8) to trigger enhancement more aggressively
- Trigger enhancement if ANY marking has `faded: true`, regardless of clarity score
- Add faded marking count as enhancement decision factor
- Consider multi-pass detection with result aggregation
- Tune system prompts for better faded marking detection
