package providers

import (
	"log/slog"
	"net/http"

	"github.com/JaimeStill/agent-lab/internal/routes"
)

// Routes returns a route group for the providers API endpoints.
// The system and logger are captured in handler closures.
func Routes(sys System, logger *slog.Logger) routes.Group {
	return routes.Group{
		Prefix:      "/api/providers",
		Tags:        []string{"Providers"},
		Description: "Provider configuration management",
		Routes: []routes.Route{
			{
				Method:  "POST",
				Pattern: "",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleCreate(w, r, sys, logger)
				},
			},
			{
				Method:  "GET",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleGetByID(w, r, sys, logger)
				},
			},
			{
				Method:  "PUT",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleUpdate(w, r, sys, logger)
				},
			},
			{
				Method:  "DELETE",
				Pattern: "/{id}",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleDelete(w, r, sys, logger)
				},
			},
			{
				Method:  "POST",
				Pattern: "/search",
				Handler: func(w http.ResponseWriter, r *http.Request) {
					HandleSearch(w, r, sys, logger)
				},
			},
		},
	}
}
