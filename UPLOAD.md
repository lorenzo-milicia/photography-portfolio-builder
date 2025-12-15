# Images Upload to Remote Storage

This guide explains how to upload processed images to remote storage (Cloudflare R2, AWS S3, etc.).

## Overview

The `images upload` command uploads processed images from the local `dist/images` directory to S3-compatible remote storage. It includes:

- **Smart caching**: Checks if files already exist remotely before uploading (saves bandwidth & time)
- **Force upload**: Use `--force` to overwrite existing files
- **Dry-run mode**: Test uploads without actually transferring files
- **Environment-based credentials**: Secure credential management via environment variables

## Quick Start

### 1. Set up credentials

For **Cloudflare R2**:
```bash
export R2_ACCESS_KEY_ID="your-r2-access-key"
export R2_SECRET_ACCESS_KEY="your-r2-secret-key"
```

For **AWS S3**:
```bash
export AWS_ACCESS_KEY_ID="your-aws-access-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret-key"
```

### 2. Process images locally

```bash
./bin/builder images process -i photos -o dist/images
```

### 3. Upload to remote storage

**Dry-run first (recommended):**
```bash
./bin/builder images upload \
  -i dist/images \
  -b my-bucket-name \
  -r auto \
  --endpoint https://account-id.r2.cloudflarestorage.com \
  --base-url https://images.example.com \
  --dry-run
```

**Actual upload:**
```bash
./bin/builder images upload \
  -i dist/images \
  -b my-bucket-name \
  -r auto \
  --endpoint https://account-id.r2.cloudflarestorage.com \
  --base-url https://images.example.com
```

## Command Flags

| Flag | Required | Description |
|------|----------|-------------|
| `-i, --input` | Yes | Input directory containing processed images |
| `-b, --bucket` | Yes | S3 bucket name |
| `-r, --region` | Yes | S3 region (use `auto` for R2) |
| `--endpoint` | No | Custom S3 endpoint (required for R2) |
| `--base-url` | No | Public base URL for accessing files |
| `--prefix` | No | Prefix to prepend to all keys (e.g., `images/`) |
| `--force` | No | Force upload even if files already exist |
| `--dry-run` | No | Simulate upload without actually uploading |

## Examples

### Cloudflare R2 with Custom Domain

```bash
export R2_ACCESS_KEY_ID="abc123..."
export R2_SECRET_ACCESS_KEY="xyz789..."

./bin/builder images upload \
  -i dist/images \
  -b my-portfolio-images \
  -r auto \
  --endpoint https://1234567890abcdef.r2.cloudflarestorage.com \
  --base-url https://images.myportfolio.com
```

### AWS S3 (US East)

```bash
export AWS_ACCESS_KEY_ID="AKIA..."
export AWS_SECRET_ACCESS_KEY="..."

./bin/builder images upload \
  -i dist/images \
  -b my-portfolio-bucket \
  -r us-east-1
```

### Force Re-upload All Files

```bash
./bin/builder images upload \
  -i dist/images \
  -b my-bucket \
  -r auto \
  --endpoint https://account.r2.cloudflarestorage.com \
  --force
```

## VS Code Tasks

Two VS Code tasks are available:

1. **Images: Upload (dry-run)** — Preview what will be uploaded
2. **Images: Upload to R2** — Perform actual upload

Run via: `Tasks` → `Run Task` → Select task

You'll be prompted for:
- Bucket name
- R2 endpoint URL
- Public base URL

## Workflow

Typical workflow for deploying a portfolio:

```bash
# 1. Process images locally
./bin/builder images process -i photos -o dist/images

# 2. Upload images to R2
export R2_ACCESS_KEY_ID="..."
export R2_SECRET_ACCESS_KEY="..."
./bin/builder images upload -i dist/images -b my-bucket -r auto \
  --endpoint https://....r2.cloudflarestorage.com \
  --base-url https://images.example.com

# 3. Generate website with remote image URLs
./bin/builder website build -o dist \
  --host https://images.example.com

# 4. Deploy dist/ to hosting (Cloudflare Pages, Netlify, etc.)
```

## Architecture

The upload system uses a clean interface pattern:

```
internal/uploader/
├── uploader.go     # Interface definition
└── s3.go           # S3-compatible implementation
```

This design allows for easy addition of other storage providers (Google Cloud Storage, Azure Blob, etc.) in the future.

## Troubleshooting

### "missing credentials" error

Ensure environment variables are set:
```bash
echo $R2_ACCESS_KEY_ID
echo $R2_SECRET_ACCESS_KEY
```

### Files not accessible after upload

- Check that `--base-url` matches your R2 custom domain or public bucket URL
- Verify bucket permissions allow public read access
- For R2: Configure a custom domain in the Cloudflare dashboard

### Slow uploads

- The first upload will transfer all files
- Subsequent uploads skip existing files (unless `--force` is used)
- Use `--dry-run` to preview what will be uploaded

### Large file counts

The upload walks the entire `dist/images` directory recursively. For very large portfolios:
- Consider using `--prefix` to upload specific projects
- Monitor upload progress in the terminal

## Security Notes

- **Never commit credentials** to version control
- Use environment variables or secure secret management
- For CI/CD, use secrets management (GitHub Actions secrets, etc.)
- R2/S3 credentials should have minimal required permissions (PutObject, HeadObject, DeleteObject on the bucket only)
