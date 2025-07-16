package geminiwebtools

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"golang.org/x/net/html"
)

// HTTPClient provides secure HTTP functionality for web content fetching.
type HTTPClient struct {
	client *http.Client
	config *HTTPClientConfig
}

// ClientPool manages a pool of reusable HTTP clients for different configurations.
type ClientPool struct {
	clients map[string]*http.Client
	mutex   sync.RWMutex
}

// Global client pool for efficient HTTP client reuse
var globalClientPool = &ClientPool{
	clients: make(map[string]*http.Client),
}

// HTTPClientConfig contains configuration for the HTTP client.
type HTTPClientConfig struct {
	Timeout         time.Duration
	FollowRedirects bool
	AllowPrivateIPs bool
	MaxContentSize  int64
	UserAgent       string
}

// DefaultHTTPClientConfig returns a default HTTP client configuration.
func DefaultHTTPClientConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		Timeout:         constants.DefaultHTTPTimeout,
		FollowRedirects: true,
		AllowPrivateIPs: false,
		MaxContentSize:  constants.DefaultHTTPMaxContentSize,
		UserAgent:       constants.DefaultUserAgent,
	}
}

// getOrCreateClient retrieves or creates an HTTP client from the pool.
func (cp *ClientPool) getOrCreateClient(config *HTTPClientConfig) *http.Client {
	key := cp.configKey(config)

	// Try to get existing client first
	cp.mutex.RLock()
	if client, exists := cp.clients[key]; exists {
		cp.mutex.RUnlock()
		return client
	}
	cp.mutex.RUnlock()

	// Create new client with optimized settings
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cp.clients[key]; exists {
		return client
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Configure secure redirect policy
	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			// Limit redirect count
			if len(via) >= constants.MaxRedirects {
				return fmt.Errorf("too many redirects (max: %d)", constants.MaxRedirects)
			}

			// Validate redirect URL
			if err := validateRedirectURL(req.URL, via); err != nil {
				return fmt.Errorf("redirect validation failed: %w", err)
			}

			return nil
		}
	}

	// Configure transport with optimized connection pooling
	transport := &http.Transport{
		// Connection pooling settings using constants
		MaxIdleConns:        constants.MaxIdleConns,
		MaxIdleConnsPerHost: constants.MaxIdleConnsPerHost,
		MaxConnsPerHost:     constants.MaxConnsPerHost,
		IdleConnTimeout:     constants.IdleConnTimeout,

		// Timeouts using constants
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Extract host and port
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address: %w", err)
			}

			// Resolve the address
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve host: %w", err)
			}

			// Check for private IPs if not allowed
			if !config.AllowPrivateIPs {
				for _, ip := range ips {
					if isPrivateIP(ip) {
						return nil, fmt.Errorf("private IP addresses are not allowed: %s", ip)
					}
				}
			}

			// Use optimized dialer with keep-alive
			dialer := &net.Dialer{
				Timeout:   constants.DefaultDialerTimeout,
				KeepAlive: constants.KeepAliveTimeout,
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   constants.TLSHandshakeTimeout,
		ResponseHeaderTimeout: constants.ResponseHeaderTimeout,
		ExpectContinueTimeout: constants.ExpectContinueTimeout,

		// Enable HTTP/2 for better performance
		ForceAttemptHTTP2: true,

		// Keep compression enabled for bandwidth optimization
		DisableCompression: false,

		// Additional optimizations
		DisableKeepAlives: false,     // Enable keep-alives for connection reuse
		WriteBufferSize:   32 * 1024, // 32KB write buffer
		ReadBufferSize:    32 * 1024, // 32KB read buffer
	}

	client.Transport = transport
	cp.clients[key] = client
	return client
}

// configKey generates a unique key for the client configuration.
func (cp *ClientPool) configKey(config *HTTPClientConfig) string {
	return fmt.Sprintf("%v_%v_%v_%d_%s",
		config.Timeout,
		config.FollowRedirects,
		config.AllowPrivateIPs,
		config.MaxContentSize,
		config.UserAgent,
	)
}

// NewHTTPClient creates a new HTTP client with the specified configuration using connection pooling.
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	if config == nil {
		config = DefaultHTTPClientConfig()
	}

	// Get or create a pooled client
	client := globalClientPool.getOrCreateClient(config)

	return &HTTPClient{
		client: client,
		config: config,
	}
}

