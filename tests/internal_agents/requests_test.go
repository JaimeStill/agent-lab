package internal_agents_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JaimeStill/agent-lab/internal/agents"
)

func TestParseVisionForm_MissingPrompt(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("images", "test.png")
	part.Write(createTestPNG())
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/vision", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := agents.ParseVisionForm(req, 32<<20)
	if err == nil {
		t.Error("ParseVisionForm() expected error for missing prompt")
	}
	if !strings.Contains(err.Error(), "prompt is required") {
		t.Errorf("ParseVisionForm() error = %q, want to contain 'prompt is required'", err.Error())
	}
}

func TestParseVisionForm_MissingImages(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("prompt", "describe this image")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/vision", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := agents.ParseVisionForm(req, 32<<20)
	if err == nil {
		t.Error("ParseVisionForm() expected error for missing images")
	}
	if !strings.Contains(err.Error(), "at least one image is required") {
		t.Errorf("ParseVisionForm() error = %q, want to contain 'at least one image is required'", err.Error())
	}
}

func TestParseVisionForm_InvalidOptionsJSON(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("prompt", "describe this image")
	writer.WriteField("options", "not valid json")
	part, _ := writer.CreateFormFile("images", "test.png")
	part.Write(createTestPNG())
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/vision", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := agents.ParseVisionForm(req, 32<<20)
	if err == nil {
		t.Error("ParseVisionForm() expected error for invalid options JSON")
	}
	if !strings.Contains(err.Error(), "invalid options JSON") {
		t.Errorf("ParseVisionForm() error = %q, want to contain 'invalid options JSON'", err.Error())
	}
}

func TestParseVisionForm_NonImageFile(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("prompt", "describe this image")
	part, _ := writer.CreateFormFile("images", "test.txt")
	part.Write([]byte("this is not an image"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/vision", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := agents.ParseVisionForm(req, 32<<20)
	if err == nil {
		t.Error("ParseVisionForm() expected error for non-image file")
	}
	if !strings.Contains(err.Error(), "is not an image") {
		t.Errorf("ParseVisionForm() error = %q, want to contain 'is not an image'", err.Error())
	}
}

func TestParseVisionForm_Success(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("prompt", "describe this image")
	writer.WriteField("token", "test-token")
	writer.WriteField("options", `{"temperature": 0.7}`)
	part, _ := writer.CreateFormFile("images", "test.png")
	part.Write(createTestPNG())
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/vision", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	form, err := agents.ParseVisionForm(req, 32<<20)
	if err != nil {
		t.Fatalf("ParseVisionForm() unexpected error: %v", err)
	}

	if form.Prompt != "describe this image" {
		t.Errorf("ParseVisionForm() Prompt = %q, want %q", form.Prompt, "describe this image")
	}

	if form.Token != "test-token" {
		t.Errorf("ParseVisionForm() Token = %q, want %q", form.Token, "test-token")
	}

	if form.Options == nil {
		t.Error("ParseVisionForm() Options = nil, want non-nil")
	} else if form.Options["temperature"] != 0.7 {
		t.Errorf("ParseVisionForm() Options[temperature] = %v, want 0.7", form.Options["temperature"])
	}

	if len(form.Images) != 1 {
		t.Errorf("ParseVisionForm() len(Images) = %d, want 1", len(form.Images))
	}

	if !strings.HasPrefix(form.Images[0], "data:image/png;base64,") {
		t.Errorf("ParseVisionForm() Images[0] should start with data:image/png;base64,, got %q", form.Images[0][:50])
	}
}

func TestParseVisionForm_MultipleImages(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("prompt", "compare these images")
	part1, _ := writer.CreateFormFile("images", "test1.png")
	part1.Write(createTestPNG())
	part2, _ := writer.CreateFormFile("images", "test2.png")
	part2.Write(createTestPNG())
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/vision", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	form, err := agents.ParseVisionForm(req, 32<<20)
	if err != nil {
		t.Fatalf("ParseVisionForm() unexpected error: %v", err)
	}

	if len(form.Images) != 2 {
		t.Errorf("ParseVisionForm() len(Images) = %d, want 2", len(form.Images))
	}
}

func createTestPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F,
		0x00, 0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59,
		0xE7, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
}
