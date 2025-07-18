// Package geminiwebtools provides OAuth2-authenticated WebFetch and WebSearch tools
// compatible with the gemini-cli TypeScript implementation.
package geminiwebtools

import (
	"time"

	"github.com/d-kuro/geminiwebtools/pkg/auth"
	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// Config holds all configuration options for the web tools library.
// This design allows all parameters to be configurable while maintaining
// compatibility with the gemini-cli zero-configuration approach.
type Config struct {
	// API Configuration
	CodeAssistEndpoint string            `json:"codeAssistEndpoint,omitempty"`
	GeminiAPIEndpoint  string            `json:"geminiApiEndpoint,omitempty"`
	OAuth2Config       auth.OAuth2Config `json:"oauth2Config,omitempty"`

	// Model Configuration
	DefaultModel string `json:"defaultModel,omitempty"`

	// HTTP Configuration
	Timeout        time.Duration `json:"timeout,omitempty"`
	MaxContentSize int           `json:"maxContentSize,omitempty"`

	// Cache Configuration (for future extension)
	CacheEnabled bool          `json:"cacheEnabled,omitempty"`
	CacheSize    int           `json:"cacheSize,omitempty"`
	CacheTTL     time.Duration `json:"cacheTTL,omitempty"`

	// Credential Storage
	CredentialStore storage.CredentialStore `json:"-"` // Not serialized

	// Processing Configuration
	CitationStyle string `json:"citationStyle,omitempty"`
	MaxSources    int    `json:"maxSources,omitempty"`

	// Tool-specific Configuration
	WebFetch  WebFetchConfig  `json:"webFetch,omitempty"`
	WebSearch WebSearchConfig `json:"webSearch,omitempty"`
}

// WebFetchConfig holds WebFetch-specific configuration options.
type WebFetchConfig struct {
	// Content processing options
	ConvertHTML     bool `json:"convertHtml,omitempty"`
	TruncateContent bool `json:"truncateContent,omitempty"`
	TruncateLength  int  `json:"truncateLength,omitempty"`

	// Security options
	AllowPrivateIPs bool `json:"allowPrivateIps,omitempty"`
	FollowRedirects bool `json:"followRedirects,omitempty"`

	// Fallback behavior
	EnableFallback  bool          `json:"enableFallback,omitempty"`
	FallbackTimeout time.Duration `json:"fallbackTimeout,omitempty"`
}

// WebSearchConfig holds WebSearch-specific configuration options.
type WebSearchConfig struct {
	// Search behavior
	MaxResults     int      `json:"maxResults,omitempty"`
	DefaultDomains []string `json:"defaultDomains,omitempty"`

	// Citation processing
	InsertCitations bool   `json:"insertCitations,omitempty"`
	CitationFormat  string `json:"citationFormat,omitempty"`
}

// ConfigOption defines a functional option for configuring the Config.
type ConfigOption func(*Config)

// WithCredentialStore sets a custom credential store.
func WithCredentialStore(store storage.CredentialStore) ConfigOption {
	return func(c *Config) {
		c.CredentialStore = store
	}
}

// WithTimeout sets the HTTP timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithMaxContentSize sets the maximum content size.
func WithMaxContentSize(size int) ConfigOption {
	return func(c *Config) {
		c.MaxContentSize = size
	}
}

// NewConfig creates a new configuration with the provided options.
// If no options are provided, returns a configuration with sensible defaults
// that match the gemini-cli implementation behavior.
func NewConfig(opts ...ConfigOption) *Config {
	// Start with default configuration
	config := &Config{
		// API endpoints (matching gemini-cli defaults)
		CodeAssistEndpoint: constants.DefaultCodeAssistEndpoint,
		GeminiAPIEndpoint:  constants.DefaultGeminiAPIEndpoint,

		// OAuth2 configuration (matching gemini-cli)
		OAuth2Config: auth.OAuth2Config{
			ClientID:     constants.DefaultOAuthClientID,
			ClientSecret: constants.DefaultOAuthClientSecret,
			AuthURL:      constants.DefaultOAuthAuthURL,
			TokenURL:     constants.DefaultOAuthTokenURL,
			Scopes:       constants.DefaultOAuthScopes,
		},

		// Model configuration
		DefaultModel: constants.DefaultModelName,

		// HTTP configuration (matching gemini-cli timeouts)
		Timeout:        constants.DefaultHTTPTimeout,
		MaxContentSize: constants.DefaultMaxContentSize,

		// Cache configuration (disabled by default for compatibility)
		CacheEnabled: false,
		CacheSize:    100,
		CacheTTL:     constants.DefaultCacheTTL,

		// Processing configuration
		CitationStyle: constants.DefaultCitationStyle,
		MaxSources:    constants.DefaultMaxSources,

		// WebFetch defaults (matching gemini-cli behavior)
		WebFetch: WebFetchConfig{
			ConvertHTML:     true,
			TruncateContent: true,
			TruncateLength:  constants.DefaultTruncateLength,
			AllowPrivateIPs: false,
			FollowRedirects: true,
			EnableFallback:  true,
			FallbackTimeout: constants.DefaultFallbackTimeout,
		},

		// WebSearch defaults
		WebSearch: WebSearchConfig{
			MaxResults:      constants.DefaultMaxSearchResults,
			InsertCitations: true,
			CitationFormat:  constants.DefaultCitationStyle,
		},

		// Set default credential store (use filesystem store for gemini-cli compatibility)
		CredentialStore: storage.MustNewFileSystemStore(""),
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Validate ensures the configuration is valid and complete.
func (c *Config) Validate() error {
	if c.CodeAssistEndpoint == "" {
		return &ConfigError{Field: "CodeAssistEndpoint", Message: constants.ValidationErrorEmpty}
	}
	if c.GeminiAPIEndpoint == "" {
		return &ConfigError{Field: "GeminiAPIEndpoint", Message: constants.ValidationErrorEmpty}
	}
	if c.OAuth2Config.ClientID == "" {
		return &ConfigError{Field: "OAuth2Config.ClientID", Message: constants.ValidationErrorEmpty}
	}
	if c.OAuth2Config.ClientSecret == "" {
		return &ConfigError{Field: "OAuth2Config.ClientSecret", Message: constants.ValidationErrorEmpty}
	}
	if c.CredentialStore == nil {
		return &ConfigError{Field: "CredentialStore", Message: constants.ValidationErrorRequired}
	}
	return nil
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return constants.ConfigErrorPrefix + e.Field + ": " + e.Message
}