// FetchContent fetches content from a URL and returns the content, content type, and size.
func (hc *HTTPClient) FetchContent(ctx context.Context, urlStr string) (content, contentType string, contentSize int, err error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return "", "", 0, ctx.Err()
	default:
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow HTTP and HTTPS
	if parsedURL.Scheme != constants.SchemeHTTP && parsedURL.Scheme != constants.SchemeHTTPS {
		return "", "", 0, fmt.Errorf("unsupported scheme: %s", parsedURL.Scheme)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set security headers
	req.Header.Set("User-Agent", hc.config.UserAgent)
	req.Header.Set("Accept", constants.DefaultAcceptHeader)
	req.Header.Set("Accept-Language", constants.DefaultAcceptLanguageHeader)
	req.Header.Set("DNT", "1")                           // Do Not Track
	req.Header.Set("X-Requested-With", "geminiwebtools") // Identify as non-browser
	req.Header.Set("Cache-Control", "no-cache")          // Prevent caching of requests
	req.Header.Set("Pragma", "no-cache")                 // HTTP/1.0 compatibility
	req.Header.Set("X-Content-Type-Options", "nosniff")  // Prevent MIME sniffing
	req.Header.Set("X-Frame-Options", "DENY")            // Prevent framing (if response is HTML)
	req.Header.Set("Referrer-Policy", "no-referrer")     // Don't send referrer

	// Make request
	resp, err := hc.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Get content type
	contentType = resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = constants.ContentTypePlain
	}

	// Read content with optimized size limit and streaming
	var reader io.Reader = resp.Body
	maxSize := hc.config.MaxContentSize
	if maxSize <= 0 {
		maxSize = constants.DefaultHTTPMaxContentSize
	}

	// Use a limited reader to avoid reading more than necessary
	reader = io.LimitReader(resp.Body, maxSize+1) // +1 to detect truncation

	// Pre-allocate buffer with estimated size based on Content-Length
	var buf []byte
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil && size > 0 {
			// Use the smaller of Content-Length or max size
			if size > maxSize {
				size = maxSize
			}
			buf = make([]byte, 0, size)
		}
	}

	// If no Content-Length header, use a reasonable default buffer size
	if buf == nil {
		bufSize := int64(64 * 1024) // 64KB default
		if maxSize < bufSize {
			bufSize = maxSize
		}
		buf = make([]byte, 0, bufSize)
	}

	// Read content in chunks to avoid large memory allocations
	const chunkSize = 32 * 1024 // 32KB chunks
	chunk := make([]byte, chunkSize)
	totalRead := int64(0)

	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			// Check if adding this chunk would exceed our limit
			if totalRead+int64(n) > maxSize {
				// Only add what we can within the limit
				remaining := maxSize - totalRead
				if remaining > 0 {
					buf = append(buf, chunk[:remaining]...)
					totalRead += remaining
				}
				return string(buf), contentType, int(totalRead), fmt.Errorf("content truncated: exceeded maximum size of %d bytes", maxSize)
			}

			buf = append(buf, chunk[:n]...)
			totalRead += int64(n)
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", 0, fmt.Errorf("failed to read response body: %w", err)
		}

		// Check for context cancellation during reading
		select {
		case <-ctx.Done():
			return "", "", 0, ctx.Err()
		default:
		}
	}

	content = string(buf)
	contentSize = int(totalRead)

	return content, contentType, contentSize, nil
}

// isPrivateIP checks if an IP address is in a private range.
func isPrivateIP(ip net.IP) bool {
	// Check for IPv4 private ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 10.0.0.0/8
		if ip4[0] == constants.PrivateIPClass10 {
			return true
		}
		// 172.16.0.0/12
		if ip4[0] == constants.PrivateIPClass172A && ip4[1] >= constants.PrivateIPClass172B && ip4[1] <= constants.PrivateIPClass172C {
			return true
		}
		// 192.168.0.0/16
		if ip4[0] == constants.PrivateIPClass192A && ip4[1] == constants.PrivateIPClass192B {
			return true
		}
		// 127.0.0.0/8 (loopback)
		if ip4[0] == constants.PrivateIPLoopback {
			return true
		}
		// 169.254.0.0/16 (link-local)
		if ip4[0] == constants.PrivateIPLinkLocalA && ip4[1] == constants.PrivateIPLinkLocalB {
			return true
		}
	}

	// Check for IPv6 private ranges
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return true
	}

	// Check for IPv6 unique local addresses (fc00::/7)
	if len(ip) == 16 && (ip[0]&constants.PrivateIPv6UniqueA) == constants.PrivateIPv6UniqueB {
		return true
	}

	return false
}

