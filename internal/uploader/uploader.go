package uploader

import (
	"context"
	"io"
)

// Uploader defines the interface for uploading files to remote storage
type Uploader interface {
	// Upload uploads a file to the remote storage
	// key is the destination path/key in the remote storage
	// content is the file content to upload
	// contentType is the MIME type of the content
	Upload(ctx context.Context, key string, content io.Reader, contentType string) error

	// Exists checks if a file exists at the given key in remote storage
	Exists(ctx context.Context, key string) (bool, error)

	// GetURL returns the public URL for accessing the uploaded file
	GetURL(key string) string

	// Delete removes a file from remote storage
	Delete(ctx context.Context, key string) error
}

// UploadOptions contains options for uploading files
type UploadOptions struct {
	// Force uploads even if the file already exists
	Force bool

	// Prefix to prepend to all keys (e.g., "images/")
	Prefix string

	// ContentType override (if empty, will be detected from file extension)
	ContentType string
}
