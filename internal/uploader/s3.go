package uploader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
)

// S3Uploader implements the Uploader interface for S3-compatible storage (AWS S3, Cloudflare R2, etc.)
type S3Uploader struct {
	client   *s3.Client
	bucket   string
	endpoint string
	baseURL  string // Public base URL for accessing files
}

// S3Config contains configuration for S3-compatible storage
type S3Config struct {
	// Endpoint is the S3-compatible endpoint URL (e.g., https://account-id.r2.cloudflarestorage.com)
	// For AWS S3, leave empty to use default
	Endpoint string

	// Region for the S3 bucket (e.g., "us-east-1", "auto" for R2)
	Region string

	// Bucket name
	Bucket string

	// AccessKeyID for authentication (can be read from env: AWS_ACCESS_KEY_ID or R2_ACCESS_KEY_ID)
	AccessKeyID string

	// SecretAccessKey for authentication (can be read from env: AWS_SECRET_ACCESS_KEY or R2_SECRET_ACCESS_KEY)
	SecretAccessKey string

	// BaseURL is the public URL base for accessing uploaded files
	// For Cloudflare R2 with custom domain: https://images.example.com
	// For AWS S3: https://bucket-name.s3.region.amazonaws.com
	BaseURL string
}

// NewS3Uploader creates a new S3-compatible uploader
func NewS3Uploader(ctx context.Context, cfg S3Config) (*S3Uploader, error) {
	// Load credentials from config or environment
	accessKey := cfg.AccessKeyID
	secretKey := cfg.SecretAccessKey

	if accessKey == "" {
		// Try R2-specific env vars first, then fall back to AWS env vars
		accessKey = os.Getenv("R2_ACCESS_KEY_ID")
		if accessKey == "" {
			accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		}
	}

	if secretKey == "" {
		secretKey = os.Getenv("R2_SECRET_ACCESS_KEY")
		if secretKey == "" {
			secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		}
	}

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing credentials: set R2_ACCESS_KEY_ID/R2_SECRET_ACCESS_KEY or AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY")
	}

	// Create AWS config
	var awsCfg aws.Config
	var err error

	if cfg.Endpoint != "" {
		// Custom endpoint (e.g., Cloudflare R2)
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	} else {
		// Standard AWS S3
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	// Create S3 client
	var client *s3.Client
	if cfg.Endpoint != "" {
		client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for R2
		})
	} else {
		client = s3.NewFromConfig(awsCfg)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		// Default to S3 URL format
		if cfg.Endpoint != "" {
			baseURL = fmt.Sprintf("%s/%s", cfg.Endpoint, cfg.Bucket)
		} else {
			baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
		}
	}

	log.Info().
		Str("bucket", cfg.Bucket).
		Str("region", cfg.Region).
		Str("endpoint", cfg.Endpoint).
		Str("baseURL", baseURL).
		Msg("S3 uploader initialized")

	return &S3Uploader{
		client:   client,
		bucket:   cfg.Bucket,
		endpoint: cfg.Endpoint,
		baseURL:  baseURL,
	}, nil
}

// Upload uploads a file to S3-compatible storage
func (u *S3Uploader) Upload(ctx context.Context, key string, content io.Reader, contentType string) error {
	log.Debug().Str("key", key).Str("contentType", contentType).Msg("Uploading to S3")

	input := &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(contentType),
	}

	_, err := u.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload %s: %w", key, err)
	}

	log.Debug().Str("key", key).Msg("Upload successful")
	return nil
}

// Exists checks if a file exists in S3-compatible storage
func (u *S3Uploader) Exists(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	}

	_, err := u.client.HeadObject(ctx, input)
	if err != nil {
		// Check if it's a "not found" error
		// AWS SDK v2 returns different error types, we check the error message
		errMsg := err.Error()
		if contains(errMsg, "NotFound") || contains(errMsg, "404") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of %s: %w", key, err)
	}

	return true, nil
}

// GetURL returns the public URL for accessing the uploaded file
func (u *S3Uploader) GetURL(key string) string {
	return fmt.Sprintf("%s/%s", u.baseURL, key)
}

// Delete removes a file from S3-compatible storage
func (u *S3Uploader) Delete(ctx context.Context, key string) error {
	log.Debug().Str("key", key).Msg("Deleting from S3")

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	}

	_, err := u.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete %s: %w", key, err)
	}

	log.Debug().Str("key", key).Msg("Delete successful")
	return nil
}

// contains checks if a string contains a substring (case-insensitive check would be better but this is simpler)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// DetectContentType detects MIME type from file extension
func DetectContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".html":
		return "text/html"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
