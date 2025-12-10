# document-context Format Support Enhancement

## Context

During Session 2c (document-context integration), we implemented shims in agent-lab for functionality that should live in the document-context library.

## Existing document-context Infrastructure

The library already provides (`pkg/document/document.go`):

```go
type ImageFormat string

const (
    PNG  ImageFormat = "png"
    JPEG ImageFormat = "jpg"
)

func (f ImageFormat) MimeType() (string, error)

type Document interface {
    PageCount() int
    ExtractPage(pageNum int) (Page, error)
    ExtractAllPages() ([]Page, error)
    Close() error
}

type Page interface {
    Number() int
    ToImage(renderer image.Renderer, c cache.Cache) ([]byte, error)
}
```

And (`pkg/encoding/image.go`):

```go
func EncodeImageDataURI(data []byte, format document.ImageFormat) (string, error)
```

## Current Shims in agent-lab

### ImageFormat Parsing (`internal/images/image.go`)

Uses `document.ImageFormat` from document-context, adds local parsing:

```go
func ParseImageFormat(s string) (document.ImageFormat, error) {
    switch strings.ToLower(s) {
    case "png":
        return document.PNG, nil
    case "jpg", "jpeg":
        return document.JPEG, nil
    case "":
        return document.PNG, nil
    default:
        return "", fmt.Errorf("format must be 'png' or 'jpg'")
    }
}
```

### Document Opening (`internal/images/document.go`)

```go
var SupportedDocumentFormats = map[string]bool{
    "application/pdf": true,
}

type PageExtractor interface {
    ExtractPage(pageNum int) (document.Page, error)
    io.Closer
}

func IsDocumentSupported(contentType string) bool
func OpenDocument(path string, contentType string) (PageExtractor, error)
```

## Proposed document-context Enhancements

### 1. ImageFormat Parsing

Add to `pkg/document/document.go`:

```go
func ParseImageFormat(s string) (ImageFormat, error) {
    switch strings.ToLower(strings.TrimSpace(s)) {
    case "png":
        return PNG, nil
    case "jpg", "jpeg":
        return JPEG, nil
    default:
        return "", fmt.Errorf("unsupported image format: %s", s)
    }
}

// ContentType returns the MIME type (alias for MimeType for convenience)
func (f ImageFormat) ContentType() string {
    mt, _ := f.MimeType()
    return mt
}
```

### 2. Document Format Support

Add to `pkg/document/document.go`:

```go
var formatRegistry = map[string]func(string) (Document, error){
    "application/pdf": func(path string) (Document, error) {
        return OpenPDF(path)
    },
}

// SupportedFormats returns content types that can be opened.
func SupportedFormats() []string {
    formats := make([]string, 0, len(formatRegistry))
    for ct := range formatRegistry {
        formats = append(formats, ct)
    }
    return formats
}

// IsSupported checks if a content type can be opened.
func IsSupported(contentType string) bool {
    _, ok := formatRegistry[contentType]
    return ok
}

// Open opens a document based on content type.
func Open(path string, contentType string) (Document, error) {
    opener, ok := formatRegistry[contentType]
    if !ok {
        return nil, fmt.Errorf("unsupported document format: %s", contentType)
    }
    return opener(path)
}
```

## Migration Path

### Session 2d Tasks

1. **Update document-context** (`~/code/document-context`)
   - Add `ParseImageFormat()` function
   - Add `ContentType()` method to ImageFormat
   - Add `Open()`, `IsSupported()`, `SupportedFormats()` functions
   - Add format registry for extensibility
   - Update tests
   - Release new version (v0.2.0)

2. **Update agent-lab**
   - Update `go.mod` to use new document-context version
   - Remove `internal/images/document.go` shim
   - Remove local `ParseImageFormat()` from `internal/images/image.go`
   - Update repository to use `document.Open()` and `document.IsSupported()`
   - Update calls to use `document.ParseImageFormat()`
   - Verify tests pass

## Benefits

- **Single source of truth**: All document/image format handling in document-context
- **Reduced duplication**: No parallel type definitions
- **Extensibility**: Adding formats requires only document-context changes
- **Cleaner agent-lab**: Web service focuses on HTTP concerns, not format dispatch

## Files Affected

### document-context
- `pkg/document/document.go` - Add `ParseImageFormat()`, `ContentType()`, `Open()`, `IsSupported()`, `SupportedFormats()`

### agent-lab (removal)
- `internal/images/document.go` - Delete entirely
- `internal/images/image.go` - Remove local `ParseImageFormat()` function

### agent-lab (update)
- `internal/images/repository.go` - Use `document.Open()`, `document.IsSupported()`
- `internal/images/params.go` - Use `document.ParseImageFormat()`
