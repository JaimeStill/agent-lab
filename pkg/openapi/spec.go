package openapi

import "net/http"

// Spec represents a complete OpenAPI 3.1 specification document.
type Spec struct {
	OpenAPI    string               `json:"openapi"`
	Info       *Info                `json:"info"`
	Servers    []*Server            `json:"servers,omitempty"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

func NewSpec(title, version string) *Spec {
	return &Spec{
		OpenAPI: "3.1.0",
		Info: &Info{
			Title:   title,
			Version: version,
		},
		Components: NewComponents(),
		Paths:      make(map[string]*PathItem),
	}
}

func (s *Spec) AddServer(url string) {
	s.Servers = append(s.Servers, &Server{URL: url})
}

func (s *Spec) SetDescription(desc string) {
	s.Info.Description = desc
}

func ServeSpec(specBytes []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(specBytes)
	}
}
