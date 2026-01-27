package logging

import "os"

// Env maps environment variable names for logging configuration.
type Env struct {
	Level  string
	Format string
}

// Config holds logging configuration settings.
type Config struct {
	Level  Level  `toml:"level"`
	Format Format `toml:"format"`
}

// Finalize applies defaults, loads environment overrides, and validates the configuration.
func (c *Config) Finalize(env *Env) error {
	c.loadDefaults()
	c.loadEnv(env)
	return c.validate()
}

// Merge applies non-zero values from the overlay configuration.
func (c *Config) Merge(overlay *Config) {
	if overlay.Level != "" {
		c.Level = overlay.Level
	}
	if overlay.Format != "" {
		c.Format = overlay.Format
	}
}

func (c *Config) loadDefaults() {
	if c.Level == "" {
		c.Level = LevelInfo
	}
	if c.Format == "" {
		c.Format = FormatText
	}
}

func (c *Config) loadEnv(env *Env) {
	if env == nil {
		return
	}
	if v := os.Getenv(env.Level); v != "" {
		c.Level = Level(v)
	}
	if v := os.Getenv(env.Format); v != "" {
		c.Format = Format(v)
	}
}

func (c *Config) validate() error {
	if err := c.Level.Validate(); err != nil {
		return err
	}
	return c.Format.Validate()
}
