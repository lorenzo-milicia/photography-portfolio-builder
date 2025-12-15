package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Position struct {
	TopLeftX     int `yaml:"top_left_x"`
	TopLeftY     int `yaml:"top_left_y"`
	BottomRightX int `yaml:"bottom_right_x"`
	BottomRightY int `yaml:"bottom_right_y"`
}

type PhotoPlacement struct {
	Filename string   `yaml:"filename"`
	Position Position `yaml:"position"`
}

type LayoutConfig struct {
	GridWidth        int              `yaml:"grid_width"`
	Placements       []PhotoPlacement `yaml:"placements"`
	MobileGridWidth  int              `yaml:"mobile_grid_width"`
	MobilePlacements []PhotoPlacement `yaml:"mobile_placements"`
}

func computeHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run migrate_layout_filenames.go <content-dir>")
		fmt.Println("Example: go run migrate_layout_filenames.go content")
		os.Exit(1)
	}

	contentDir := os.Args[1]
	projectsDir := filepath.Join(contentDir, "projects")

	// Map from original filename to hash (12 chars)
	hashMap := make(map[string]string)

	// Walk projects directory
	projects, err := os.ReadDir(projectsDir)
	if err != nil {
		fmt.Printf("Error reading projects directory: %v\n", err)
		os.Exit(1)
	}

	for _, project := range projects {
		if !project.IsDir() {
			continue
		}

		projectName := project.Name()
		projectPath := filepath.Join(projectsDir, projectName)
		layoutPath := filepath.Join(projectPath, "layout.yaml")
		photosPath := filepath.Join("photos", projectName)

		// Check if layout.yaml exists
		if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
			fmt.Printf("Skipping %s: no layout.yaml\n", projectName)
			continue
		}

		fmt.Printf("\nProcessing project: %s\n", projectName)

		// Read layout.yaml
		data, err := os.ReadFile(layoutPath)
		if err != nil {
			fmt.Printf("Error reading layout.yaml: %v\n", err)
			continue
		}

		var layout LayoutConfig
		if err := yaml.Unmarshal(data, &layout); err != nil {
			fmt.Printf("Error parsing layout.yaml: %v\n", err)
			continue
		}

		// Collect all unique filenames
		uniqueFilenames := make(map[string]bool)
		for _, p := range layout.Placements {
			uniqueFilenames[p.Filename] = true
		}
		for _, p := range layout.MobilePlacements {
			uniqueFilenames[p.Filename] = true
		}

		// Compute hashes for each filename
		for filename := range uniqueFilenames {
			photoPath := filepath.Join(photosPath, filename)
			hash, err := computeHash(photoPath)
			if err != nil {
				fmt.Printf("  Warning: couldn't hash %s: %v\n", filename, err)
				continue
			}
			hashMap[filename] = hash[:12]
			fmt.Printf("  %s -> %s\n", filename, hash[:12])
		}

		// Update layout with new filenames
		for i := range layout.Placements {
			oldName := layout.Placements[i].Filename
			if newName, ok := hashMap[oldName]; ok {
				layout.Placements[i].Filename = newName
			}
		}
		for i := range layout.MobilePlacements {
			oldName := layout.MobilePlacements[i].Filename
			if newName, ok := hashMap[oldName]; ok {
				layout.MobilePlacements[i].Filename = newName
			}
		}

		// Write updated layout.yaml
		outData, err := yaml.Marshal(&layout)
		if err != nil {
			fmt.Printf("Error marshaling layout: %v\n", err)
			continue
		}

		// Backup original
		backupPath := layoutPath + ".bak"
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			fmt.Printf("Error creating backup: %v\n", err)
			continue
		}

		// Write new layout
		if err := os.WriteFile(layoutPath, outData, 0644); err != nil {
			fmt.Printf("Error writing layout.yaml: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ Updated layout.yaml (backup: %s)\n", backupPath)

		// Clear hashMap for next project
		for k := range hashMap {
			delete(hashMap, k)
		}
	}

	fmt.Println("\nMigration complete!")
}
