package agents

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/JaimeStill/go-agents/pkg/agent"
)

// ChatRequest contains the data for chat execution requests.
type ChatRequest struct {
	Prompt  string         `json:"prompt"`
	Options map[string]any `json:"options,omitempty"`
	Token   string         `json:"token,omitempty"`
}

// ToolsRequest contains the data for tool-calling execution requests.
type ToolsRequest struct {
	Prompt  string         `json:"prompt"`
	Tools   []agent.Tool   `json:"tools"`
	Options map[string]any `json:"options,omitempty"`
	Token   string         `json:"token,omitempty"`
}

// EmbedRequest contains the data for embedding execution requests.
type EmbedRequest struct {
	Input   string         `json:"input"`
	Options map[string]any `json:"options,omitempty"`
	Token   string         `json:"token,omitempty"`
}

// VisionForm contains the parsed multipart form data for vision requests.
type VisionForm struct {
	Prompt  string
	Images  []string
	Options map[string]any
	Token   string
}

// ParseVisionForm parses a multipart form request into a VisionForm.
// It validates required fields and converts uploaded images to base64 data URIs.
func ParseVisionForm(r *http.Request, maxMemory int64) (*VisionForm, error) {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	form := &VisionForm{
		Prompt: r.FormValue("prompt"),
		Token:  r.FormValue("token"),
	}

	if form.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	if optStr := r.FormValue("options"); optStr != "" {
		if err := json.Unmarshal([]byte(optStr), &form.Options); err != nil {
			return nil, fmt.Errorf("invalid options JSON: %w", err)
		}
	}

	files := r.MultipartForm.File["images"]
	if len(files) <= 0 {
		return nil, fmt.Errorf("at least one image is required")
	}

	images, err := prepareImages(files)
	if err != nil {
		return nil, err
	}
	form.Images = images

	return form, nil
}

func prepareImages(files []*multipart.FileHeader) ([]string, error) {
	prepared := make([]string, len(files))

	for i, fh := range files {
		file, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", fh.Filename, err)
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", fh.Filename, err)
		}

		mimeType := http.DetectContentType(data)
		if !strings.HasPrefix(mimeType, "image/") {
			return nil, fmt.Errorf("file %s is not an image (detected: %s)", fh.Filename, mimeType)
		}

		encoded := base64.StdEncoding.EncodeToString(data)
		prepared[i] = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
	}

	return prepared, nil
}
