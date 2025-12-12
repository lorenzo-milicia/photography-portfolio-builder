# Quick Start Guide

Get your photography portfolio builder up and running in minutes!

## 1. Build the Application

```bash
make build
```

Or manually:
```bash
go build -o bin/builder ./cmd/builder
```

## 2. Start the Server

```bash
make run
```

Or manually:
```bash
./bin/builder
```

The server will start on `http://localhost:8080`

## 3. Create Your First Project

1. Open your browser to `http://localhost:8080`
2. Click "**+ New Project**" in the sidebar
3. Enter a project name (e.g., "My First Gallery")
4. Add a description
5. Click "**Create Project**"

## 4. Upload Photos

1. Select your project from the sidebar
2. In the Photos section, click "**Choose File**"
3. Select one or more photos
4. Click "**Upload Photo**"
5. Repeat for all photos you want to add

## 5. Configure Layout

In the Layout Settings section:

- **Justified Layout**: Best for varying photo sizes
  - Set row height (e.g., 300px)
  - Set gap between images (e.g., 10px)

- **Grid Layout**: Best for uniform presentation
  - Set number of columns (e.g., 3)
  - Set gap between images (e.g., 15px)

- **Manual Layout**: For custom positioning

Click "**Update Layout**" to save changes.

## 6. Generate Your Site

1. Click "**🚀 Generate Site**" in the sidebar
2. Wait for the success message
3. Click "**View Preview**" to see your portfolio

## 7. Find Your Generated Site

Your static website is now in:
```
output/public/
```

Deploy these files to any static hosting service!

## Tips

- **Preview Changes**: Generate the site after making changes to see updates
- **Multiple Projects**: Create multiple projects for different photo collections
- **File Organization**: Photos are organized by project in `content/photos/<slug>/`
- **Custom Port**: Run on a different port with `./bin/builder -port 3000`

## Troubleshooting

**Port already in use?**
```bash
./bin/builder -port 3001
```

**Need to start fresh?**
```bash
make clean-all  # Removes generated content
make build      # Rebuild
make run        # Start fresh
```

**Photos not showing?**
- Check file format (JPG, PNG, WebP supported)
- Ensure file size is reasonable (< 10MB recommended)
- Check browser console for errors

---

Enjoy building your photography portfolio! 📷
