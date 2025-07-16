package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"golang.org/x/oauth2"
)

// getDefaultStorageDir returns the default directory for storing credentials.
// This matches the gemini-cli default location.
func getDefaultStorageDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Use .gemini to match gemini-cli convention
	return filepath.Join(homeDir, constants.DefaultStorageDir), nil
}

// ensureDir creates the directory if it doesn't exist.
func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, constants.DirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

// loadTokenFromFile loads an OAuth2 token from a JSON file.
func loadTokenFromFile(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("token file does not exist at %s: %w", path, ErrStorageNotFound)
		}
		return nil, fmt.Errorf("failed to read token file at %s: %w", path, err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token JSON at %s: %w", path, ErrStorageCorrupted)
	}

	return &token, nil
}

// storeTokenToFile stores an OAuth2 token to a JSON file.
func storeTokenToFile(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token to JSON for %s: %w", path, err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}

	// Write with restricted permissions
	if err := os.WriteFile(path, data, constants.FilePermissions); err != nil {
		return fmt.Errorf("failed to write token file at %s: %w", path, err)
	}

	return nil
}

// removeFile removes a file.
func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file at %s: %w", path, err)
	}
	return nil
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
