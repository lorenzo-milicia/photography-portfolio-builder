package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/uploader"
)

var (
	uploadInputDir   string
	uploadBucket     string
	uploadRegion     string
	uploadEndpoint   string
	uploadBaseURL    string
	uploadPrefix     string
	uploadForce      bool
	uploadDryRun     bool
	uploadSkipThumbs bool
)

var imagesUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload processed images to remote storage (S3/R2)",
	Long: `Upload processed images to S3-compatible remote storage (AWS S3, Cloudflare R2, etc.).
	
Credentials are read from environment variables:
  - R2_ACCESS_KEY_ID / AWS_ACCESS_KEY_ID
  - R2_SECRET_ACCESS_KEY / AWS_SECRET_ACCESS_KEY

Example usage:
  # Upload to Cloudflare R2
  export R2_ACCESS_KEY_ID="your-access-key"
  export R2_SECRET_ACCESS_KEY="your-secret-key"
  builder images upload -i dist/images -b my-bucket -r auto --endpoint https://account-id.r2.cloudflarestorage.com --base-url https://images.example.com

  # Upload to AWS S3
  export AWS_ACCESS_KEY_ID="your-access-key"
  export AWS_SECRET_ACCESS_KEY="your-secret-key"
  builder images upload -i dist/images -b my-bucket -r us-east-1
`,
	Run: func(cmd *cobra.Command, args []string) {
		if uploadInputDir == "" {
			fmt.Println("Error: input directory is required (-i)")
			os.Exit(1)
		}

		if uploadBucket == "" {
			fmt.Println("Error: bucket name is required (-b)")
			os.Exit(1)
		}

		if uploadRegion == "" {
			fmt.Println("Error: region is required (-r)")
			os.Exit(1)
		}

		ctx := context.Background()

		// Create S3 uploader
		s3Config := uploader.S3Config{
			Endpoint:        uploadEndpoint,
			Region:          uploadRegion,
			Bucket:          uploadBucket,
			BaseURL:         uploadBaseURL,
			AccessKeyID:     "", // Will be read from env
			SecretAccessKey: "", // Will be read from env
		}

		fmt.Printf("Initializing uploader...\n")
		fmt.Printf("  Bucket: %s\n", uploadBucket)
		fmt.Printf("  Region: %s\n", uploadRegion)
		if uploadEndpoint != "" {
			fmt.Printf("  Endpoint: %s\n", uploadEndpoint)
		}
		if uploadBaseURL != "" {
			fmt.Printf("  Base URL: %s\n", uploadBaseURL)
		}
		if uploadPrefix != "" {
			fmt.Printf("  Prefix: %s\n", uploadPrefix)
		}
		if uploadForce {
			fmt.Printf("  Force: enabled (will overwrite existing files)\n")
		}
		if uploadDryRun {
			fmt.Printf("  Dry run: enabled (no files will be uploaded)\n")
		}

		var ul uploader.Uploader
		if !uploadDryRun {
			var err error
			ul, err = uploader.NewS3Uploader(ctx, s3Config)
			if err != nil {
				fmt.Printf("Error initializing uploader: %v\n", err)
				os.Exit(1)
			}
		}

		// Walk the input directory and upload all files
		fmt.Printf("\nScanning directory: %s\n", uploadInputDir)

		var uploadedCount, skippedCount, errorCount int

		err := filepath.Walk(uploadInputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error accessing %s: %v\n", path, err)
				errorCount++
				return nil // Continue walking
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Calculate relative path from input dir
			relPath, err := filepath.Rel(uploadInputDir, path)
			if err != nil {
				fmt.Printf("Error calculating relative path for %s: %v\n", path, err)
				errorCount++
				return nil
			}

			// Optionally skip thumbnail files stored in .thumbs folders
			if uploadSkipThumbs {
				if strings.HasPrefix(relPath, ".thumbs") || strings.Contains(relPath, "/.thumbs/") {
					fmt.Printf("⏭️  %s (thumbs skipped)\n", relPath)
					skippedCount++
					return nil
				}
			}

			// Construct remote key (use forward slashes for S3)
			key := filepath.ToSlash(relPath)
			if uploadPrefix != "" {
				// Use explicit prefix provided by user
				key = strings.TrimSuffix(uploadPrefix, "/") + "/" + key
			} else {
				// Default to `images/` prefix so remote mirrors local `dist/images/...` layout
				key = "images/" + key
			}

			// Check if file already exists (unless force is enabled)
			if !uploadForce && !uploadDryRun {
				exists, err := ul.Exists(ctx, key)
				if err != nil {
					fmt.Printf("⚠️  Error checking existence of %s: %v\n", key, err)
					errorCount++
					return nil
				}

				if exists {
					fmt.Printf("⏭️  %s (already exists)\n", key)
					skippedCount++
					return nil
				}
			}

			// Detect content type
			contentType := uploader.DetectContentType(path)

			if uploadDryRun {
				fmt.Printf("🔍 %s (would upload, %s)\n", key, contentType)
				uploadedCount++
				return nil
			}

			// Open file for reading
			file, err := os.Open(path)
			if err != nil {
				fmt.Printf("❌ Error opening %s: %v\n", path, err)
				errorCount++
				return nil
			}
			defer file.Close()

			// Upload
			err = ul.Upload(ctx, key, file, contentType)
			if err != nil {
				fmt.Printf("❌ Error uploading %s: %v\n", key, err)
				errorCount++
				return nil
			}

			fmt.Printf("✅ %s\n", key)
			uploadedCount++

			return nil
		})

		if err != nil {
			fmt.Printf("Error walking directory: %v\n", err)
			os.Exit(1)
		}

		// Print summary
		fmt.Printf("\n─────────────────────────────────\n")
		fmt.Printf("Upload complete!\n")
		if uploadDryRun {
			fmt.Printf("  Would upload: %d files\n", uploadedCount)
		} else {
			fmt.Printf("  Uploaded: %d files\n", uploadedCount)
			fmt.Printf("  Skipped: %d files\n", skippedCount)
		}
		if errorCount > 0 {
			fmt.Printf("  Errors: %d\n", errorCount)
		}
		fmt.Printf("─────────────────────────────────\n")

		if errorCount > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	imagesCmd.AddCommand(imagesUploadCmd)

	imagesUploadCmd.Flags().StringVarP(&uploadInputDir, "input", "i", "", "Input directory containing processed images (required)")
	imagesUploadCmd.Flags().StringVarP(&uploadBucket, "bucket", "b", "", "S3 bucket name (required)")
	imagesUploadCmd.Flags().StringVarP(&uploadRegion, "region", "r", "", "S3 region (e.g., 'us-east-1', 'auto' for R2) (required)")
	imagesUploadCmd.Flags().StringVar(&uploadEndpoint, "endpoint", "", "Custom S3 endpoint URL (for R2: https://account-id.r2.cloudflarestorage.com)")
	imagesUploadCmd.Flags().StringVar(&uploadBaseURL, "base-url", "", "Public base URL for accessing files (e.g., https://images.example.com)")
	imagesUploadCmd.Flags().StringVar(&uploadPrefix, "prefix", "images/", "Prefix to prepend to all keys (e.g., 'images/')")
	imagesUploadCmd.Flags().BoolVar(&uploadForce, "force", false, "Force upload even if files already exist")
	imagesUploadCmd.Flags().BoolVar(&uploadDryRun, "dry-run", false, "Simulate upload without actually uploading files")
	imagesUploadCmd.Flags().BoolVar(&uploadSkipThumbs, "skip-thumbs", true, "Skip files in '.thumbs' directories")

	imagesUploadCmd.MarkFlagRequired("input")
	imagesUploadCmd.MarkFlagRequired("bucket")
	imagesUploadCmd.MarkFlagRequired("region")
}
