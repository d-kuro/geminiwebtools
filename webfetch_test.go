package geminiwebtools

import (
	"strings"
	"testing"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
)

func TestExtractUrls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single HTTP URL",
			input:    "Visit http://example.com for more info",
			expected: []string{"http://example.com"},
		},
		{
			name:     "single HTTPS URL",
			input:    "Visit https://example.com for more info",
			expected: []string{"https://example.com"},
		},
		{
			name:     "multiple URLs",
			input:    "Check out https://google.com and http://github.com",
			expected: []string{"https://google.com", "http://github.com"},
		},
		{
			name:     "URL with path and query",
			input:    "API endpoint: https://api.example.com/v1/users?id=123",
			expected: []string{"https://api.example.com/v1/users?id=123"},
		},
		{
			name:     "URL with fragment",
			input:    "See https://docs.example.com/guide#section1",
			expected: []string{"https://docs.example.com/guide#section1"},
		},
		{
			name:     "no URLs",
			input:    "This text has no URLs in it",
			expected: []string{},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "URL at start of string",
			input:    "https://example.com is a good site",
			expected: []string{"https://example.com"},
		},
		{
			name:     "URL at end of string",
			input:    "Visit my site at https://example.com",
			expected: []string{"https://example.com"},
		},
		{
			name:     "GitHub blob URL",
			input:    "Check https://github.com/user/repo/blob/main/file.go",
			expected: []string{"https://github.com/user/repo/blob/main/file.go"},
		},
		{
			name:     "URLs with various protocols (only http/https should match)",
			input:    "ftp://files.com https://web.com mailto:test@example.com",
			expected: []string{"https://web.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUrls(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("extractUrls() = %v, want %v", result, tt.expected)
				return
			}

			for i, url := range result {
				if url != tt.expected[i] {
					t.Errorf("extractUrls()[%d] = %s, want %s", i, url, tt.expected[i])
				}
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid HTTP URL",
			url:         "http://example.com",
			expectError: false,
		},
		{
			name:        "valid HTTPS URL",
			url:         "https://example.com",
			expectError: false,
		},
		{
			name:        "valid URL with path",
			url:         "https://example.com/path/to/resource",
			expectError: false,
		},
		{
			name:        "valid URL with query params",
			url:         "https://example.com/search?q=test&limit=10",
			expectError: false,
		},
		{
			name:        "valid URL with fragment",
			url:         "https://example.com/docs#section1",
			expectError: false,
		},
		{
			name:        "invalid URL format",
			url:         "not-a-url",
			expectError: true,
			errorMsg:    "unsupported URL scheme",
		},
		{
			name:        "unsupported scheme - ftp",
			url:         "ftp://files.example.com",
			expectError: true,
			errorMsg:    "unsupported URL scheme",
		},
		{
			name:        "unsupported scheme - file",
			url:         "file:///etc/passwd",
			expectError: true,
			errorMsg:    "unsupported URL scheme",
		},
		{
			name:        "missing host",
			url:         "https://",
			expectError: true,
			errorMsg:    "URL missing host",
		},
		{
			name:        "localhost URL",
			url:         "https://localhost/api",
			expectError: true,
			errorMsg:    "localhost URLs are not allowed",
		},
		{
			name:        "127.0.0.1 URL",
			url:         "https://127.0.0.1:8080",
			expectError: true,
			errorMsg:    "localhost URLs are not allowed",
		},
		{
			name:        "IPv6 localhost",
			url:         "https://[::1]:8080",
			expectError: true,
			errorMsg:    "localhost URLs are not allowed",
		},
		{
			name:        "URL with null character",
			url:         "https://example.com/path\x00/file",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
		{
			name:        "URL with carriage return",
			url:         "https://example.com/path\r/file",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
		{
			name:        "URL with newline",
			url:         "https://example.com/path\n/file",
			expectError: true,
			errorMsg:    "invalid URL format",
		},
		{
			name:        "URL exceeds maximum length",
			url:         "https://example.com/" + strings.Repeat("a", constants.MaxURLLength),
			expectError: true,
			errorMsg:    "URL exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateURL() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateURL() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateURL() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConvertGitHubBlobURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub blob URL",
			input:    "https://github.com/user/repo/blob/main/file.go",
			expected: "https://raw.githubusercontent.com/user/repo/main/file.go",
		},
		{
			name:     "GitHub blob URL with branch name",
			input:    "https://github.com/owner/project/blob/feature-branch/src/main.js",
			expected: "https://raw.githubusercontent.com/owner/project/feature-branch/src/main.js",
		},
		{
			name:     "GitHub blob URL with commit hash",
			input:    "https://github.com/user/repo/blob/abc123def456/README.md",
			expected: "https://raw.githubusercontent.com/user/repo/abc123def456/README.md",
		},
		{
			name:     "GitHub blob URL with nested path",
			input:    "https://github.com/org/repo/blob/main/docs/api/reference.md",
			expected: "https://raw.githubusercontent.com/org/repo/main/docs/api/reference.md",
		},
		{
			name:     "non-GitHub URL should not be converted",
			input:    "https://gitlab.com/user/repo/blob/main/file.go",
			expected: "https://gitlab.com/user/repo/blob/main/file.go",
		},
		{
			name:     "GitHub URL without blob should not be converted",
			input:    "https://github.com/user/repo/tree/main",
			expected: "https://github.com/user/repo/tree/main",
		},
		{
			name:     "GitHub URL with blob in different context should not be converted",
			input:    "https://github.com/user/repo/issues/123",
			expected: "https://github.com/user/repo/issues/123",
		},
		{
			name:     "already converted raw URL should remain unchanged",
			input:    "https://raw.githubusercontent.com/user/repo/main/file.go",
			expected: "https://raw.githubusercontent.com/user/repo/main/file.go",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "non-URL string",
			input:    "not a url",
			expected: "not a url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGitHubBlobURL(tt.input)
			if result != tt.expected {
				t.Errorf("convertGitHubBlobURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsHTMLContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{
			name:        "HTML content type",
			contentType: "text/html",
			expected:    true,
		},
		{
			name:        "XHTML content type",
			contentType: "application/xhtml+xml",
			expected:    true,
		},
		{
			name:        "HTML with charset",
			contentType: "text/html; charset=utf-8",
			expected:    false, // The function does exact string comparison
		},
		{
			name:        "plain text",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "JSON content type",
			contentType: "application/json",
			expected:    false,
		},
		{
			name:        "XML content type",
			contentType: "application/xml",
			expected:    false,
		},
		{
			name:        "empty content type",
			contentType: "",
			expected:    false,
		},
		{
			name:        "mixed case HTML",
			contentType: "TEXT/HTML",
			expected:    false, // The function does exact string comparison
		},
		{
			name:        "image content type",
			contentType: "image/png",
			expected:    false,
		},
		{
			name:        "JavaScript content type",
			contentType: "application/javascript",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isHTMLContent(tt.contentType)
			if result != tt.expected {
				t.Errorf("isHTMLContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewWebFetcher(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		checkFields bool
	}{
		{
			name:        "nil config should use default",
			config:      nil,
			expectError: false,
			checkFields: true,
		},
		{
			name:        "valid config",
			config:      NewConfig(),
			expectError: false,
			checkFields: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher, err := NewWebFetcher(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewWebFetcher() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewWebFetcher() unexpected error = %v", err)
				return
			}

			if fetcher == nil {
				t.Errorf("NewWebFetcher() returned nil fetcher")
				return
			}

			if tt.checkFields {
				// Check that all required fields are initialized
				if fetcher.config == nil {
					t.Errorf("NewWebFetcher() config is nil")
				}
				if fetcher.auth == nil {
					t.Errorf("NewWebFetcher() auth is nil")
				}
				if fetcher.codeAssist == nil {
					t.Errorf("NewWebFetcher() codeAssist is nil")
				}
				if fetcher.grounding == nil {
					t.Errorf("NewWebFetcher() grounding is nil")
				}
				if fetcher.httpClient == nil {
					t.Errorf("NewWebFetcher() httpClient is nil")
				}
			}
		})
	}
}
