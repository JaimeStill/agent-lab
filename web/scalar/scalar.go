// Package docs provides the interactive API documentation handler using Scalar UI.
// Assets are embedded at compile time for zero-dependency deployment.
package scalar

import (
	_ "embed"
	"net/http"

	"github.com/JaimeStill/agent-lab/pkg/routes"
)

//go:embed index.html
var indexHTML []byte

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexHTML)
	}
}

func Routes() routes.Group {
	return routes.Group{
		Prefix: "/scalar",
		Routes: []routes.Route{
			{Method: "GET", Pattern: "", Handler: Handler()},
		},
	}
}
