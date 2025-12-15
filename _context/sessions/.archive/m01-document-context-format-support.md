# Maintenance Session m01: document-context Format Support

## Overview

This maintenance session migrates format support functionality from agent-lab shims to the document-context library, and improves agent config validation using go-agents' native patterns.

## Prerequisites

- document-context repository at `~/code/document-context`
- agent-lab repository at `~/code/agent-lab`
- Go 1.25.2+

---

## Phase 1: document-context Changes

### 1.1 Add ParseImageFormat Function

**File:** `~/code/document-context/pkg/document/document.go`

Add import:

```go
"strings"
```

Add function after the `MimeType()` method:

```go
func ParseImageFormat(s string) (ImageFormat, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "png":
		return PNG, nil
	case "jpg", "jpeg":
		return JPEG, nil
	default:
		return "", fmt.Errorf("unsupported image format: %s", s)
	}
}
```

### 1.2 Add Format Registry

**File:** `~/code/document-context/pkg/document/document.go`

Add after the interfaces (at end of file):

```go
var formatRegistry = map[string]func(string) (Document, error){
	"application/pdf": func(path string) (Document, error) {
		return OpenPDF(path)
	},
}

func SupportedFormats() []string {
	formats := make([]string, 0, len(formatRegistry))
	for contentType := range formatRegistry {
		formats = append(formats, contentType)
	}
	return formats
}

func IsSupported(contentType string) bool {
	_, ok := formatRegistry[contentType]
	return ok
}

func Open(path string, contentType string) (Document, error) {
	opener, ok := formatRegistry[contentType]
	if !ok {
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
	return opener(path)
}
```

### 1.3 Validate Changes

```bash
cd ~/code/document-context
go vet ./...
```

### 1.4 Commit and Tag

```bash
git add .
git commit -m "feat: add ParseImageFormat and format registry functions"
git tag v0.1.1
git push origin main --tags
```

---

## Phase 2: agent-lab Shim Removal

### 2.1 Update go.mod

```bash
cd ~/code/agent-lab
go get github.com/JaimeStill/document-context@v0.1.1
go mod tidy
```

### 2.2 Delete Shim File

Delete the file:

```bash
rm internal/images/document.go
```

### 2.3 Update repository.go

**File:** `internal/images/repository.go`

Update the import block to add `document` import:

```go
"github.com/JaimeStill/document-context/pkg/document"
```

Change line ~105 from:

```go
if !IsSupported(doc.ContentType) {
```

To:

```go
if !document.IsSupported(doc.ContentType) {
```

Change line ~128 from:

```go
openDoc, err := OpenDocument(docPath, doc.ContentType)
```

To:

```go
openDoc, err := document.Open(docPath, doc.ContentType)
```

Change line ~173 function signature from:

```go
func (r *repo) renderPage(ctx context.Context, documentID uuid.UUID, doc PageExtractor, renderer image.Renderer, pageNum int, opts RenderOptions) (*Image, error) {
```

To:

```go
func (r *repo) renderPage(ctx context.Context, documentID uuid.UUID, doc document.Document, renderer image.Renderer, pageNum int, opts RenderOptions) (*Image, error) {
```

### 2.4 Update image.go

**File:** `internal/images/image.go`

Change line ~49 in `Validate()` method from:

```go
format, err := ParseImageFormat(string(o.Format))
if err != nil {
    return err
}
```

To:

```go
format, err := document.ParseImageFormat(string(o.Format))
if err != nil {
    return fmt.Errorf("%w: format must be 'png' or 'jpg'", ErrInvalidRenderOption)
}
```

**Remove** the local `ParseImageFormat` function (lines 146-159).

### 2.5 Update mapping.go

**File:** `internal/images/mapping.go`

Change line ~76 from:

```go
if parsed, err := ParseImageFormat(format); err == nil {
```

To:

```go
if parsed, err := document.ParseImageFormat(format); err == nil {
```

### 2.6 Validate Changes

```bash
go vet ./...
```

---

## Phase 3: Config Pattern Improvement

### 3.1 Update validateConfig

**File:** `internal/agents/repository.go`

Replace the `validateConfig` function (lines ~131-141):

```go
func (r *repo) validateConfig(config json.RawMessage) error {
	cfg := agtconfig.DefaultAgentConfig()

	var userCfg agtconfig.AgentConfig
	if err := json.Unmarshal(config, &userCfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	cfg.Merge(&userCfg)

	if _, err := agent.New(&cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	return nil
}
```

### 3.2 Update constructAgent

**File:** `internal/agents/handler.go`

Replace lines ~344-347 in `constructAgent` function:

From:

```go
var cfg agtconfig.AgentConfig
if err := json.Unmarshal(record.Config, &cfg); err != nil {
    return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
}
```

To:

```go
cfg := agtconfig.DefaultAgentConfig()

var storedCfg agtconfig.AgentConfig
if err := json.Unmarshal(record.Config, &storedCfg); err != nil {
    return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
}

cfg.Merge(&storedCfg)
```

### 3.3 Validate Changes

```bash
go vet ./...
```

---

## Validation Criteria

### Phase 1 (document-context)
- [ ] `go vet ./...` passes
- [ ] Version tagged v0.1.1

### Phase 2 (agent-lab images)
- [ ] `go mod tidy` succeeds
- [ ] `go vet ./...` passes
- [ ] `internal/images/document.go` deleted
- [ ] All references use `document.` prefix

### Phase 3 (agent-lab agents)
- [ ] `go vet ./...` passes
- [ ] Partial configs validate correctly (defaults applied)

---

## Files Modified

### document-context
| File | Action |
|------|--------|
| `pkg/document/document.go` | Modify - add ParseImageFormat, format registry |

### agent-lab
| File | Action |
|------|--------|
| `go.mod` | Modify - update document-context to v0.1.1 |
| `internal/images/document.go` | Delete |
| `internal/images/repository.go` | Modify - use document.Open, document.IsSupported, document.Document |
| `internal/images/image.go` | Modify - use document.ParseImageFormat, remove local function |
| `internal/images/mapping.go` | Modify - use document.ParseImageFormat |
| `internal/agents/repository.go` | Modify - Default + Merge pattern in validateConfig |
| `internal/agents/handler.go` | Modify - Default + Merge pattern in constructAgent |
