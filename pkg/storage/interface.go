// Package storage provides interfaces for credential storage,
// compatible with the gemini-cli authentication system.
package storage

import (
	"errors"

	"github.com/d-kuro/geminiwebtools/pkg/constants"
	"golang.org/x/oauth2"
)

// CredentialStore defines the interface for storing and retrieving OAuth2 credentials.
// This interface allows for different storage backends while maintaining compatibility
// with the gemini-cli credential storage system.
type CredentialStore interface {
	// LoadToken loads the stored OAuth2 token.
	// Returns nil if no token is stored or if the token cannot be loaded.
	LoadToken() (*oauth2.Token, error)

	// StoreToken stores the OAuth2 token.
	// The token should be stored securely and persistently.
	StoreToken(token *oauth2.Token) error

	// ClearToken removes the stored OAuth2 token.
	// This is used during logout operations.
	ClearToken() error

	// HasToken checks if a valid token is stored.
	// This is a convenience method to check token existence without loading it.
	HasToken() bool

	// GetStoragePath returns the path where credentials are stored.
	// This is used for informational purposes and debugging.
	GetStoragePath() string
}

// FileSystemStore implements CredentialStore using the filesystem.
// This is compatible with the gemini-cli credential storage format.
type FileSystemStore struct {
	baseDir string
}

// NewFileSystemStore creates a new filesystem-based credential store.
// If baseDir is empty, it will use the default directory (~/.gemini or equivalent).
func NewFileSystemStore(baseDir string) (*FileSystemStore, error) {
	if baseDir == "" {
		var err error
		baseDir, err = getDefaultStorageDir()
		if err != nil {
			return nil, err
		}
	}

	// Ensure the directory exists
	if err := ensureDir(baseDir); err != nil {
		return nil, err
	}

	return &FileSystemStore{
		baseDir: baseDir,
	}, nil
}

// MustNewFileSystemStore creates a new file system store and panics if an error occurs.
// This is useful for initialization code where errors are not expected.
func MustNewFileSystemStore(baseDir string) *FileSystemStore {
	store, err := NewFileSystemStore(baseDir)
	if err != nil {
		panic(err)
	}
	return store
}

// LoadToken implements CredentialStore.LoadToken.
func (fs *FileSystemStore) LoadToken() (*oauth2.Token, error) {
	return loadTokenFromFile(fs.getTokenPath())
}

// StoreToken implements CredentialStore.StoreToken.
func (fs *FileSystemStore) StoreToken(token *oauth2.Token) error {
	return storeTokenToFile(fs.getTokenPath(), token)
}

// ClearToken implements CredentialStore.ClearToken.
func (fs *FileSystemStore) ClearToken() error {
	return removeFile(fs.getTokenPath())
}

// HasToken implements CredentialStore.HasToken.
func (fs *FileSystemStore) HasToken() bool {
	return fileExists(fs.getTokenPath())
}

// GetStoragePath implements CredentialStore.GetStoragePath.
func (fs *FileSystemStore) GetStoragePath() string {
	return fs.baseDir
}

// getTokenPath returns the full path to the token file.
func (fs *FileSystemStore) getTokenPath() string {
	return fs.baseDir + constants.TokenFileName
}

// Sentinel errors for storage operations
var (
	ErrStorageNotFound   = errors.New("storage item not found")
	ErrStorageCorrupted  = errors.New("storage data corrupted")
	ErrStoragePermission = errors.New("storage permission denied")
)
