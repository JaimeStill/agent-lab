package routes

import "net/http"

// Group represents a collection of routes with a common prefix.
type Group struct {
	Prefix      string
	Tags        []string
	Description string
	Routes      []Route
}

// Route represents an HTTP route with method, pattern, and handler.
type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}