// validateRedirectURL validates redirect URLs for security
func validateRedirectURL(redirectURL *url.URL, via []*http.Request) error {
	// Don't allow redirects to different schemes (downgrade attacks)
	if len(via) > 0 {
		originalScheme := via[0].URL.Scheme
		if redirectURL.Scheme != originalScheme {
			// Allow HTTP -> HTTPS upgrade, but not HTTPS -> HTTP downgrade
			if originalScheme != "http" || redirectURL.Scheme != "https" {
				return fmt.Errorf("scheme change not allowed: %s -> %s", originalScheme, redirectURL.Scheme)
			}
		}
	}

	// Don't allow redirects to private IPs
	host := redirectURL.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve redirect host: %w", err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("redirect to private IP not allowed: %s", ip)
		}
	}

	return nil
}

// ExtractTextFromHTML safely extracts text content from HTML using the standard html parser.
func ExtractTextFromHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fallback to simple tag removal if parsing fails
		return fallbackTextExtraction(htmlContent)
	}

	// Extract text nodes while skipping dangerous elements
	var result strings.Builder
	extractTextNodes(doc, &result)

	// Clean up whitespace
	content := result.String()
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	return strings.TrimSpace(content)
}

// extractTextNodes recursively extracts text from HTML nodes while filtering out dangerous content
func extractTextNodes(node *html.Node, result *strings.Builder) {
	if node == nil {
		return
	}

	// Skip dangerous elements
	if node.Type == html.ElementNode {
		switch strings.ToLower(node.Data) {
		case "script", "style", "noscript", "iframe", "object", "embed":
			return // Skip these elements and their children entirely
		}
	}

	// Extract text content
	if node.Type == html.TextNode {
		text := strings.TrimSpace(node.Data)
		if text != "" {
			result.WriteString(text)
			result.WriteString(" ")
		}
	}

	// Process child nodes
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		extractTextNodes(child, result)
	}
}

// fallbackTextExtraction provides simple fallback text extraction
func fallbackTextExtraction(htmlContent string) string {
	// Remove script and style tags with their content
	content := removeHTMLTagsWithContent(htmlContent, constants.HTMLTagsToRemove)

	// Remove all other HTML tags
	content = removeHTMLTags(content)

	// Clean up whitespace
	content = strings.ReplaceAll(content, constants.WhitespaceNewline, " ")
	content = strings.ReplaceAll(content, constants.WhitespaceTab, " ")

	// Collapse multiple spaces
	for strings.Contains(content, constants.WhitespaceDouble) {
		content = strings.ReplaceAll(content, constants.WhitespaceDouble, " ")
	}

	return strings.TrimSpace(content)
}

// removeHTMLTagsWithContent removes specified HTML tags along with their content.
func removeHTMLTagsWithContent(content string, tags []string) string {
	for _, tag := range tags {
		startTag := "<" + tag
		endTag := "</" + tag + ">"

		for {
			start := strings.Index(strings.ToLower(content), startTag)
			if start == -1 {
				break
			}

			// Find the end of the opening tag
			tagEnd := strings.Index(content[start:], ">")
			if tagEnd == -1 {
				break
			}
			tagEnd += start + 1

			// Find the closing tag
			end := strings.Index(strings.ToLower(content[tagEnd:]), endTag)
			if end == -1 {
				break
			}
			end += tagEnd + len(endTag)

			// Remove the entire tag and its content
			content = content[:start] + content[end:]
		}
	}

	return content
}

// removeHTMLTags removes all HTML tags from content.
func removeHTMLTags(content string) string {
	inTag := false
	var result strings.Builder

	for _, char := range content {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(char)
		}
	}

	return result.String()
}
