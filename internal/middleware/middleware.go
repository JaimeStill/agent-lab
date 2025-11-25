// Package middleware provides HTTP middleware management and application.
package middleware

import "net/http"

// System manages a stack of HTTP middleware functions.
type System interface {
	Use(mw func(http.Handler) http.Handler)
	Apply(handler http.Handler) http.Handler
}

type middleware struct {
	stack []func(http.Handler) http.Handler
}

// New creates a middleware system.
func New() System {
	return &middleware{
		stack: []func(http.Handler) http.Handler{},
	}
}

// Use adds a middleware function to the stack.
func (m *middleware) Use(mw func(http.Handler) http.Handler) {
	m.stack = append(m.stack, mw)
}

// Apply wraps the handler with all middleware in the stack, applying them in reverse order.
func (m *middleware) Apply(handler http.Handler) http.Handler {
	for i := len(m.stack) - 1; i >= 0; i-- {
		handler = m.stack[i](handler)
	}
	return handler
}
