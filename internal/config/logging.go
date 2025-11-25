package config

import "os"

const (
	// EnvLoggingLevel overrides the logging level.
	EnvLoggingLevel = "LOGGING_LEVEL"

	// EnvLoggingFormat overrides the logging format.
	EnvLoggingFormat = "LOGGING_FORMAT"
)

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level  LogLevel  `toml:"level"`
	Format LogFormat `toml:"format"`
}

// Finalize applies defaults, loads environment overrides, and validates the logging configuration.
func (c *LoggingConfig) Finalize() error {
	c.loadDefaults()
	c.loadEnv()
	return c.validate()
}

// Merge applies values from overlay configuration that differ from zero values.
func (c *LoggingConfig) Merge(overlay *LoggingConfig) {
	if overlay.Level != "" {
		c.Level = overlay.Level
	}
	if overlay.Format != "" {
		c.Format = overlay.Format
	}
}

func (c *LoggingConfig) loadEnv() {
	if v := os.Getenv(EnvLoggingLevel); v != "" {
		c.Level = LogLevel(v)
	}
	if v := os.Getenv(EnvLoggingFormat); v != "" {
		c.Format = LogFormat(v)
	}
}

func (c *LoggingConfig) loadDefaults() {
	if c.Level == "" {
		c.Level = LogLevelInfo
	}
	if c.Format == "" {
		c.Format = LogFormatJSON
	}
}

func (c *LoggingConfig) validate() error {
	if err := c.Level.Validate(); err != nil {
		return err
	}
	if err := c.Format.Validate(); err != nil {
		return err
	}
	return nil
}
