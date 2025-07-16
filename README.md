# Gemini Web Tools

A Go library that provides web search and web fetch functionality compatible with gemini-cli interfaces using OAuth2 authentication.

## Features

- **Web Search**: AI-powered web search using Google's Gemini model with grounding support
- **Web Fetch**: Intelligent web content fetching with AI processing and fallback to direct HTTP
- **OAuth2 Authentication**: Compatible with Google OAuth2 authentication flow
- **CodeAssist Integration**: Uses Google's internal CodeAssist Server for AI operations
- **Grounding Support**: Includes citation processing and source attribution
- **Security**: Built-in security features including private IP protection and content size limits
- **Configurable**: Flexible configuration system with sensible defaults

## Installation

```bash
go get github.com/d-kuro/geminiwebtools
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/d-kuro/geminiwebtools"
)

func main() {
    ctx := context.Background()
    
    // Create an authenticated client (opens browser for OAuth2)
    client, err := geminiwebtools.NewClientWithBrowserAuth(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Perform a web search
    searchResult, err := client.Search(ctx, "Go programming language")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Search Result:")
    fmt.Printf("Summary: %s\n", searchResult.Summary)
    fmt.Println(searchResult.DisplayText)
    fmt.Printf("Sources: %d\n", len(searchResult.Sources))
    
    // Fetch web content
    fetchResult, err := client.Fetch(ctx, "https://golang.org Summarize the main features of Go")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("\nFetch Result:")
    fmt.Printf("Summary: %s\n", fetchResult.Summary)
    fmt.Println(fetchResult.DisplayText)
}
```

### Manual Authentication Flow

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/d-kuro/geminiwebtools"
    "github.com/d-kuro/geminiwebtools/pkg/storage"
)

func main() {
    ctx := context.Background()
    
    // Create client with custom configuration
    store, err := storage.NewFileSystemStore(".gemini_credentials")
    if err != nil {
        log.Fatal(err)
    }
    
    client, err := geminiwebtools.NewClient(
        geminiwebtools.WithCredentialStore(store),
        geminiwebtools.WithTimeout(60*time.Second),
        geminiwebtools.WithMaxContentSize(10*1024*1024), // 10MB
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Check authentication status
    if !client.IsAuthenticated() {
        err = client.AuthenticateWithBrowser(ctx)
        if err != nil {
            log.Fatal(err)
        }
    }
    
    // Use the client...
}
```

### Quick Setup (Recommended for CLI Apps)

```go
package main

import (
    "context"
    "log"
    
    "github.com/d-kuro/geminiwebtools"
)

func main() {
    ctx := context.Background()
    
    // One-line setup with browser authentication
    client, err := geminiwebtools.NewClientWithBrowserAuth(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    // Ready to use immediately
    result, err := client.Search(ctx, "Go modules tutorial")
    if err != nil {
        log.Fatal(err)
    }
    
    // Web fetch with URL and instructions in the prompt
    fetchResult, err := client.Fetch(ctx, "https://example.com Extract and summarize the main content")
    if err != nil {
        log.Fatal(err)
    }
    
    // Process results...
}
```

## Configuration

The library supports flexible configuration through the `Config` struct:

### Available Configuration Options

The library supports these configuration options through functional options:

```go
client, err := geminiwebtools.NewClient(
    geminiwebtools.WithCredentialStore(store),         // Custom credential storage
    geminiwebtools.WithTimeout(60*time.Second),        // Request timeout
    geminiwebtools.WithMaxContentSize(10*1024*1024),   // Content size limit (10MB)
)
```

### Configuration Options

- **Credential Store**: Where OAuth2 tokens are stored (default: `~/.gemini`)
- **Timeout**: HTTP request timeout (configurable)
- **Max Content Size**: Limit for fetched content size
- **Storage**: File system-based credential storage with custom paths

## Authentication

The library uses OAuth2 authentication compatible with Google's authentication flow:

```go
// Check authentication status
if !client.IsAuthenticated() {
    // Trigger browser authentication
    err := client.AuthenticateWithBrowser(ctx)
    if err != nil {
        log.Fatal(err)
    }
}

// Get detailed authentication status
status, err := client.GetAuthStatus()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Auth status: %+v\n", status)

// Clear stored authentication
err = client.ClearAuthentication()
if err != nil {
    log.Fatal(err)
}
```

## Results

### Web Search Results

```go
type WebSearchResult struct {
    Summary     string              // One-line summary
    Content     string              // Processed content for LLM
    DisplayText string              // Formatted content for display
    Sources     []GroundingChunk    // Source citations
    Metadata    WebSearchMetadata   // Additional metadata
}
```

### Web Fetch Results

```go
type WebFetchResult struct {
    Summary     string             // One-line summary
    Content     string             // Processed content for LLM
    DisplayText string             // Formatted content for display
    Sources     []GroundingChunk   // Source citations if available
    Metadata    WebFetchMetadata   // Additional metadata
}
```

## Error Handling

The library uses standard Go error handling patterns:

```go
result, err := client.Search(ctx, "query")
if err != nil {
    log.Fatal(err)
}
```

## Examples

The `examples/` directory contains complete working examples:

- **browser_auth_example.go**: Demonstrates browser-based OAuth2 authentication flows
  - Manual authentication with status checking
  - Convenience function for quick setup
  - Custom configuration with file storage

Run the example:
```bash
cd examples
go run browser_auth_example.go
```

## Security Features

- **Private IP Protection**: Prevents access to private IP ranges by default
- **Content Size Limits**: Configurable limits on fetched content size  
- **URL Validation**: Validates URLs before making requests
- **Secure Transport**: Uses secure HTTP transport configuration
- **OAuth2 Authentication**: Secure authentication using Google OAuth2 flow

## Requirements

- Go 1.24 or later
- Google OAuth2 credentials (compatible with gemini-cli)
- Internet connection for authentication and API calls

## License

This project is open source. Please check the LICENSE file for details.
