package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	// EnvCORSEnabled overrides the CORS enabled flag.
	EnvCORSEnabled = "CORS_ENABLED"

	// EnvCORSOrigins overrides the allowed CORS origins (comma-separated).
	EnvCORSOrigins = "CORS_ORIGINS"

	// EnvCORSAllowedMethods overrides the allowed HTTP methods (comma-separated).
	EnvCORSAllowedMethods = "CORS_ALLOWED_METHODS"

	// EnvCORSAllowedHeaders overrides the allowed HTTP headers (comma-separated).
	EnvCORSAllowedHeaders = "CORS_ALLOWED_HEADERS"

	// EnvCORSAllowCredentials overrides the allow credentials flag.
	EnvCORSAllowCredentials = "CORS_ALLOW_CREDENTIALS"

	// EnvCORSMaxAge overrides the preflight cache duration in seconds.
	EnvCORSMaxAge = "CORS_MAX_AGE"
)

// CORSConfig contains Cross-Origin Resource Sharing configuration.
type CORSConfig struct {
	Enabled          bool     `toml:"enabled"`
	Origins          []string `toml:"origins"`
	AllowedMethods   []string `toml:"allowed_methods"`
	AllowedHeaders   []string `toml:"allowed_headers"`
	AllowCredentials bool     `toml:"allow_credentials"`
	MaxAge           int      `toml:"max_age"`
}

// Finalize applies defaults, loads environment overrides, and validates the CORS configuration.
func (c *CORSConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return nil
}

// Merge applies values from overlay configuration, including boolean and array fields.
func (c *CORSConfig) Merge(overlay *CORSConfig) {
	c.Enabled = overlay.Enabled
	c.AllowCredentials = overlay.AllowCredentials

	if overlay.Origins != nil {
		c.Origins = overlay.Origins
	}
	if overlay.AllowedMethods != nil {
		c.AllowedMethods = overlay.AllowedMethods
	}
	if overlay.AllowedHeaders != nil {
		c.AllowedHeaders = overlay.AllowedHeaders
	}
	if overlay.MaxAge >= 0 {
		c.MaxAge = overlay.MaxAge
	}
}

func (c *CORSConfig) loadDefaults() {
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 3600
	}
}

func (c *CORSConfig) loadEnv() {
	if v := os.Getenv(EnvCORSEnabled); v != "" {
		if enabled, err := strconv.ParseBool(v); err == nil {
			c.Enabled = enabled
		}
	}

	if v := os.Getenv(EnvCORSOrigins); v != "" {
		origins := strings.Split(v, ",")
		c.Origins = make([]string, 0, len(origins))
		for _, origin := range origins {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				c.Origins = append(c.Origins, trimmed)
			}
		}
	}

	if v := os.Getenv(EnvCORSAllowedMethods); v != "" {
		methods := strings.Split(v, ",")
		c.AllowedMethods = make([]string, 0, len(methods))
		for _, method := range methods {
			if trimmed := strings.TrimSpace(method); trimmed != "" {
				c.AllowedMethods = append(c.AllowedMethods, trimmed)
			}
		}
	}

	if v := os.Getenv(EnvCORSAllowedHeaders); v != "" {
		headers := strings.Split(v, ",")
		c.AllowedHeaders = make([]string, 0, len(headers))
		for _, header := range headers {
			if trimmed := strings.TrimSpace(header); trimmed != "" {
				c.AllowedHeaders = append(c.AllowedHeaders, trimmed)
			}
		}
	}

	if v := os.Getenv(EnvCORSAllowCredentials); v != "" {
		if creds, err := strconv.ParseBool(v); err == nil {
			c.AllowCredentials = creds
		}
	}

	if v := os.Getenv(EnvCORSMaxAge); v != "" {
		if maxAge, err := strconv.Atoi(v); err == nil {
			c.MaxAge = maxAge
		}
	}
}
