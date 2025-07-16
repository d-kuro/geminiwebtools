package constants

import "time"

const (
	LibraryVersion = "0.0.1"
	LibraryName    = "geminiwebtools"

	DefaultCodeAssistEndpoint = "https://cloudcode-pa.googleapis.com"
	DefaultGeminiAPIEndpoint  = "https://generativelanguage.googleapis.com"
	DefaultAPIVersion         = "v1internal"

	// DefaultOAuthClientID referenced from gemini-cli
	// https://github.com/google-gemini/gemini-cli/blob/v0.1.12/packages/core/src/code_assist/oauth2.ts#L32
	DefaultOAuthClientID = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	// DefaultOAuthClientSecret referenced from gemini-cli
	// https://github.com/google-gemini/gemini-cli/blob/v0.1.12/packages/core/src/code_assist/oauth2.ts#L41
	DefaultOAuthClientSecret = "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"

	DefaultOAuthAuthURL  = "https://accounts.google.com/o/oauth2/auth"
	DefaultOAuthTokenURL = "https://oauth2.googleapis.com/token"

	DefaultModelName = "gemini-2.5-flash"

	DefaultHTTPTimeout        = 30 * time.Second
	DefaultDialerTimeout      = 10 * time.Second
	DefaultMaxContentSize     = 5 * 1024 * 1024
	DefaultHTTPMaxContentSize = 10 * 1024 * 1024
	DefaultUserAgent          = "geminiwebtools/1.0"
	MaxURLLength              = 2048              // Maximum URL length for security
	MaxAPIRequestSize         = 1 * 1024 * 1024   // 1MB max request size
	MaxAPIResponseSize        = 10 * 1024 * 1024  // 10MB max response size
	APIRequestTimeout         = 60 * time.Second  // API request timeout
	AIRequestTimeout          = 120 * time.Second // AI request timeout
	HTTPFetchTimeout          = 30 * time.Second  // HTTP fetch timeout
	MaxRedirects              = 5                 // Maximum redirects to follow

	// Connection pool optimizations
	MaxIdleConns        = 100              // Maximum number of idle connections across all hosts
	MaxIdleConnsPerHost = 10               // Maximum idle connections per host
	MaxConnsPerHost     = 100              // Maximum connections per host
	IdleConnTimeout     = 90 * time.Second // How long an idle connection can remain idle

	// Fine-grained timeouts
	TLSHandshakeTimeout   = 10 * time.Second // TLS handshake timeout
	ResponseHeaderTimeout = 30 * time.Second // Response header timeout
	ExpectContinueTimeout = 1 * time.Second  // Expect: 100-continue timeout
	KeepAliveTimeout      = 30 * time.Second // Connection keep-alive timeout

	// API-specific connection limits
	APIMaxIdleConns        = 50               // API-specific maximum idle connections
	APIMaxIdleConnsPerHost = 5                // API-specific maximum idle connections per host
	APIMaxConnsPerHost     = 50               // API-specific maximum connections per host
	APIIdleConnTimeout     = 60 * time.Second // API-specific idle connection timeout

	DefaultCacheTTL = 15 * time.Minute

	DefaultCitationStyle    = "numbered"
	DefaultMaxSources       = 20
	DefaultTruncateLength   = 100000
	DefaultFallbackTimeout  = 10 * time.Second
	DefaultMaxSearchResults = 20
	DefaultMaxCitations     = 10
	DefaultMaxQueryDisplay  = 3

	ContentTypeHTML  = "text/html"
	ContentTypeXHTML = "application/xhtml+xml"
	ContentTypePlain = "text/plain"
	ContentTypeJSON  = "application/json"

	DefaultAcceptHeader         = "text/html,application/xhtml+xml,application/xml;q=0.9,text/plain;q=0.8,*/*;q=0.1"
	DefaultAcceptLanguageHeader = "en-US,en;q=0.9"

	SchemeHTTP  = "http"
	SchemeHTTPS = "https"

	PrivateIPClass10    = 10
	PrivateIPClass172A  = 172
	PrivateIPClass172B  = 16
	PrivateIPClass172C  = 31
	PrivateIPClass192A  = 192
	PrivateIPClass192B  = 168
	PrivateIPLoopback   = 127
	PrivateIPLinkLocalA = 169
	PrivateIPLinkLocalB = 254
	PrivateIPv6UniqueA  = 0xfe
	PrivateIPv6UniqueB  = 0xfc

	DirPermissions  = 0700
	FilePermissions = 0600

	AuthTimeout           = 5 * time.Minute
	TokenRefreshThreshold = 5 * time.Minute
	TokenRefreshTimeout   = 30 * time.Second // Timeout for token refresh operations
	ServerShutdownTimeout = 5 * time.Second
	StateRandomBytes      = 32
	MinTokenLength        = 10   // Minimum token length
	MaxTokenLength        = 4096 // Maximum token length

	DefaultStorageDir = ".gemini"
	TokenFileName     = "/oauth_creds.json"

	MinPhraseLength   = 10
	WhitespaceNewline = "\n"
	WhitespaceTab     = "\t"
	WhitespaceDouble  = "  "

	URLRegexPattern = `https?://[^\s]+`
	GitHubDomain    = "github.com"
	GitHubRawDomain = "raw.githubusercontent.com"
	GitHubBlobPath  = "/blob/"
	GitHubRawPath   = "/"

	SourcesHeader       = "\n\n**Sources:**\n"
	CitationsHeader     = "\n\n**Citations:**\n"
	SearchQueriesHeader = "\n\n**Search queries used:**\n"
	MoreSourcesFormat   = "... and %d more sources\n"
	MoreQueriesFormat   = "... and %d more\n"

	ValidationErrorEmpty    = "cannot be empty"
	ValidationErrorRequired = "must be provided"
	ConfigErrorPrefix       = "config error in "

	AuthSuccessURL = "https://developers.google.com/gemini-code-assist/auth_success_gemini"
	AuthFailureURL = "https://developers.google.com/gemini-code-assist/auth_failure_gemini"

	TierIDFree = "free-tier"
)

var DefaultOAuthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

var HTMLTagsToRemove = []string{"script", "style", "head"}

var BrowserCommands = map[string][]string{
	"windows": {"cmd", "/c", "start"},
	"darwin":  {"open"},
	"linux":   {"xdg-open"},
}
