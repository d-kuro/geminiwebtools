// Package main demonstrates browser-based authentication with geminiwebtools.
// This example shows how to use the library with browser authentication
// similar to gemini-cli's authentication flow.
//
// Run this example with: go run browser_auth_example.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/d-kuro/geminiwebtools"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

func main() {
	fmt.Println("Gemini Web Tools - Browser Authentication Example")
	fmt.Println("================================================")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Option 1: Create client and authenticate manually
	fmt.Println("\n1. Manual Authentication Flow:")
	if err := manualAuthExample(ctx); err != nil {
		log.Printf("Manual auth example failed: %v", err)
	}

	// Option 2: Use convenience function (recommended for CLI apps)
	fmt.Println("\n2. Convenience Function (Recommended):")
	if err := convenienceAuthExample(ctx); err != nil {
		log.Printf("Convenience auth example failed: %v", err)
	}

	// Option 3: Custom configuration with filesystem storage
	fmt.Println("\n3. Custom Configuration with File Storage:")
	if err := customConfigExample(ctx); err != nil {
		log.Printf("Custom config example failed: %v", err)
	}
}

// manualAuthExample demonstrates manual authentication flow
func manualAuthExample(ctx context.Context) error {
	// Create client with default configuration
	client, err := geminiwebtools.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Check authentication status
	status, err := client.GetAuthStatus()
	if err != nil {
		return fmt.Errorf("failed to get auth status: %w", err)
	}

	fmt.Printf("Authentication status: %+v\n", status)

	// Authenticate if not already authenticated
	if !client.IsAuthenticated() {
		fmt.Println("Not authenticated. Starting browser authentication...")

		err = client.AuthenticateWithBrowser(ctx)
		if err != nil {
			return fmt.Errorf("browser authentication failed: %w", err)
		}

		fmt.Println("✅ Authentication successful!")
	} else {
		fmt.Println("✅ Already authenticated!")
	}

	// Perform a web search
	fmt.Println("\nPerforming web search...")
	searchResult, err := client.Search(ctx, "Go programming language features")
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Printf("Search Summary: %s\n", searchResult.Summary)
	fmt.Printf("Sources found: %d\n", len(searchResult.Sources))
	fmt.Printf("Content length: %d characters\n", len(searchResult.Content))

	// Perform a web fetch
	fmt.Println("\nPerforming web fetch...")
	fetchResult, err := client.Fetch(ctx, "https://golang.org Summarize the main features of Go")
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	fmt.Printf("Fetch Summary: %s\n", fetchResult.Summary)
	fmt.Printf("Content length: %d characters\n", len(fetchResult.Content))
	if len(fetchResult.DisplayText) > 200 {
		fmt.Printf("Preview: %s...\n", fetchResult.DisplayText[:200])
	} else {
		fmt.Printf("Content: %s\n", fetchResult.DisplayText)
	}

	return nil
}

// convenienceAuthExample demonstrates the convenience function
func convenienceAuthExample(ctx context.Context) error {
	// This function creates a client and authenticates in one step
	// Perfect for CLI applications
	client, err := geminiwebtools.NewClientWithBrowserAuth(ctx)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	fmt.Println("✅ Client created and authenticated!")

	// Perform operations immediately
	searchResult, err := client.Search(ctx, "TypeScript vs Go comparison")
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Printf("Search completed: %s\n", searchResult.Summary)

	return nil
}

// customConfigExample demonstrates custom configuration
func customConfigExample(ctx context.Context) error {
	// Create filesystem storage for persistent credentials
	store, err := storage.NewFileSystemStore(".gemini_credentials")
	if err != nil {
		return fmt.Errorf("failed to create file storage: %w", err)
	}

	// Create client with custom configuration
	client, err := geminiwebtools.NewClient(
		geminiwebtools.WithCredentialStore(store),
		geminiwebtools.WithTimeout(60*time.Second),
		geminiwebtools.WithMaxContentSize(10*1024*1024), // 10MB
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Authenticate if needed
	if !client.IsAuthenticated() {
		fmt.Println("Authenticating with custom config...")

		err = client.AuthenticateWithBrowser(ctx)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		fmt.Println("✅ Authenticated with file storage!")
	}

	// Test both search and fetch
	fmt.Println("Testing with custom configuration...")

	searchResult, err := client.Search(ctx, "Rust programming language memory safety")
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Printf("Custom config search: %s\n", searchResult.Summary)

	fetchResult, err := client.Fetch(ctx, "https://rust-lang.org Explain Rust's memory safety features")
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	fmt.Printf("Custom config fetch: %s\n", fetchResult.Summary)

	return nil
}
